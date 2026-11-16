# ADR-0021 — Relatórios em PDF, DOCX e XLSX gerados na Central

- **Status:** Aprovado
- **Data:** 2026-11-14
- **Substitui / Substituído por:** define a "biblioteca PDF (Fase 5)" de `desktop/docs/arquitetura.md` (P9)

## Contexto
O usuário pediu exportar as informações em PDF, planilha XLSX e DOCX (A26): um panorama geral e o detalhamento por família/propriedade. Pediu também:
- PDF e DOCX com o brasão (`logo-cabecalho-docs.png`, na raiz do projeto) e, abaixo dele, "Polícia Militar de Rondônia" e "7º Batalhão de Polícia Militar";
- planilha só com os textos, bonita e interativa.

Tudo precisa funcionar offline (AGENTS.md §6), com versões exatas e licenças compatíveis.

## Decisão
**Aprovada pelo usuário em 14/11/2026** (bibliotecas; tabela + fichas sem fotos).
- **XLSX:** `github.com/xuri/excelize/v2` **v2.11.1** (BSD-3, publicada em 02/11/2026). A planilha tem:
  - uma aba por visão (Panorama, Por equipe, Por área, Por família, Itens por tipo, Tamanho, Sobre), cada uma com o cabeçalho institucional em texto;
  - tabelas do Excel com filtro e ordenação, totais com `SUBTOTAL(109, …)` (somam só as linhas visíveis), painéis congelados, gráficos de barras, links entre as abas e formatos numéricos reais.
- **PDF:** `github.com/signintech/gopdf` **v0.38.4** (MIT, publicada em 31/10/2026), com a fonte **Archivo estática** (OFL-1.1, `internal/fontes/OFL.txt`) embutida no executável. O desenho é próprio:
  - brasão e textos centralizados em todas as páginas;
  - tabelas com cabeçalho repetido a cada página, divisórias e texto centralizado;
  - rodapé "Página X de Y".
- **DOCX:** gerado **só com a biblioteca padrão** (`archive/zip` + WordprocessingML). As bibliotecas DOCX em Go estão paradas desde 2024 (`gomutex/godocx` v0.1.5) ou viraram pagas (unioffice). O documento tem:
  - cabeçalho com o brasão e os textos, rodapé com campos `PAGE`/`NUMPAGES`;
  - estilos Título e Título 1 (navegação do Word);
  - tabelas com `tblHeader` (cabeçalho repetido) e divisórias `insideV`.
- PDF e DOCX saem do **mesmo modelo de documento** (`relatorios.Documento`), então têm o mesmo conteúdo. A largura das colunas garante que nenhuma palavra seja partida ao meio.
- **Conteúdo:**
  - panorama (totais), por equipe e por área identificada, itens por tipo, tamanho das propriedades, tabela por família;
  - **fichas** de cada família, opcionais, sem fotos;
  - a ficha avulsa de uma propriedade sai em PDF ou DOCX pela página dela.

  Os números vêm do mesmo `estatisticas.Resultado` do Painel, **com os filtros ativos** (RF-C07, RN-81), e os filtros são listados no documento.
- **Gravação:** janela "Salvar como" do Windows (fornecida pelo `main.go`; a camada `app` não importa o Wails e continua testável no Linux), arquivo temporário + fsync + rename.

## Dependências transitivas
`richardlehane/mscfb` e `msoleps` (Apache-2.0), `xuri/efp` e `nfp` (BSD-3), `tiendc/go-deepcopy` (MIT), `phpdave11/gofpdi` (MIT), `pkg/errors` (BSD-2). Atualizadas: `golang.org/x/crypto`, `x/net`, `x/term`. Nenhuma acessa a rede.

## Alternativas consideradas
- **PDF pela impressão do WebView2 ("Microsoft Print to PDF"):** passa pela janela de impressão e o layout fica menos controlado.
- **`go-pdf/fpdf`:** sem versão nova desde 2023 (provavelmente arquivada).
- **Bibliotecas DOCX de terceiros:** imaturas ou comerciais.

## Consequências
- Validado no Office real (Windows, via COM): o Word abriu o DOCX sem pedir reparo (19 páginas) e exportou para PDF; o Excel abriu a planilha com 6 tabelas e 6 gráficos, e o `SUBTOTAL` calculou certo.
- Atualizar excelize ou gopdf exige rever a licença (lição do ADR-0019).
- O DOCX usa Calibri, presente no Windows. As fontes não são embutidas no DOCX.

## Fontes
- excelize: `go doc github.com/xuri/excelize/v2` (v2.11.1). gopdf: `go doc github.com/signintech/gopdf` (v0.38.4).
- ECMA-376 (Office Open XML), partes WordprocessingML e Open Packaging Conventions.
- Situação do go-pdf/fpdf (fonte secundária): [gpdf-cloud.pages.dev/blog/go-pdf-fpdf-archived](https://gpdf-cloud.pages.dev/blog/go-pdf-fpdf-archived).
