# Arquitetura — Android

Status: aprovado com o plano; o projeto será criado na Fase 0.

## Módulos Gradle (proposta)
| Módulo | Conteúdo |
|---|---|
| `:app` | Activity, navegação, injeção de dependência manual (sem framework, por ora) |
| `:core:dominio` | Modelos, regras (área, normalização, validações), máquina de estados do lote. Kotlin puro, testável na JVM |
| `:core:dados` | Room 3 (entidades, DAOs, migrações), repositórios, armazenamento de fotos |
| `:core:sync` | Cliente HTTPS (OkHttp + pinning), lotes, pacote `.s7pkg`, recibos |
| `:feature:equipe`, `:feature:propriedades`, `:feature:sync`, `:feature:config` | Telas Compose + ViewModels |

Concentrar o Room em `:core:dados` segue a recomendação do Room 3 de isolar o KSP.

## Camadas
UI (Compose) → ViewModel (`StateFlow`, `SavedStateHandle`) → Repositório → DAO Room / FotoStore / GnssService / SyncClient.

## Stack
Kotlin ≥ 2.2 · Compose BOM (fixar na Fase 0, linha 2026.08+) · Material 3 · Navigation Compose · Room 3.0.x + BundledSQLiteDriver · CameraX · ML Kit Barcode (empacotado) · OkHttp · kotlinx.serialization · coroutines.
WorkManager **não** é usado para sincronização: ela é iniciada pelo usuário e acompanhada na tela, em primeiro plano. Avaliar uso na Fase 3 só para retomada automática.

## SDK
`minSdk 26` (A15), `targetSdk` e `compileSdk 37` (Compose 1.12 exige compileSdk 37; P7). Distribuição por sideload (P7).

## Rede de sincronização
- `ConnectivityManager.requestNetwork(NetworkRequest{TRANSPORT_WIFI}, ...)` **sem** exigir `NET_CAPABILITY_INTERNET`; o OkHttp usa `network.socketFactory` (e `network.openConnection` quando aplicável).
- Permissões: `INTERNET`, `ACCESS_NETWORK_STATE`, `ACCESS_WIFI_STATE`, `ACCESS_LOCAL_NETWORK` (API 37+), `CAMERA`, `ACCESS_FINE_LOCATION`.
- Fonte sobre Android 17: https://developer.android.com/privacy-and-security/local-network-permission
