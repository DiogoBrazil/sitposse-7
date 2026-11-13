# Arquitetura — Central (Windows)

## Estrutura de pastas (proposta)
```
desktop/
├── main.go                 # inicialização Wails
├── app/                    # bindings finos (serviços expostos ao frontend)
├── internal/
│   ├── banco/              # conexão, PRAGMAs, migrações (embed), visões
│   ├── dominio/            # tipos, área, normalização, validações
│   ├── importacao/         # núcleo único: validar, publicar fotos, aplicar, recibo
│   ├── syncserver/         # HTTPS (net/http + crypto/tls), pareamento, endpoints v1
│   ├── conflitos/
│   ├── auditoria/
│   ├── consultas/          # FiltroEstudo → SQL, estatísticas, drill-down
│   ├── relatorios/         # PDF/CSV
│   ├── backup/             # VACUUM INTO, cópia de fotos, verificação, restauração
│   ├── cartografia/        # provedor plugável, teste de conectividade, carta-imagem
│   └── reconciliador/
└── frontend/               # Vue 3 + TS + Vite (PrimeVue, ECharts, Leaflet)
```

## Dados em disco
`%LOCALAPPDATA%\SITPOSSE-7\` (configurável):
- `sitposse7.db` (+ `-wal`, `-shm`)
- `fotos/{aa}/{uuid}.jpg`
- `staging/`
- `tls/` (chave e certificado da Central; entram no backup)
- `logs/`

## Servidor de sincronização
- `net/http` + `crypto/tls` (TLS 1.3 preferido; TLS 1.2 só com ECDHE-ECDSA + AEAD, ADR-0014), certificado ECDSA P‑256 autoassinado, SAN com os IPs locais, porta fixa (padrão **38807**; ver `porta_test.go`).
- `cmd/central-teste`: só o servidor de sincronização, sem interface, usado pelos testes cruzados do Android (`mobile/core/sync`). Não faz parte do produto.
- Escuta em todas as interfaces; recusa clientes de fora de faixas privadas (RFC 1918 e link-local).
- `SetThreadExecutionState(ES_CONTINUOUS|ES_SYSTEM_REQUIRED)` durante a recepção (não impede a suspensão pela tampa).

## Instalador
NSIS (build do Wails) com:
- WebView2 embutido (bootstrapper offline/standalone) para instalar sem internet;
- regra de firewall de entrada para o executável, `RemoteAddress=LocalSubnet`, perfis Privado e Público;
- atalho no menu Iniciar.
Validado em Windows 10 22H2 e 11 limpos, sem internet (Fase 0).

## Stack
Go (estável atual, fixar na Fase 0) · Wails v3.1.x · Vue 3 + TypeScript + Vite · PrimeVue · Apache ECharts (vue-echarts) · Leaflet 1.9 + markercluster · biblioteca PDF (Fase 5).
