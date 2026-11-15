# Contratos

Contratos previstos (protocolo v1, ver `docs/synchronization/protocolo.md`):

| Contrato | Arquivo (a criar) | Uso |
|---|---|---|
| QR de pareamento | `qr-pareamento.v1.md` | Central → Android |
| Pareamento | `pareamento.v1.md` | `POST /v1/pareamento` |
| Status | `status.v1.md` | `GET /v1/status` |
| Lote (manifesto + registros) | `lote.v1.md` | `PUT /v1/lotes/{id}`, pacote USB |
| Recibo | `recibo.v1.md` | Resposta da importação, `.s7rcb`, QR de recibo |
| Pacote USB | `pacote-s7pkg.v1.md` | Contingência e backup Android |
| Restauração | `restauracao.v1.md` | `POST /v1/restauracao` |

Todos os payloads levam `protocolo_versao` e `schema_versao`. Timestamps em ISO 8601 UTC com ms e `Z` (ADR-0010).
