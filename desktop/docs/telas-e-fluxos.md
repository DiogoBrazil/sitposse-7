# Telas e fluxos — Central

Layout: barra lateral fixa + área de conteúdo; responsivo de 1366×768 a 4K; tema claro e escuro; tabelas com busca, filtros, ordenação, paginação, estados vazio, carregando e erro.

| Menu | Conteúdo |
|---|---|
| **Painel** | Cartões (propriedades, pessoas, veículos, bovinos, equinos, aves, eletrodomésticos, móveis, área total em ha + "N sem área", com/sem GPS, com/sem fotos), gráficos por equipe e por tipo, alertas (conflitos, lotes com erro, último backup) |
| **Propriedades** | Tabela filtrável → detalhes (dados, itens, fotos, coordenadas em texto, histórico e auditoria, "Localização" online) → editar |
| **Equipes / Operações** | Operação ativa, equipes, aparelhos pareados (revogar), última sincronização |
| **Fotos** | Galeria filtrável por propriedade e equipe |
| **Sincronização** | QR de pareamento, IPs e porta, diagnóstico de rede e firewall, recepção ao vivo, histórico de lotes, "Restaurar equipe" |
| **Importar pacote** | Selecionar `.s7pkg` → validação → resultado → recibo (QR e arquivo) |
| **Conflitos** | Fila; tela de três colunas (Base / Central / Android) |
| **Estatísticas** | Barra de filtros (chips + "Limpar filtros"), cartões, tabela analítica (linhas e colunas escolhíveis), gráficos, drill-down. **Sempre offline** |
| **Mapa** *(online)* | Mapa geral com os mesmos filtros; mensagem RN-91 se offline |
| **Relatórios** | PDF/CSV com os filtros ativos; carta-imagem individual e geral *(online)* |
| **Backup** | Fazer backup, verificar, restaurar (assistente), histórico |
| **Configurações** | Nome do operador, pasta de dados, porta, provedor cartográfico e chave |
| **Auditoria** | Consulta por entidade, período e origem |

## Estatísticas: `FiltroEstudo`
Dimensões: operação · equipe(s) · identificação da área · propriedade · categoria e tipo de item (incluindo grupos de "Outros") · presença/ausência de tipo · faixas numéricas por categoria (p. ex. bovinos > 5) · tem GPS · tem foto · tem área · faixa de área (ha).
O mesmo objeto alimenta cartões, tabelas, gráficos, lista, mapa e exportações (RN-81). Cada número tem drill-down para a lista de propriedades que o compõem.
