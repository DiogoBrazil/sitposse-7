# ADR-0004 — modernc.org/sqlite na Central

- **Status:** Aprovado
- **Data:** 2026-11-13

## Decisão
Usar `modernc.org/sqlite` (Go puro, sem CGO) via `database/sql`, com uma conexão de escrita dedicada e PRAGMAs `journal_mode=WAL`, `synchronous=FULL`, `foreign_keys=ON` e `busy_timeout`. O backup usa `VACUUM INTO` ou a API de backup do driver.

## Alternativas consideradas
- **mattn/go-sqlite3**: muito usado, mas exige CGO e toolchain C no Windows, o que complica o build.

## Consequências
- Build cruzado simples. O desempenho é inferior ao CGO, mas suficiente para o volume aprovado (RNF-07).
- Atenção: a v1.56.0 passou a interpretar parâmetros de DSN no estilo mattn (`_busy_timeout` etc.). Configure os PRAGMAs explicitamente e teste.

## Fontes
- https://gitlab.com/cznic/sqlite
