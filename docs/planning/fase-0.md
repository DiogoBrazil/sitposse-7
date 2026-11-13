# Fase 0 — Fundação e spikes de risco (diário)

Iniciada em 2026-11-13. Plano: [plano.md](plano.md) · Rede: [rede-em-campo.md](../operacao/rede-em-campo.md).

## Ambiente de desenvolvimento (WSL2, sem sudo)
| Ferramenta | Versão | Local |
|---|---|---|
| JDK | Temurin 21.0.13+9 (SHA-256 verificado) | `~/.local/jdk/current` |
| Go | 1.27.4 (SHA-256 verificado) | `~/.local/go` |
| Android SDK | cmdline-tools 16111833; platforms;android-37.0 r02; build-tools 37.0.0 e 36.0.0 (padrão do AGP); platform-tools 37.0.3 | `~/Android/Sdk` |
| Gradle | 9.9.1 (wrapper com `distributionSha256Sum`; SHA-256 do `gradle-wrapper.jar` conferido com o oficial) | `mobile/gradlew` |
| wails3 | v3.1.0 | `~/go/bin` |
| Node | 24.20.0 | sistema |

Variáveis: `source ~/.config/sitposse7-env.sh` (também carregado pelo `~/.bashrc` em shells interativos).
Os primeiros pacotes foram baixados pelo Windows porque a rede do WSL estava a 40–110 KB/s. Com `networkingMode=mirrored` no `.wslconfig`, o WSL passou a ~7,5 MB/s (medido em 13/11) e baixa direto.
Licença do Android SDK aceita pelo usuário em 13/11 (`~/Android/Sdk/licenses`). O `sdkmanager` diz que `--licenses` "não é mais necessário", mas o AGP 9.5.1 continua exigindo o aceite. A plataforma 37 tinha sido extraída sem `package.xml` e foi reinstalada pelo `sdkmanager`.
**Atenção:** o `sdkmanager` agora delega para o novo "Android CLI", que coleta métricas de uso por padrão. Isso afeta só a máquina de desenvolvimento; para desligar, use `--no-metrics`.

**Segunda máquina (13/11, PC desktop, Windows 11 Pro build 26300, WSL2 em NAT):** mesmas versões de JDK e Go (SHA-256 conferidos); cmdline-tools 16111833 (SHA-1 conferido com o `repository2-3.xml`); `platforms;android-37.0` r02, `build-tools;36.0.0` e platform-tools 37.0.1, instalados pelo Android CLI com métricas desligadas. A rede do WSL em NAT mediu ~15 MB/s, sem precisar da rede espelhada. Achados:
- O wrapper `sdkmanager` **não aceita** `--no-metrics` (mostra a ajuda e sai com erro). Use o binário `android` diretamente: `android --no-metrics --sdk="$ANDROID_HOME" sdk install "platforms/android-37.0" "build-tools/36.0.0" "platform-tools"`. Ele grava o `package.xml` e o aceite em `licenses/`.
- O `unzip` não estava instalado (sem sudo). O cmdline-tools foi extraído com o `zipfile` do Python, preservando as permissões de execução.
- Cada máquina gera a própria chave de debug (`~/.android/debug.keystore`). O APK de outra máquina não instala por cima ("conflito com pacote existente"): desinstale o app antes.

## Versões fixadas (P9)
| Componente | Versão | Observação |
|---|---|---|
| Wails | 3.1.0 | `wails3 build GOOS=windows` funciona do WSL, sem CGO |
| modernc.org/sqlite | 1.62.0 | |
| golang.org/x/text | (go.sum) | NFD para normalização |
| santhosh-tekuri/jsonschema | 6.0.3 | só testes |
| Vue / @wailsio/runtime / qrcode | 3.5.45 / 3.1.0 / 1.5.4 | |
| AGP | 9.5.1 | exige Gradle ≥ 9.6, JDK 17+, API máx. 37; Kotlin embutido (sem `kotlin-android`) |
| Gradle | 9.9.1 | |
| Kotlin | 2.4.21 | |
| KSP | 2.3.14 | 2.3.13 tem regressão conhecida (issue 3286) |
| Compose BOM | 2026.09.00 | |
| Room 3 | 3.0.3 | `androidx.room3` |
| androidx.sqlite (bundled) | 2.7.1 | |
| activity-compose / lifecycle / navigation | 1.13.0 / 2.11.0 / 2.10.2 | |
| core-ktx / exifinterface | 1.19.1 / 1.4.2 | |
| CameraX | 1.6.2 | |
| ZXing core | 3.5.4 | substitui o ML Kit (ADR-0015) |
| JUnit (BOM) | 6.1.3 | testes JVM, com kotlin-test |
| coroutines / serialization / OkHttp | 1.11.0 / 1.13.0 / 5.5.1 | |

## Spikes
| Spike | Situação | Resultado |
|---|---|---|
| (a) HTTPS com pin via QR | **Central e cliente prontos; teste cruzado JVM×Go passando**; falta o aparelho real | Go: pin SPKI base64url (43 caracteres), pin estável ao reemitir o certificado com IP novo, pin errado recusado, cliente sem pin recusa o certificado autoassinado, só rede local (RFC 1918 / link-local / loopback). Contratos validados por JSON Schema. **Kotlin** (`:core:sync`): `TrustManager` e `HostnameVerifier` pelo pin; mesmo pin que Go e OpenSSL (fixture `tls/pin.json`); contra o servidor Go real: TLS 1.3, TLS 1.2 forçado (Android 8/9), pin de outra Central recusado, porta fechada. TLS 1.2 restrito adotado (ADR-0014). **App de diagnóstico** (`:app`): lê o QR (CameraX + ZXing), pede `ACCESS_LOCAL_NETWORK` no Android 17, usa `requestNetwork(TRANSPORT_WIFI)` sem exigir internet e mostra o resultado de `/v1/status` |
| (b) Hotspot Windows / tampa | **Cancelado** (A21, ADR-0016, 13/11) | O hotspot do Windows só será usado com internet na origem e com o computador ligado, e isso já foi validado duas vezes (ver §Testes em aparelho real). Os testes sem internet, com Adaptador de Loopback e com tampa fechada saíram do escopo. P8 encerrada |
| (c) PRAGMAs Room 3 | **Concluído:** JVM e aparelho real (moto g56, Android 16, 13/11: escrita e leitura `wal / 2 / 1`, SQLite 3.50.1, "Configuração de durabilidade correta") | **Hipótese confirmada:** Room 3.0.3 + WAL deixava todas as conexões com `synchronous=1` (NORMAL) e `foreign_keys=0`. O código do Room (`RoomConnectionManager`) executa `PRAGMA synchronous = NORMAL` em cada conexão do pool depois do `driver.open()`, e `Callback.onOpen` só roda na primeira conexão. **Correção:** `DriverDuravel` envolve o `BundledSQLiteDriver`, liga `foreign_keys=ON` e `synchronous=FULL` na abertura e eleva para FULL qualquer `PRAGMA synchronous` menor. Resultado (`PragmasTest`, escrita + 4 leituras + reabertura): `wal / 2 / 1`, SQLite 3.50.1. `setJournalMode(WRITE_AHEAD_LOGGING)` fixo |
| (d) Instalador NSIS | **Instalador gerado (13/11)**; teste pendente em VM limpa | WebView2 **offline** (Evergreen Standalone x64 em `build/windows/nsis/`), regra de firewall `localsubnet` na porta 38807 (perfis Privado e Público) e remoção na desinstalação. Dados da Central nunca são apagados. Gerado pelo WSL com `makensis` 3.09 (`wails3 package GOOS=windows`): `sitposse7-central-amd64-installer.exe`, 222 MB, SHA-256 `c4906bdb…d4fa05a7`. WebView2 Evergreen Standalone x64 baixado do link oficial (fwlink 2124701), 215 MB, SHA-256 `d3cbe8fe…a974f9c3`, assinatura Authenticode **válida** (Microsoft Corporation). Falta: testar em Windows 11 (VM Hyper-V sem rede) e, se possível, Windows 10 22H2. O Windows 11 já traz o WebView2, então a instalação offline do WebView2 só é exercitada no Windows 10 |

## Achados
- O template do Wails gera artefatos para várias plataformas. Mantido só Windows; a geração de ícones precisa de `-macfilename ""`.
- O bootstrapper padrão do WebView2 no Wails é **online**; foi trocado pelo instalador offline (RNF-15).
- O gerador de bindings do Wails expõe **todos** os métodos públicos do serviço. Por isso o ciclo de vida fica em `app.Central`, e só `SincronizacaoService` (leitura) é exposto.
- Porta padrão da sincronização: **38807**. A anterior (47807) ficou inacessível no primeiro teste real (13/11): a rede espelhada do WSL reserva no Windows a faixa 44620–48715 (portas efêmeras do Linux), e o `netsh ... excludedportrange` **não** mostra essa reserva. `porta_test.go` impede voltar para uma faixa reservada.
- A Central aceita várias instâncias abertas; a segunda mostra "porta em uso". Corrigir com instância única (Fase 2).
- O QR inclui IPs de adaptadores virtuais (WSL/Hyper-V/Docker, ex.: 172.18.0.1); o app tenta os IPs em sequência, com até 5 s por IP que não responde. Corrigir: filtrar adaptadores virtuais e tentar os IPs em paralelo.
- **TLS 1.3 × Android 8/9:** a plataforma só tem TLS 1.3 a partir do Android 10, e o `minSdk` é 26. Resolvido com TLS 1.2 restrito (ECDHE + AEAD) como mínimo (A19, ADR-0014).
- **ML Kit traz telemetria:** `barcode-scanning:17.3.0` (empacotado) depende de `transport-backend-cct` (métricas de uso do Google), `play-services-*` e Firebase. Trocado por ZXing (A20, ADR-0015). APK de debug sem nenhuma dessas dependências (conferido em `:app:dependencies` e no manifesto final).
- **Room 3 rebaixa `synchronous`:** ver spike (c).
- **AGP 9 + KMP:** `:core:dados` usa `com.android.kotlin.multiplatform.library` (bloco `kotlin { android { } }`); o `room3-runtime` precisa ser `api`, porque o `BancoCampo` herda de `RoomDatabase`. Com `@ConstructedBy`, é preciso `-Xexpect-actual-classes`.
- `applicationId` provisório `br.gov.ro.pm.sitposse7.campo` (P10).

## Testes automatizados (13/11)
- Kotlin: `./gradlew test :core:dados:jvmTest :app:testDebugUnitTest` → 22 testes passando (domínio com fixtures 8, sync 7, banco 4, QR 3).
- Go: `go test ./internal/... ./app/...` passando (novos: `TestPinFixtureTLS`, `tls_test.go`). O pacote raiz (`main` do Wails) só compila para Windows.

## Testes em aparelho real
| Data | Aparelho | Rede | Resultado |
|---|---|---|---|
| 13/11 | Motorola moto g56 5G, **Android 16** | Hotspot do Windows 11 (origem com internet: "Wi‑Fi 2"); Central `sitposse7-central-38807.exe` | **Conectado em TLSv1.3, 108 ms** pela leitura do QR da Central. No Android 16 não há a permissão de rede local (só no 17). Pendente: o cartão "Banco de dados" |
| 13/11 | Motorola moto g56 5G, **Android 16** (APK gerado na 2ª máquina) | Hotspot do Windows 11 num PC desktop (origem com internet: Ethernet; banda 5 GHz; "Economia de energia" desativada). Celular em `192.168.137.64`; QR com `192.168.0.24 · 192.168.137.1 · 172.18.64.1` | **Cartão "Banco de dados": escrita e leitura `journal=wal · synchronous=2 · foreign_keys=1`, SQLite 3.50.1, "Configuração de durabilidade correta"** (fecha o spike (c) no aparelho). **Conectado a 192.168.0.24 em TLSv1.3, 92 ms.** Achado: o celular alcançou a Central pelo IP da **Ethernet**, não pelo do hotspot, porque o compartilhamento de internet do Windows encaminha entre as interfaces. Sem a origem, esse IP some e o app perde até 5 s nele antes de tentar o `192.168.137.1` (reforça a melhoria "tentar os IPs em paralelo / filtrar adaptadores") |

## Próximos passos
Lista ordenada e atualizada: [continuacao.md](continuacao.md) §2.

1. ~~Projeto Android~~ · ~~teste cruzado Kotlin × Go~~ · ~~app de diagnóstico~~ (13/11).
2. **Usuário, em aparelho real** (APK: `mobile/app/build/outputs/apk/debug/app-debug.apk`; instalar com `adb install -r`):
   - ~~a tela de diagnóstico mostra "Configuração de durabilidade correta" (`wal / 2 / 1`)~~ (13/11, moto g56);
   - ler o QR da Central e testar a conexão no **hotspot do celular** (sem internet); anotar a versão TLS e o tempo. O hotspot do Windows com internet já foi validado (13/11), e sem internet não será usado (A21);
   - um aparelho com Android 17 (permissão de rede local) e, se houver, um com Android 8 ou 9 (TLS 1.2);
   - instalador NSIS em Windows 10 22H2 e 11 limpos. ~~Roteiro de rede/tampa~~ cancelado (A21, ADR-0016).
3. Com os resultados: fechar P7 (P8 já encerrada pela A21), registrar aqui e dar a Fase 0 por concluída.

**Adiados pelo usuário em 13/11/2026, sem bloquear a conclusão da Fase 0 (A22)** (fazer depois): o teste no hotspot do celular (cenário A) e o teste em Android 17 (P7) e em Android 8/9. Lista em [continuacao.md](continuacao.md) §2.
