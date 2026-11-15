# AGENTS.md — Desktop (Central)

Complementa o [`AGENTS.md`](../AGENTS.md) raiz, **sem contradizê-lo**.

- O **núcleo Go** (`internal/...`) **não importa Wails**. Os bindings ficam em uma camada fina (`app/`). Tudo que contém regra é testável com `go test` (ADR-0003).
- SQLite com `modernc.org/sqlite`: **uma** conexão de escrita; PRAGMAs `WAL`, `synchronous=FULL`, `foreign_keys=ON` e `busy_timeout` explícitos (ADR-0004).
- Migrações SQL numeradas e embutidas; backup automático antes de migrar; nunca destrutivas.
- Wi‑Fi e pacote USB usam o **mesmo** núcleo de importação (`internal/importacao`).
- Fotos: **arquivo publicado antes do registro**; reconciliador obrigatório ([integridade.md](../docs/synchronization/integridade.md)).
- Toda alteração feita pelo operador grava `auditoria` na mesma transação.
- Consultas de estatística: só SQL **parametrizado** sobre as visões `v_*_vigentes`; nunca concatenar valores de filtro.
- Frontend sem CDN: todas as bibliotecas empacotadas no build (a Central funciona offline).
- Só `internal/cartografia` faz requisições à internet; nada de dados de cadastro nessas requisições.
- Textos da UI em português do Brasil.
