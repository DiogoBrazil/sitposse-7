# Visão geral da arquitetura

## Componentes

```
Android "Campo"                                  Windows "Central"
┌───────────────────────────────┐               ┌──────────────────────────────────────┐
│ UI Compose (telas, ViewModels)│               │ Vue 3 + TS (Painel, Cadastros, Sync, │
│ Domínio (regras, validação,   │               │  Estatísticas, Mapa*, Relatórios)    │
│  área, estados)               │               │      ▲ bindings Wails v3             │
│ Repositórios ──► Room3/SQLite │  HTTPS+pin    │ Serviços Go                          │
│ Fotos (FS privado + SHA‑256)  │ ────────────► │  ├ servidor de sync (net/http + TLS) │
│ GNSS (LocationManager)        │  ou pacote    │  ├ importador (núcleo único p/       │
│ Cliente de sync (lotes,       │  .s7pkg (USB) │  │  Wi‑Fi e USB, idempotente)        │
│  retomada)                    │ ◄──────────── │  ├ conflitos / auditoria             │
│ Exportador pacote/backup      │  recibo       │  ├ consultas / estatísticas (SQL)    │
└───────────────────────────────┘  (QR/.s7rcb)  │  ├ backup / restauração              │
                                                │  └ cartografia* (provedor plugável)  │
                                                │ SQLite (modernc) + pasta de fotos    │
                                                └──────────────────────────────────────┘
                                                 * única parte que exige internet
```

**Princípios**
1. **Banco local é a fonte da verdade** em cada lado. A UI nunca guarda dado "salvo" só em memória.
2. **Um núcleo de importação** (`desktop/internal/importacao`) atende Wi‑Fi e pacote USB, com as mesmas validações, a mesma idempotência e os mesmos conflitos.
3. O **núcleo Go não depende do Wails**: domínio, banco, sync e relatórios são pacotes Go puros. O Wails é só a casca de UI (ADR-0003).
4. **Contratos compartilhados** em `shared/schemas` (JSON Schema 2020‑12) e **fixtures** comuns em `shared/test-fixtures`.

## Stack (versões verificadas em 13/11/2026; versões exatas fixadas na Fase 0)

| Camada | Escolha | ADR |
|---|---|---|
| Android | Kotlin ≥ 2.2, Jetpack Compose + Material 3, Coroutines/Flow, ViewModel, Navigation Compose | — |
| Persistência Android | Room 3 (`androidx.room3`, 3.0.3 estável) + `BundledSQLiteDriver` + KSP | [0002](decisions/0002-room3-bundled-sqlite.md) |
| Câmera | CameraX ImageCapture | — |
| QR Code | ZXing core (offline, sem Play Services nem telemetria) | [0015](decisions/0015-zxing-leitor-qr.md) |
| Localização | `LocationManager` / GNSS (sem Google Play Services) | — |
| HTTP Android | OkHttp + TrustManager com pinning SPKI; TLS 1.3/1.2 restrito | [0007](decisions/0007-tls-pinning-qr.md), [0014](decisions/0014-tls12-minimo-android8.md) |
| Desktop | Go (estável atual) + Wails v3.1 + Vue 3 + TypeScript + Vite | [0003](decisions/0003-wails3-nucleo-go.md) |
| SQLite Go | modernc.org/sqlite (Go puro, sem CGO) | [0004](decisions/0004-modernc-sqlite.md) |
| UI Desktop | PrimeVue (tabelas), Apache ECharts (gráficos), Leaflet 1.9 (mapa), todos empacotados localmente | — |
| PDF | Biblioteca Go mantida (escolha na Fase 5) | — |
| Instalador | NSIS + WebView2 embutido + regra de firewall | — |

## Fluxos principais
- **Coleta:** formulário → (rascunho em `rascunhos`) → Salvar → transação Room (entidade + `rev++`) → "Salvo no aparelho ✓".
- **Foto:** CameraX → `tmp` → redimensiona e grava `.part` com SHA‑256 → fsync → rename → transação no banco. Detalhes em [mobile/docs/persistencia-offline.md](../../mobile/docs/persistencia-offline.md).
- **Sincronização:** QR → pareamento TLS com pinning → lote (registros) → fotos em blocos → importar → recibo → confirmação local. Ver [protocolo.md](../synchronization/protocolo.md).
- **Contingência:** pacote `.s7pkg` → USB → importação na Central → recibo (QR ou `.s7rcb`) → confirmação no Android.
- **Consulta:** `FiltroEstudo` → SQL parametrizado sobre as visões vigentes → cartões, tabelas, gráficos, listas, mapa e exportações.
- **Mapa:** teste de conectividade com o provedor → Leaflet (tiles online) ou mensagem amigável (RN-91).
