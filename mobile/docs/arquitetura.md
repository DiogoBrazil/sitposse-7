# Arquitetura — Android

Status: aprovado com o plano; esqueleto criado na Fase 0 (`:app`, `:core:dominio`, `:core:dados`, `:core:sync`). Os módulos `:feature:*` entram a partir da Fase 1.

## Módulos Gradle (proposta)
| Módulo | Conteúdo |
|---|---|
| `:app` | Activity, navegação, injeção de dependência manual (sem framework, por ora) |
| `:core:dominio` | Modelos, regras (área, normalização, validações, leitura do QR), máquina de estados do lote. Kotlin puro, testável na JVM |
| `:core:dados` | Room 3 (entidades, DAOs, migrações), repositórios, armazenamento de fotos. Módulo KMP (android + jvm): os testes JVM do host usam o mesmo `BundledSQLiteDriver` do aparelho |
| `:core:sync` | Cliente HTTPS (OkHttp + pinning), lotes, pacote `.s7pkg`, recibos. Kotlin JVM puro; recebe o `SocketFactory` da `Network` Wi‑Fi |
| `:feature:equipe`, `:feature:propriedades`, `:feature:sync`, `:feature:config` | Telas Compose + ViewModels |

Concentrar o Room em `:core:dados` segue a recomendação do Room 3 de isolar o KSP.

## Camadas
UI (Compose) → ViewModel (`StateFlow`, `SavedStateHandle`) → Repositório → DAO Room / FotoStore / GnssService / SyncClient.

## Stack
Kotlin · Compose BOM · Material 3 · Navigation Compose · Room 3.0.x + BundledSQLiteDriver · CameraX · ZXing core (ADR-0015) · OkHttp · kotlinx.serialization · coroutines. Versões exatas: `mobile/gradle/libs.versions.toml` (resumo em `docs/planning/fase-0.md`).
WorkManager **não** é usado para sincronização: ela é iniciada pelo usuário e acompanhada na tela, em primeiro plano. Avaliar uso na Fase 3 só para retomada automática.

## SDK
`minSdk 26` (A15), `targetSdk` e `compileSdk 37` (Compose 1.12 exige compileSdk 37; P7). Distribuição por sideload (P7).

## Rede de sincronização
- `ConnectivityManager.requestNetwork(NetworkRequest{TRANSPORT_WIFI}, ...)` **sem** exigir `NET_CAPABILITY_INTERNET`; o OkHttp usa `network.socketFactory` (e `network.openConnection` quando aplicável).
- Permissões: `INTERNET`, `ACCESS_NETWORK_STATE`, `CHANGE_NETWORK_STATE` (exigida por `requestNetwork`), `ACCESS_LOCAL_NETWORK` (API 37+, pedida em tempo de execução), `CAMERA`, `ACCESS_FINE_LOCATION` (Fase 1).
- TLS 1.3 preferido; TLS 1.2 só com ECDHE + AEAD, para Android 8/9 (ADR-0014).
- Fonte sobre Android 17: https://developer.android.com/privacy-and-security/local-network-permission
