# Plano de implementação

Aprovado em **13/11/2026**. As estimativas são para **um desenvolvedor em tempo integral**; a faixa é ampla por causa da incerteza com dispositivos reais.

| Fase | Entregas | Critério de aceite | Estim. | Status |
|---|---|---|---|---|
| **Docs** | Plano gravado em `docs/`, ADRs, AGENTS.md, estrutura do monorepo, fixtures de área e normalização | Revisão do usuário | — | **Concluída** |
| **0. Fundação e spikes de risco** | Esqueleto `mobile/` (Gradle, Compose, Room 3) e `desktop/` (Wails 3 + Vue); `shared/schemas` iniciais. **Spikes:** (a) Android → Central por HTTPS com pin pelo QR, no hotspot do celular sem internet e no hotspot do Windows com internet na origem, incluindo Android 17; (b) ~~teste de hotspot e tampa do Windows~~ cancelado pela A21/ADR-0016 (13/11/2026); (c) PRAGMAs efetivos do Room 3 com BundledSQLiteDriver; (d) instalador NSIS + WebView2 + firewall em Windows 10 22H2 e 11 limpos | Spikes documentados com resultado; P7 fechada ou adiada com aprovação do usuário (A22: testes de hotspot do celular e Android 17/8/9 adiados em 13/11/2026); P8 encerrada pela A21; versões fixadas | 1,5–2,5 sem | **Concluída** (14/11/2026, A23) |
| **MVP (A24)** | Fatia vertical das Fases 1 e 2 + transferência segura da Fase 3: cadastro completo offline no celular; operação, pareamento, recepção idempotente com recibo e fotos verificadas, e consulta somente leitura na Central; telas responsivas | (1) testes JVM/Go: domínio, migrações, fotos (arquivo antes do registro), e cliente Kotlin × servidor Go real nos cenários 1, 4, 5, 6, 7 e 8 de `testing/estrategia.md` §4; (2) teste real: cadastros offline em área rural, envio pelo hotspot do Windows, contagens, UUIDs e hashes conferidos na Central | — | **Em andamento** |
| **Estudos e coletas (A28)** | Estudo → Jornada → Equipe na Central; código da equipe e coletas no celular; confirmação do destino antes de importar; formulário em 7 etapas com revisão; responsável obrigatório; Observações relevantes; reinício dos bancos de teste. Etapas E0–E6 do plano de 15/11 (ADR-0023/0024/0025) | Testes da A28 (Go, Kotlin e cruzados) + bancada no notebook e no moto g56 | — | **Em andamento** (aprovado em 15/11/2026) |
| **1. Campo: persistência e coleta** | Equipe, propriedades, campos fixos, itens dinâmicos, área, GNSS, fotos, rascunhos, lista e busca, estados | Testes Android de `testing/estrategia.md` §2 passando | 3–4 sem | — |
| **2. Central: base e recepção mínima** | Banco, migrações, operações e equipes, pareamento, recepção de lote com fotos, CRUD e auditoria | Primeira carga de 2 celulares com contagens, UUIDs e hashes conferidos | 2–3 sem | — |
| **3. Sincronização robusta** | Incremental, tombstones, retomada de fotos, recibos, ACK perdido, conflitos com tela de resolução, pacote USB com recibo, restauração de equipe | Cenários 1–14 de `testing/estrategia.md` §4 passando | 3–4 sem | — |
| **4. Estatísticas e filtros** | `FiltroEstudo`, painel, tabelas, gráficos, drill-down, aliases, sugestão de duplicatas (P5) | Fixture de estatísticas com todos os cruzamentos do prompt §18 (lista derivada, A25) | 2–3 sem | **Antecipada (A25, 14/11/2026)**: em andamento, junto com a melhoria visual da Central |
| **5. Backups e relatórios** | Backup e restauração nos dois lados; liberar espaço (P3/P4); relatórios PDF/CSV | Restauração real testada | 1,5–2 sem | Relatórios **antecipados** (A26, 14/11/2026): PDF, DOCX e XLSX feitos; backups pendentes |
| **6. Cartografia online** | Provedor (P1), mapas individual e geral, cartas-imagem, indisponibilidade amigável | Testes de `testing/estrategia.md` §6 | 1,5–2 sem | **Antecipada (A27, 14/11/2026)**: cartas-imagem Sentinel-2/CDSE (ADR-0022) aprovadas no geral pelo usuário em 15/11; ajustes de detalhe adiados para etapa futura |
| **7. Campo, empacotamento e docs** | Teste com equipes reais, ajustes de UX, APK assinado, instalador, manuais | Checklist de aceite (§7 da estratégia) | 1,5–2 sem | — |

**Total estimado: 16–23 semanas.** O MVP (A24) antecipa partes das Fases 1, 2 e 3; o que ficou de fora delas continua nas respectivas fases. A Fase 4 pode correr em paralelo com a 3, e a Fase 6 depende de P1.

**Por que a ordem difere da sugerida no prompt:** a infraestrutura de rede (hotspot Windows/Android, Android 17) é o maior desconhecido técnico, então foi antecipada para spikes da Fase 0, antes de investir na UI.

## Antecipação A29 — Central (15/11/2026)
Autorizadas as correções/exclusões e a melhoria das listagens, conforme ADR-0026. Fila de conflitos continua adiada por escolha explícita do usuário (recusa temporária de novas versões); sem ampliação para USB ou backups. Sem mudança de protocolo ou esquema. Instalador 0.7.0-mvp; validação Windows pendente.

## Antecipação A30 — Cartas-imagem inteligentes (15/11/2026)
Plano aprovado pelo usuário (ADR-0027): imagens do INPE (CBERS-4A 2 m, Sentinel-2 L2A, mosaico trimestral), seleção automática, mosaicos sinalizados, carta geral + setores, número fixo da propriedade no estudo e pinos interativos. Etapas:
- **E0:** decisões e documentação;
- **E1:** numeração (migração 0003);
- **E2:** geo/UTM, setores, leitor COG, STAC, INPE;
- **E3:** seleção e raster;
- **E4:** cartas multifolha e serviço;
- **E5:** telas;
- **E6:** build 0.8.0-mvp e teste regional.

O app Campo não muda.

## Detalhes por produto
- Android: [mobile/docs](../../mobile/docs/arquitetura.md)
- Central: [desktop/docs](../../desktop/docs/arquitetura.md)
