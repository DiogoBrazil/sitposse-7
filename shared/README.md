# shared/

Artefatos **compartilhados** entre Android (Kotlin) e Central (Go). São a garantia de que os dois lados interpretam os dados igualmente.

| Pasta | Conteúdo | Situação |
|---|---|---|
| [`contracts/`](contracts/README.md) | Descrição dos contratos de payload (sincronização, pacote, recibo, QR) | Especificado em `docs/synchronization/protocolo.md`; arquivos na Fase 0/2 |
| [`schemas/`](schemas/README.md) | JSON Schema 2020‑12 versionados | Fase 0 |
| [`test-fixtures/`](test-fixtures/README.md) | Casos de teste executados pelos **dois** lados | Área e normalização prontos |

Regra: qualquer mudança aqui exige ADR (se for contrato), incremento de versão e atualização dos testes Kotlin **e** Go no mesmo commit.
