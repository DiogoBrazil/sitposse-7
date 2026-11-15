# Estatísticas, filtros e consultas dinâmicas (prompt §22-A)

Texto do usuário, recebido em 14/11/2026. Substitui a "lista derivada" provisória da A25 como **critério de aceite da Fase 4**. Na seção "Atendimento", cada item aponta para a implementação.

> **Regra fundamental:** toda informação quantitativa cadastrada permite visão geral, detalhamento por categoria, por tipo e por propriedade/família, sempre que aplicável.

## 22-A.1 Visão geral
Totais consolidados da área do estudo: propriedades e famílias; pessoas (adultos, jovens, crianças); veículos; eletrodomésticos; móveis; bovinos; equinos; aves; área total em hectares; propriedades com e sem coordenadas. Só registros válidos, sem duplicação por sincronizações sucessivas.

## 22-A.2 Filtros dinâmicos e combináveis
- **Veículos:** total; caminhões; automóveis, caminhonetes e utilitários; motocicletas e ciclomotores; por propriedade/família; por equipe; propriedades com determinado tipo.
- **Pessoas:** total; adultos, jovens e crianças; por propriedade; por equipe ou área identificada.
- **Animais:** total por categoria; bovinos adultos e bezerros; equinos adultos e potros; aves por tipo, incluindo personalizadas; por propriedade.
- **Eletrodomésticos e móveis:** total; por tipo; por propriedade; itens predefinidos e personalizados.
- **Propriedades:** nome; responsável; equipe; identificação da área; com/sem coordenadas; faixas de tamanho; presença ou ausência de bens ou animais.

## 22-A.3 Cruzamento de filtros (exemplos)
1. Quantos caminhões existem na área levantada pela Equipe Alfa?
2. Quais propriedades possuem bovinos e quantos animais há em cada uma?
3. Quantas crianças existem nas propriedades com mais de cinco bovinos?
4. Quantas motocicletas existem em determinada área do estudo?
5. Quais propriedades possuem fotografias e coordenadas?
6. Qual a área total das propriedades cadastradas por determinada equipe?
7. Quais famílias possuem determinado tipo de eletrodoméstico?

Voltar à visão geral removendo os filtros deve ser fácil.

## 22-A.4 Comportamento da interface
Indicadores, tabelas e gráficos acompanham os filtros ativos. Os filtros ativos ficam visíveis, e uma ação simples limpa todos. Ao escolher uma categoria ou indicador, abre a relação das propriedades que compõem o total, e daí o cadastro individual. Não se limitar a cartões com totais.

## 22-A.5 Consistência
Só registros persistidos e consolidados; sem excluídos, sem versões anteriores, sem duplicar itens. Cartões, gráficos, tabelas e relatórios coerentes entre si. Novos filtros devem poder ser incluídos sem reestruturar o sistema.

## Atendimento
| Item | Como | Onde |
|---|---|---|
| 22-A.1 | Totais calculados de um único conjunto filtrado, com detalhamento de cada categoria | `desktop/internal/estatisticas`, Painel |
| 22-A.2 | `Filtro` com equipe, área identificada, busca por nome/responsável, período, localização, foto, área (faixas e mínimo/máximo) e **condições** sobre qualquer quantidade, inclusive cada tipo de item (predefinido ou "Outros" agrupado pelo nome normalizado). Os campos vêm de um registro único (`Campos`) | idem |
| 22-A.3 | Fixture `shared/test-fixtures/estatisticas/` com as 7 perguntas e as respostas esperadas, testada em Go | `estatisticas_test.go` |
| 22-A.4 | Barra de filtros com chips e "Limpar filtros"; todo número, barra e linha abre a relação das propriedades (drill-down) e daí a página da propriedade | Painel |
| 22-A.5 | Só as visões vigentes (RN-80); um único cálculo alimenta cartões, gráficos e tabelas (RN-81); novo filtro = nova entrada no registro de campos | idem |
