# Cartas-imagem (Sentinel-2 / Copernicus)

Decisões: **A27** e **[ADR-0022](../../docs/architecture/decisions/0022-cartas-imagem-sentinel2-cdse.md)** (altera RN-90), sobre o [ADR-0013](../../docs/architecture/decisions/0013-cartografia-plugavel.md).

## O que faz
O operador escolhe as propriedades: uma, pela página dela ("Carta-imagem"), ou várias, pelos mesmos filtros do Painel. Depois busca as datas de imagem Sentinel-2 disponíveis para a área, ajusta o enquadramento na prévia e gera a carta em **PDF** (A4 paisagem; A3 com mais de 25 propriedades) e **PNG**.

As cartas e as imagens consultadas ficam na pasta de dados (`cartografia/`), **sem validade**, e reabrem sem internet.

## Arquitetura
| Parte | Onde | Papel |
|---|---|---|
| Geoespacial | `internal/cartografia/geo` | Validação de coordenadas, Web Mercator (EPSG:3857), enquadramento 4:3, `Pixel` (posição do pino), escala (só válida se variar < 1%), barra de escala, agrupamento de pontos distantes |
| Provedor | `internal/cartografia` (`Provedor`) + `internal/cartografia/cdse` | OAuth2 `client_credentials` (token só em memória, renovado 1 min antes de vencer), Catalog API (dias de passagem com nuvens), Process API (PNG em EPSG:3857, cores reais + `dataMask` no alfa) |
| Armazém | `internal/cartografia/armazem.go` | Imagens por chave (área, tamanho, data/período, versão do script) e cartas (`cartas/<id>/carta.pdf`, `carta.png`, `carta.json`); gravação atômica; "Limpar imagens" não apaga cartas |
| Carta | `internal/cartografia/carta` | Mapa (imagem + pinos + norte + escala + atribuição) desenhado uma vez e usado no PNG e no PDF; quadro de informações; tabela de coordenadas |
| Credenciais | `internal/credenciais` | Gerenciador de Credenciais do Windows (`advapi32`), alvo `SITPOSSE-7/Copernicus-CDSE` |
| App | `app/cartas.go` (`CartaService`) | Planejar, passagens, prévia, gerar, cartas salvas; rota local `/cartografia/` (só imagens e PNG das cartas, nomes validados) |
| Tela | `frontend/src/telas/Cartas.vue`, `componentes/MapaCarta.vue`, `ConfigCopernicus.vue` | Leaflet 1.9.4 sem mapa-base de terceiros; imagem e pinos em camadas separadas |

**O mesmo pino na tela e no arquivo:** a imagem é pedida em EPSG:3857, a projeção do Leaflet. Na tela, o Leaflet projeta a coordenada; no arquivo, `geo.Pixel` faz a mesma conta. O teste `TestPinoNoPixelDaProjecao` confere o centro de cada pino desenhado.

**Data da imagem:** a sugestão é a passagem de menos nuvens entre as dos últimos 45 dias com até 20% de nuvens; senão, a de menos nuvens de todas. A mais recente não é presumida a melhor. Com uma passagem única, a carta cita a data; com mosaico, cita o período. Imagem com área sem dados (mais de 0,05%) não gera carta (RN-93).

**Limites de zoom e área:** a prévia vai até o zoom 16 (≈ 2,4 m por pixel de tela). Acima disso, cada pixel de 10 m do Sentinel-2 viraria um borrão (visto no teste real de 14/11). Nenhuma imagem é pedida com menos de 1,5 km de largura (`geo.GarantirLargura`). O enquadramento automático só é aplicado depois que o mapa tem tamanho real (`ResizeObserver` + `invalidateSize`).

**Pontos distantes:** se a carta geral passar de 20 m por pixel, a tela propõe dividi-la em grupos de até 40 km.

## O que sai para a internet
Só para o Copernicus: o token (identificador e segredo do cliente), o retângulo da área em EPSG:3857 (ou em graus, no catálogo), o período e os parâmetros da imagem. **Nenhum dado de cadastro** (teste `TestNadaDeCadastroSaiParaAInternet`).

## Cadastro gratuito (uma vez)
1. Criar uma conta em `dataspace.copernicus.eu` (de preferência com e-mail institucional).
2. Abrir `shapps.dataspace.copernicus.eu/dashboard` → **User Settings → OAuth clients → Create**, dar um nome e escolher a validade.
3. Copiar o **Client ID** e o **Client secret** (o segredo aparece uma vez só).
4. Na Central: **Cartas-imagem → Configurar acesso** → colar → **Salvar e testar**.

Não contratar planos pagos. A cota gratuita é de 10.000 unidades e 10.000 requisições por mês, e 300 por minuto; cada carta de 2000 × 1500 px custa cerca de 11 unidades. Quando a cota acaba, o serviço responde 429 e a tela informa quando tentar de novo.

## Limitações (explicadas na carta)
- **Resolução:** 10 m por pixel. Aproximar a imagem não aumenta a resolução.
- **O pino:** é a coordenada coletada em campo, com a precisão do GPS, e não confirma visualmente a construção.
- **Barra de escala:** só aparece quando é tecnicamente correta (RN-92).

## Testes
| Teste | O que garante |
|---|---|
| `geo_test.go` | Projeção com valores independentes (Porto Velho), ordem lat/lon, pino ↔ pixel nas duas direções, norte para cima, não é interpolação linear em graus, escala/barra, carta continental sem barra, agrupamento determinístico |
| `cdse_test.go` (servidor falso) | Token reaproveitado e renovado, sem credenciais, credenciais recusadas, cota (429 + `Retry-After`), sem internet, queda no meio do download, resposta ilegível, imagem com área sem dados, limite de tamanho, só área/período/parâmetros enviados, dia da passagem única |
| `armazem_test.go` | Imagens e cartas sobrevivem ao reinício; nomes maliciosos recusados; limpar não apaga cartas; carta interrompida não aparece |
| `carta_test.go` | Pino desenhado em `geo.Pixel`; PNG e PDF (individual, geral, geral grande em A3); data × período; campos obrigatórios |
| `credenciais_test.go` | Gravar, ler (com acentos) e apagar; rodado também no Gerenciador de Credenciais real do Windows (`GOOS=windows go test -c`) |
| `app/cartas_test.go` (Central real + provedor falso) | Sem localização listada à parte, prévia guardada sem gastar cota de novo, carta salva reaberta após reiniciar, rota local segura, imagem incompleta e ponto fora do enquadramento recusados, sem credenciais, pontos distantes agrupados |

**Falta o teste com o Copernicus real:** depende da conta do usuário (§ Cadastro gratuito).
