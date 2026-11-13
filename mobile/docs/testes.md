# Testes — Android

A estratégia geral está em [`docs/testing/estrategia.md`](../../docs/testing/estrategia.md). Específico do Android:

| Tipo | Ferramenta | Casos-chave |
|---|---|---|
| Unidade (JVM) | JUnit + kotlinx-coroutines-test | Área e normalização (fixtures `shared`), validações, máquina de estados do lote |
| Banco | Room in-memory + instrumentado | CHECKs, índice único parcial de itens, `rev` incrementado em cada escrita, PRAGMAs efetivos |
| Migração | `MigrationTestHelper` | Todas as versões publicadas → atual |
| Fotos | Instrumentado com falha injetada | Falha após `.part`, após rename e antes do banco; reconciliador |
| GNSS | Provedor de localização *mock* | ≤ 15 m, > 15 m, timeout, permissão negada |
| UI | Compose UI Test | Cadastro completo; vários "Outros"; rascunho em rotação; mensagens de erro |
| Robustez manual | Aparelho real | Modo avião; `adb shell am kill` durante salvar; reinício; armazenamento cheio |
| Sync | Cliente JVM × servidor Go | Cenários de `estrategia.md` §4 |
