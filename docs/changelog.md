# Changelog

Formato: data · área · descrição. Mudanças de **esquema** e **protocolo** sempre citam o ADR e a migração.

## 2026-11-15 (A28, E1) — Central: estudos, jornadas, equipes, coletas e confirmação do destino
- **shared** · Contratos **v2** (`shared/schemas/*.v2`, fixtures `contratos/*.v2.json`): lote de uma coleta (`coleta` no lugar de `equipe`), `responsavel_familia` obrigatório, `observacoes_relevantes` (até 4000 caracteres), estado do lote com `aguardando_confirmacao`/`cancelado` e `motivo`, recibo com `destino` (e a linha `destino|…` no `recibo_sha256`), pareamento e QR sem equipe nem operação. Nova fixture `codigos/codigos.v1.json` (Crockford Base32 + dígito Luhn mod 32). Estatísticas: fixture `*.v2` com estudos, jornadas e outro estudo (23 casos). Os `*.v1` de contratos ficam até a E4 (app Campo). `shared/contracts/README.md` corrigido.
- **desktop** · Banco: migração **0002 reescrita** (geração 2, ADR-0025): `estudos`, `jornadas`, `equipes` (código único, nome único na jornada), `coletas` (destino travado por gatilho), `decisoes_destino` (somente acréscimo), `lotes_recebidos` com os estados novos, `propriedades` com `coleta_uuid`, responsável obrigatório e observações, `v_propriedades_vigentes` com equipe, jornada e estudo. **Guarda:** banco do MVP é recusado sem ser alterado; a Central abre só com a mensagem de bloqueio.
- **desktop** · Pacotes novos `internal/codigos` e `internal/estudos`. `pareamento` sem operação/equipe. `importacao`: `importar` deixa o lote **aguardando a decisão**; `Aguardando()` (resumo para o modal: novos, atualizados, excluídos, fotos, destino pelo código, destino travado) e `Decidir()` (confirmar, alterar ou cancelar; recibo v2). Reconciliador mantém o staging de lotes aguardando e cancelados. `syncserver` v2 (erros `lote_cancelado`, `ja_decidido`; sem `outra_operacao`/`sem_operacao`). Consultas, `FiltroEstudo` (estudo em foco, jornadas, equipes; aba por jornada), relatórios (Estudo, Jornada, Equipe; observações na ficha e no XLSX) e cartas (Estudo, Jornada(s)) sem operação nem área identificada.
- **desktop** · Serviços Wails: `EstudoService` (estudos, jornadas, equipes, estudo em foco), `SincronizacaoService.Aguardando/DecidirDestino`; `OperacaoService` e `ConsultaService.Equipes` removidos; `Info.bloqueio`. `central-teste` cria um estudo de teste e aceita `aguardando`, `confirmar`, `cancelar` e `-destino=auto|manual`. Central `0.6.0-mvp` (ainda sem as telas novas: E2).
- **testes** · Go: códigos (fixture, troca de um caractere, colisão), estudos, guarda de geração, destino travado no banco e fluxo HTTPS (aguardando, código inexistente corrigido, código de outra equipe, cancelamento e reenvio, destino travado, reinício, coleta de outro aparelho). **Os testes cruzados do Android (`:core:sync`) ficam vermelhos até a E4**, porque a Central passou a falar só a v2.

## 2026-11-15 (A28, E0) — estudos, jornadas, equipes e coletas: decisões e documentação
- **docs** · Planejamento aprovado pelo usuário: **Estudo de Situação → Jornada → Equipe** na Central, **coleta pelo código da equipe** no celular, **confirmação do destino** antes de importar, formulário em 7 etapas, responsável obrigatório, **Observações relevantes** e reinício único dos bancos de teste.
  - ADRs novos (Propostos): **0023** (hierarquia e códigos `EQ-XXXX-XXXX`/`SITPOSSE7-XXXX-XXXX`, Crockford Base32 + dígito Luhn mod 32; substitui o ADR-0012), **0024** (coletas, protocolo v2, confirmação/correção/cancelamento do destino, destino travado) e **0025** (geração 2 dos esquemas, com guarda contra banco antigo).
  - `matriz-decisoes.md`: **A28**; A3, A10 e R6 substituídas; A4 e A12 alteradas.
  - `regras-de-negocio.md`: seção "Estudo, jornada, equipe e coleta" (RN-01..09 reescritas/novas), **RN-11** (responsável obrigatório), **RN-15** (observações, até 4000 caracteres), RN-70 e RN-74 ajustadas.
  - `funcionais.md` (RF-A01..03, A15, C01, C02, C10), `protocolo.md` (v2), `modelo-de-dados.md`, `integridade.md` (invariantes 7–9 e novos pontos de falha), `pendencias.md` (P6 deixa de existir), `plano.md` (fase "Estudos e coletas").
  - Telas e documentos de `mobile/docs` e `desktop/docs` serão atualizados nas etapas que mudam as telas (E2 e E5). Nenhuma mudança de código nesta etapa.

## 2026-11-15 (A27) — teste das cartas-imagem pelo usuário
- **docs** · O usuário testou a Central `0.5.1-mvp` e aprovou as cartas-imagem no geral. Os ajustes de detalhe ficaram para uma etapa futura (detalhes a enviar pelo usuário). Registrado em `continuacao.md` e `plano.md`.

## 2026-11-14 (A27) — prévia da carta: zoom e área mínima
- **desktop** · Teste real com o Copernicus: credenciais e busca de datas funcionaram, mas a prévia ficou no zoom máximo e a imagem virou borrões (pixels de 10 m esticados). A causa exata (enquadramento automático ou aproximação manual) não foi reproduzida. Correções:
  - zoom máximo da prévia 16 (≈ 2,4 m/px de tela);
  - nenhuma imagem pedida com menos de 1,5 km de largura (`geo.GarantirLargura`, com teste);
  - enquadramento só com o mapa já medido (`ResizeObserver`).

  Central `0.5.1-mvp`, instalador `61e7c91b…`.

## 2026-11-14 (A27) — cartas-imagem com Sentinel-2 (Copernicus)
- **docs** · A27 e **ADR-0022**: imagem Sentinel-2 L2A do Copernicus CDSE (gratuito); **RN-90 alterada** (imagens consultadas e cartas guardadas sem validade; continua proibido mapa-base offline e pré-download); AGENTS.md §6 e §8 atualizados; P1 resolvida para imagem de satélite. Novo `desktop/docs/cartografia.md`.
- **desktop** · Novos pacotes:
  - `internal/cartografia/geo` (Web Mercator, enquadramento, pino ↔ pixel, escala, agrupamento);
  - `internal/cartografia/cdse` (OAuth2, catálogo, imagem, erros de rede, credenciais e cota);
  - armazém de imagens e cartas;
  - `internal/cartografia/carta` (mapa com pinos, norte, escala e atribuição; folhas PNG e PDF A4/A3 com cabeçalho institucional, legenda, quadro de informações, ressalva RN-92 e tabela de coordenadas);
  - `internal/credenciais` (Gerenciador de Credenciais do Windows);
  - `internal/fontes` (fontes compartilhadas).
- **desktop** · `CartaService` e rota local `/cartografia/`. Tela **Cartas-imagem** (Leaflet): carta individual ou geral com os filtros do Painel, datas com nuvens e sugestão, prévia, geração, cartas salvas e configuração do acesso. Botão "Carta-imagem" na página da propriedade.
- **desktop** · **Dependências novas** (versões exatas): `leaflet` 1.9.4 (BSD-2), `@types/leaflet` 1.9.22 (dev), `golang.org/x/image` v0.48.0 (BSD-3; `x/sys` → v0.50.0). Central `0.5.0-mvp`, instalador `0188293a…` (o anterior ficou como `…-installer-11e36a52.exe`).
- **desktop** · Testes: `geo`, `cdse` (servidor falso), `armazem`, `carta`, `credenciais` (também no Windows real) e `app/cartas_test.go`. Falta o teste com o Copernicus real (depende da conta do usuário).

## 2026-11-14 (A26) — exportação em PDF, Word e Excel; listas centralizadas
- **desktop** · Listas: textos e cabeçalhos centralizados e divisórias entre colunas (`TabelaDados`).
- **desktop** · `internal/relatorios` (ADR-0021):
  - um modelo de documento comum para PDF (gopdf, fonte Archivo embutida) e DOCX (só biblioteca padrão), com brasão e "Polícia Militar de Rondônia / 7º Batalhão de Polícia Militar" no cabeçalho de todas as páginas, tabelas com cabeçalho repetido e divisórias, e "Página X de Y";
  - XLSX (excelize) com 7 abas, tabelas filtráveis, totais `SUBTOTAL`, gráficos e links.

  Os relatórios usam os filtros ativos. `RelatorioService` no app: "Salvar como" do Windows, gravação atômica, "Abrir arquivo" e "Mostrar na pasta". `estatisticas.Familia` ganhou coordenadas e origem do GPS (para as fichas).
- **desktop** · **Dependências novas** (versões exatas): `github.com/xuri/excelize/v2` v2.11.1 (BSD-3) e `github.com/signintech/gopdf` v0.38.4 (MIT), com as transitivas listadas no ADR-0021. Fontes Archivo estáticas (OFL-1.1) e o brasão embutidos no executável.
- **desktop** · Frontend: botão "Exportar" no Painel e em Propriedades e "Exportar ficha" na página da propriedade (`ExportarRelatorio`). Central `0.4.0-mvp`, instalador `11e36a52…` (o anterior ficou como `…-installer-391936a4.exe`).
- **desktop** · Testes: `relatorios_test.go` (PDF, DOCX, XLSX, formatação), `app/relatorios_test.go` (gravação atômica). Validado no Word e no Excel do Windows. Sem mudança de esquema nem de protocolo.

## 2026-11-14 (A25, Etapas 2 e 3) — estatísticas com filtros e drill-down (§22-A)
- **docs** · Especificação do usuário registrada em `docs/requirements/estatisticas.md` (§22-A). Ela substitui a lista derivada provisória como critério de aceite da Fase 4. **ADR-0020:** FiltroEstudo calculado em memória sobre as visões vigentes.
- **desktop** · Pacote `internal/estatisticas`:
  - `Filtro` com equipe, área identificada, busca sem acento, período (cadastro ou recebimento), localização, foto, área informada, faixas e hectares;
  - **condições** sobre qualquer quantidade ou tipo de item: possui, não possui, mais de, pelo menos, menos de, exatamente;
  - `Carregar` (consultas fixas nas visões vigentes) e `Calcular`: totais, por equipe, por área, itens por tipo ("Outros" agrupados pelo nome normalizado), faixas de área e famílias, todos do mesmo conjunto;
  - `ConsultaService.Estatisticas` exposto ao frontend.
- **shared** · Fixture `test-fixtures/estatisticas/`: base com excluídos e 20 casos, incluindo as 7 perguntas da §22-A.3.
- **desktop** · Painel novo:
  - barra de filtros com chips e "Limpar filtros";
  - cartões redesenhados, sem as linhas coloridas: número principal + detalhamento com barra de proporção;
  - abas por equipe ou área (colunas escolhíveis), por família, itens por tipo e tamanho das propriedades;
  - drill-down em todo número, barra e linha, até o cadastro;
  - ao voltar do cadastro, filtros, aba e rolagem são mantidos.

  Propriedades usa a mesma barra de filtros. Testes: `estatisticas_test.go`; Vitest 11 (`filtro.test.ts`). Central `0.3.0-mvp`, instalador `391936a4…` (o anterior ficou como `…-installer-cc3e49b7.exe`). Sem mudança de esquema, de protocolo nem de dependência.

## 2026-11-14 (A25, Etapa 1) — novo visual da Central
- **docs** · A25: Fase 4 antecipada inteira, com lista de cruzamentos derivada (`telas-e-fluxos.md` §Cruzamentos) e P5 resolvida. **ADR-0019:** tabelas próprias no lugar do PrimeVue, porque o PrimeVue 5 deixou de ser MIT e exige licença comercial para a PMRO.
- **desktop** · Identidade visual nova (`telas-e-fluxos.md` §Identidade visual): tokens claro/escuro, fonte Archivo empacotada, barra lateral verde com a operação ativa, faixa de totais.
  - Componentes `TabelaDados` (divisórias, ordenação, paginação, totais, layout empilhado), `Grafico`/`graficos.ts` (ECharts sob demanda, paleta validada), `FaixaTotais` e `InfoTecnica`.
  - **Painel:** faixa de totais, cobertura, recepção, tabelas por equipe e por família com total geral, gráficos por família.
  - **Propriedades:** tabela nova; o clique abre a **página da propriedade** (gráficos + tabelas de pessoas, rebanho, veículos, bens e criações; área, localização, fotos, registro).
  - **Sincronização:** passos + QR, lotes em tabela, recusas em modal; dados técnicos movidos para o modal **Informações técnicas**. O botão "Gerar QR" fica bloqueado enquanto gera (14/11 saíram dois QRs em 1,5 s).
  - **Equipes:** tabela nova.
- **desktop** · **Dependências novas** (versões exatas, ADR-0019):
  - `echarts` 6.1.0 (Apache‑2.0), `vue-echarts` 8.3.2 (MIT), `@fontsource-variable/archivo` 5.3.0 (OFL‑1.1);
  - só para desenvolvimento: `vitest` 5.1.0 (MIT).

  Nenhum recurso externo no build (conferido no `dist`). `npm audit`: 0 vulnerabilidades.
- **desktop** · Testes: `tabela.test.ts` (7). Modo `npm run demo` com dados fictícios, fora do build de produção (conferido).
- **desktop** · Central `0.2.0-mvp`, instalador `cc3e49b7…` (o anterior ficou como `…-installer-99706c25.exe`). Sem mudança de esquema, de protocolo nem de backend.

## 2026-11-14 (MVP) — correção: app fechava ao confirmar o pareamento
- **mobile** · **Causa:** `ApiCentral.close()`/`ClienteCentral.fechar()` fechavam o socket TLS na thread que chamava (a principal, no fim do `api.use { }` do pareamento). O Android lança `NetworkOnMainThreadException`, e o OkHttp a repassa. Agora o fechamento roda na thread do OkHttp (`Conexoes.kt`, `liberarForaDaThreadAtual`). Detalhes e evidências: [`mobile/docs/incidentes/2026-11-14-crash-pareamento.md`](../mobile/docs/incidentes/2026-11-14-crash-pareamento.md).
- **mobile** · Defesas: erro inesperado na sincronização vira mensagem na tela com referência (`sinc_erro_inesperado`), e o app não fecha; o lote fica `erro_recuperavel` com a causa. Registro local no logcat (`Sitposse7Sinc`), por tentativa, com as etapas e a duração, sem segredos.
- **mobile** · App `0.1.1-mvp` (versionCode 3), APK `f4a7fd54…`, mesma chave de debug (`ab61d27f…`): instala por cima sem perder dados. O anterior ficou como `sitposse7-campo-debug-a0de8bde.apk`.
- **mobile** · Testes novos: `FechamentoConexaoTest` (2, reproduzem o crash) e `PareamentoGoTest` (4: recusado, repetido, tempo esgotado, Central caída). Total 48.
- **desktop** · `central-teste`: `-espera` e comandos `pendentes`/`aceitar`/`recusar`. A Central e o instalador não mudaram.
- **Validado no aparelho** (moto g56, 14/11): pareamento e envio OK no notebook (2 cadastros) e no PC (1 cadastro). P12 aberta: o que fazer ao parear com outra Central.
- **docs** · `protocolo.md` §2: o token é consumido antes da resposta (o aparelho que perde a resposta precisa de novo QR). Sem mudança de esquema, protocolo ou dependência.

## 2026-11-14 (MVP) — telas da Central e roteiro do teste em campo
- **desktop** · Frontend novo (Vue, sem biblioteca nova): painel, propriedades (busca, filtro por equipe, detalhe com itens, área, coordenadas e fotos servidas por `/fotos/{uuid}.jpg`), equipes, sincronização (criar operação, QR de pareamento com token de 10 min e uso único, confirmação do pareamento em qualquer tela, lotes recebidos ao vivo com recusas e diagnóstico da Central). Layout adaptável (barra lateral → barra superior abaixo de 1000 px; detalhe lateral → abaixo em janelas menores), tema claro/escuro, janela mínima 900×600. Versão `0.1.0-mvp`; bindings regenerados.
- **desktop** · Instalador `99706c25…` (inclui o WebView2 offline, como antes).
- **docs** · `operacao/teste-de-campo-mvp.md` (antes de sair, no campo, na volta); `continuacao.md` com o próximo passo.

## 2026-11-14 (MVP) — telas do app de campo
- **mobile** · App `0.1.0-mvp` (versionCode 2): `CampoApp` (dependências e reconciliador de fotos na abertura), navegação por pilha salva (sem biblioteca nova), tema claro de alto contraste com modo escuro, layout adaptável pela largura (grade na lista; conteúdo centralizado em telas largas), ícones vetoriais locais (Material Icons, Apache 2.0).
- **mobile** · Telas: configurar/editar equipe; início (números, rascunhos para retomar, última sincronização confirmada, alarme de foto sem arquivo); lista com busca sem acento e filtros (pendentes, sem GPS, sem foto, erro); detalhe com excluir; formulário por seções com rascunho automático (~1 s), quantidades com +/−, itens do catálogo e vários "Outros" (aviso de repetido e somar, RN-33), área com conversão, GPS (GNSS sem internet com progresso, melhor precisão, meta de 15 m e 90 s; ou digitado sem precisão) e fotos (CameraX, redução a 2560 px/JPEG 85, `.part` + fsync + rename antes do registro); sincronização (pareamento pelo QR com espera do operador, envio com progresso de fotos, resultado e recusas). Diagnóstico continua acessível.
- **mobile** · Permissões novas: `ACCESS_FINE_LOCATION` e `ACCESS_COARSE_LOCATION` (GNSS). `kotlinx-serialization-json` (já fixada) no `:app` para o rascunho. Teste `ArmazemFotosTest`.
- **docs** · P11 (mensagens de erro dos módulos `:core` no código, não em `strings.xml`).

## 2026-11-14 (MVP) — banco v2 e sincronização no celular (ADR-0018)
- **mobile** · **Esquema:** banco Room v2 com a migração não destrutiva `MIGRACAO_1_2` (SQL exportado pelo Room; testada com `MigrationTestHelper`): equipe, propriedades, itens, fotografias, rascunhos, lotes de envio (JSON congelado), itens do lote e pareamento.
- **mobile** · `:core:dominio`: `Catalogo` (conferido com a fixture compartilhada), modelo do formulário `CadastroPropriedade` (serializável para o rascunho) e `ValidacaoCadastro` (RN-11..55, RN-32/33/34).
- **mobile** · `:core:dados`: `RepositorioCampo` (equipe e instalação, salvar/editar/excluir com `rev` na mesma transação e tombstones, foto só com arquivo existente, rascunhos).
- **mobile** · `:core:sync`: contratos v1 (`ContratosTest` com as mesmas fixtures do Go), `ApiCentral`, `Pareador` e `Sincronizador` (lote congelado, fotos em blocos com retomada, recibo conferido e gravado em transação). `SincronizacaoGoTest`: banco Room real × Central Go real. `ClienteCentralGoTest` lê o `central_id` real.
- **Dependência nova (só testes):** `androidx.room3:room3-testing:3.0.3`, oficial, exigida pelo `mobile/AGENTS.md` para testar migrações (`MigrationTestHelper`, variante JVM). `:core:sync` passa a usar `kotlinx-coroutines-core` (já fixada).

## 2026-11-14 (MVP) — contratos v1 e recepção na Central (ADR-0017)
- **shared** · Schemas v1 `comum`, `pareamento`, `lote`, `estado-lote` e `recibo`; fixtures válidas e inválidas em `test-fixtures/contratos` (com o `recibo_sha256` esperado) e o catálogo inicial em `test-fixtures/catalogo/itens.v1.json`.
- **desktop** · **Esquema:** migração `0002_mvp_recepcao.sql` (não destrutiva; testada a partir de um banco da Fase 0 com dados): operações, equipes, tokens, dispositivos, propriedades, catálogo semeado, itens, fotografias, lotes recebidos, fotos esperadas, aplicações, auditoria somente acréscimo e visões vigentes.
- **desktop** · Novos pacotes: `contratos` (tipos e validação v1, conferida contra os schemas), `importacao` (lote idempotente pelo hash do manifesto, fotos em blocos de 1 MiB com fsync e retomada por offset, SHA‑256 conferido, arquivo publicado antes do registro, aplicação em `SAVEPOINT` por registro, recibo persistido, reconciliador), `pareamento` (operação ativa, token de 10 min e uso único, pedido aguardando o operador por até 90 s, credencial com hash), `consultas` (painel, propriedades com busca sem acento, detalhe, equipes, lotes) e `nucleo` (monta tudo igual para o app, o `central-teste` e os testes). API v1 completa no `syncserver`. Camada `app` com os serviços de operação, sincronização e consulta, e rota `/fotos/{uuid}.jpg` para a tela.
- **desktop** · `cmd/central-teste` sobe a Central completa (operação criada, token, confirmação automática) para os testes cruzados do Android.
- **docs** · ADR-0017 (Proposto): detalhes da v1 (espera longa no pareamento, códigos de erro sem 408, idempotência por hash do manifesto, hash do recibo, validação em Go, esquema 0002). `protocolo.md` §9, `modelo-de-dados.md`, `importacao-e-consolidacao.md`, READMEs de `shared/`.
- Sem dependência nova.

## 2026-11-14 — Fase 0 concluída (A23) e MVP para teste em campo (A24)
- **docs** · Matriz: A23 (Fase 0 concluída, com os adiados da A22 e o Windows 10 sem teste) e A24 (recorte do MVP: Fases 1 e 2 em fatia vertical + transferência segura da Fase 3; Central só recebe e consulta; dados de teste descartáveis). `plano.md` com a linha do MVP e a Fase 0 concluída; `continuacao.md` com as 7 etapas do MVP.
- Sem mudança de código.

## 2026-11-14 (Fase 0) — reteste do instalador na VM
- **desktop** · Instalador NSIS: o texto do `netsh` deixa de ir para o log. Redirecionado, o `netsh` escreve em UTF‑8, e o `nsExec` o mostrava com acentos quebrados com ou sem `/OEM` (a correção de 13/11 partia de um diagnóstico errado). Agora `nsExec::Exec` + `Pop $0`, e o log usa mensagens próprias conforme o código de saída; se a regra de firewall não puder ser criada, o instalador avisa na tela. Build `01040f2b…6aa8f1f`, com o mesmo `sitposse7-central.exe` (`39c266d1…`). Sem mudança de esquema ou de protocolo.
- **docs** · `planning/fase-0.md` §Reteste do instalador: achados 1–4 confirmados como corrigidos (o 1 com o certificado antigo, só com `127.0.0.1`, reaproveitado); achado 6 novo (perfil do WebView2 fica em `%APPDATA%` após desinstalar); ponto "Windows limpo" recriado à mão, após exclusão por engano; teste do zero dispensado pelo usuário. `planning/continuacao.md`: estado e próximos passos (aprovação da Fase 0 e pedido de MVP para teste em campo).

## 2026-11-13 (Fase 0) — correções do teste do instalador e ícones (Central e app)
- **desktop** · `syncserver.CarregarOuCriarIdentidade`: ao reaproveitar o certificado salvo, os IPs anunciados (tela de diagnóstico e QR) passam a ser os atuais, e não o SAN do certificado (que traz `127.0.0.1` e pode trazer IPs de redes antigas). Teste novo: `TestIdentidadeReaproveitadaAnunciaSoIPsAtuais`. Pin e certificado inalterados; sem mudança de esquema ou de protocolo.
- **desktop** · Instalador NSIS: `nsExec::ExecToLog /OEM` nas chamadas ao `netsh` (acentos corretos no log) e remoção da pasta da empresa em `Program Files` quando ela fica vazia na desinstalação (`RMDir` sem `/r`).
- **desktop** · Ícone próprio da Central: `build/appicon.svg` (fonte), `build/appicon.png` e `build/windows/icon.ico`. Renderizado com o Edge headless já presente no Windows; nenhuma dependência nova.
- **mobile** · Ícone adaptativo do app (mesmo desenho da Central): `res/drawable/ic_launcher_background.xml`, `ic_launcher_foreground.xml`, `ic_launcher_monochrome.xml` e `res/mipmap-anydpi/ic_launcher{,_round}.xml`; `android:icon`/`android:roundIcon` no manifesto. Só vetores, sem dependência nova.
- **docs** · `fase-0.md` e `continuacao.md`: achados 1 a 4 marcados como corrigidos; reteste na VM pendente; `continuacao.md` com o roteiro da próxima sessão (reteste, ícone no celular, limpeza da ISO, aprovação da Fase 0) e o comando de regerar o ícone (§4).

## 2026-11-13 (Fase 0) — instalador testado no Windows 11
- **docs** · `planning/fase-0.md`: spike (d) concluído no Windows 11 (VM Hyper-V limpa, sem rede): instalação, Central abrindo sem rede, regra de firewall (`LocalSubnet`, TCP 38807, presa ao executável), desinstalação preservando os dados e reinstalação com o mesmo ID e pin. Nova seção §Teste do instalador com o ambiente (ativação online da avaliação antes do ponto de verificação) e cinco achados não bloqueantes, entre eles o `127.0.0.1` na tela de diagnóstico e possíveis IPs antigos no QR. Windows 10 22H2 não testado.
- **docs** · `planning/continuacao.md`: item 2 da Fase 0 feito; roteiro para repetir o teste na VM; achados em "Melhorias".
- Sem mudança de código, esquema ou protocolo.

## 2026-11-13 (Fase 0) — instalador gerado e A22
- **desktop** · Instalador NSIS gerado pelo WSL (`wails3 package GOOS=windows`, `makensis` 3.09), com o WebView2 Evergreen Standalone x64 (assinatura da Microsoft conferida). Sem mudança no script; o binário do WebView2 continua fora do git. Teste em VM limpa pendente.
- **docs** · A22 (matriz): a Fase 0 pode ser concluída com os testes de hotspot do celular e de Android 17/8/9 adiados; P7 segue aberta (prazo recomendado: Fase 3). `plano.md` (critério da Fase 0), `pendencias.md`, `continuacao.md` (§2 com o roteiro da VM; §4 com o comando do instalador) e `fase-0.md`.

## 2026-11-13 (Fase 0) — testes adiados
- **docs** · A pedido do usuário, o teste no hotspot do celular (cenário A) e o teste em Android 17 (P7), além de Android 8/9, ficam para depois: seção "Adiados" em `planning/continuacao.md` §2; notas em `fase-0.md` e `pendencias.md` (P7). O critério da Fase 0 não muda; o fechamento depende de fazê-los ou de o usuário aprovar levá-los adiante.

## 2026-11-13 (Fase 0) — hotspot do Windows só com internet (A21, ADR-0016)
- **docs** · Decisão do usuário: o hotspot do Windows é usado **somente com internet na origem** e com o computador (PC ou notebook) **ligado**; sem internet, hotspot do celular ou pacote USB. ADR-0016 substitui a condição "hotspot do Windows condicionado ao teste" da A16; matriz com A21.
- **docs** · P8 encerrada sem teste de bancada (`pendencias.md`); spike (b) cancelado (`plano.md`, `fase-0.md`); `operacao/rede-em-campo.md` reescrito no cenário B (passo a passo com internet, tampa aberta, economia de energia desativada, queda de internet durante o uso); risco K2 reavaliado (`riscos.md`); `continuacao.md` sem o item P8; `README.md`.
- Sem mudança de código, esquema ou protocolo. A Central continua sem depender de internet em nenhuma função (A2): a internet é requisito operacional apenas do cenário B.

## 2026-11-13 (Fase 0) — segunda máquina e cartão "Banco de dados" no aparelho
- **docs** · `planning/fase-0.md`: spike (c) concluído no aparelho (moto g56, Android 16: `wal · synchronous=2 · foreign_keys=1`, SQLite 3.50.1); nova conexão pelo hotspot do Windows em TLSv1.3, 92 ms. Achado: o celular alcançou a Central pelo IP da origem (Ethernet), encaminhado pelo compartilhamento de internet do Windows. Ambiente da 2ª máquina registrado.
- **docs** · `planning/continuacao.md`: item 1 da Fase 0 concluído; P8 com o passo "Atualizar e reler o QR depois de desligar a origem"; armadilhas novas (`--no-metrics` só no binário `android`, ausência de `unzip`, chave de debug por máquina).
- Sem mudança de código, esquema ou protocolo. Bateria do §4 rodada na 2ª máquina: 22 testes Kotlin (0 pulados) e pacotes Go passando.

## 2026-11-13 (Fase 0) — passagem de bastão
- **docs** · `planning/continuacao.md`: estado, pendências da Fase 0 em ordem, preparo de máquina nova, armadilhas, comandos de teste e proteções contra regressão. Links no `AGENTS.md`, `README.md` e `docs/README.md`; READMEs de `desktop/` e `mobile/` com o estado atual.

## 2026-11-13 (Fase 0, em andamento) — Android e testes cruzados
- **mobile** · Projeto Gradle 9.9.1 (AGP 9.5.1, Kotlin 2.4.21): `:core:dominio` (área, normalização e leitura do QR com as fixtures `shared`), `:core:dados` (KMP android+jvm, Room 3.0.3 + `BundledSQLiteDriver` + `DriverDuravel`, banco v1 só com `meta`, esquema exportado em `core/dados/schemas`), `:core:sync` (OkHttp com pinning SPKI, `GET /v1/status`), `:app` (tela de diagnóstico: PRAGMAs, leitura do QR, conexão pela rede Wi‑Fi). `allowBackup=false`, `dataExtractionRules` excluindo tudo, `targetSdk 37` com `ACCESS_LOCAL_NETWORK`.
- **mobile** · Achado: o Room 3 rebaixa `synchronous` para NORMAL em WAL e deixa `foreign_keys` desligado. Corrigido pelo `DriverDuravel` (ADR-0002 mantido; ver `mobile/docs/persistencia-offline.md`).
- **protocolo** · TLS 1.2 restrito (ECDHE-ECDSA + AES-GCM/ChaCha20) aceito como mínimo, para Android 8/9; TLS 1.3 continua preferido (ADR-0014). Sem mudança de `protocolo_versao` (o formato das mensagens não muda).
- **desktop** · `syncserver` com TLS 1.2 restrito e testes (`tls_test.go`, `pin_fixture_test.go`); `cmd/central-teste` (servidor sem interface para os testes cruzados do Android).
- **shared** · Fixture `tls/pin.json` + `tls/central-fixture.crt.pem` (pin calculado com OpenSSL; Go e Kotlin conferem).
- **dependências** · ZXing core 3.5.4 no lugar do ML Kit, que traz telemetria (ADR-0015). Novas: OkHttp 5.5.1, kotlinx.serialization 1.13.0, JUnit 6.1.3 (só testes). Todas fixadas em `mobile/gradle/libs.versions.toml`.
- **protocolo** · Porta padrão da Central passa de 47807 para **38807**: a 47807 cai na faixa que a rede espelhada do WSL reserva no Windows (44620–48715). Regra de firewall do instalador e fixtures atualizadas; teste `porta_test.go`.
- **docs** · ADRs 0014 e 0015; matriz (A19, A20); pendências (P7 parcial, P10 nova; duas resolvidas); textos de transporte e stack atualizados.

## 2026-11-13 (Fase 0, em andamento)
- **desktop** · Projeto Wails v3.1.0 + Vue 3 + TS (só Windows). Núcleo Go: `dominio` (área, normalização), `banco` (migração 0001_meta; PRAGMAs verificados), `syncserver` (identidade TLS, pin SPKI, `GET /v1/status`, QR). Tela de diagnóstico. Instalador NSIS com WebView2 offline e regra de firewall.
- **shared** · `schemas/qr-pareamento.v1.schema.json`, `schemas/status.v1.schema.json`, fixture `contratos/qr-pareamento.v1.json`.
- **protocolo** · Porta padrão 47807; pin = SHA-256 do SPKI em base64url sem padding (ADR-0007).
- **docs** · Diário da Fase 0 (`planning/fase-0.md`) com versões fixadas e achados.

## 2026-11-13
- **docs** · Planejamento aprovado. Criada a estrutura do monorepositório e da documentação: requisitos, regras de negócio (fonte única), arquitetura, modelo de dados, protocolo de sincronização v1, integridade, conflitos, segurança, testes, plano, matriz de decisões, pendências, riscos e rede em campo.
- **docs** · ADRs 0001–0013 aprovados.
- **shared** · Fixtures iniciais: `area/conversao.json` (RN-44) e `normalizacao/nomes.json` (RN-35).
- Nenhum código de aplicativo implementado.
