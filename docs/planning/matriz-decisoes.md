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
| A16 | Rede: hotspot de celular e hotspot nativo do Windows; sem roteador por ora. **A condição "hotspot do Windows condicionado ao teste" foi substituída pela A21 em 13/11/2026** | usuário | rede-em-campo.md, ADR-0016 |
| A17 | Central sem login na v1 | usuário | segurança |
| A18 | Segurança moderada; foco em integridade | prompt §20 | segurança |
| A19 | Transporte: TLS 1.3 preferido; TLS 1.2 só com ECDHE + AEAD, para Android 8/9 (13/11/2026) | usuário (Fase 0) | ADR-0014 |
| A20 | Leitor de QR: ZXing no lugar do ML Kit, que traz telemetria do Google (13/11/2026) | usuário (Fase 0) | ADR-0015 |
| A21 | Hotspot do Windows só com internet na origem e com o computador (PC ou notebook) ligado; sem internet, hotspot do celular ou pacote USB. Sem teste de hotspot sem internet nem de tampa fechada; P8 encerrada (13/11/2026) | usuário (Fase 0) | ADR-0016 |
| A22 | A Fase 0 pode ser concluída com o teste no hotspot do celular (cenário A) e os testes em Android 17 (P7) e 8/9 **adiados**; P7 continua aberta até o teste (13/11/2026). Decisão de cronograma, sem mudança de arquitetura | usuário (Fase 0) | plano.md, continuacao.md §2 |
| A23 | **Fase 0 concluída** (14/11/2026), com os itens adiados pela A22 e o Windows 10 22H2 sem teste; teste do instalador do zero dispensado após o reteste por cima | usuário | fase-0.md §Reteste |
| A24 | **MVP para teste em campo** (14/11/2026): as Fases 1 e 2 são feitas juntas, como uma fatia vertical, incluindo da Fase 3 só o que torna a transferência segura (lote idempotente, recibo persistido, `rev_confirmada`, fotos com SHA‑256 conferido e retomada por offset, reconciliadores). Celular: **cadastro completo** (equipe, propriedade, quantitativos, itens dinâmicos, área, GNSS, fotos, rascunho, lista e busca, estados). Central: operação, pareamento pelo QR com confirmação do operador, recepção e **consulta somente leitura** (sem correção pela Central, logo sem conflitos). Adiados: correção na Central e conflitos, pacote USB, restauração de equipe, estatísticas, backups, mapas e relatórios. Transferência pelo hotspot do Windows com internet (A21). Telas responsivas desde já. Dados do teste são **descartáveis**: `applicationId` provisório e chave de debug (P10 segue aberta) | usuário | plano.md |
| A25 | **Fase 4 antecipada inteira** (14/11/2026), antes do teste em campo, junto com a melhoria visual da Central: `FiltroEstudo`, painel com gráficos e tabelas, tabela analítica, drill-down, aliases e sugestão de duplicatas. **Bibliotecas:** tabelas próprias e Apache ECharts (gráficos), empacotadas localmente. O PrimeVue foi escolhido e depois trocado no mesmo dia, porque a v5 deixou de ser MIT (ADR-0019). **Filtros:** equipe, período, GPS/fotos, área e rebanho, mais os do `FiltroEstudo`; sempre com o total geral, por família (propriedade) e por equipe. **Cruzamentos:** especificação do usuário §22-A em `docs/requirements/estatisticas.md` (recebida em 14/11; substitui a lista derivada provisória). **P5 resolvida:** aliases editáveis só na Central (RN-35) e duplicatas sugeridas por nome normalizado + distância do GPS, com "vincular" sem fundir nem apagar; migração 0003 com ADR. **Ordem:** (1) base visual, (2) `FiltroEstudo` no Go + fixture, (3) Painel/Estatísticas, (4) aliases e duplicatas. O teste em campo segue com um só computador como Central (P12) | usuário | plano.md, telas-e-fluxos.md |
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
