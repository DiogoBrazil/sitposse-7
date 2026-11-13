# ADR-0010 — Timestamps

- **Status:** Aprovado
- **Data:** 2026-11-13

## Decisão
- SQLite: `INTEGER` com epoch em **milissegundos UTC**.
- JSON: string **ISO 8601 UTC com milissegundos e `Z`** (`2026-11-13T19:40:38.123Z`).
- UI: horário local do dispositivo (`America/Porto_Velho` na prática).
- `created_at`/`updated_at` de origem são preservados; a Central registra `recebido_em` à parte. Nenhuma lógica de sincronização usa esses horários (ADR-0005).
