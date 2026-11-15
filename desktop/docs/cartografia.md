# Cartas-imagem (INPE CBERS-4A / Sentinel-2; Copernicus como reserva)

Decisões:
- **A30** e **[ADR-0027](../../docs/architecture/decisions/0027-cartas-imagem-inpe-selecao-automatica.md)**, que alteram a A27 e o [ADR-0022](../../docs/architecture/decisions/0022-cartas-imagem-sentinel2-cdse.md);
- [ADR-0013](../../docs/architecture/decisions/0013-cartografia-plugavel.md);
- RN-90 a RN-97.

## O que faz
1. **Escolha das propriedades:** o operador escolhe uma propriedade (pela página dela, "Carta-imagem") ou várias, pelos mesmos filtros do Painel.
2. **Preparo da imagem:** a Central prepara **sozinha** a imagem da área:
   - busca as cenas no INPE;
   - avalia cada uma sobre a área das propriedades;
   - escolhe a melhor composição: CBERS-4A de 2 m até 12 meses, depois Sentinel-2 de 10 m, depois o mosaico trimestral;
   - harmoniza as cores;
   - gera a prévia e as folhas.

   Se a mesma área já foi preparada, a composição guardada abre na hora e sem internet. **Atualizar melhor imagem** refaz a busca.
3. **Mapa:**
   - **Tiles e enquadramento:** mostra os tiles locais da composição e abre sempre no enquadramento das propriedades (há o botão "Ajustar às propriedades").
   - **Pinos:** têm o **número fixo do estudo** (RN-96). Passar o mouse mostra nome e responsável; clicar abre o cartão (nº, nome, responsável), que fecha no ×, no Esc ou ao clicar no mapa.
   - **Pinos sobrepostos:** viram um selo com a contagem, cujo cartão lista todas as propriedades.
   - **Folhas da carta:** aparecem tracejadas (geral e setores).
4. **Gerar carta:**
   - **PDF:** a folha geral (com o índice dos setores), uma folha por setor e a tabela de coordenadas.
   - **PNG:** um por folha (`carta-01-geral.png`, `carta-02-setor-A.png`…).
   - Tudo fica guardado em `cartografia/cartas/<id>/` e abre sem internet.

## Fluxo e componentes
| Parte | Onde | Papel |
|---|---|---|
| Plano | `app/cartas.go` (`PlanejarGeral`, `PlanejarIndividual`) | Propriedades com número fixo, nome, responsável e coordenada; sem localização; coordenadas suspeitas (fora do Brasil ou a mais de 50 km da mediana) |
| Preparo | `app/cartas.go` (`Preparar`, `Andamento`, `Cancelar`) → `composicao.Servico.Preparar` | Tarefa em segundo plano com fases e contadores reais ("Buscando imagens disponíveis", "Avaliando a qualidade das imagens", "Preparando a imagem da região"), bytes baixados e cancelamento |
| Carta | `app/cartas.go` (`GerarCarta`) + `carta/` | Folhas a partir das imagens guardadas da composição; recusa folha com área sem imagem (RN-93); quadro de proveniência (fonte, data, resolução e % de cada parte; "mosaico"; nuvem estimada) |
| Rotas locais | `HandlerCartografia` | `/cartografia/composicoes/<id>/<z>/<x>/<y>.jpg` (tile gerado e guardado na 1ª vez; fora da composição, PNG transparente), PNG das cartas e imagens antigas; nomes validados por expressão regular |
| Tela | `telas/Cartas.vue`, `componentes/MapaCarta.vue`, `CartaoPropriedade.vue`, `src/cartas.ts` | Leaflet com duas camadas de tiles locais (geral até o zoom da carta geral; detalhe nos setores até a resolução nativa); `agruparPinos` (testado no Vitest); textos do cadastro nos tooltips sempre como texto, nunca HTML |
| Reserva | `cdse/` + `ConfigCopernicus.vue` ("Fonte de reserva (opcional)") | Só quando o INPE não responde e há credenciais; sem credenciais vale a mensagem da RN-91 |
| Armazém | `armazem.go` | Cartas (`carta.pdf` + PNGs; as da A27 continuam abrindo); "Limpar imagens guardadas" apaga imagens brutas, `blocos/` e `composicoes/`, nunca as cartas |

**Enquadramento:** a carta geral cobre todos os pontos, com margem de 8% da extensão (mínimo de 300 m) e proporção 4:3.

**Setores:** são criados quando a geral passa de 2,5 × a resolução da imagem por pixel ou tem pinos sobrepostos que um setor separa.
- **Escala:** todos têm a mesma escala (1,5 × a resolução: 3 m/px com 2 m; 15 m/px com 10 m).
- **Montagem:** são determinísticos; grupos distantes viram setores próprios.
- **Imagens:** a área buscada é só a união dos setores, mais a geral em baixa resolução.

**Mesmo pino na tela e no arquivo:** tudo é EPSG:3857; o Leaflet e `geo.Pixel` fazem a mesma conta (`TestPinoNoPixelDaProjecao`).

## Imagens do INPE, seleção e composição (A30, ADR-0027)
| Parte | Onde | Papel |
|---|---|---|
| UTM e setores | `geo/utm.go`, `geo/setores.go` | Krüger 6ª ordem (ida e volta); carta geral + setores de escala fixa (2 m → 3 m/px; 10 m → 15 m/px), determinísticos |
| Leitor COG | `cog/` | TIFF/BigTIFF em blocos, deflate, preditor 2, overviews, GeoKeys UTM/4326; HTTP Range com cache em disco (`blocos/`, por URL + ETag); tabelas de blocos enormes lidas por pedaços (mosaico do Brasil: 13,7 milhões de blocos) |
| Catálogo | `stac/`, `inpe/` | STAC do INPE: CBERS-4A WPM fusionado (≤ 12 meses), Sentinel-2 L2A (TCI + SCL), mosaico trimestral |
| Raster | `raster/` | Leitura numa grade 3857 (coordenada exata a cada 16 px, interpolada entre eles; bilinear), SCL como máscara, cor por ajuste afim robusto contra o mosaico ou casamento de percentis, nuvem do CBERS **estimada** pelo resíduo do brilho, composição por célula com costura de ≈ 40 m, tom único, tiles XYZ |
| Seleção | `selecao/` | Critérios num só struct; elegibilidade por célula (≤ 0,5% sem dados, ≤ 2% nuvem); regra hierárquica: nível de resolução, depois pontuação; desempate determinístico |
| Composição | `composicao/` | Orquestra tudo; refaz os setores na escala de 10 m quando não há 2 m; guarda `composicoes/<id>/` (receita, folhas, tiles) e reabre offline pela área; reserva CDSE |

**Calibração em Ariquemes (15/11/2026, teste `SITPOSSE7_REDE=1`):**
- **Cenas usadas:** CBERS-4A de 13/10/2026 em 94–96% da área; Sentinel-2 nas células com nuvem estimada.
- **Primeiro preparo, sem cache:** 5 propriedades levaram cerca de 13 s e 24 MB; 20 propriedades em 2 × 10 km (4 setores) levaram 38 s e 89 MB.
- **Ajuste de cor afim:** no CBERS, o ajuste afim irrestrito chegou a R² = 0,69, mas apagava o detalhe (soma dos ganhos de 0,17) e escondia nuvens. Por isso a transformação exige soma dos ganhos ≥ 0,6, e nessa cena valeu o casamento de percentis.

## O que sai para a internet
- **INPE:** a pesquisa STAC (retângulo, coleções e período) e pedidos de faixas de bytes dos arquivos públicos de imagem.
- **Copernicus (só como reserva):** token, retângulo, período e parâmetros.
- **Nenhum dado de cadastro** (AGENTS.md §8).

## Fonte de reserva (Copernicus, opcional)
1. Criar uma conta em `dataspace.copernicus.eu` (de preferência com e-mail institucional).
2. Abrir `shapps.dataspace.copernicus.eu/dashboard` → **User Settings → OAuth clients → Create**, dar um nome e escolher a validade.
3. Copiar o **Client ID** e o **Client secret** (o segredo aparece uma vez só).
4. Na Central: **Cartas-imagem → Fonte de reserva (opcional)** → colar → **Salvar e testar**.

Não contratar planos pagos. Cota gratuita: 10.000 unidades e 10.000 requisições por mês, e 300 por minuto.

## Limitações (explicadas na carta)
- **Resolução nominal:** 2 m (CBERS-4A) ou 10 m (Sentinel-2). Aproximar não aumenta a resolução, e a resolução nominal não garante identificar objetos.
- **Nuvem no CBERS-4A:** é **estimada** pela Central (a fonte não traz máscara). As células com nuvem estimada usam outra imagem; nuvens pequenas abaixo do limite podem aparecer.
- **Mosaico:** junta datas e fontes diferentes, e a carta diz isso (não é uma fotografia de uma data só).
- **O pino:** é a coordenada coletada em campo, com a precisão do GPS, e não confirma visualmente a construção.
- **Barra de escala:** só aparece quando é tecnicamente correta (RN-92).

## Testes
| Teste | O que garante |
|---|---|
| `geo/*_test.go` | Projeção, pino ↔ pixel, escala; UTM contra referências; setores (5, 20 em 2 × 10 km, 100 em 10 × 30 km, grupos distantes, repetidos, inválidos, determinismo) |
| `cog/`, `stac/`, `inpe/` | Leitor COG (sintéticos, Range, cache, offline, tabelas sob demanda, recusas), STAC paginado e instável, candidatas filtradas e deduplicadas; real opcional (`SITPOSSE7_REDE=1`) |
| `raster/`, `selecao/` | Reprojeção preserva a posição, SCL, cor, costura, buracos, tiles; seleção hierárquica, retalhos, células limitadas ou sem imagem, independência da ordem |
| `composicao/` | Ponta a ponta com servidor falso (escolha, folhas, tiles, reabrir offline, reusar sem internet); real opcional em Ariquemes |
| `carta/` | Pino em `geo.Pixel`; PNG e PDF; carta com várias folhas, proveniência do mosaico, índice de setores |
| `app/cartas_test.go` | Plano com número fixo e responsável, preparo assíncrono, carta com setores (arquivos por folha), reinício offline, rotas seguras (tiles, PNG), limpar imagens, reserva CDSE (inclusive incompleta), sem reserva = RN-91, cancelamento |
| `frontend/src/cartas.test.ts` | Agrupamento de pinos (sem perder nenhum, determinístico, coordenadas iguais) e escolha do PNG da carta |
| `cdse_test.go`, `armazem_test.go`, `credenciais_test.go` | Como na A27 |

Revisão visual: `npm run demo` → Cartas-imagem (preparo simulado, tooltip, cartão, grupos, setores).
