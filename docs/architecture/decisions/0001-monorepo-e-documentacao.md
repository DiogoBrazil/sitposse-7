# ADR-0001 — Monorepositório e documentação em camadas

- **Status:** Aprovado
- **Data:** 2026-11-13

## Contexto
Dois produtos (Android e Windows) compartilham contratos de sincronização, regras de negócio e fixtures. Divergência entre eles pode corromper dados.

## Decisão
Um único repositório com `mobile/`, `desktop/` e `shared/`. A documentação raiz (`docs/`) é a fonte de requisitos, regras e contratos; `mobile/docs` e `desktop/docs` trazem só especificidades. O `AGENTS.md` raiz tem precedência sobre os específicos. As regras de negócio vivem só em `docs/requirements/regras-de-negocio.md`.

## Alternativas consideradas
- Repositórios separados: facilitam builds independentes, mas tornam difícil manter contratos e fixtures sincronizados.

## Consequências
- Mudanças de contrato são atômicas (um commit altera Kotlin, Go, schema e fixture).
- A CI precisa de jobs separados por plataforma.
