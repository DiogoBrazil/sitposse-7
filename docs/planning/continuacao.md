# Continuação do trabalho (passagem de bastão)

> **Leia isto primeiro** ao retomar o projeto em outra máquina ou com outro agente.
> Atualizado em **13/11/2026**, ao encerrar a sessão. Mantenha este arquivo em dia ao final de cada sessão.

## 1. Onde estamos

- **Fase atual:** Fase 0 (fundação e spikes de risco). O plano está em [plano.md](plano.md); o diário detalhado da fase, em [fase-0.md](fase-0.md).
- **Branch de trabalho:** `development`. A `main` só recebe o que for aprovado.
- **Últimos commits relevantes:**
  - `02752bb` feat(mobile): projeto Android, teste cruzado com a Central e spikes da Fase 0
  - `fb8382b` fix(desktop): porta padrão da Central 47807 → 38807
  - `a3dc8eb` docs(fase-0): resultado do teste no moto g56
- **Nenhuma funcionalidade da Fase 1 foi iniciada, e não deve ser** até a Fase 0 ser dada como concluída pelo usuário (AGENTS.md §2).

### Já feito e verificado
| Item | Situação | Evidência |
|---|---|---|
| Central (Go + Wails 3) | Esqueleto, banco com migração 0001, servidor HTTPS `GET /v1/status`, QR na tela de diagnóstico | `go test ./internal/... ./app/...` |
| App Android | Esqueleto (`:app`, `:core:dominio`, `:core:dados`, `:core:sync`) e tela de diagnóstico | `./gradlew test ...` (22 testes) |
| Spike (a) HTTPS + pin via QR | Cliente Kotlin × servidor Go real: TLS 1.3, TLS 1.2, pin divergente. **Aparelho real OK:** moto g56, Android 16, hotspot do Windows **com** internet na origem, TLSv1.3, 108 ms | `ClienteCentralGoTest`; `fase-0.md` §Testes em aparelho real |
| Spike (c) PRAGMAs Room 3 | Room rebaixava `synchronous` para NORMAL e deixava `foreign_keys` desligado; corrigido com `DriverDuravel`. **Aparelho real OK** (moto g56, 13/11: `wal · 2 · 1`, SQLite 3.50.1); na mesma sessão, nova conexão em TLSv1.3, 92 ms | `PragmasTest` (JVM); `fase-0.md` §Testes em aparelho real |
| Spike (d) Instalador NSIS | Script pronto (WebView2 offline, firewall na porta 38807) | **Não testado** |
| Spike (b) Hotspot do Windows / tampa | **Cancelado** (A21/ADR-0016): hotspot do Windows só com internet e computador ligado, já validado 2× (notebook com Wi‑Fi, PC com Ethernet). P8 encerrada | `fase-0.md`; `rede-em-campo.md` |
| Decisões novas | A19/ADR-0014 (TLS 1.2 restrito), A20/ADR-0015 (ZXing no lugar do ML Kit), porta 38807, A21/ADR-0016 (hotspot do Windows só com internet) | [matriz-decisoes.md](matriz-decisoes.md) |

## 2. O que falta para fechar a Fase 0 (em ordem)

Quase tudo depende do **usuário com aparelhos e computador reais**. O P8 (hotspot do Windows sem internet e tampa fechada) foi **encerrado sem teste** pela A21/ADR-0016: o hotspot do Windows só é usado com internet e com o computador ligado. O agente prepara os builds, orienta e registra os resultados em `fase-0.md`.

1. ~~**Cartão "Banco de dados" no aparelho**~~: **feito em 13/11** (moto g56: "Configuração de durabilidade correta", `wal · synchronous=2 · foreign_keys=1`).
2. **Spike (d), instalador:** instalar o NSIS no Windows, gerar o instalador (`wails3 package` no Windows) e testar em Windows 10 22H2 e 11 **limpos e sem internet**: WebView2 offline, regra de firewall, desinstalação sem apagar os dados.
3. Registrar tudo em `fase-0.md`, atualizar o changelog e **pedir ao usuário que aprove a conclusão da Fase 0**. Como os testes abaixo foram adiados, o critério do plano ("P7 fechada") só é cumprido fazendo-os antes ou com o usuário aprovando, no fechamento, levá-los para depois. **Não decidir isso sozinho.**

### Adiados pelo usuário (13/11/2026): fazer depois
1. **Hotspot do celular (cenário A):** um celular cria o hotspot (sem dados móveis); o computador da Central (PC com Wi‑Fi ou notebook) e outro celular se conectam a ele; testar a conexão dos dois celulares, inclusive o que criou o hotspot ([rede-em-campo.md](../operacao/rede-em-campo.md) §Cenário A).
2. **Android 17 (P7):** testar em um aparelho com Android 17 (a permissão de rede local é pedida antes de conectar). Se houver, testar também Android 8 ou 9 (deve conectar em **TLSv1.2**).

### Melhorias já identificadas (não bloqueiam a Fase 0; fazer quando o usuário aprovar ou na Fase 2)
- A Central permite várias janelas abertas ao mesmo tempo; deveria aceitar só uma (instância única do Wails). A segunda janela mostra "porta em uso".
- O QR inclui IPs de adaptadores virtuais (WSL/Hyper-V/Docker, ex.: `172.18.0.1`); o app tenta os IPs em sequência (até 5 s cada). Filtrar adaptadores virtuais na Central e/ou tentar os IPs em paralelo no app.
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
wails3 build GOOS=windows               # → bin/sitposse7-central.exe (no Windows: wails3 build)
```

Contagem esperada (13/11): domínio 8, sync 7 (`PinTest` 1 + `ClienteCentralGoTest` 6), banco 4 (`PragmasTest` 2 + `PragmaDuravelTest` 2), QR 3. Se a contagem cair, algo foi pulado.

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
