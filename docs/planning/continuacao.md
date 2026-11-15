# Continuação do trabalho (passagem de bastão)

> **Leia isto primeiro** ao retomar o projeto em outra máquina ou com outro agente.
> Atualizado em **15/11/2026**. Mantenha este arquivo em dia ao final de cada sessão.
> Ambiente da §3 pronto no PC desktop, com a VM de teste. Comece pela §2.

## 1. Onde estamos

- **Fase atual:** **MVP para teste em campo** (A24). A Fase 0 foi concluída em 14/11 (A23); diário em [fase-0.md](fase-0.md). Plano: [plano.md](plano.md).
- **Branch de trabalho:** `development`. A `main` só recebe o que for aprovado.
- **Últimos commits relevantes** (o mais recente primeiro; confira com `git log --oneline -5`):
  - `fix(fase-0)`: reteste do instalador na VM e mensagens próprias do firewall no log do NSIS (14/11)
  - `8681b87` fix(fase-0): correções do teste do instalador, ícone da Central e do app Android (13/11, noite)
  - `8911bc0` docs(fase-0): instalador gerado pelo WSL e adiamento aprovado dos testes de rede/Android (A22)
  - `02752bb` feat(mobile): projeto Android, teste cruzado com a Central e spikes da Fase 0
- Implementar só o que está no recorte da A24; o resto das Fases 1–3 segue no plano (AGENTS.md §2).

### Já feito e verificado
| Item | Situação | Evidência |
|---|---|---|
| Central (Go + Wails 3) | Esqueleto, banco com migração 0001, servidor HTTPS `GET /v1/status`, QR na tela de diagnóstico, ícone próprio (`desktop/build/appicon.svg`) | `go test ./internal/... ./app/...` |
| App Android | Esqueleto (`:app`, `:core:dominio`, `:core:dados`, `:core:sync`), tela de diagnóstico e ícone adaptativo (mesmo desenho da Central, com camada monocromática) | `./gradlew test ...` (22 testes) |
| Spike (a) HTTPS + pin via QR | Cliente Kotlin × servidor Go real: TLS 1.3, TLS 1.2, pin divergente. **Aparelho real OK:** moto g56, Android 16, hotspot do Windows **com** internet na origem, TLSv1.3, 108 ms | `ClienteCentralGoTest`; `fase-0.md` §Testes em aparelho real |
| Spike (c) PRAGMAs Room 3 | Room rebaixava `synchronous` para NORMAL e deixava `foreign_keys` desligado; corrigido com `DriverDuravel`. **Aparelho real OK** (moto g56, 13/11: `wal · 2 · 1`, SQLite 3.50.1); na mesma sessão, nova conexão em TLSv1.3, 92 ms | `PragmasTest` (JVM); `fase-0.md` §Testes em aparelho real |
| Spike (d) Instalador NSIS | **Testado no Windows 11** (13/11, VM Hyper-V sem rede) e **retestado em 14/11** com o build `01040f2b…6aa8f1f`: achados 1–4 corrigidos (o 3 só na 2ª tentativa). Teste do zero dispensado pelo usuário. Windows 10 22H2 não testado | `fase-0.md` §Teste e §Reteste do instalador |
| Spike (b) Hotspot do Windows / tampa | **Cancelado** (A21/ADR-0016): hotspot do Windows só com internet e computador ligado, já validado 2× (notebook com Wi‑Fi, PC com Ethernet). P8 encerrada | `fase-0.md`; `rede-em-campo.md` |
| Decisões novas | A19/ADR-0014 (TLS 1.2 restrito), A20/ADR-0015 (ZXing no lugar do ML Kit), porta 38807, A21/ADR-0016 (hotspot do Windows só com internet) | [matriz-decisoes.md](matriz-decisoes.md) |

## 2. O que falta para fechar a Fase 0 (em ordem)

### MVP para teste em campo (A24, aprovado em 14/11): etapas
A Fase 0 foi **concluída** (A23). Recorte do MVP: [matriz-decisoes.md](matriz-decisoes.md) A24 e [plano.md](plano.md). Ordem de trabalho (marque ao concluir):
1. ~~**Contratos**~~ **feito (14/11, ADR-0017):** schemas v1 + fixtures, conferidos no Go e no Kotlin.
2. **Celular, dados:** **feito (14/11, ADR-0018):** banco v2 com migração testada (`MigrationTestHelper`), catálogo, `RepositorioCampo` (rev na mesma transação, tombstones, foto só com arquivo), rascunhos. Falta: gravação do arquivo da foto e reconciliador (no `:app`, com a câmera).
3. **Celular, telas:** **feito (14/11), falta testar no aparelho:** configurar equipe, início (números, rascunhos, última sincronização, alarme de foto sem arquivo), lista com busca sem acento e filtros, detalhe com excluir (tombstone), formulário por seções com rascunho automático, itens com "Outros" (aviso de repetido, RN-33), área com conversão, GPS (GNSS com progresso e meta de 15 m, ou digitado), câmera (foto reduzida a 2560 px, arquivo antes do registro), tela de sincronização (pareamento pelo QR, progresso, resultado). Layout adaptável (grade na lista, conteúdo centralizado em telas largas). Tema claro de alto contraste.
4. ~~**Central, recepção:**~~ **feito (14/11):** migração 0002, operação, pareamento com confirmação do operador, `PUT/GET /v1/lotes`, `PUT/GET /v1/fotos`, importação com recibo, reconciliador, consultas; `central-teste` com a API completa. Testes Go de ponta a ponta (`internal/nucleo`) nos cenários 1, 4, 5, 6, 7 e 8.
5. **Celular, sincronização:** **núcleo feito (14/11):** `Pareador`, `Sincronizador` e `ApiCentral` no `:core:sync`; `SincronizacaoGoTest` (banco Room real × Central Go real) nos cenários 1, 4, 5, 6 e 8 + queda de rede. Falta a tela e a rede Wi‑Fi do aparelho (`requestNetwork`).
6. ~~**Central, telas**~~ **feito (14/11):** painel (totais das visões vigentes, área com "N sem área"), propriedades (busca sem acento, filtro por equipe, detalhe com itens, área, coordenadas e fotos ampliáveis), equipes, sincronização (criar operação, QR de pareamento com validade e uso único, confirmação do pareamento em qualquer tela, lotes recebidos ao vivo com recusas). Barra lateral que vira barra superior em janela estreita; tema claro/escuro.
7. **Builds e roteiro:** **feito (14/11):** `Downloads\sitposse7-teste\sitposse7-central-amd64-installer.exe` (`99706c25…`) e `sitposse7-campo-debug.apk` (`a0de8bde…`, 0.1.0-mvp); versões anteriores guardadas com sufixo. Roteiro: [teste-de-campo-mvp.md](../operacao/teste-de-campo-mvp.md).

**Teste de bancada (14/11, noite): o app fechava ao confirmar o pareamento.** Corrigido no `0.1.1-mvp` (`sitposse7-campo-debug.apk`, `f4a7fd54…`; instala por cima, mesma chave). Causa: socket TLS fechado na thread principal. Ver [incidente](../../mobile/docs/incidentes/2026-11-14-crash-pareamento.md). **Validado no aparelho em 14/11 às 19:28:** os 2 cadastros antigos foram enviados ao notebook e o 3º ao PC, com recibo e sem crash. Cada Central ficou com o que recebeu: ver **P12**. Decisão do usuário (14/11): por enquanto, **um só computador** como Central; usar mais de um fica para depois (alternativa sugerida: exportar e importar entre Centrais). Os reinícios do PC desktop são de hardware (WHEA 18 / MCE no Ryzen 5 5600X, desde setembro): use o notebook como Central nos testes.

**Fase 4 antecipada (A25, 14/11, noite), em etapas:**
1. ~~**Base visual**~~ **feito:** ADR-0019, tabelas próprias, ECharts, Archivo, Painel/Propriedades/página da propriedade/Equipes/Sincronização e modal técnico. Central `0.2.0-mvp` (`cc3e49b7…`). **Falta o usuário olhar no Windows.**
2. ~~**`FiltroEstudo` no Go**~~ **feito (14/11):** `internal/estatisticas` (ADR-0020), fixture `shared/test-fixtures/estatisticas/` com as 7 perguntas da especificação do usuário (`docs/requirements/estatisticas.md`, §22-A). 900 propriedades em ~14 ms.
3. ~~**Painel/Estatísticas**~~ **feito (14/11):** barra de filtros, cartões, abas, drill-down; Propriedades com os mesmos filtros. Central `0.3.0-mvp` (`391936a4…`). **Falta o usuário testar no Windows.**
**Exportação (A26, 14/11, noite):** PDF, Word e Excel com os filtros ativos + listas centralizadas com divisórias entre colunas. Feito; ver ADR-0021. Central `0.4.0-mvp` (`11e36a52…`). **Falta o usuário testar no Windows.**

**Cartas-imagem (A27, 14/11, noite):** Sentinel-2/Copernicus, ADR-0022, `desktop/docs/cartografia.md`. Central atual **`0.5.1-mvp` (`61e7c91b…`)**, instalada no PC.
- **Conta Copernicus criada pelo usuário** (categoria *Public*, finalidade *Public sector*, tipo *National/regional/local public authorities*). Cliente OAuth "SITPOSSE-7 Central" (*Client Credentials*, sem SPA, validade de 90 dias: **vence em 13/02/2027**; depois disso, criar outro e colar em Cartas-imagem → Configurar). As credenciais ficam só no Gerenciador de Credenciais do Windows (alvo `SITPOSSE-7/Copernicus-CDSE`).
- **1º teste real:** a conexão e a busca de datas funcionaram. A prévia ficou no zoom máximo e a imagem virou borrões. Corrigido na 0.5.1 (zoom máximo 16, área mínima de 1,5 km, enquadramento com `ResizeObserver`); a causa exata não foi reproduzida.
- **Coordenadas de teste (14/11, ~23h):** a pedido do usuário, Jaraguá e Lua De Prata foram movidas para área rural (-10.0754, -63.1741 e -10.0750, -63.1689) direto no banco da Central, como correção da Central (`corrigido_desde_rev_origem = 1`, `rev_central = 1`, auditoria `corrigir`). Cópia anterior: `%LOCALAPPDATA%\SITPOSSE-7\sitposse7.db.antes-coordenadas-teste-20261114-232248`. Uma versão nova dessas duas vinda do celular será **recusada** ("corrigido na Central") até existir a resolução de conflitos.
- **Teste da 0.5.1 (15/11):** o usuário testou e **aprovou no geral**. Ficaram só **ajustes de detalhe**, adiados por decisão do usuário (15/11) para uma **etapa futura**: os detalhes ainda serão enviados por ele. Ao retomar, conferir também as perguntas da 1ª rodada que seguem sem resposta: (1) se a prévia abriu com a régua em torno de 500 m sem aproximar; (2) se a nitidez de 10 m atende ou se é preciso enxergar construções (nesse caso, avaliar outro provedor pela interface `Provedor`: Brasil MAIS/Planet, ~3 m, ou Esri World Imagery, < 1 m); e a calibração das cores (névoa leve; ganho fixo de 2,5).

4. **Aliases e duplicatas** (P5 resolvida): migração 0003 + ADR, telas. *(Depois da A28.)*

**Estudos, jornadas, equipes e coletas (A28, aprovado em 15/11):** plano completo em ADR-0023/0024/0025 e na fase "Estudos e coletas" do [plano.md](plano.md). Etapas (marque ao concluir):
- [x] **E0** decisões e documentação (15/11).
- [x] **E1** (15/11) Central, núcleo: migração 0002 reescrita + guarda, códigos, estudos/jornadas/equipes, pareamento sem equipe, importação com `aguardando_confirmacao`/decisão/destino travado, recibo v2, estatísticas/relatórios/cartas por estudo, `central-teste` com comandos de decisão. Contratos v2 (`shared/schemas/*.v2`, fixtures). Os testes cruzados do Android voltaram a passar na E4.
- [ ] **E2** Central, telas: Estudos, estudo em foco, modal de destino, filtros por jornada/equipe.
- [x] **E3** (15/11) Android, dados: Room v3 + guarda, coletas, códigos, responsável obrigatório, observações.
- [x] **E4** (15/11) Android, sincronização v2 por coleta: `:core:sync` com todos os testes cruzados verdes contra a Central Go.
- [x] **E5** (15/11; falta testar no aparelho) Android, telas: código da equipe, coletas, formulário em 7 etapas com revisão.
- [ ] **E6** builds, reinício controlado dos bancos (com confirmação do usuário na hora) e bancada.

Revisão visual sem o Windows: `cd desktop/frontend && npm run demo` (dados fictícios) e capturas pelo navegador.

**Próximo passo: o usuário faz o teste em campo** seguindo o roteiro e traz os resultados. Antes, conferir na bancada (PC + moto g56): instalar, parear, 2–3 cadastros com foto e GPS, enviar, conferir na Central. Pendências de decisão: ADR-0017 e ADR-0018 (Propostos) e P11.

**Atenção (dados descartáveis, A24):** o APK é assinado com a chave de debug **desta máquina**. Gere todas as versões do teste aqui; um APK de outra máquina obriga a desinstalar o app e apaga os cadastros do celular.

### Pendências menores da Fase 0
1. **(Opcional) Ícone no celular:** instalar `Downloads\sitposse7-teste\sitposse7-campo-debug.apk` no moto g56. Não feito em 14/11.
2. **(Opcional) Limpeza do Windows:** a VM não usa mais a ISO (unidade de DVD sem mídia desde 14/11), então `Downloads\26300.9457…pt-br.iso` (8,2 GB) pode ser apagada. Para recriar a VM do zero seria preciso baixá-la de novo.

### Situação dos itens da Fase 0

Quase tudo depende do **usuário com aparelhos e computador reais**. O P8 (hotspot do Windows sem internet e tampa fechada) foi **encerrado sem teste** pela A21/ADR-0016: o hotspot do Windows só é usado com internet e com o computador ligado. O agente prepara os builds, orienta e registra os resultados em `fase-0.md`.

1. ~~**Cartão "Banco de dados" no aparelho**~~: **feito em 13/11** (moto g56: "Configuração de durabilidade correta", `wal · synchronous=2 · foreign_keys=1`).
2. ~~**Spike (d), instalador**~~: **feito em 13/11 no Windows 11** (VM limpa, sem rede) e **retestado em 14/11** (achados 1–4 corrigidos). Achados não bloqueantes em `fase-0.md` §Teste do instalador e em "Melhorias" abaixo. Windows 10 22H2 (único que exercita o WebView2 offline) não testado: fazer se houver uma máquina ou ISO.
   - **Gerado pelo WSL** (o `makensis` 3.09 está em `/usr/bin`; não precisa de NSIS, Go nem Node no Windows): `bin/sitposse7-central-amd64-installer.exe`, 222 MB, copiado para `Downloads\sitposse7-teste`. Build atual: `01040f2b…6aa8f1f` (14/11). O anterior ficou como `…-installer-9d120cce.exe`. Ver §4.
   - **Ambiente de teste (pronto no PC desktop):** VM Hyper-V `SITPOSSE7-Win11-teste` (Geração 2, 4 GB, 2 vCPUs, TPM virtual, adaptador **não conectado**) com o **Windows 11 Enterprise Evaluation** ativado até ~11/02/2027, conta local `teste`. Para repetir: reverter ao ponto de verificação **"Windows limpo"** (recriado em 14/11 depois de limpar a VM à mão; pontos automáticos desligados; DVD sem mídia) e colar o instalador pela sessão avançada (área de transferência; não usa rede). Sem ativação online a avaliação nasce vencida (0xC004F009); se recriar a VM, ligue-a ao *Default Switch* só para `slmgr.vbs /ato` e desconecte antes do ponto de verificação.
   - **Atenção:** o Windows 11 já vem com o WebView2, então o instalador **pula** o WebView2 embutido. Só um Windows 10 22H2 limpo exercita a instalação offline do WebView2.
3. ~~Registrar tudo em `fase-0.md` e no changelog~~ (14/11). ~~Aprovação da conclusão~~: **aprovada em 14/11 (A23)**. Os testes abaixo foram adiados **com aprovação do usuário** (A22, 13/11): não bloqueiam a conclusão da Fase 0, e a P7 continua aberta até o teste em Android 17.

### Adiados pelo usuário (13/11/2026): fazer depois
1. **Hotspot do celular (cenário A):** um celular cria o hotspot (sem dados móveis); o computador da Central (PC com Wi‑Fi ou notebook) e outro celular se conectam a ele; testar a conexão dos dois celulares, inclusive o que criou o hotspot ([rede-em-campo.md](../operacao/rede-em-campo.md) §Cenário A).
2. **Android 17 (P7):** testar em um aparelho com Android 17 (a permissão de rede local é pedida antes de conectar). Se houver, testar também Android 8 ou 9 (deve conectar em **TLSv1.2**).

### Melhorias já identificadas (não bloqueiam a Fase 0; fazer quando o usuário aprovar ou na Fase 2)
- **Diagnóstico da sincronização (propostas do incidente de 14/11, aguardam aprovação):**
  - (a) log de eventos local na Central (arquivo na pasta de dados com rotação; hoje a Central não registra recusas nem erros da API);
  - (b) identificador de correlação enviado pelo app num cabeçalho opcional e registrado pela Central (mudança de protocolo: exige ADR);
  - (c) registro local de crashes no app, visível no Diagnóstico;
  - (d) botão "Gerar QR" bloqueado enquanto gera (14/11: dois QRs com 1,5 s de diferença).
- A Central permite várias janelas abertas ao mesmo tempo; deveria aceitar só uma (instância única do Wails). A segunda janela mostra "porta em uso".
- O QR inclui IPs de adaptadores virtuais (WSL/Hyper-V/Docker, ex.: `172.18.0.1`); o app tenta os IPs em sequência (até 5 s cada). Filtrar adaptadores virtuais na Central e/ou tentar os IPs em paralelo no app.
- **Achados do teste do instalador:** 1 a 4 **corrigidos e retestados em 14/11** (`fase-0.md` §Reteste). Faltam: (5) assinatura de código; (6) a desinstalação deixa o perfil do WebView2 em `%APPDATA%\sitposse7-central.exe` (`$AppData` aponta para `C:\ProgramData` no contexto de administrador do template do Wails). Corrigir no empacotamento (Fase 7).
- P10: `applicationId` provisório `br.gov.ro.pm.sitposse7.campo`. Decidir antes do primeiro uso em campo.

## 3. Preparar uma máquina nova

Resumo do ambiente de referência (WSL2 Ubuntu, sem sudo). Versões exatas: [fase-0.md](fase-0.md) §Versões fixadas.

| Ferramenta | Versão | Observação |
|---|---|---|
| JDK | Temurin **21** | `JAVA_HOME` |
| Go | **1.27.4** | o `go.mod` exige ≥ 1.26 |
| Android SDK | `platforms;android-37.0`, `build-tools;36.0.0` (padrão do AGP 9.5.1), `platform-tools` | Licença aceita pelo usuário (ver abaixo) |
| Node | 24.x | frontend da Central |
| wails3 | **v3.1.0** | `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0` |
| Gradle | 9.9.1 | via wrapper (`mobile/gradlew`); não precisa instalar |

Arquivo de ambiente usado nesta máquina (`~/.config/sitposse7-env.sh`; carregue-o em **toda** chamada de shell não interativa):
```bash
export JAVA_HOME="$HOME/.local/jdk/current"
export ANDROID_HOME="$HOME/Android/Sdk"
export ANDROID_SDK_ROOT="$ANDROID_HOME"
export PATH="$HOME/.local/go/bin:$HOME/go/bin:$JAVA_HOME/bin:$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$PATH"
```

**Armadilhas já encontradas (não repetir):**
- **Licença do SDK:** o `sdkmanager` atual diz que `--licenses` "não é mais necessário", mas o AGP 9.5.1 exige o aceite. Instale os pacotes pelo `sdkmanager --install "platforms;android-37.0"` (ele grava `package.xml` e o aceite). Plataforma extraída à mão, sem `package.xml`, não é reconhecida pelo AGP. Aceitar a licença é decisão do usuário: pergunte antes.
- **Android CLI coleta métricas:** o `sdkmanager` agora delega para o "Android CLI" do Google, que envia métricas de uso por padrão. Use `--no-metrics` e evite chamá-lo sem necessidade.
- **O wrapper `sdkmanager` não aceita `--no-metrics`** (mostra a ajuda e falha). Chame o binário `android` diretamente, com a opção **antes** do subcomando:
  `android --no-metrics --sdk="$ANDROID_HOME" sdk install "platforms/android-37.0" "build-tools/36.0.0" "platform-tools"` (grava `package.xml` e `licenses/`).
- **Sem `unzip`** (e sem sudo): extraia o cmdline-tools com o `zipfile` do Python, aplicando as permissões de `external_attr` (senão os scripts de `bin/` perdem o `+x`).
- **Chave de debug por máquina:** o APK gerado em outra máquina não instala por cima do anterior ("conflito com pacote existente"). Desinstale o app no celular antes (só há dados de diagnóstico na Fase 0).
- **Meça a rede antes de baixar:** na 2ª máquina o WSL em NAT fez ~15 MB/s e não precisou da rede espelhada.
- **WSL com `networkingMode=mirrored`:** deixa a rede rápida, mas **reserva no Windows as portas 44620–48715**. Por isso a Central usa a porta 38807 (`porta_test.go` impede regressão). O `netsh ... excludedportrange` não mostra essa reserva.
- **JUnit 5 ignora em silêncio teste que não devolve `Unit`:** `fun x() = runBlocking { …; assertNotNull(y) }` devolve valor e o teste **não roda**. Use `runBlocking<Unit> { }` e confira a contagem de testes.
- **Comentário aninhado no Kotlin:** `/* … schemas/*.v1 … */` abre outro comentário dentro do primeiro e "some" com o resto do arquivo (erros de referência não resolvida em outro arquivo).
- **Testes de porta no Windows:** use só `127.0.0.1`. Abrir um socket em `0.0.0.0` dispara o pedido de permissão do Firewall, e o clique em "Permitir" cria uma regra de entrada (aconteceu com o `powershell.exe`).
- **Sem emulador** nesta máquina (sem acesso a `/dev/kvm`). Teste em aparelho real.
- O USB do celular não chega ao WSL: instale o APK pelo Windows (`adb` do Windows) ou copie o arquivo para o aparelho.

## 4. Compilar e testar (o que deve passar antes de qualquer commit)

```bash
source ~/.config/sitposse7-env.sh

# Android (de mobile/): 22 testes, incluindo o teste cruzado com o servidor Go real
cd mobile
./gradlew test :core:dados:jvmTest :app:testDebugUnitTest
./gradlew :app:assembleDebug            # → app/build/outputs/apk/debug/app-debug.apk

# Central (de desktop/)
cd ../desktop
go vet ./internal/... ./app/... ./cmd/...
go test ./internal/... ./app/...
(cd frontend && npm ci)                 # numa máquina nova
(cd frontend && npm test)               # Vitest: tabelas e filtros (11)
wails3 build GOOS=windows               # → bin/sitposse7-central.exe (no Windows: wails3 build)

# Instalador NSIS (pelo WSL, com o makensis do Linux). Antes, coloque o WebView2 offline
# (Evergreen Standalone x64, https://go.microsoft.com/fwlink/?linkid=2124701; confira a assinatura
# da Microsoft com Get-AuthenticodeSignature no Windows) em build/windows/nsis/ com o nome
# MicrosoftEdgeWebView2RuntimeInstallerX64.exe (ignorado pelo git).
wails3 package GOOS=windows             # → bin/sitposse7-central-amd64-installer.exe (~222 MB)

# Ícone (só se mudar build/appicon.svg). Renderiza com o Edge do Windows (sem dependência nova)
# e gera o .ico. Confira o PNG antes: um SVG inválido vira uma página de erro sem falhar o comando.
W=/mnt/c/Users/$USER/AppData/Local/Temp/sitposse7-icone; mkdir -p $W; cp build/appicon.svg $W/
"/mnt/c/Program Files (x86)/Microsoft/Edge/Application/msedge.exe" --headless --disable-gpu \
  --hide-scrollbars --default-background-color=00000000 --window-size=1024,1024 \
  --screenshot="$(wslpath -w $W/appicon.png)" "$(wslpath -w $W/appicon.svg)"
cp $W/appicon.png build/appicon.png
(cd build && wails3 generate icons -input appicon.png -windowsfilename windows/icon.ico -macfilename "")
```

Contagem esperada (14/11, noite): domínio 13 (área/normalização/QR 8 + `CadastroTest` 5), sync 20 (`PinTest` 1, `ClienteCentralGoTest` 6, `ContratosTest` 4, `SincronizacaoGoTest` 3, `FechamentoConexaoTest` 2, `PareamentoGoTest` 4), banco 9 (`PragmasTest` 2, `PragmaDuravelTest` 2, `MigracaoTest` 1, `RepositorioCampoTest` 4), app 6 (QR 3, `ArmazemFotosTest` 3): total 48. Se a contagem cair, algo foi pulado.

Observações:
- O `ClienteCentralGoTest` compila `desktop/cmd/central-teste` e **falha** (não é pulado) sem Go no `PATH` (ou em `SITPOSSE7_GO`).
- O `go test ./...` na raiz do `desktop/` falha no Linux porque o `main` do Wails precisa de GTK e de `frontend/dist`. Isso é esperado; teste os pacotes `internal`, `app` e `cmd`.
- O `.exe` fica travado no Windows enquanto a Central estiver aberta: feche-a antes de copiar uma versão nova.

## 5. Teste em aparelho real (roteiro rápido)
1. Gerar o APK e o `.exe` (§4) e copiá-los para o Windows.
2. Ligar o hotspot (Windows: Configurações → Rede e Internet → Hotspot móvel; desativar a "Economia de energia") e conectar o celular a ele (pelo QR **do Windows**, com a câmera comum).
3. Abrir a Central **uma única vez**. Na primeira execução, permitir no Firewall marcando **redes públicas**. Conferir "Servidor: Ativo", com a porta 38807. Clicar em **Atualizar** se o hotspot foi ligado depois de abrir a Central.
4. No app: conferir o cartão **Banco de dados**, tocar em **Ler QR da Central** (o QR **da Central**) e depois em **Testar conexão**.
5. Anotar: aparelho, versão do Android, tipo de rede, versão TLS, tempo em ms e mensagens de erro. Registrar em `fase-0.md` §Testes em aparelho real.

## 6. Proteções contra regressão (não desfazer)

| Proteção | Onde | Por quê |
|---|---|---|
| Banco aberto **só** por `configurarPadrao()` (`DriverDuravel` + WAL) | `mobile/core/dados` | Sem isso o Room usa `synchronous=NORMAL` e desliga `foreign_keys`. O `PragmasTest` detecta |
| Nenhuma dependência com telemetria ou Play Services | `mobile/app` | Conferir com `./gradlew :app:dependencies --configuration releaseRuntimeClasspath \| grep -iE "datatransport\|firebase\|play-services\|gms\|mlkit"` → deve sair vazio (ADR-0015) |
| Suítes TLS espelhadas nos dois lados | `SuitesTLS12` (Go) e `SUITES_TLS` (Kotlin) | Mudar só um lado quebra Android 8/9 ou abre suítes fracas (ADR-0014). Testes: `tls_test.go`, `ClienteCentralGoTest` |
| Nunca aceitar qualquer certificado | `TrustManagerComPin`, `ConfigClienteComPin` | ADR-0007 / AGENTS.md §5 |
| Pin calculado igual em Go, Kotlin e OpenSSL | `shared/test-fixtures/tls/pin.json` | Uma divergência impede o pareamento |
| Porta fora de faixas reservadas | `porta_test.go` | Bug real de 13/11 |
| Decodificação do QR em UTF‑8 | `DecodificadorQr` | A Central gera o QR em UTF‑8 sem ECI; sem a dica, os acentos se corrompem |
| `allowBackup=false` + `dataExtractionRules` excluindo tudo | `AndroidManifest.xml` | R10 |
| Esquema Room exportado e versionado | `mobile/core/dados/schemas/` | Migrações não destrutivas (AGENTS.md §3) |

## 7. Regras de trabalho deste projeto (lembrete)
- Siga o [AGENTS.md](../../AGENTS.md) raiz e os de `mobile/` e `desktop/`.
- **Commits sem linha `Co-Authored-By`** nem outra atribuição a assistente (preferência do usuário). Mensagens em português, no padrão `tipo(escopo): descrição`.
- Commit com código, testes, documentação e `docs/changelog.md` juntos. O push é feito pelo usuário.
- Mudou uma decisão aprovada? Escreva um ADR novo e atualize a matriz. Dúvida funcional, de segurança ou de UX: pergunte e registre em `pendencias.md`.
- Antes de usar uma biblioteca, consulte a documentação atual (Context7 ou o site oficial). Fixe versões exatas em `mobile/gradle/libs.versions.toml` ou no `go.mod`.
