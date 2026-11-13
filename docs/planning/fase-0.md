# Fase 0 — Fundação e spikes de risco (diário)

Iniciada em 2026-11-13. Plano: [plano.md](plano.md) · Rede: [rede-em-campo.md](../operacao/rede-em-campo.md).

## Ambiente de desenvolvimento (WSL2, sem sudo)
| Ferramenta | Versão | Local |
|---|---|---|
| JDK | Temurin 21.0.13+9 (SHA-256 verificado) | `~/.local/jdk/current` |
| Go | 1.27.4 (SHA-256 verificado) | `~/.local/go` |
| Android SDK | cmdline-tools 16111833; platforms;android-37.0 r02; build-tools 37.0.0; platform-tools 37.0.3 (SHA-1 verificados) | `~/Android/Sdk` |
| wails3 | v3.1.0 | `~/go/bin` |
| Node | 24.20.0 | sistema |

Variáveis: `source ~/.config/sitposse7-env.sh` (também carregado pelo `~/.bashrc` em shells interativos).
Os pacotes foram baixados pelo Windows porque a rede do WSL estava a 40–110 KB/s. A correção (rede espelhada) está sendo aplicada.
O `sdkmanager` atual dispensa `--licenses` ("no longer needed").

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
| ML Kit barcode-scanning (empacotado) | 17.3.0 | |
| coroutines / serialization / OkHttp | 1.11.0 / 1.13.0 / 5.5.1 | |

## Spikes
| Spike | Situação | Resultado |
|---|---|---|
| (a) HTTPS com pin via QR | **Central pronta**; cliente Android pendente | Go: TLS 1.3, pin SPKI base64url (43 caracteres), pin estável ao reemitir o certificado com IP novo, pin errado recusado, cliente sem pin recusa o certificado autoassinado, só rede local (RFC 1918 / link-local / loopback). Contratos validados por JSON Schema |
| (b) Hotspot Windows / tampa | Pendente (usuário, notebook real) | Roteiro em `rede-em-campo.md`. Executável: `desktop/bin/sitposse7-central.exe` |
| (c) PRAGMAs Room 3 | Pendente | Hipótese: o Room 3 pode forçar `synchronous=NORMAL` em WAL. Plano: `:core:dados` como módulo KMP (android + jvm), com testes jvm no host usando o mesmo BundledSQLiteDriver e diagnóstico no app real. Também fixar `setJournalMode(WRITE_AHEAD_LOGGING)`, porque o modo automático do Room usa TRUNCATE em aparelhos com pouca RAM |
| (d) Instalador NSIS | Script pronto; teste pendente (Windows) | WebView2 **offline** (Evergreen Standalone x64 em `build/windows/nsis/`), regra de firewall `localsubnet` na porta 47807 (perfis Privado e Público) e remoção na desinstalação. Dados da Central nunca são apagados. Falta: instalar NSIS no Windows, gerar e testar em Windows 10 22H2 e 11 limpos sem internet |

## Achados
- O template do Wails gera artefatos para várias plataformas. Mantido só Windows; a geração de ícones precisa de `-macfilename ""`.
- O bootstrapper padrão do WebView2 no Wails é **online**; foi trocado pelo instalador offline (RNF-15).
- O gerador de bindings do Wails expõe **todos** os métodos públicos do serviço. Por isso o ciclo de vida fica em `app.Central`, e só `SincronizacaoService` (leitura) é exposto.
- Porta padrão da sincronização: **47807**.

## Próximos passos
1. Projeto Android: `:app` (Compose), `:core:dominio` (JVM puro + fixtures), `:core:dados` (KMP android+jvm, Room 3, testes jvm), `:core:sync` (OkHttp + TrustManager com pin SPKI).
2. Teste cruzado Kotlin × Go: mesmo pin para o mesmo certificado (fixture gerada pelo Go) e cliente Kotlin JVM contra o servidor Go real.
3. App de diagnóstico: ler QR (CameraX + ML Kit), forçar a rede Wi-Fi, pedir `ACCESS_LOCAL_NETWORK` (API 37), chamar `/v1/status` e mostrar os PRAGMAs efetivos do banco.
4. Usuário: testes em aparelho real (hotspot de celular e do Windows, Android 17) e do instalador.
