# ADR-0020 — FiltroEstudo calculado em memória sobre as visões vigentes

- **Status:** Aprovado
- **Data:** 2026-11-14
- **Substitui / Substituído por:** detalha "Consulta: `FiltroEstudo` → SQL parametrizado" (`visao-geral.md`)

## Contexto
A Fase 4 (A25, `docs/requirements/estatisticas.md` §22-A) pede filtros combináveis sobre qualquer quantidade, inclusive cada tipo de item e os grupos de "Outros". Pede também que cartões, gráficos, tabelas e listas sejam coerentes entre si (RN-81, 22-A.5), e que novos filtros entrem "sem reestruturação". O volume de projeto é pequeno: até 300 propriedades por operação, testes com 900 (RNF-07). O limite de resposta é 300 ms (RNF-08).

## Decisão
**Aprovada pelo usuário com a antecipação da Fase 4 (A25).**
- `internal/estatisticas.Carregar` lê a base com consultas **fixas**, sem nenhum valor de filtro no SQL, e **só das visões `v_*_vigentes`** (RN-80). O `desktop/AGENTS.md` exige SQL parametrizado e as visões vigentes; aqui nenhum valor de filtro chega ao SQL.
- `estatisticas.Calcular(base, filtro)` aplica o filtro **em memória** e produz, do **mesmo** conjunto de famílias:
  - totais e grupos por equipe e por área identificada;
  - tipos de item e faixas de área;
  - a lista de famílias para o drill-down.
- Os campos filtráveis vêm de um **registro único** (`camposFixos` + catálogo + grupos de "Outros"). Um filtro novo sobre uma quantidade é uma entrada nesse registro.
- A área é somada em **dm² inteiros** e formatada uma vez (`dominio.FormatarHa`). Ausência de área nunca vira zero.

## Alternativas consideradas
- **Montar SQL dinâmico por filtro:** cada cartão e cada tabela teria a própria consulta, com risco de números divergentes entre si e muito mais código por filtro novo.
- **Visões materializadas por indicador:** complexidade sem ganho no volume do projeto.

## Consequências
- Medido: carregar + calcular **900 propriedades e 4.500 fotos em ~14 ms** (`TestDesempenhoVolume3x`, limite 300 ms).
- A coerência é verificada em teste: em todo caso da fixture, a soma das famílias e a soma das equipes batem com os totais.
- Se o volume crescer em ordens de grandeza, o `Carregar` pode passar a filtrar por operação no SQL (parametrizado) sem mudar o `Calcular`.

## Fontes
- `shared/test-fixtures/estatisticas/` (base e 20 casos, incluindo as 7 perguntas da §22-A.3).
