# JSON Schemas

JSON Schema **2020‑12**, um arquivo por entidade e mensagem, com versão no nome (`propriedade.v1.schema.json`, `lote.v1.schema.json`, ...).

- Kotlin: tipos escritos à mão (kotlinx.serialization), validados contra as fixtures de `test-fixtures/contratos`.
- Go: tipos escritos à mão (`encoding/json`) e validação com uma biblioteca de JSON Schema escolhida na Fase 0.
- Não há geração de código nesta versão (menos toolchain). Avaliar se os contratos crescerem.

| Schema | Uso |
|---|---|
| `comum.v1` | Definições compartilhadas (UUID, instante, equipe…) |
| `qr-pareamento.v1` | Conteúdo do QR da Central |
| `status.v1` | `GET /v1/status` |
| `pareamento.v1` | `POST /v1/pareamento` (`#/$defs/pedido` e `#/$defs/resposta`) |
| `lote.v1` | `PUT /v1/lotes/{lote}` |
| `estado-lote.v1` | Respostas de `PUT/GET /v1/lotes/{lote}` e do importar sem sucesso |
| `recibo.v1` | Recibo de importação |

Go: a Central valida com código próprio (`desktop/internal/contratos`); os testes conferem schema **e** código Go em todas as fixtures (ADR-0017). Os `$id` usam a base `https://sitposse7.local/schemas/` só como identificador: nada é baixado.
