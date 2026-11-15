# Telas e fluxos — Central

Layout: barra lateral fixa (com o **estudo em foco**, A28) + área de conteúdo; responsivo de 1366×768 a 4K; tema claro e escuro; tabelas com busca, filtros, ordenação, paginação, estados vazio, carregando e erro.

| Menu | Conteúdo |
|---|---|
| **Painel** (estatísticas, §22-A) | Sempre do **estudo em foco** (nome e código no cabeçalho). Barra de filtros (busca, jornada, equipe; "Mais filtros": período de cadastro ou recebimento, localização, fotos, área informada, faixas e hectares, condições sobre qualquer quantidade ou tipo de item) com chips e "Limpar filtros", e "N de M famílias". Cartões com detalhamento (famílias, pessoas, veículos, rebanho, aves, eletrodomésticos, móveis, área). Abas: por jornada ou equipe (colunas escolhíveis + gráfico), por família, itens por tipo (predefinidos e "Outros" agrupados), tamanho das propriedades. Todo número, barra ou linha abre a relação das famílias (drill-down) e daí o cadastro; "Filtrar o Painel por estes critérios". Depois: alertas (conflitos, lotes com erro, último backup) |
| **Propriedades** | Mesma barra de filtros do Painel + tabela com totais → **página da propriedade** (faixa de totais; pessoas, rebanho e veículos em gráfico + tabela; bens e criações por categoria; área e localização em texto; fotos ampliáveis; registro e sincronização); depois: histórico e auditoria, "Localização" online, editar |
| **Estudos** *(A28, ADR-0023)* | Lista com busca (nome, código, jornadas, equipes, famílias, última importação) e **Novo estudo** (nome, data da 1ª jornada, equipes iniciais). Página do estudo: código, renomear, pôr em foco, **Nova jornada** (data; copiar nomes de equipes de jornadas anteriores; equipes novas; cada uma com código novo); por jornada, as equipes com o **código grande** e **Copiar**, "Copiar todos os códigos", **+ Equipe** e excluir equipe sem cadastros |
| **Confirmar destino da sincronização** *(modal global, ADR-0024)* | Aparece em qualquer tela quando um lote chega: aparelho, código recebido, cadastros novos/atualizados/excluídos, fotos e o destino pelo código (estudo, jornada, equipe, código oficial) → **Confirmar e importar** · **Alterar destino** · **Cancelar sincronização**. Código inexistente: "Código de equipe não encontrado" → **Corrigir no Central** (Estudo → Jornada → Equipe, com o código oficial) ou cancelar. Coleta já importada: destino travado. Fila "1 de N" |
| **Fotos** | Galeria filtrável por propriedade e equipe |
| **Sincronização** | Passos e QR de pareamento do aparelho (botão bloqueado enquanto gera), estado da recepção, histórico de lotes (código recebido, destino, corrigido, situação: recebendo, aguardando confirmação, importado, cancelado) com recusas; IPs, porta, impressão digital e pasta ficam em **Informações técnicas** (modal no rodapé da barra lateral); depois: diagnóstico de rede e firewall, "Restaurar equipe" |
| **Importar pacote** | Selecionar `.s7pkg` → validação → resultado → recibo (QR e arquivo) |
| **Conflitos** | Fila; tela de três colunas (Base / Central / Android) |
| **Estatísticas** | Barra de filtros (chips + "Limpar filtros"), cartões, tabela analítica (linhas e colunas escolhíveis), gráficos, drill-down. **Sempre offline** |
| **Cartas-imagem** *(A27; A30, ADR-0027)* | Carta individual (também pela página da propriedade) ou geral (filtros do Painel). A imagem é escolhida e preparada **automaticamente** (INPE: CBERS-4A 2 m, Sentinel-2 10 m; mosaico sinalizado), com progresso real e cancelamento; "Atualizar melhor imagem". Mapa com tiles locais, pinos com número fixo, tooltip (nome e responsável), cartão ao clicar e grupos de pinos sobrepostos; folhas tracejadas. Carta: geral + setores automáticos, PDF multipágina e PNG por folha, guardada e reaberta sem internet. Fonte de reserva (Copernicus) opcional. Mensagem RN-91 sem internet. Detalhes: [cartografia.md](cartografia.md) |
| **Relatórios** | **Feito (A26, ADR-0021):** botão "Exportar" no Painel e em Propriedades: PDF, Word (.docx) ou Excel (.xlsx) com os filtros ativos (panorama, por jornada e equipe, itens, tamanhos, por família e, opcionalmente, uma ficha por família); "Exportar ficha" na página da propriedade (PDF ou Word). Depois: carta-imagem individual e geral *(online)* |
| **Backup** | Fazer backup, verificar, restaurar (assistente), histórico |
| **Configurações** | Nome do operador, pasta de dados, porta, provedor cartográfico e chave |
| **Auditoria** | Consulta por entidade, período e origem |

## Estatísticas: `FiltroEstudo`
Dimensões (A28): estudo em foco · jornada(s) · equipe(s) · propriedade · categoria e tipo de item (incluindo grupos de "Outros") · presença/ausência de tipo · faixas numéricas por categoria (p. ex. bovinos > 5) · tem GPS · tem foto · tem área · faixa de área (ha).
O mesmo objeto alimenta cartões, tabelas, gráficos, lista, mapa e exportações (RN-81). Cada número tem drill-down para a lista de propriedades que o compõem.

## Cruzamentos (lista derivada, A25)
**Substituída em 14/11 pela especificação do usuário: [`docs/requirements/estatisticas.md`](../../docs/requirements/estatisticas.md) (§22-A), que é o critério de aceite da Fase 4.** A lista abaixo continua coberta por ela.

**Totais:** sempre há o **geral**, **por equipe** e **por família** (cada propriedade é uma família).

**Filtros combináveis (`FiltroEstudo`):**
- estudo (em foco), jornada(s), equipe(s);
- período (cadastro no celular ou recebimento na Central);
- tem GPS, tem foto, tem área, faixa de área (ha);
- categoria e tipo de item, incluindo os grupos de "Outros" (aliases);
- presença ou ausência de um tipo (ex.: sem geladeira);
- faixa numérica por categoria (ex.: bovinos > 5).

**Cruzamentos:**
- pessoas por faixa etária × equipe;
- rebanho (bovinos adultos, bezerros, equinos adultos, potros, aves) × equipe;
- veículos por tipo × equipe;
- itens (eletrodomésticos, móveis, aves, "Outros") × equipe;
- área total e média × equipe, e distribuição por faixas de área;
- cobertura de GPS, foto e área × equipe;
- propriedades sem o item X; propriedades com N acima de um limite.

Sempre: drill-down até o cadastro e só registros vigentes (RN-80..82).

## Identidade visual (A25, ADR-0019)
O assunto é um levantamento de campo da PM em linhas vicinais de Rondônia. As referências vêm desse mundo: verde da mata (marca), terra de laterita, azul dos rios e o amarelo do ipê. O ipê é o **único acento**: marca o item ativo do menu.

- **Tokens** em `frontend/src/estilo.css`, com tema claro e escuro (o escuro segue o Windows):
  - superfícies `--papel`, `--folha`, `--folha-2`; réguas `--regua` e `--regua-forte`;
  - texto `--tinta`, `--tinta-2`, `--tinta-3`;
  - marca `--mata`; barra lateral `--lateral`;
  - estados `--ok`, `--alerta`, `--erro`.
- **Tipografia:** Archivo variável, empacotada, uma família só.
  - Títulos de página largos (`font-stretch` 112%), como placa de estrada.
  - Números de destaque condensados (78%) e tabulares.
  - Texto a 15 px.
- **Elemento de destaque:** a **faixa de totais** (`FaixaTotais`), no estilo de caderneta de campo. São grupos separados por réguas verticais, com uma marca curta da cor do assunto. O resto é plano: blocos com borda, sem sombra.
- **Tabelas** (`TabelaDados`):
  - texto e cabeçalho centralizados (A26), divisórias entre linhas **e entre colunas**, cabeçalho fixo, ordenação acessível (`aria-sort`), paginação e linha de totais;
  - linhas abertas com Enter;
  - abaixo de 1000 px de largura, as colunas secundárias saem; abaixo de 640 px, cada linha vira um bloco "rótulo: valor" (exceto em tabelas `compacta`).
- **Gráficos** (`Grafico` + `graficos.ts`, ECharts):
  - a cor segue o assunto em toda a Central: eletrodomésticos `--s1` (mata), pessoas `--s2` (rio), rebanho `--s3` (laterita), veículos `--s4` (açaí), móveis `--s5` (ipê), aves `--s6` (urucum);
  - barras finas com ponta arredondada, valor na ponta na cor do texto, grade tracejada discreta, tooltip;
  - um gráfico de uma série não tem legenda;
  - **todo gráfico tem tabela ao lado** com os mesmos números.
- **Paleta validada** com `validate_palette.js` (daltonismo e contraste):
  - claro em `#ffffff`: `#2f7d4a, #2b6fc0, #d9692e, #7a4fa8, #d39200, #c94f86`. Pior par: ΔE 18,2 com daltonismo, 20,1 sem. O ipê fica abaixo de 3:1, então o valor vai sempre escrito na ponta da barra e na tabela.
  - escuro em `#18201b`: `#3f9a5f, #4a8ad8, #d06a34, #9b7fd6, #b8860b, #d8669a`. Todos acima de 3:1.
- **Modo de demonstração:** `npm run demo` sobe o frontend no navegador com dados fictícios (`src/demo`, só no modo `demo`) para revisar o visual sem o Windows.

## Central 0.7.0-mvp (A29)
Implementação e critérios em [ADR-0026](../../docs/architecture/decisions/0026-correcoes-e-exclusoes-central.md). Tabelas com 5 registros/página e totais externos. Editor por seções e exclusão lógica em cascata; fotos/recibos/cartas preservados. Novas versões de cadastros corrigidos são recusadas temporariamente, inclusive itens/fotos isolados. Nenhum dado é alterado no celular. Testes de fluxo em `internal/nucleo/correcoes_test.go`; validar ações/modais, download e relatórios no Windows.

Ajuste 0.7.1-mvp: Ação é a última coluna em Propriedades; ícones permanecem lado a lado, com 8 px de intervalo (também nos estudos, jornadas e equipes). Nome da propriedade ocupa 18%; ações têm largura reservada.

Ajuste 0.7.2-mvp: Ação também é a última coluna da lista principal de Estudos de situação, seguindo Propriedades, Jornadas e Equipes.
