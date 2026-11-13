-- 0001: metadados da instalação da Central (Fase 0).
-- O esquema de negócio (docs/architecture/modelo-de-dados.md) entra na Fase 2.
CREATE TABLE meta (
    chave      TEXT PRIMARY KEY,
    valor      TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
) STRICT;
