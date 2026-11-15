# Modelo de dados (SQLite)

Status: **Recomendado R1/R2, aprovado com o plano em 13/11/2026; estrutura de estudos e coletas pela A28 (15/11/2026).** O DDL definitivo é escrito nas Fases 1 e 2; mudanças exigem ADR e migração (ver `AGENTS.md`).

## Convenções
- PK `uuid TEXT` (UUID v4, minúsculas, com hífens). IDs locais sequenciais não são usados como identidade.
- Timestamps: `INTEGER` com epoch em **ms UTC** no banco; **ISO 8601 UTC** (`2026-11-13T19:40:38.123Z`) no JSON; exibição em horário local (ADR-0010).
- Tabelas `STRICT`, `PRAGMA foreign_keys=ON`, `CHECK` para todas as regras de domínio.
- Exclusão é **tombstone** `excluido_em INTEGER NULL`. Não há DELETE físico de entidade de negócio.
- `created_at`/`updated_at` vêm do relógio do aparelho de origem e são **preservados**. A Central grava `recebido_em` à parte.
- Toda tabela operacional mutável tem `created_at` e `updated_at`. `catalogo_itens` é versionado por *seed* (tabela de referência imutável).

## Tabelas de negócio (contrato compartilhado)

> **A28 (15/11/2026, ADR-0023/0024/0025):** a operação e a equipe configurada no celular foram substituídas por **estudo → jornada → equipe** (na Central) e **coleta** (no celular). Os bancos de teste foram reiniciados na geração 2.

### estudos (Central)
| coluna | tipo | regra |
|---|---|---|
| uuid | TEXT PK | |
| codigo | TEXT NOT NULL UNIQUE | `SITPOSSE7` + 8 caracteres (RN-02), guardado sem hífens |
| nome | TEXT NOT NULL | não vazio; não precisa ser único |
| created_at, updated_at | INTEGER NOT NULL | |
| excluido_em | INTEGER NULL | reservado (estudos não são excluídos nesta fase) |

### jornadas (Central)
| coluna | tipo | regra |
|---|---|---|
| uuid | TEXT PK | |
| estudo_uuid | TEXT NOT NULL FK | |
| numero | INTEGER NOT NULL | sequencial no estudo; `UNIQUE(estudo_uuid, numero)` |
| data | TEXT NOT NULL | data civil `AAAA-MM-DD` (não é instante nem chave) |
| created_at, updated_at | INTEGER NOT NULL | |
| excluido_em | INTEGER NULL | só sem coletas vinculadas |

### equipes (Central)
| coluna | tipo | regra |
|---|---|---|
| uuid | TEXT PK | |
| jornada_uuid | TEXT NOT NULL FK | |
| nome | TEXT NOT NULL | livre, não vazio |
| nome_normalizado | TEXT NOT NULL | RN-35; `UNIQUE(jornada_uuid, nome_normalizado) WHERE excluido_em IS NULL` |
| codigo | TEXT NOT NULL UNIQUE | `EQ` + 8 caracteres (RN-02), único em toda a Central |
| created_at, updated_at | INTEGER NOT NULL | |
| excluido_em | INTEGER NULL | só sem coletas vinculadas |

### coletas
| coluna | Android | Central |
|---|---|---|
| uuid | TEXT PK (gerado no celular) | TEXT PK (o mesmo) |
| codigo | `codigo_equipe` (atual, normalizado) e `codigo_original` | `codigo_informado` (o último recebido) e `codigo_original` |
| status | `em_andamento` \| `concluida`; `iniciada_em`, `concluida_em` | — |
| destino | `destino_confirmado_em`, `destino_json` (vindos do recibo) | `equipe_uuid` NULL até a 1ª confirmação; depois **imutável** (gatilho); `destino_confirmado_em` |
| instalação | — | `instalacao_uuid` FK |
| created_at, updated_at, rev | INTEGER NOT NULL (+ `rev_confirmada` no Android) | `created_at`, `updated_at`, `rev_origem` |

`coleta_codigos(uuid, coleta_uuid, codigo_anterior, codigo_novo, alterado_em)` no Android: histórico das correções do código (somente acréscimo).

### instalacoes
`uuid` (gerado no celular, em `meta`); na Central, `dispositivos(instalacao_uuid PK, credencial_hash, rotulo_aparelho, pareado_em, revogado_em)`. Sem ID de hardware (RN-03).

### propriedades
| coluna | tipo | regra |
|---|---|---|
| uuid | TEXT PK | |
| coleta_uuid | TEXT NOT NULL FK | a coleta de origem (RN-05) |
| equipe_uuid | TEXT NOT NULL FK | só na Central: a equipe de destino confirmada |
| nome_propriedade | TEXT NOT NULL | `CHECK(length(trim(nome_propriedade)) > 0)` |
| responsavel_familia | TEXT NOT NULL | não vazio (RN-11, A28) |
| observacoes_relevantes | TEXT NULL | até 4000 caracteres (RN-15) |
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
- **Banco v3 (A28, ADR-0025):** `coletas`, `coleta_codigos`, `propriedades` (com `coleta_uuid` e `observacoes_relevantes`), `itens_propriedade`, `fotografias`, `rascunhos` (+ `coleta_uuid`), `lotes_envio` (+ `coleta_uuid`; um aberto por coleta), `lote_itens`, `pareamento` (sem operação). A tabela `equipe` deixou de existir.
- **MVP (banco v2, ADR-0018), substituído pelo v3:** `equipe`, `propriedades`, `itens_propriedade`, `fotografias`, `rascunhos`, `lotes_envio` (+ `payload_json` congelado), `lote_itens` (+ `resultado`, `motivo`), `pareamento`; UUID da instalação em `meta`. Sem `CHECK` nem índice parcial (limite do Room): regras no domínio e no repositório. Catálogo no código (`Catalogo`), conferido com a fixture.
- Em cada entidade sincronizável: `rev INTEGER NOT NULL` (incrementado **na mesma transação** de toda criação, edição ou exclusão) e `rev_confirmada INTEGER NOT NULL DEFAULT 0`. Pendente: `rev > rev_confirmada`.
- `lotes_envio(uuid, coleta_uuid, status [preparado|enviando|aguardando_confirmacao|confirmado|cancelado|erro_recuperavel|erro_fatal], canal [wifi|usb], payload_json, criado_em, atualizado_em, recibo_json NULL, erro NULL)`.
- `lote_itens(lote_uuid, entidade, entidade_uuid, rev, PRIMARY KEY(lote_uuid, entidade, entidade_uuid))`.
- `rascunhos(uuid, propriedade_uuid NULL, coleta_uuid, json, atualizado_em)`: nunca sincronizado nem contado.
- `pareamento(central_id, spki_sha256, ips_json, porta, credencial, pareado_em)`.

### Central
- Em cada entidade: `rev_origem` (última rev do Android aplicada), `rev_central` (contador de correções), `corrigido_desde_rev_origem INTEGER` (0/1), `recebido_em`, `origem_instalacao`.
- `lotes_recebidos(uuid PK, instalacao_uuid, coleta_uuid, codigo_informado, canal, status [recebendo|aguardando_confirmacao|importado|cancelado], equipe_destino_uuid, decidido_em, motivo_cancelamento, manifesto_json, manifesto_sha256, recibo_json, recebido_em, importado_em, ack_em)`.
- `decisoes_destino(id, lote_uuid, coleta_uuid, acao [confirmar|alterar|cancelar], codigo_informado, codigo_confirmado, equipe_uuid, jornada_uuid, estudo_uuid, quando, operador)`, somente acréscimo.
- `aplicacoes(lote_uuid, entidade, entidade_uuid, rev, resultado [aplicado|duplicado|conflito|rejeitado], motivo)`.
- `conflitos(uuid, cadastro_uuid, lote_uuid, base_json, snapshot_android_json, snapshot_central_json, status [aberto|resolvido], decisao_json, resolvido_por, resolvido_em, created_at)`.
- `auditoria(id, quando, operador, origem [central|coleta:<uuid>], entidade, entidade_uuid, acao, antes_json, depois_json)`, somente acréscimo (*append-only*).
- `dispositivos(instalacao_uuid PK, credencial_hash, rotulo_aparelho, pareado_em, revogado_em)`.
- `tokens_pareamento(token_hash, criado_em, expira_em, usado_em)`.
- `aliases_itens(nome_normalizado PK, grupo_canonico, created_at, updated_at)`.
- `backups(uuid, caminho, criado_em, verificado_em NULL, cobre_ate_recebido_em, manifesto_sha256)`.
- **Geração 2 (A28, migração 0002 reescrita, ADR-0025):** `estudos`, `jornadas`, `equipes`, `coletas`, `decisoes_destino`, `lotes_recebidos` com os estados novos, `meta.esquema_geracao = '2'`; o resto como no MVP, sem `operacoes`.
- **Migração 0003 (A30, ADR-0027, RN-96):** `numeracao_propriedades(propriedade_uuid PK, estudo_uuid, numero, atribuido_em)`, `UNIQUE(estudo_uuid, numero)`, sem UPDATE/DELETE (gatilhos). O número é atribuído na primeira aplicação da propriedade, na mesma transação. A migração numera os cadastros já existentes (inclusive excluídos) por `recebido_em, uuid` e recria só a visão `v_propriedades_vigentes`, que passa a expor `numero_no_estudo`.
- **MVP (migração 0002 original, ADR-0017), substituído:** `operacoes` (no máximo uma `ativa`), `equipes`, `tokens_pareamento`, `dispositivos`, `propriedades`, `catalogo_itens` (semeado; igual a `shared/test-fixtures/catalogo/itens.v1.json`), `itens_propriedade`, `fotografias`, `lotes_recebidos` (+ `manifesto_sha256`, `ack_em`), `fotos_esperadas`, `aplicacoes`, `auditoria` (somente acréscimo, por gatilhos) e as visões vigentes. Cada entidade recebida tem `registro_json` (último payload aplicado, usado como "antes" na auditoria). `conflitos`, `aliases_itens` e `backups` ficam para as fases deles.

## Índices e visões
- `propriedades(equipe_uuid) WHERE excluido_em IS NULL` e `propriedades(coleta_uuid)`
- `jornadas(estudo_uuid, data)`, `equipes(jornada_uuid)`
- `itens_propriedade(propriedade_uuid, categoria, codigo_catalogo) WHERE excluido_em IS NULL`
- `itens_propriedade(nome_normalizado) WHERE excluido_em IS NULL`
- `fotografias(propriedade_uuid) WHERE excluido_em IS NULL`
- Visões `v_propriedades_vigentes`, `v_itens_vigentes` e `v_fotos_vigentes` (filtram tombstones; na Central, `v_propriedades_vigentes` traz também equipe, jornada e estudo). São a **única** base das estatísticas (RN-80).

## Migrações
- **Android:** `exportSchema` com `room3 { schemaDirectory(...) }` e pasta `schemas/` versionada; `Migration`/`AutoMigration` testadas com `MigrationTestHelper`; `fallbackToDestructiveMigration` **proibido**.
- **Central:** arquivos `NNNN_descricao.sql` embutidos (`embed`), aplicados em transação, com `PRAGMA user_version` e **backup automático antes de migrar**.
- **Contratos:** `schema_versao` e `protocolo_versao` em todo payload; a Central aceita uma faixa de versões e rejeita as demais com mensagem clara.
