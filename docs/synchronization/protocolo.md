# Protocolo de sincronização — v2

Status: **v1 aprovado com o plano (13/11/2026); v2 proposto pelo ADR-0024 (A28, 15/11/2026).** Mudanças exigem ADR, incremento de `protocolo_versao` e fixtures.
Regras de negócio: RN-70 a RN-74. Integridade: [integridade.md](integridade.md). Conflitos: [conflitos.md](conflitos.md).

## 1. Transporte
- HTTPS na rede Wi‑Fi local, sem internet, por IP: TLS 1.3 preferido; TLS 1.2 só com ECDHE + AEAD, para Android 8/9 (ADR-0014). Pinning do SPKI vindo do QR (ADR-0007).
- Porta padrão **38807** (fora das faixas reservadas pelo Windows/Hyper-V, ≥ 49152, e pela rede espelhada do WSL, 44620–48715). O Android não fixa a porta: ela vem no QR.
- JSON UTF‑8 (gzip opcional). Fotos em binário (`application/octet-stream`).
- Autenticação: `Authorization: Bearer <credencial_dispositivo>`, exceto em `/v1/pareamento`, que usa o token do QR.
- Todo payload JSON leva `protocolo_versao` e `schema_versao`.
- No Android a sincronização roda na `Network` Wi‑Fi específica (ver `mobile/docs/arquitetura.md`) e exige `ACCESS_LOCAL_NETWORK` quando `targetSdk ≥ 37`.

## 2. Pareamento (só do aparelho)
```
Operador (Central)          Central                         Android
      │ "Parear aparelho"     │                                 │
      │──────────────────────►│ gera token (10 min, uso único)  │
      │                       │ exibe QR {v, central_id, ips[], │
      │                       │  porta, spki_sha256, token}     │
      │                       │◄──────── lê QR ─────────────────│
      │                       │◄─ TLS (pin SPKI) POST /v1/pareamento {token, instalacao, rotulo_aparelho}
      │◄── "Aparelho X quer parear. Confirmar?"
      │── confirma ──────────►│ grava dispositivo (hash da credencial)
      │                       │── 200 {credencial_dispositivo} ─────────────────────────►│ grava pareamento
```
- O pareamento identifica o **aparelho**; a equipe e o estudo vêm de cada coleta, pelo código (ADR-0024).
- Token expirado ou já usado → `410` com mensagem "Gere um novo QR na Central".
- A Central consome o token e grava o dispositivo **antes** de responder `200`. Se o aparelho não recebe ou não grava a resposta (queda de rede, app encerrado), ele continua sem pareamento e precisa de um **novo QR**: o mesmo responde `410`. O novo pareamento substitui a credencial do mesmo `instalacao_uuid`. Recusa (`403 recusado`) e tempo esgotado (`409 tempo_esgotado`) não consomem o token. Incidente relacionado: [crash após a confirmação](../../mobile/docs/incidentes/2026-11-14-crash-pareamento.md).

## 3. Endpoints
| Método e rota | Função |
|---|---|
| `GET /v1/status` | versões aceitas, `central_id`, horário |
| `POST /v1/pareamento` | pareamento (seção 2) |
| `PUT /v1/lotes/{lote}` | manifesto + registros de **uma coleta** (idempotente) |
| `GET /v1/lotes/{lote}` | estado do lote (`recebendo`, `aguardando_confirmacao`, `importado` com recibo, `cancelado` com motivo) |
| `PUT /v1/fotos/{foto}?offset=N` | bloco de até 1 MiB da foto (append em staging) |
| `GET /v1/fotos/{foto}` | bytes já recebidos (para retomar) ou `existe=true` |
| `POST /v1/lotes/{lote}/importar` | confere as fotos e põe o lote em `aguardando_confirmacao` |
| `POST /v1/lotes/{lote}/ack` | o Android informa que gravou o recibo (informativo) |
| `POST /v1/restauracao` | restauração de cadastros autorizada pelo operador (seção 7, futura) |

As rotas continuam com o prefixo `/v1`; a versão do conteúdo é a de `protocolo_versao`.

## 4. Máquina de estados do lote (Android)
```
 PREPARADO ──► ENVIANDO ──(importar ok)──► AGUARDANDO_CONFIRMACAO ──(operador confirma)──► CONFIRMADO
                  │                               │                                    (recibo gravado em transação)
                  │ falha de rede                 ├──(operador cancela)──► CANCELADO (terminal; novo lote no próximo envio)
                  ▼                               │
          ERRO_RECUPERAVEL ◄──────────────────────┘ falha de rede
                  │ retomar: GET /v1/lotes/{id} → recebendo | aguardando | importado | cancelado | 404
   ERRO_FATAL: versão incompatível, pareamento revogado, cert. diferente → exige ação do usuário; dados intactos.
   Canal USB: PREPARADO → EXPORTADO_AGUARDANDO_CONFIRMACAO → CONFIRMADO (ao ler o recibo).
```
- **Um lote aberto por coleta.** Várias coletas pendentes geram vários lotes, enviados em sequência; cada um é decidido separadamente.
- O lote `CANCELADO` nunca é reenviado com o mesmo `lote_uuid`. Os registros continuam pendentes (`rev > rev_confirmada`) e entram no próximo lote da coleta.

## 5. Sequência de envio
```
Android                                   Central                               Operador
  │ GET /v1/status ─────────────────────────►│ versões                               │
  │ PUT /v1/lotes/L {coleta, registros} ────►│ grava lote "recebendo" (idempotente)  │
  │◄─ {fotos_faltantes:[{uuid, offset}]} ────│                                       │
  │ PUT /v1/fotos/F?offset=0 (1 MiB) ───────►│ append + fsync em staging/F.part      │
  │ POST /v1/lotes/L/importar ──────────────►│ valida tamanho + SHA‑256              │
  │◄─ 200 {status: aguardando_confirmacao} ──│ lote "aguardando_confirmacao" ───────►│ modal "Confirmar destino"
  │ GET /v1/lotes/L (a cada 2 s) ───────────►│                                       │
  │                                          │◄──── confirmar | alterar | cancelar ──│
  │                                          │ confirmar/alterar:                    │
  │                                          │ 1. publica arquivos (rename + fsync)  │
  │                                          │ 2. transação: destino da coleta,      │
  │                                          │    decisao, registros, aplicacoes,    │
  │                                          │    auditoria, lote=importado + recibo │
  │◄─ 200 {status: importado, recibo} ───────│ cancelar: lote=cancelado + decisao    │
  │ transação: recibo; rev_confirmada; destino da coleta                             │
  │ POST /v1/lotes/L/ack ───────────────────►│ registra ack                          │
```
- **Resposta perdida** ou celular fora da rede quando o operador decide: o Android consulta `GET /v1/lotes/L` e recebe **o mesmo recibo** (ou `cancelado`).
- **Reenvio** de um lote já importado: devolve o recibo existente, sem reaplicar. Lote cancelado: `409 lote_cancelado`.
- **Hash divergente**: a Central apaga o `.part` e pede a foto de novo; o lote não passa para `aguardando_confirmacao` até todas as fotos baterem.
- **Edição durante o envio**: gera `rev` maior, que continua pendente para o próximo lote.
- **Destino travado:** se a coleta já foi importada antes, o operador só confirma ou cancela (ADR-0024).

## 6. Recibo (v2)
```json
{
  "protocolo_versao": 2,
  "lote_uuid": "…",
  "central_id": "…",
  "importado_em": "2026-11-13T19:40:38.123Z",
  "destino": {
    "estudo_codigo": "SITPOSSE7A82F91C4", "estudo_nome": "Fazenda Santa Maria",
    "jornada_numero": 1, "jornada_data": "2026-10-10",
    "equipe_nome": "Equipe Alfa", "equipe_codigo": "EQ7K3M9QX2",
    "codigo_informado": "EQ7K3M9QX2", "corrigido": false
  },
  "resultados": [
    {"entidade": "coleta", "uuid": "…", "rev": 1, "resultado": "aplicado"},
    {"entidade": "propriedade", "uuid": "…", "rev": 4, "resultado": "aplicado"},
    {"entidade": "foto", "uuid": "…", "rev": 1, "resultado": "rejeitado", "motivo": "…"}
  ],
  "recibo_sha256": "…"
}
```
O Android faz `rev_confirmada = max(rev_confirmada, rev)` para `aplicado`, `duplicado` e `conflito`. `rejeitado` continua pendente e é exibido como erro. O `destino` é gravado na coleta.

## 7. Restauração de cadastros (RF-A15, futura)
1. O aparelho reinstalado gera nova instalação e lê o QR de pareamento.
2. O operador escolhe **"Restaurar cadastros"** e seleciona a coleta ou a equipe.
3. A Central responde com os cadastros **vigentes** (já com as correções). Cada entidade vem com `rev = rev_origem` e as correções passam a ser a nova base (a Central desliga `corrigido_desde_rev_origem` para esses cadastros, com registro em auditoria).
4. Fotos: por padrão só metadados e miniaturas. Os originais ficam na Central (opção de baixar, se houver espaço).
5. O Android grava tudo em **uma transação**, com `rev_confirmada = rev`.
6. O app avisa: "Dados não enviados antes da perda do aparelho não podem ser recuperados."

## 8. Pacote USB
Formato e fluxo: ADR-0008. O conteúdo é o mesmo do `PUT /v1/lotes` com as fotos dentro do ZIP. A importação passa pelo mesmo núcleo, e o recibo é igual ao da seção 6.

## 9. Detalhes da v2 (ADR-0017, alterado pelo ADR-0024)
- **Schemas:** `shared/schemas/{comum,pareamento,qr-pareamento,status,lote,estado-lote,recibo}.v2.schema.json`; fixtures em `shared/test-fixtures/contratos/*.v2.json`.
- **Versões:** `protocolo_versao = 2` e `schema_versao = 2` em todo payload JSON. A Central desta versão aceita só a 2.
- **Pareamento:** o `POST /v1/pareamento` espera a decisão do operador por até **90 s**. Token do QR: 10 min, uso único, marcado como usado só na confirmação.
- **Erros:** corpo `{"codigo","erro"}`. Códigos: `invalido` 400 · `nao_autorizado` 401 · `proibido`, `recusado` 403 · `nao_encontrado` 404 · `lote_divergente`, `lote_cancelado`, `ja_decidido`, `tempo_esgotado` 409 · `token_invalido` 410 · `versao_incompativel` 422 · `erro_interno` 500. **Nunca 408** (o OkHttp repete sozinho).
- **`PUT /v1/fotos/{foto}?offset=N`:** offset diferente do já recebido → 409 `{"codigo":"offset","offset":M}`; o app continua de M. Resposta 200: `{"uuid","offset","existe"}`; `GET` igual.
- **`POST /v1/lotes/{lote}/importar`:** com foto faltando ou hash divergente → **409 com o estado do lote** (`fotos_faltantes`); nada muda. Com tudo certo → 200 com `status = aguardando_confirmacao` (ou o estado atual, se já decidido).
- **Mesmo `lote_uuid`:** bytes idênticos → aceito (e, se importado, o mesmo recibo); bytes diferentes → 409 `lote_divergente`. O Android congela o JSON do lote ao prepará-lo.
- **`recibo_sha256`:** SHA‑256 de `lote_uuid\ncentral_id\nimportado_em\n` + a linha `destino|estudo_codigo|jornada_numero|jornada_data|equipe_codigo|codigo_informado|corrigido\n` + uma linha `entidade|uuid|rev|resultado|motivo\n` por resultado, na ordem do recibo.
- **A coleta vai em todo lote** (com a sua `rev`); no recibo ela vem como `duplicado` se a Central já tinha a mesma `rev`.
- **Estado cancelado:** `GET` devolve `{"status":"cancelado","motivo":"cancelado_operador"|"codigo_inexistente"}`.

## 10. Compatibilidade de versões
- A Central declara `protocolo_versao_min/max` e `schema_versao_min/max` em `/v1/status`.
- O Android fora da faixa vê "Atualize o aplicativo (Campo vX / Central vY)". Nenhum dado é descartado.
- Toda nova versão de protocolo é acompanhada de fixtures da versão anterior, para testes de compatibilidade.
