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
| (b) Hotspot Windows / tampa | Pendente (usuário, notebook real) | Roteiro em `rede-em-campo.md`. Executável: `desktop/bin/sitposse7-central.exe` |
| (c) PRAGMAs Room 3 | **Resolvido na JVM**; falta conferir no aparelho (tela de diagnóstico) | **Hipótese confirmada:** Room 3.0.3 + WAL deixava todas as conexões com `synchronous=1` (NORMAL) e `foreign_keys=0`. O código do Room (`RoomConnectionManager`) executa `PRAGMA synchronous = NORMAL` em cada conexão do pool depois do `driver.open()`, e `Callback.onOpen` só roda na primeira conexão. **Correção:** `DriverDuravel` envolve o `BundledSQLiteDriver`, liga `foreign_keys=ON` e `synchronous=FULL` na abertura e eleva para FULL qualquer `PRAGMA synchronous` menor. Resultado (`PragmasTest`, escrita + 4 leituras + reabertura): `wal / 2 / 1`, SQLite 3.50.1. `setJournalMode(WRITE_AHEAD_LOGGING)` fixo |
| (d) Instalador NSIS | Script pronto; teste pendente (Windows) | WebView2 **offline** (Evergreen Standalone x64 em `build/windows/nsis/`), regra de firewall `localsubnet` na porta 47807 (perfis Privado e Público) e remoção na desinstalação. Dados da Central nunca são apagados. Falta: instalar NSIS no Windows, gerar e testar em Windows 10 22H2 e 11 limpos sem internet |

## Achados
- O template do Wails gera artefatos para várias plataformas. Mantido só Windows; a geração de ícones precisa de `-macfilename ""`.
- O bootstrapper padrão do WebView2 no Wails é **online**; foi trocado pelo instalador offline (RNF-15).
- O gerador de bindings do Wails expõe **todos** os métodos públicos do serviço. Por isso o ciclo de vida fica em `app.Central`, e só `SincronizacaoService` (leitura) é exposto.
- Porta padrão da sincronização: **47807**.
- **TLS 1.3 × Android 8/9:** a plataforma só tem TLS 1.3 a partir do Android 10, e o `minSdk` é 26. Resolvido com TLS 1.2 restrito (ECDHE + AEAD) como mínimo (A19, ADR-0014).
- **ML Kit traz telemetria:** `barcode-scanning:17.3.0` (empacotado) depende de `transport-backend-cct` (métricas de uso do Google), `play-services-*` e Firebase. Trocado por ZXing (A20, ADR-0015). APK de debug sem nenhuma dessas dependências (conferido em `:app:dependencies` e no manifesto final).
- **Room 3 rebaixa `synchronous`:** ver spike (c).
- **AGP 9 + KMP:** `:core:dados` usa `com.android.kotlin.multiplatform.library` (bloco `kotlin { android { } }`); o `room3-runtime` precisa ser `api`, porque o `BancoCampo` herda de `RoomDatabase`. Com `@ConstructedBy`, é preciso `-Xexpect-actual-classes`.
- `applicationId` provisório `br.gov.ro.pm.sitposse7.campo` (P10).

## Testes automatizados (13/11)
- Kotlin: `./gradlew test :core:dados:jvmTest :app:testDebugUnitTest` → 22 testes passando (domínio com fixtures 8, sync 7, banco 4, QR 3).
- Go: `go test ./internal/... ./app/...` passando (novos: `TestPinFixtureTLS`, `tls_test.go`). O pacote raiz (`main` do Wails) só compila para Windows.

## Próximos passos
1. ~~Projeto Android~~ · ~~teste cruzado Kotlin × Go~~ · ~~app de diagnóstico~~ (13/11).
2. **Usuário, em aparelho real** (APK: `mobile/app/build/outputs/apk/debug/app-debug.apk`; instalar com `adb install -r`):
   - a tela de diagnóstico mostra "Configuração de durabilidade correta" (`wal / 2 / 1`);
   - ler o QR da Central e testar a conexão: hotspot do celular e hotspot do Windows, sem internet; anotar a versão TLS e o tempo;
   - um aparelho com Android 17 (permissão de rede local) e, se houver, um com Android 8 ou 9 (TLS 1.2);
   - roteiro de rede/tampa ([rede-em-campo.md](../operacao/rede-em-campo.md)) e instalador NSIS em Windows 10 22H2 e 11 limpos.
3. Com os resultados: fechar P7 e P8, registrar aqui e dar a Fase 0 por concluída.
