# Registros de decisão de arquitetura (ADRs)

Cada ADR registra contexto, decisão, alternativas, consequências e status.

**Status possíveis:** `Proposto` · `Aprovado` · `Substituído por ADR-NNNN` · `Rejeitado`.
Um ADR aprovado **não é editado** para mudar a decisão: cria-se um novo ADR que o substitui. O texto antigo só recebe a marcação de status.

| ADR | Título | Status |
|---|---|---|
| [0001](0001-monorepo-e-documentacao.md) | Monorepositório e documentação em camadas | Aprovado |
| [0002](0002-room3-bundled-sqlite.md) | Room 3 + BundledSQLiteDriver no Android | Aprovado |
| [0003](0003-wails3-nucleo-go.md) | Wails v3 com núcleo Go independente | Aprovado (validar na Fase 0) |
| [0004](0004-modernc-sqlite.md) | modernc.org/sqlite na Central | Aprovado |
| [0005](0005-rev-sentido-unico.md) | Versionamento por `rev` e sincronização em sentido único | Aprovado |
| [0006](0006-unidade-de-conflito.md) | Unidade de conflito = cadastro | Aprovado |
| [0007](0007-tls-pinning-qr.md) | TLS com pinning via QR Code | Aprovado |
| [0008](0008-pacote-usb-recibo.md) | Pacote USB de contingência com recibo | Aprovado |
| [0009](0009-area-milesimos-dm2.md) | Área em milésimos e cálculo em dm² | Aprovado |
| [0010](0010-timestamps.md) | Timestamps epoch ms / ISO 8601 UTC | Aprovado |
| [0011](0011-fotos-reduzidas.md) | Fotos reduzidas a ~2560 px | Aprovado |
| [0012](0012-entidade-operacao.md) | Entidade operação | Aprovado |
| [0013](0013-cartografia-plugavel.md) | Cartografia online com provedor plugável | Aprovado (provedor pendente: P1) |
| [0014](0014-tls12-minimo-android8.md) | TLS 1.2 restrito como mínimo (Android 8 e 9) | Aprovado |
| [0015](0015-zxing-leitor-qr.md) | ZXing para ler o QR da Central (substitui o ML Kit) | Aprovado |
| [0016](0016-hotspot-windows-com-internet.md) | Hotspot do Windows só com internet de origem (encerra P8) | Aprovado |
| [0017](0017-contratos-v1-e-esquema-central-mvp.md) | Contratos v1 de sincronização e esquema 0002 da Central (MVP) | Proposto |
| [0018](0018-esquema-v2-celular-mvp.md) | Esquema v2 do banco do celular e sincronização no `:core:sync` (MVP) | Proposto |
| [0019](0019-frontend-central-tabelas-proprias-echarts.md) | Frontend da Central: tabelas próprias, ECharts e fonte empacotada (substitui o PrimeVue) | Aprovado |

Modelo para novos ADRs: copie [`template.md`](template.md).
