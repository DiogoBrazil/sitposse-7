# ADR-0008 — Pacote USB de contingência com recibo

- **Status:** Aprovado
- **Data:** 2026-11-13

## Decisão
- Pacote `.s7pkg` = ZIP com `manifest.json` (versões, lote, equipe, instalação, entidades e fotos com SHA‑256), `registros.json` e `fotos/`. **Sem criptografia nesta versão** (A11); a integridade é garantida por hashes.
- A Central importa pelo **mesmo núcleo** do Wi‑Fi (idempotente, com conflitos) e gera um **recibo** `.s7rcb` (JSON + hash), também exibido como **QR**.
- O Android só confirma ao ler o QR ou importar o `.s7rcb`. Até lá, o lote fica em `EXPORTADO_AGUARDANDO_CONFIRMACAO`.
- O mesmo formato, em modo "completo", serve de **backup** do celular.

## Consequências
- Se o recibo for grande para um QR, o QR traz só `lote_uuid` + hash do recibo + lista compacta, ou se usa o arquivo. Validar na Fase 3.
- Pacotes contêm dados pessoais **sem cifra**: a orientação operacional é guardar pen drives com cuidado. Cifrar fica para evolução (novo ADR).
