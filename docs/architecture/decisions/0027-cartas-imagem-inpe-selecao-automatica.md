# ADR-0027 — Cartas-imagem com imagens do INPE, seleção automática, mosaicos e setores

- **Status:** Aprovado pelo usuário em 15/11/2026 (plano A30)
- **Data:** 2026-11-15
- **Altera:** ADR-0022 (provedor único CDSE, escolha manual da data, imagem única por carta) e ADR-0013 (provedor de imagens); RN-93 reescrita

## Contexto
A carta-imagem da A27 usa só o Sentinel-2 de 10 m pelo Copernicus (CDSE). O operador escolhe período, data e nuvens, e a carta tem uma única folha. A qualidade visual fica muito abaixo do esperado, e áreas extensas viram uma carta ilegível.

Verificado em 15/11/2026 no catálogo STAC do INPE (`https://data.inpe.br/bdc/stac/v1/`), sobre Ariquemes (−10,075; −63,169):
- **`CB4A-WPM-PCA-FUSED-1`** — CBERS-4A/WPM fusionado (PCA), RGB de 2 m.
  - 20 cenas sobre o ponto entre 2023 e 2026 (a mais recente é de 13/10/2026).
  - Formato: COG BigTIFF com deflate, blocos de 512 px, overviews, UTM e `nodata = 0`.
  - Licença CC BY 4.0. Não exige login e aceita leitura por faixa (HTTP `Range`).
  - **Sem metadado de nuvens.** As cores brutas vêm com névoa e um tom arroxeado.
- **`S2_L2A-1`** — Sentinel-2 L2A em COG.
  - Ativos `TCI` (cores reais, 10 m) e `SCL` (classificação de cena, 20 m), com `eo:cloud_cover`.
  - Termos do Copernicus. Não exige login.
- **`mosaic-s2-brazil-3m-1`** — mosaico trimestral do Sentinel-2 sobre o Brasil, feito pelo INPE.
  - 10 m, em EPSG:4326, a partir do cubo LCF de 16 dias.
  - O metadado de licença diz "proprietary". Uso com crédito ao INPE e ao Copernicus.

Num recorte de 1,6 km, o CBERS de 2 m mostrou árvores isoladas, estradas e açudes que o Sentinel-2 não mostra.

## Decisão
1. **Fontes, em ordem de preferência:**
   - **CBERS-4A WPM fusionado (2 m)**, com até **12 meses**;
   - **Sentinel-2 L2A (10 m)** mais recente sem nuvens na área;
   - **mosaico trimestral do Sentinel-2 do INPE**, usado como referência de cor e como último recurso de cobertura;
   - o **CDSE** (Process API) continua configurável e só é usado automaticamente quando o INPE está inacessível.
2. **Seleção automática e reproduzível.**
   - **Unidade de avaliação:** células de cerca de 250 m sobre a área de interesse, que é a união dos setores da carta, não o retângulo inteiro.
   - **Filtros eliminatórios:** área sem dados e nuvem ou sombra sobre a célula.
   - **Pontuação:** resolução, nuvem, névoa e idade, com penalidade por cena adicional.
   - **Escolha:** gulosa e determinística, com desempate por pontuação, data e id.
   - **Configuração:** pesos e limiares ficam num único struct (`selecao.Criterios`).
   - **Nuvem no Sentinel-2:** é medida pela SCL.
   - **Nuvem no CBERS:** é **estimada** pela comparação com a referência sem nuvens e aparece como "estimativa".
   - **Reprodutibilidade:** a escolha fica gravada com os ids das cenas.
3. **Mosaicos permitidos e sinalizados** (RN-93 reescrita, RN-97): uma carta pode juntar cenas de datas e resoluções diferentes. A folha traz o quadro de proveniência (fonte, data e resolução de cada parte) e nunca apresenta um mosaico como imagem de uma data só. Área sem nenhuma imagem continua bloqueando a carta.
4. **Processamento próprio, em Go puro, sem GDAL.**
   - Leitor COG por faixa (TIFF/BigTIFF com deflate, 8 bits RGB/cinza, blocos e overviews).
   - Reprojeção UTM ou EPSG:4326 → EPSG:3857.
   - Harmonização de cor do CBERS por transformação afim ajustada contra a referência Sentinel-2, com ajuste por percentis como alternativa.
   - Costura suavizada entre cenas.
   - Pirâmide de tiles JPEG local para a prévia.
   - Super-resolução por IA não é usada.
5. **Setores.** A carta tem sempre uma **folha geral**. Quando a geral fica com muito menos detalhe que a imagem, ou com pinos sobrepostos, são acrescentadas **folhas de setor** calculadas automaticamente. Saída: um PDF com todas as folhas e um PNG por folha.
6. **Número fixo da propriedade no estudo** (RN-96). É atribuído quando o cadastro chega à Central, nunca é reutilizado e é o mesmo no mapa, na legenda e em todas as folhas.
7. **Armazenamento** (continua o ADR-0022):
   - blocos COG baixados, composições (pirâmides de tiles) e cartas ficam na pasta de dados, sem validade;
   - "Limpar imagens guardadas" apaga blocos, composições e imagens brutas, nunca as cartas;
   - só a área consultada pelo operador é baixada, e o mapa-base offline continua proibido.
8. **Privacidade** (AGENTS.md §8): ao INPE vão apenas o retângulo da área (busca STAC) e pedidos de faixas de bytes dos arquivos públicos. Nenhum dado de cadastro.

## Consequências
- A carta deixa de precisar de credenciais. O Copernicus vira reserva opcional.
- Mais uso de disco: cerca de 10 a 60 MB por estudo, com limpeza manual.
- O leitor COG é código próprio, coberto por testes com COGs sintéticos e um teste regional opcional com rede.
- A nuvem no CBERS é estimativa: os filtros são conservadores e o operador sempre vê fonte e data.
- O INPE vira dependência externa, mitigada pela reserva CDSE e pelas composições guardadas.

## Complemento (15/11/2026, A31): referências atualizadas
**Problema:** a coordenada de uma propriedade pode mudar depois do preparo da imagem (edição na Central ou nova sincronização). A tela de cartas fica montada em memória, e a carta usava as coordenadas novas sobre a imagem preparada para as antigas: a tela mostrava outro lugar, e um pino que saísse do seu setor sumia das folhas de setor sem aviso.

**Decisão (RN-98):**
- **Composição:** grava a chave da área para a qual foi preparada (`chave` em `composicao.json`). Nas composições anteriores, que não têm o campo, vale o índice `composicoes/areas/`.
- **Recusa:** `GerarCarta` recusa a composição que não corresponde às coordenadas atuais.
- **Exceção sem internet:** a recusa não vale para a imagem anterior aceita sem internet que ainda cobre todos os pinos (folha geral e algum setor).
- **Tela:** relê as propriedades ao voltar e antes de gerar, compara pelo UUID com os pinos usados no preparo e oferece "Atualizar referências".
- **Sem internet:** a última composição de cada seleção (tipo + propriedades) fica lembrada em `composicoes/selecoes/`. Ela é usada quando as coordenadas mudaram, a busca falhou e a composição ainda cobre todos os pinos.
- **Cartas salvas:** não são regeneradas nem sobrescritas.

**Sem mudança de banco, de protocolo ou da chave de área:** as composições guardadas continuam sendo reaproveitadas.

## Alternativas consideradas
- **GDAL:** exige DLLs nativas e cgo, aumenta o instalador e complica a distribuição no Windows.
- **Só o Sentinel-2 com melhor tratamento de cor:** continua com 10 m e sem os detalhes necessários.
- **Planet/NICFI ou Brasil MAIS:** exigem convênio ou conta, e o NICFI foi encerrado. Fica registrado como possibilidade institucional.
- **Google ou Esri:** proibidos pelos termos de uso.
- **Numeração por carta, em ordem alfabética:** muda quando os filtros mudam. Descartada pelo usuário.

## Fontes
- https://data.inpe.br/bdc/stac/v1/collections/CB4A-WPM-PCA-FUSED-1
- https://data.inpe.br/bdc/stac/v1/collections/S2_L2A-1
- https://data.inpe.br/bdc/stac/v1/collections/mosaic-s2-brazil-3m-1
- https://creativecommons.org/licenses/by/4.0/
- https://sentinels.copernicus.eu/documents/247904/690755/Sentinel_Data_Legal_Notice
- https://brazil-data-cube.github.io/specifications/input-data.html
