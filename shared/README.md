# shared/

Artefatos **compartilhados** entre Android (Kotlin) e Central (Go). São a garantia de que os dois lados interpretam os dados igualmente.

| Pasta | Conteúdo | Situação |
|---|---|---|
| [`contracts/`](contracts/README.md) | Descrição dos contratos de payload (sincronização, pacote, recibo, QR) | Especificado em `docs/synchronization/protocolo.md` (§9: detalhes da v1, ADR-0017) |
| [`schemas/`](schemas/README.md) | JSON Schema 2020‑12 versionados | v1: QR, status, pareamento, lote, estado do lote, recibo |
| [`test-fixtures/`](test-fixtures/README.md) | Casos de teste executados pelos **dois** lados | Área, normalização, TLS, catálogo e contratos v1 |

Regra: qualquer mudança aqui exige ADR (se for contrato), incremento de versão e atualização dos testes Kotlin **e** Go no mesmo commit.
