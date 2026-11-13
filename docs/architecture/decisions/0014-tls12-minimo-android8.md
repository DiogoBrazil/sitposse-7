# ADR-0014 — TLS 1.2 restrito como mínimo (Android 8 e 9)

- **Status:** Aprovado
- **Data:** 2026-11-13
- **Substitui / Substituído por:** ajusta o "TLS 1.3" de `protocolo.md` §1 e `seguranca-e-privacidade.md`; complementa o ADR-0007 (pinning inalterado)

## Contexto
Os documentos aprovados fixavam o transporte em **TLS 1.3**. A plataforma Android só oferece TLS 1.3 a partir do **Android 10 (API 29)**, mas o `minSdk` aprovado é **26 (Android 8)** (A15). Com TLS 1.3 obrigatório, aparelhos com Android 8 e 9 não conseguiriam sincronizar. Achado do spike (a) da Fase 0.

## Decisão
- A Central aceita **TLS 1.3 (preferido) e TLS 1.2**. Nada abaixo de 1.2.
- Em TLS 1.2, **somente** suítes ECDHE (sigilo futuro) com cifra autenticada (AEAD), compatíveis com o certificado ECDSA P‑256:
  `TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256`, `TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384`, `TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256`.
- O Android oferece TLS 1.3 + 1.2 com as mesmas suítes; aparelhos com Android 10 ou superior negociam 1.3.
- O pinning do SPKI (ADR-0007) vale igualmente nas duas versões.

## Alternativas consideradas
- **Empacotar o Conscrypt** no app: mantém TLS 1.3 em todos os aparelhos, mas adiciona dependência nativa e aumenta o APK em cerca de 2–4 MB por ABI.
- **Subir o `minSdk` para 29**: exclui aparelhos com Android 8 e 9 (muda A15).

## Consequências
- Sem dependência nova. Segurança equivalente para o modelo de ameaça desta versão (A18): ECDHE + AEAD e autenticação do servidor pelo pin.
- Testes: `desktop/internal/syncserver/tls_test.go` (aceita 1.2 com AEAD; recusa CBC e versões < 1.2) e `mobile/core/sync` `ClienteCentralGoTest` (cliente forçado a 1.2 contra o servidor Go real).
- Revisar se o `minSdk` subir para 29 ou mais: nesse caso basta voltar a exigir TLS 1.3.

## Fontes
- https://developer.android.com/about/versions/10/behavior-changes-all#tls-1.3
- https://pkg.go.dev/crypto/tls#Config
