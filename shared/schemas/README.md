# JSON Schemas

JSON Schema **2020‑12**, um arquivo por entidade e mensagem, com versão no nome (`propriedade.v1.schema.json`, `lote.v1.schema.json`, ...).

- Kotlin: tipos escritos à mão (kotlinx.serialization), validados contra as fixtures de `test-fixtures/contratos`.
- Go: tipos escritos à mão (`encoding/json`) e validação com uma biblioteca de JSON Schema escolhida na Fase 0.
- Não há geração de código nesta versão (menos toolchain). Avaliar se os contratos crescerem.

**Versão atual: v2** (A28, ADR-0024). Os arquivos `*.v1` ficam até o app Campo passar para a v2 (etapa E4) e depois são removidos; a Central desta versão só aceita a v2.

| Schema | Uso |
|---|---|
| `comum.v2` | Definições compartilhadas (UUID, instante, coleta, código de equipe, texto longo…) |
| `qr-pareamento.v2` | Conteúdo do QR da Central (sem operação) |
| `status.v2` | `GET /v1/status` |
| `pareamento.v2` | `POST /v1/pareamento` (`#/$defs/pedido` e `#/$defs/resposta`), só do aparelho |
| `lote.v2` | `PUT /v1/lotes/{lote}`: uma coleta por lote |
| `estado-lote.v2` | Respostas de `PUT/GET /v1/lotes/{lote}` e do importar (`recebendo`, `aguardando_confirmacao`, `importado`, `cancelado`) |
| `recibo.v2` | Recibo de importação, com o destino da coleta |

**Versões nos payloads:** todos levam `protocolo_versao`; os que transportam registros ou pedem algo à Central (lote, pedido de pareamento) levam também `schema_versao`.

Go: a Central valida com código próprio (`desktop/internal/contratos`); os testes conferem schema **e** código Go em todas as fixtures (ADR-0017). Os `$id` usam a base `https://sitposse7.local/schemas/` só como identificador: nada é baixado.
