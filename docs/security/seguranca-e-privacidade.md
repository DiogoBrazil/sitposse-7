# Segurança e privacidade

Postura aprovada (A18): **prioridade em integridade e não perda de dados**. O restante fica moderado nesta versão e evolui depois. Itens marcados como *evolução* exigem novo ADR.

## Medidas desta versão
| Tema | Medida | Ref. |
|---|---|---|
| Transporte | TLS 1.3 preferido; TLS 1.2 só com ECDHE + AEAD (Android 8/9); pinning do SPKI via QR; **nunca** desabilitar a verificação | ADR-0007, ADR-0014 |
| Pareamento | Token de uso único (10 min) + confirmação do operador; credencial de 256 bits por dispositivo (Central guarda o hash); revogação na Central | protocolo §2 |
| Acesso à Central | Sem login na v1; depende da conta Windows. O operador informa o nome nas Configurações para a auditoria | A17 |
| Auditoria | Tabela somente de acréscimo, com antes e depois de toda alteração na Central | modelo de dados |
| Android: armazenamento | Banco e fotos no armazenamento **privado** do app; `allowBackup=false`; `dataExtractionRules` excluindo tudo (nada vai à nuvem) | R10 |
| Pacote USB | ZIP **sem cifra**, com hashes para integridade. Orientação operacional de guarda | ADR-0008 |
| Identificadores | Sem ID de hardware; UUIDs aleatórios | RN-03 |
| Rede | Sem telemetria (inclusive de bibliotecas: por isso o leitor de QR é o ZXing, ADR-0015); tráfego externo só para tiles do provedor cartográfico aprovado, que não levam dados de cadastro | AGENTS §6 |
| Firewall | Regra de entrada restrita a `LocalSubnet`, só para o executável da Central | `desktop/docs/arquitetura.md` |

## Evolução (fora do escopo atual)
- Cifrar o banco Android (SQLCipher ou chave no Keystore) e as fotos.
- Cifrar os pacotes USB (AES‑GCM/age) com gestão de senha.
- Login e perfis na Central.
- Cifrar os backups da Central.

## Riscos aceitos
- Perda ou roubo de um celular expõe os dados ali armazenados (sem cifra em repouso).
- Pen drive com pacote exposto.
