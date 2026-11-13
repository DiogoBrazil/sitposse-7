# Matriz de decisões

> **Regra:** as três categorias nunca se misturam. Para mudar algo **Aprovado**, crie um ADR e registre aqui com data.

## APROVADO
Fonte: prompt-mestre e respostas do usuário em 13/11/2026; recomendações R1–R10 aprovadas junto com o plano na mesma data.

| # | Decisão | Origem | Ref. |
|---|---|---|---|
| A1 | Monorepo com Campo (Android) e Central (Windows) | prompt | ADR-0001 |
| A2 | Android 100% offline; Central offline exceto mapa e carta-imagem; sem tiles offline | prompt | RN-90/91 |
| A3 | Equipe: nome* e responsável*; área opcional; UUID estável; sem login | prompt | RN-01..04 |
| A4 | Propriedade: só o nome é obrigatório; o UUID é a identidade | prompt | RN-10..13 |
| A5 | Campos fixos e itens dinâmicos; catálogo de eletrodomésticos com ar-condicionado e ventilador | prompt + usuário | RN-20..35 |
| A6 | Faixas ECA: crianças 0–11, jovens 12–17, adultos 18+ | usuário | RN-20 |
| A7 | 1 alqueire paulista = 2,42 ha; valor e unidade preservados | prompt | RN-40..45 |
| A8 | GPS WGS84, meta ≤ 15 m, manual permitido | prompt | RN-50..55 |
| A9 | Sincronização em sentido único Android → Central; correção na Central + edição no Android = conflito manual | usuário | RN-70..73, ADR-0005 |
| A10 | Entidade operação | usuário | ADR-0012 |
| A11 | TLS + pinning via QR; pacote USB com hashes, sem cifra | usuário | ADR-0007/0008 |
| A12 | Restauração da identidade da equipe pela Central | usuário | RN-06 |
| A13 | Volume pequeno (~5 equipes, ~300 propriedades, ~1.500 fotos); testes 3× | usuário | RNF-07 |
| A14 | Fotos reduzidas (~2560 px, JPEG 85) | usuário | ADR-0011 |
| A15 | Android 8+ (minSdk 26); Windows 10 22H2+ 64 bits | usuário | RNF-10 |
| A16 | Rede: hotspot de celular e hotspot nativo do Windows (este condicionado ao teste); sem roteador por ora | usuário | rede-em-campo.md |
| A17 | Central sem login na v1 | usuário | segurança |
| A18 | Segurança moderada; foco em integridade | prompt §20 | segurança |
| R1 | Stack: Room 3 + BundledSQLiteDriver; Compose/M3; Wails v3.1 + Vue 3 + TS; modernc SQLite; JSON Schema | plano | ADR-0002..0004 |
| R2 | `rev` por entidade + `rev_confirmada` | plano | ADR-0005 |
| R3 | Unidade de conflito = cadastro | plano | ADR-0006 |
| R4 | WAL + `synchronous=FULL`; arquivo antes do registro | plano | integridade.md |
| R5 | Rascunhos em tabela própria, nunca sincronizados | plano | mobile/persistencia |
| R6 | Equipe ↔ operação 1:1, vínculo no 1º pareamento | plano | ADR-0012 |
| R7 | Normalização de nomes "Outros" + aliases na Central | plano | RN-35 |
| R8 | Carta-imagem montada em Go, se o provedor permitir | plano | ADR-0013 |
| R9 | Backup Android por pacote via SAF; Central com `VACUUM INTO` + fotos, verificado | plano | — |
| R10 | `allowBackup=false` | plano | segurança |

## RECOMENDADO (aguardando aprovação)
_Nenhum no momento._ Novas recomendações entram aqui até serem aprovadas.

## PENDENTE
Ver [pendencias.md](pendencias.md).
