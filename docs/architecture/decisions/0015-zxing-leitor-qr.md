# ADR-0015 — ZXing para ler o QR da Central (substitui o ML Kit)

- **Status:** Aprovado
- **Data:** 2026-11-13
- **Substitui / Substituído por:** substitui a escolha "ML Kit Barcode, variante empacotada" (`visao-geral.md`, `mobile/AGENTS.md`, P9)

## Contexto
O ML Kit `barcode-scanning:17.3.0` (variante empacotada) traz como dependência `com.google.android.datatransport:transport-backend-cct`, além de `play-services-base`/`basement` e componentes do Firebase. O `transport-backend-cct` é o canal de **métricas de uso** do Google, que envia eventos sempre que há internet. O AGENTS.md proíbe telemetria (§6) e o envio de dados a serviços externos (§8). Achado na Fase 0 ao montar o app de diagnóstico.

## Decisão
- Ler o QR com **ZXing core** (`com.google.zxing:core`, Apache‑2.0, Java puro): sem rede, sem Google Play Services, sem Firebase.
- Os quadros vêm do CameraX `ImageAnalysis` (plano Y do YUV_420_888) e são decodificados localmente.
- A decodificação força **UTF‑8**, porque a Central gera o QR em modo byte UTF‑8 sem ECI (biblioteca `qrcode`), e o nome da operação tem acentos.

## Alternativas consideradas
- **ML Kit removendo o datatransport pelo manifesto** (`tools:node="remove"`): não é um caminho suportado, precisaria de verificação por captura de rede a cada atualização, e o ML Kit continuaria trazendo dependências do Play Services.

## Consequências
- O APK diminui (sem modelos do ML Kit nem GMS). O caso de uso é favorável: QR grande, exibido em tela, a curta distância.
- O ZXing core recebe só manutenção; a API é estável e pequena.
- Verificação: `./gradlew :app:dependencies` não lista `datatransport`, `firebase`, `play-services` nem `mlkit`. Teste: `DecodificadorQrTest` (UTF‑8 sem ECI, `rowStride` maior que a largura).

## Fontes
- https://github.com/zxing/zxing
- https://developers.google.com/ml-kit/terms
