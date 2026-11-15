# Testes — Android

A estratégia geral está em [`docs/testing/estrategia.md`](../../docs/testing/estrategia.md). Específico do Android:

| Tipo | Ferramenta | Casos-chave |
|---|---|---|
| Unidade (JVM) | JUnit 6 + kotlin-test | Área, normalização e QR (fixtures `shared`), validações, máquina de estados do lote, decodificação do QR (ZXing) |
| Banco | Room em arquivo na JVM (`:core:dados:jvmTest`, mesmo driver empacotado) + instrumentado | CHECKs, índice único parcial de itens, `rev` incrementado em cada escrita, PRAGMAs efetivos em todas as conexões |
| Migração | `MigrationTestHelper` | Todas as versões publicadas → atual |
| Fotos | Instrumentado com falha injetada | Falha após `.part`, após rename e antes do banco; reconciliador |
| GNSS | Provedor de localização *mock* | ≤ 15 m, > 15 m, timeout, permissão negada |
| UI | Compose UI Test | Cadastro completo; vários "Outros"; rascunho em rotação; mensagens de erro |
| Robustez manual | Aparelho real | Modo avião; `adb shell am kill` durante salvar; reinício; armazenamento cheio |
| Sync | Cliente JVM × servidor Go real (`desktop/cmd/central-teste`) | Pin (fixture `tls/pin.json`), TLS 1.3 e 1.2, pin divergente; depois os cenários de `estrategia.md` §4 |
| Sync: pareamento | `PareamentoGoTest` (Central sem confirmação automática: `-confirmar=false`, `-espera`) | Recusado, repetido, tempo esgotado, Central derrubada durante a espera ou fora do ar: sempre `FalhaCentral`, nada pareado, cadastros pendentes |
| Sync: fechamento | `FechamentoConexaoTest` (fábrica de sockets que imita o StrictMode) | `close()`/`fechar()` na thread principal não fecham sockets nela ([incidente de 14/11](incidentes/2026-11-14-crash-pareamento.md)) |
