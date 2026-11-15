# ADR-0022 — Cartas-imagem com Sentinel-2 (Copernicus CDSE) e imagens guardadas localmente

- **Status:** Aprovado
- **Data:** 2026-11-14
- **Substitui / Substituído por:** altera RN-90, AGENTS.md §6 e §8 e o ADR-0013 (cache só em memória); resolve a P1 para imagem de satélite

## Contexto
O usuário especificou as cartas-imagem (A27): imagens **Sentinel-2 L2A** do **Copernicus Data Space Ecosystem (CDSE)**, gratuitas, com pinos georreferenciados das propriedades, cartas individual e geral, e exportação PDF/PNG. Pediu também:
- guardar localmente as cartas geradas e as imagens baixadas, **sem validade**;
- reabrir as cartas sem internet.

Até agora a RN-90 e o AGENTS.md §6 proibiam qualquer cache persistente de imagens.

Fatos da documentação oficial (consultada em 14/11/2026):
- Cota gratuita do Sentinel Hub no CDSE: 10.000 unidades de processamento (PU) e 10.000 requisições por mês, e 300 por minuto. Uma imagem de 512 × 512 px com 3 bandas custa 1 PU; a `dataMask` não conta.
- Autenticação por OAuth2 `client_credentials` (cliente criado em *User Settings → OAuth clients*; o segredo aparece uma vez só).
- Catálogo (`/catalog/v1/search`) e imagem (`/process/v1`): `mosaickingOrder` `mostRecent`/`leastRecent`/`leastCC` e `maxCloudCoverage` por tile.
- Dados Sentinel: acesso livre, completo e aberto. Atribuição: "Contains modified Copernicus Sentinel data [ano]".

## Decisão
**Aprovada pelo usuário em 14/11/2026.**
1. **Provedor de imagem:** CDSE Sentinel-2 L2A, cores reais (B04, B03, B02) + `dataMask`, atrás da interface `Provedor` do ADR-0013. Outros provedores podem ser incluídos depois sem mudar as cartas. **P1 resolvida para imagem de satélite.**
2. **Imagens e cartas guardadas na pasta de dados, sem validade** (altera RN-90):
   - a imagem de uma passagem é imutável; é guardada pela chave (retângulo, data ou período, parâmetros), sem prazo de validade;
   - **só** a área que o operador consultou é guardada; continua **proibido** baixar mapas antecipadamente ou montar mapa-base offline;
   - as cartas geradas (PNG/PDF) ficam guardadas e podem ser reabertas sem internet;
   - a tela mostra o espaço usado e oferece "Limpar imagens guardadas", que apaga só as imagens brutas, nunca as cartas.
3. **O que sai para a internet** (AGENTS.md §8): só o retângulo geográfico (com margem), o período e os parâmetros da imagem. **Nenhum** dado de cadastro (nome, responsável, equipe, quantidades) e nenhuma coordenada isolada de propriedade.
4. **Credenciais** (ID e segredo do cliente OAuth) ficam no **Gerenciador de Credenciais do Windows**, nunca no código, em texto puro, nos documentos ou nos logs. O token fica só em memória. Não é "login em nuvem" do sistema (AGENTS.md §6): é a credencial de serviço do provedor de imagens.
5. **Projeção única Web Mercator (EPSG:3857)** para a imagem pedida, a prévia (Leaflet) e a carta exportada. O pino é calculado pela fórmula da projeção, igual na tela e no arquivo. A barra de escala é calculada na latitude central e só aparece se a variação dentro da carta for menor que 1% (RN-92). O norte fica para cima.
6. **Data da imagem:** por padrão, uma passagem única escolhida pelo catálogo (menos nuvens entre as recentes; o operador pode trocar), o que permite citar a data exata. Num mosaico de várias passagens, a carta cita o **período**. Imagem com área sem dados (`dataMask`) **não gera carta** (RN-93).
7. **Nenhum plano pago:** sem cobrança automática; quando a cota acaba, as requisições falham com 429 e o operador vê uma mensagem clara.

## Dependências
- **Leaflet 1.9.4** (BSD-2), já previsto no ADR-0013; `@types/leaflet` 1.9.22 (MIT, só desenvolvimento).
- **`golang.org/x/image` v0.48.0** (BSD-3, mantido pelo time do Go): texto (fonte Archivo) e formas na imagem da carta. Atualizou `golang.org/x/sys` para v0.50.0.
- Credenciais via `advapi32` com `golang.org/x/sys/windows` (já presente): nenhuma biblioteca nova.

## Alternativas consideradas
- **Validade de 30 dias para as imagens:** descartada pelo usuário; a imagem de uma data não muda.
- **Imagem só em memória (RN-90 original):** cada visualização gastaria cota e exigiria internet, e não seria possível reabrir as cartas offline.
- **Pedir a imagem em EPSG:4326 e posicionar os pinos por interpolação linear:** distorce a imagem frente ao mapa da tela e complica a escala.

## Consequências
- Cresce o uso de disco (cerca de 3 a 6 MB por imagem); há limpeza manual.
- Carta só é gerada com imagem completa; com nuvens, o operador escolhe outra data.
- Resolução de 10 m: a carta explica que o pino é a coordenada coletada (com a precisão do GPS) e que aproximar a imagem não aumenta a resolução.

## Fontes
- https://documentation.dataspace.copernicus.eu/Quotas.html
- https://documentation.dataspace.copernicus.eu/APIs/SentinelHub/Overview/Authentication.html
- https://documentation.dataspace.copernicus.eu/APIs/SentinelHub/Overview/ProcessingUnit.html
- https://documentation.dataspace.copernicus.eu/APIs/SentinelHub/Catalog.html
- https://documentation.dataspace.copernicus.eu/APIs/SentinelHub/Data/S2L2A.html
- https://documentation.dataspace.copernicus.eu/APIs/SentinelHub/Process/Examples/S2L2A.html
- Sentinel Data Legal Notice (ESA)
