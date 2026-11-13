# ADR-0011 — Fotos reduzidas a ~2560 px

- **Status:** Aprovado
- **Data:** 2026-11-13

## Decisão
- Após a captura (CameraX), redimensionar para lado maior ≤ 2560 px, JPEG qualidade 85, aplicando a orientação EXIF e preservando `DateTimeOriginal`.
- O arquivo resultante é o "original do sistema". O SHA‑256 é calculado sobre ele e o arquivo bruto da câmera é descartado.
- Decodificação com `inSampleSize`, sem carregar a imagem inteira em memória.

## Consequências
- Cerca de 0,8–1,5 MB por foto e sincronização bem mais rápida.
- Perde-se a resolução máxima da câmera (o usuário aceitou).
