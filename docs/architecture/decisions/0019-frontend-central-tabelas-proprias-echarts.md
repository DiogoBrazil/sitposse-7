# ADR-0019 — Frontend da Central: tabelas próprias, ECharts e fonte empacotada (substitui o PrimeVue)

- **Status:** Aprovado
- **Data:** 2026-11-14
- **Substitui / Substituído por:** substitui o PrimeVue previsto em `visao-geral.md` e em `desktop/docs/arquitetura.md` (P9)

## Contexto
A A25 antecipou a Fase 4 (estatísticas) e a melhoria visual da Central. A arquitetura previa o PrimeVue para tabelas e o Apache ECharts para gráficos.

Ao fixar as versões, achamos uma mudança de licença. O **PrimeVue 5** (outubro de 2026) e o tema `@primeuix/themes@3` deixaram de ser MIT: exigem chave de licença (`@primeui/license-manager`, verificação offline). A licença gratuita "Community" só vale para organizações com menos de 10 funcionários e menos de US$ 1 milhão de receita, o que exclui a PMRO. A última versão MIT é a 4.5.5 (abril de 2026), sem atualizações gratuitas futuras.

## Decisão
**Aprovada pelo usuário em 14/11/2026.**
- **Tabelas próprias**, num componente Vue do projeto (`TabelaDados`). Recursos:
  - divisórias visíveis, cabeçalho fixo, ordenação com `aria-sort` e pagina;
  - linhas acessíveis pelo teclado;
  - estados vazio, carregando e erro;
  - layout empilhado em largura estreita (container query).
  
  A lógica de ordenar e paginar fica num módulo puro, testado com Vitest.
- **Gráficos com Apache ECharts 6.1.0** (Apache‑2.0) via **vue-echarts 8.3.2** (MIT). Os módulos são importados sob demanda (`use([...])`), e tudo é empacotado pelo Vite, sem CDN (AGENTS.md §6).
- **Fonte Archivo** (variável, eixos de peso e largura; OFL‑1.1) por `@fontsource-variable/archivo` 5.3.0, empacotada. Hoje a Central usa a fonte do sistema. Com a Archivo, o visual fica igual em todo Windows e os números ganham largura condensada nos destaques.
- **Paleta dos gráficos** validada para daltonismo e contraste (claro e escuro), em `desktop/docs/telas-e-fluxos.md` §Identidade visual.
- **Testes do frontend** com **Vitest 5.1.0** (MIT, só desenvolvimento), como prevê `testing/estrategia.md` §5.

## Alternativas consideradas
- **PrimeVue 5 com licença comercial:** custo por desenvolvedor e dependência de chave para compilar e usar.
- **PrimeVue 4.5.5 (MIT):** biblioteca congelada, sem caminho de atualização sem licença.
- **Gráficos SVG feitos à mão:** nenhuma dependência nova, mas sem tooltips, sem animação e sem a variedade de que a Fase 4 precisa.

## Consequências
- Mais código nosso para manter na tabela. Em troca, não há risco de licença, e o visual fica sob controle do projeto.
- O ECharts aumenta o pacote do frontend (só os módulos usados entram). Ele roda offline (RN-83).
- Versões fixadas exatamente no `package.json`. Atualizar exige rever a licença (lição deste ADR).

## Fontes
- Licença do PrimeUI (`@primeui/license-manager@1.1.0`, `LICENSE.md`): Community License e Commercial License.
- `npm view primevue@4.5.5 license` → MIT; `npm view primevue@5.0.5 license` → "SEE LICENSE IN LICENSE.md".
- vue-echarts README: importação sob demanda com `use()` de `echarts/core`.
