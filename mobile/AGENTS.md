# AGENTS.md — Android (Campo)

Complementa o [`AGENTS.md`](../AGENTS.md) raiz, **sem contradizê-lo**.

- **Nenhuma** dependência que exija internet, Google Play Services ou que envie telemetria (ex.: `datatransport`/Firebase): QR com ZXing (ADR-0015); localização via `LocationManager`. Conferir com `./gradlew :app:dependencies` ao adicionar biblioteca.
- Room 3 (`androidx.room3`) com `BundledSQLiteDriver` envolto pelo `DriverDuravel` (`:core:dados`); PRAGMAs `WAL`, `synchronous=FULL` e `foreign_keys=ON` em **toda** conexão, verificados em teste (`PragmasTest`). Abrir o banco só por `configurarPadrao()`.
- **Proibido** `fallbackToDestructiveMigration*`. Todo `@Database(version = N)` novo exige migração e teste com `MigrationTestHelper`. A pasta `schemas/` é versionada.
- Toda escrita de entidade sincronizável incrementa `rev` **na mesma transação**.
- Toda operação de banco e de arquivo fica fora da main thread (coroutines com dispatcher de I/O).
- Fotos: decodificar com amostragem; nunca `BitmapFactory.decodeFile` sem `inSampleSize` em imagem de câmera. Ordem obrigatória: arquivo com fsync e rename, depois banco.
- A UI só mostra "Salvo no aparelho" após o retorno bem-sucedido da transação.
- Rascunhos nunca são contados nem sincronizados.
- `android:allowBackup="false"` e `dataExtractionRules` excluindo tudo.
- Rede de sincronização: usar a `Network` Wi‑Fi obtida por `requestNetwork` e seu `socketFactory`; pinning SPKI conforme ADR-0007; **nunca** um `TrustManager` que aceite tudo.
- Com `targetSdk ≥ 37`: declarar e pedir `ACCESS_LOCAL_NETWORK` antes do pareamento, com explicação.
- Textos da UI em português do Brasil, em `strings.xml` (sem texto fixo no código).
- Testes JVM rodam as fixtures de `../shared/test-fixtures`.
