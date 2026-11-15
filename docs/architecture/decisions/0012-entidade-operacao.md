# ADR-0012 — Entidade operação

- **Status:** Substituído por ADR-0023 (15/11/2026)
- **Data:** 2026-11-13

## Decisão
- Tabela `operacoes` desde a primeira versão. A Central tem uma operação **ativa** por vez para pareamento.
- Uma equipe pertence a uma operação (1:1), com o vínculo feito no primeiro pareamento (o QR traz a operação).
- Antes do pareamento, o Android funciona normalmente com `operacao_uuid` nulo.
- Reuso de aparelho em nova operação: "Encerrar operação no aparelho" só depois de tudo confirmado; cria nova equipe (detalhe final: pendência P6).

## Consequências
- Estatísticas e relatórios filtram por operação. Evita migração difícil no futuro.
