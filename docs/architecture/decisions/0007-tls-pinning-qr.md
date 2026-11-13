# ADR-0007 — TLS com pinning via QR Code

- **Status:** Aprovado
- **Data:** 2026-11-13

## Contexto
A transferência ocorre em rede local sem internet, por IP, sem CA pública. O prompt proíbe desabilitar a verificação TLS. O usuário optou por segurança moderada (A11/A18).

## Decisão
- A Central gera uma vez um par **ECDSA P‑256** e um certificado autoassinado, com SAN contendo os IPs locais (regerado se os IPs mudarem, **sem trocar a chave**).
- O QR traz `spki_sha256`. O Android usa um `X509TrustManager` próprio que aceita **somente** a cadeia cujo SPKI tem esse hash, e um `HostnameVerifier` que exige o mesmo certificado.
- O token de pareamento é de uso único e expira em 10 min. Depois do pareamento confirmado pelo operador, a Central emite uma **credencial do dispositivo** (256 bits aleatórios) e guarda apenas o hash.
- Se a chave mudar, o Android recusa a conexão e exige novo pareamento.

## Alternativas consideradas
- `CertificatePinner` do OkHttp: trabalha por hostname e aqui o acesso é por IP.
- HTTP puro com token: rejeitado pelo usuário/prompt.
- CA própria instalada no aparelho: operação complexa em campo.

## Consequências
- Perder a chave da Central (reinstalação sem backup) exige parear de novo todos os aparelhos. A chave entra no backup da Central.

## Fontes
- https://developer.android.com/training/articles/security-config
