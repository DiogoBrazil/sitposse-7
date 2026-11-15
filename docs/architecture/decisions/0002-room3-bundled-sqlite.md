# ADR-0002 — Room 3 + BundledSQLiteDriver no Android

- **Status:** Aprovado
- **Data:** 2026-11-13

## Contexto
A persistência local é o requisito mais crítico. Com minSdk 26 (Android 8), o SQLite da plataforma pode ser 3.18, sem UPSERT (3.24), sem window functions (3.25) e sem `VACUUM INTO` (3.27). O comportamento varia por fabricante.

## Decisão
- Room **3.0.x** (`androidx.room3`, estável desde 01/07/2026; 3.0.3 em 09/09/2026), com KSP e o plugin Gradle `androidx.room3` (`schemaDirectory`).
- `BundledSQLiteDriver`: SQLite moderno e **igual em todos os aparelhos**.
- Ao abrir a conexão: `PRAGMA journal_mode=WAL`, `PRAGMA synchronous=FULL`, `PRAGMA foreign_keys=ON`. A Fase 0 verifica que o Room não sobrescreve esses valores.
- `fallbackToDestructiveMigration` é **proibido**.

## Alternativas consideradas
- **Room 2.8.x**: maduro, mas o desenvolvimento novo migrou para o Room 3 e o SQLite da plataforma continua variando.
- **SQLDelight**: bom, mas fora da stack Jetpack sugerida e sem ganho claro.
- **AndroidSQLiteDriver**: APK menor, mas com SQLite variável.

## Consequências
- O APK cresce cerca de 1–2 MB por ABI por causa das bibliotecas nativas. Mitigação: distribuir por ABI ou aceitar o tamanho.
- O Room 3 exige coroutines (DAO `suspend`/`Flow`) e Kotlin ≥ 2.2 (requisito da 3.1).
- Usar `allowDataLossOnRecovery()` (3.1-alpha) está **proibido** sem novo ADR.

## Fontes
- https://developer.android.com/jetpack/androidx/releases/room3
- https://developer.android.com/training/data-storage/room/migrating-db-versions
