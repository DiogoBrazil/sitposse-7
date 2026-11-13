# JSON Schemas

JSON Schema **2020‑12**, um arquivo por entidade e mensagem, com versão no nome (`propriedade.v1.schema.json`, `lote.v1.schema.json`, ...).

- Kotlin: tipos escritos à mão (kotlinx.serialization), validados contra as fixtures de `test-fixtures/contratos`.
- Go: tipos escritos à mão (`encoding/json`) e validação com uma biblioteca de JSON Schema escolhida na Fase 0.
- Não há geração de código nesta versão (menos toolchain). Avaliar se os contratos crescerem.

A criar na Fase 0.
