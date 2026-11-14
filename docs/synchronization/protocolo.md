# Protocolo de sincronização — v1

Status: **Aprovado com o plano (13/11/2026).** Mudanças exigem ADR, incremento de `protocolo_versao` e fixtures.
Regras de negócio: RN-70 a RN-74. Integridade: [integridade.md](integridade.md). Conflitos: [conflitos.md](conflitos.md).

## 1. Transporte
- HTTPS na rede Wi‑Fi local, sem internet, por IP: TLS 1.3 preferido; TLS 1.2 só com ECDHE + AEAD, para Android 8/9 (ADR-0014). Pinning do SPKI vindo do QR (ADR-0007).
- Porta padrão **38807** (fora das faixas reservadas pelo Windows/Hyper-V, ≥ 49152, e pela rede espelhada do WSL, 44620–48715). O Android não fixa a porta: ela vem no QR.
- JSON UTF‑8 (gzip opcional). Fotos em binário (`application/octet-stream`).
- Autenticação: `Authorization: Bearer <credencial_dispositivo>`, exceto em `/v1/pareamento`, que usa o token do QR.
- Todo payload JSON leva `protocolo_versao` e `schema_versao`.
- No Android a sincronização roda na `Network` Wi‑Fi específica (ver `mobile/docs/arquitetura.md`) e exige `ACCESS_LOCAL_NETWORK` quando `targetSdk ≥ 37`.

## 2. Pareamento
```
Operador (Central)          Central                         Android
      │ "Parear aparelho"     │                                 │
      │──────────────────────►│ gera token (10 min, uso único)  │
      │                       │ exibe QR {v, central_id, ips[], │
      │                       │  porta, spki_sha256, token,     │
      │                       │  operacao_uuid, operacao_nome}  │
      │                       │◄──────── lê QR ─────────────────│
      │                       │◄─ TLS (pin SPKI) POST /v1/pareamento {token, instalacao, equipe}
      │◄── "Equipe X quer parear. Confirmar?" (ou "Restaurar equipe existente")
      │── confirma ──────────►│ vincula equipe↔operação; grava dispositivo (hash da credencial)
      │                       │── 200 {credencial_dispositivo, operacao} ──────────────►│ grava pareamento
```
- Token expirado ou já usado → `410` com mensagem "Gere um novo QR na Central".
- Equipe já vinculada a **outra** operação → `409`: o operador decide (P6).

## 3. Endpoints
| Método e rota | Função |
|---|---|
| `GET /v1/status` | versões aceitas, `central_id`, operação, lotes conhecidos desta instalação, cobertura de backup (P4) |
| `POST /v1/pareamento` | pareamento (seção 2) |
| `PUT /v1/lotes/{lote}` | manifesto + registros (idempotente) |
| `GET /v1/lotes/{lote}` | estado do lote; recibo, se já importado |
| `PUT /v1/fotos/{foto}?offset=N` | bloco de até 1 MiB da foto (append em staging) |
| `GET /v1/fotos/{foto}` | bytes já recebidos (para retomar) ou `existe=true` |
| `POST /v1/lotes/{lote}/importar` | valida, publica e importa; devolve o recibo |
| `POST /v1/lotes/{lote}/ack` | o Android informa que gravou o recibo (informativo) |
| `POST /v1/restauracao` | restauração de equipe autorizada pelo operador (seção 7) |

## 4. Máquina de estados do lote (Android)
```
           ┌──────────────┐
           │  PREPARADO   │  (lote_uuid + lista (entidade, uuid, rev) congelada em transação)
           └──────┬───────┘
                  ▼
     ┌──────────────────────┐   falha de rede    ┌──────────────────┐
     │ ENVIANDO_REGISTROS   │──────────────────► │ ERRO_RECUPERAVEL │──┐
     └──────────┬───────────┘                    └──────────────────┘  │ retomar:
                ▼                                          ▲           │ GET /v1/lotes/{id}
     ┌──────────────────────┐                              │           │
     │   ENVIANDO_FOTOS     │──────────────────────────────┘           │
     └──────────┬───────────┘                                          │
                ▼                                                      │
     ┌──────────────────────┐                                          │
     │AGUARDANDO_IMPORTACAO │──────────────────────────────────────────┘
     └──────────┬───────────┘
                ▼  recibo gravado em transação
     ┌──────────────────────┐
     │     CONFIRMADO       │
     └──────────────────────┘
   ERRO_FATAL: versão incompatível, pareamento revogado, cert. diferente → exige ação do usuário; dados intactos.
   Canal USB: PREPARADO → EXPORTADO_AGUARDANDO_CONFIRMACAO → CONFIRMADO (ao ler o recibo).
```

## 5. Sequência de envio
```
Android                                   Central
  │ GET /v1/status ─────────────────────────►│ versões / operação
  │ PUT /v1/lotes/L {manifesto, registros} ─►│ grava lote "recebendo" (idempotente)
  │◄─ {fotos_faltantes:[{uuid, offset}]} ────│
  │ PUT /v1/fotos/F?offset=0 (1 MiB) ───────►│ append + fsync em staging/F.part
  │ ... (retoma por GET /v1/fotos/F) ...     │
  │ POST /v1/lotes/L/importar ──────────────►│ 1. valida tamanho + SHA‑256
  │                                          │ 2. publica arquivos (rename + fsync dir)
  │                                          │ 3. transação: aplica registros, aplicacoes,
  │                                          │    auditoria, conflitos, lote=importado + recibo
  │◄─ 200 {recibo} ──────────────────────────│ 4. responde
  │ transação: grava recibo; rev_confirmada  │
  │ POST /v1/lotes/L/ack ───────────────────►│ registra ack
```
- **Resposta perdida** (queda após o passo 3): o Android consulta `GET /v1/lotes/L` e recebe **o mesmo recibo**.
- **Reenvio** de um lote já importado: devolve o recibo existente, sem reaplicar.
- **Hash divergente**: a Central apaga o `.part` e marca a foto como `reenviar`; o lote não é importado até todas as fotos baterem.
- **Edição durante o envio**: gera `rev` maior, que continua pendente para o próximo lote.

## 6. Recibo
```json
{
  "protocolo_versao": 1,
  "lote_uuid": "…",
  "central_id": "…",
  "importado_em": "2026-11-13T19:40:38.123Z",
  "resultados": [
    {"entidade": "propriedade", "uuid": "…", "rev": 4, "resultado": "aplicado"},
    {"entidade": "item", "uuid": "…", "rev": 2, "resultado": "duplicado"},
    {"entidade": "propriedade", "uuid": "…", "rev": 7, "resultado": "conflito", "conflito_uuid": "…"},
    {"entidade": "foto", "uuid": "…", "rev": 1, "resultado": "rejeitado", "motivo": "…"}
  ],
  "recibo_sha256": "…"
}
```
O Android faz `rev_confirmada = max(rev_confirmada, rev)` para `aplicado`, `duplicado` e `conflito`. `rejeitado` continua pendente e é exibido como erro.

## 7. Restauração de equipe (RN-06)
1. O aparelho reinstalado gera nova instalação e lê o QR de pareamento.
2. O operador escolhe **"Restaurar equipe existente"** e seleciona a equipe.
3. A Central responde com o UUID da equipe e os cadastros **vigentes** (já com as correções). Cada entidade vem com `rev = rev_origem` e as correções passam a ser a nova base (a Central desliga `corrigido_desde_rev_origem` para esses cadastros, com registro em auditoria).
4. Fotos: por padrão só metadados e miniaturas. Os originais ficam na Central (opção de baixar, se houver espaço).
5. O Android grava tudo em **uma transação**, com `rev_confirmada = rev`.
6. O app avisa: "Dados não enviados antes da perda do aparelho não podem ser recuperados."

## 8. Pacote USB
Formato e fluxo: ADR-0008. O conteúdo é o mesmo do `PUT /v1/lotes` com as fotos dentro do ZIP. A importação passa pelo mesmo núcleo, e o recibo é igual ao da seção 6.

## 9. Detalhes da v1 implementada (ADR-0017)
- **Schemas:** `shared/schemas/{comum,pareamento,lote,estado-lote,recibo}.v1.schema.json`; fixtures em `shared/test-fixtures/contratos`.
- **Pareamento:** o `POST /v1/pareamento` espera a decisão do operador por até **90 s**. Token do QR: 10 min, uso único, marcado como usado só na confirmação.
- **Erros:** corpo `{"codigo","erro"}`. Códigos: `invalido` 400 · `nao_autorizado` 401 · `proibido`, `recusado` 403 · `nao_encontrado` 404 · `lote_divergente`, `outra_operacao`, `tempo_esgotado`, `sem_operacao` 409 · `token_invalido` 410 · `versao_incompativel` 422 · `erro_interno` 500. **Nunca 408** (o OkHttp repete sozinho).
- **`PUT /v1/fotos/{foto}?offset=N`:** offset diferente do já recebido → 409 `{"codigo":"offset","offset":M}`; o app continua de M. Resposta 200: `{"uuid","offset","existe"}`; `GET` igual.
- **`POST /v1/lotes/{lote}/importar`:** com foto faltando ou hash divergente → **409 com o estado do lote** (`fotos_faltantes`); nada é aplicado.
- **Mesmo `lote_uuid`:** bytes idênticos → aceito (e, se importado, o mesmo recibo); bytes diferentes → 409 `lote_divergente`. O Android congela o JSON do lote ao prepará-lo.
- **`recibo_sha256`:** SHA‑256 de `lote_uuid\ncentral_id\nimportado_em\n` + uma linha `entidade|uuid|rev|resultado|motivo\n` por resultado, na ordem do recibo.
- **A equipe vai em todo lote** (com a sua `rev`); no recibo ela vem como `duplicado` se já estava na Central com a mesma `rev` (gravada no pareamento).

## 10. Compatibilidade de versões
- A Central declara `protocolo_versao_min/max` e `schema_versao_min/max` em `/v1/status`.
- O Android fora da faixa vê "Atualize o aplicativo (Campo vX / Central vY)". Nenhum dado é descartado.
- Toda nova versão de protocolo é acompanhada de fixtures da versão anterior, para testes de compatibilidade.
