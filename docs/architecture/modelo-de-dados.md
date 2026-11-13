# Modelo de dados (SQLite)

Status: **Recomendado R1/R2, aprovado com o plano em 13/11/2026.** O DDL definitivo é escrito nas Fases 1 e 2; mudanças exigem ADR e migração (ver `AGENTS.md`).

## Convenções
- PK `uuid TEXT` (UUID v4, minúsculas, com hífens). IDs locais sequenciais não são usados como identidade.
- Timestamps: `INTEGER` com epoch em **ms UTC** no banco; **ISO 8601 UTC** (`2026-11-13T19:40:38.123Z`) no JSON; exibição em horário local (ADR-0010).
- Tabelas `STRICT`, `PRAGMA foreign_keys=ON`, `CHECK` para todas as regras de domínio.
- Exclusão é **tombstone** `excluido_em INTEGER NULL`. Não há DELETE físico de entidade de negócio.
- `created_at`/`updated_at` vêm do relógio do aparelho de origem e são **preservados**. A Central grava `recebido_em` à parte.
- Toda tabela operacional mutável tem `created_at` e `updated_at`. `catalogo_itens` é versionado por *seed* (tabela de referência imutável).

## Tabelas de negócio (contrato compartilhado)

### operacoes
| coluna | tipo | regra |
|---|---|---|
| uuid | TEXT PK | |
| nome | TEXT NOT NULL | não vazio |
| descricao | TEXT NULL | |
| status | TEXT NOT NULL | `ativa` \| `encerrada` |
| created_at, updated_at | INTEGER NOT NULL | |

### equipes
| coluna | tipo | regra |
|---|---|---|
| uuid | TEXT PK | RN-02 |
| operacao_uuid | TEXT NULL FK | null no Android até o 1º pareamento |
| nome | TEXT NOT NULL | não vazio (RN-01) |
| responsavel | TEXT NOT NULL | não vazio |
| identificacao_area | TEXT NULL | |
| created_at, updated_at, rev | INTEGER NOT NULL | |

### instalacoes
`uuid` PK, `equipe_uuid` FK, `created_at`, `rotulo_aparelho TEXT NULL` (livre, opcional). Sem ID de hardware (RN-03).

### propriedades
| coluna | tipo | regra |
|---|---|---|
| uuid | TEXT PK | |
| equipe_uuid | TEXT NOT NULL FK | |
| nome_propriedade | TEXT NOT NULL | `CHECK(length(trim(nome_propriedade)) > 0)` |
| responsavel_familia | TEXT NULL | |
| pessoas_adultos, pessoas_jovens, pessoas_criancas | INTEGER NOT NULL DEFAULT 0 | `CHECK(>= 0)` |
| veic_caminhoes, veic_automoveis_utilitarios, veic_motos_ciclomotores | INTEGER NOT NULL DEFAULT 0 | `CHECK(>= 0)` |
| bov_adultos, bov_bezerros | INTEGER NOT NULL DEFAULT 0 | `CHECK(>= 0)` |
| equ_adultos, equ_potros | INTEGER NOT NULL DEFAULT 0 | `CHECK(>= 0)` |
| area_valor_milesimos | INTEGER NULL | `CHECK(>= 0)` (RN-44) |
| area_unidade | TEXT NULL | `ha` \| `alq_paulista`; `CHECK((area_valor_milesimos IS NULL) = (area_unidade IS NULL))` |
| lat, lon | REAL NULL | faixas RN-51; `CHECK((lat IS NULL) = (lon IS NULL))` |
| gps_origem | TEXT NULL | `gnss` \| `manual`; não nulo se houver lat |
| gps_precisao_m | REAL NULL | `CHECK(>= 0)`; nulo se manual |
| gps_capturado_em | INTEGER NULL | |
| gps_satelites | INTEGER NULL | |
| created_at, updated_at, rev | INTEGER NOT NULL | |
| excluido_em | INTEGER NULL | tombstone |

### catalogo_itens (referência)
`codigo TEXT PK` (estável, p. ex. `eletro.geladeira`, `eletro.outros`, `movel.sofa`, `ave.galinha`), `categoria` (`eletrodomestico`|`movel`|`ave`), `nome_exibicao`, `ordem`, `ativo`, `versao_catalogo`. Itens novos ganham código novo; códigos nunca são reutilizados.

### itens_propriedade
| coluna | tipo | regra |
|---|---|---|
| uuid | TEXT PK | |
| propriedade_uuid | TEXT NOT NULL FK | |
| categoria | TEXT NOT NULL | igual à categoria do código |
| codigo_catalogo | TEXT NOT NULL FK | |
| nome_personalizado | TEXT NULL | obrigatório se o código terminar em `.outros`; texto original |
| nome_normalizado | TEXT NULL | RN-35 |
| quantidade | INTEGER NOT NULL | `CHECK(>= 0)` (UI exige ≥ 1, RN-34) |
| created_at, updated_at, rev | INTEGER NOT NULL | |
| excluido_em | INTEGER NULL | |

Índice único parcial (RN-32): `UNIQUE(propriedade_uuid, codigo_catalogo) WHERE excluido_em IS NULL AND codigo_catalogo NOT LIKE '%.outros'`.

### fotografias
`uuid` PK, `propriedade_uuid` FK, `caminho_relativo`, `mime`, `tamanho_bytes`, `sha256` (hex, 64), `largura`, `altura`, `capturada_em`, `created_at`, `updated_at`, `rev`, `excluido_em`. Só no Android: `arquivo_presente INTEGER NOT NULL DEFAULT 1` (liberar espaço, P3).

## Controle de sincronização

### Android
- Em cada entidade sincronizável: `rev INTEGER NOT NULL` (incrementado **na mesma transação** de toda criação, edição ou exclusão) e `rev_confirmada INTEGER NOT NULL DEFAULT 0`. Pendente: `rev > rev_confirmada`.
- `lotes_envio(uuid, status, canal [wifi|usb], criado_em, atualizado_em, recibo_json NULL, erro NULL)`.
- `lote_itens(lote_uuid, entidade, entidade_uuid, rev, PRIMARY KEY(lote_uuid, entidade, entidade_uuid))`.
- `rascunhos(uuid, propriedade_uuid NULL, json, atualizado_em)`: nunca sincronizado nem contado.
- `pareamento(central_id, spki_sha256, ips_json, porta, credencial, operacao_uuid, operacao_nome, pareado_em)`.

### Central
- Em cada entidade: `rev_origem` (última rev do Android aplicada), `rev_central` (contador de correções), `corrigido_desde_rev_origem INTEGER` (0/1), `recebido_em`, `origem_instalacao`.
- `lotes_recebidos(uuid PK, instalacao_uuid, equipe_uuid, canal, status, manifesto_json, recibo_json, recebido_em, importado_em)`.
- `aplicacoes(lote_uuid, entidade, entidade_uuid, rev, resultado [aplicado|duplicado|conflito|rejeitado], motivo)`.
- `conflitos(uuid, cadastro_uuid, lote_uuid, base_json, snapshot_android_json, snapshot_central_json, status [aberto|resolvido], decisao_json, resolvido_por, resolvido_em, created_at)`.
- `auditoria(id, quando, operador, origem [central|equipe:<uuid>], entidade, entidade_uuid, acao, antes_json, depois_json)`, somente acréscimo (*append-only*).
- `dispositivos(instalacao_uuid PK, equipe_uuid, credencial_hash, pareado_em, revogado_em)`.
- `tokens_pareamento(token_hash, operacao_uuid, expira_em, usado_em)`.
- `aliases_itens(nome_normalizado PK, grupo_canonico, created_at, updated_at)`.
- `backups(uuid, caminho, criado_em, verificado_em NULL, cobre_ate_recebido_em, manifesto_sha256)`.

## Índices e visões
- `propriedades(equipe_uuid) WHERE excluido_em IS NULL`
- `itens_propriedade(propriedade_uuid, categoria, codigo_catalogo) WHERE excluido_em IS NULL`
- `itens_propriedade(nome_normalizado) WHERE excluido_em IS NULL`
- `fotografias(propriedade_uuid) WHERE excluido_em IS NULL`
- Visões `v_propriedades_vigentes`, `v_itens_vigentes` e `v_fotos_vigentes` (filtram tombstones e, na Central, cadastros de operações encerradas quando o filtro pedir). São a **única** base das estatísticas (RN-80).

## Migrações
- **Android:** `exportSchema` com `room3 { schemaDirectory(...) }` e pasta `schemas/` versionada; `Migration`/`AutoMigration` testadas com `MigrationTestHelper`; `fallbackToDestructiveMigration` **proibido**.
- **Central:** arquivos `NNNN_descricao.sql` embutidos (`embed`), aplicados em transação, com `PRAGMA user_version` e **backup automático antes de migrar**.
- **Contratos:** `schema_versao` e `protocolo_versao` em todo payload; a Central aceita uma faixa de versões e rejeita as demais com mensagem clara.
