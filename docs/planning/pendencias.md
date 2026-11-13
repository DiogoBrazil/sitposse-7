# Pendências (decisões em aberto)

Nenhuma destas decisões pode ser tomada em silêncio. Cada uma tem recomendação provisória e fase-limite.

| # | Pendência | Recomendação provisória | Decidir até | Status |
|---|---|---|---|---|
| P1 | Provedor cartográfico (custos, termos, exportação de cartas) | Provedor plugável. Candidatos: Esri World Imagery, Brasil MAIS (verificar adesão da PMRO e serviço WMS/XYZ), Google, Mapbox. Ver ADR-0013 | Início da Fase 6 | Aberta |
| P2 | Modelo institucional de relatórios e cartas-imagem | Layout neutro com título e ressalva obrigatória | Fase 5 | Aberta |
| P3 | Retenção e exclusão definitiva | Nunca apagar cadastros; liberar só arquivos de foto confirmados e cobertos por backup verificado, por ação explícita | Fase 5 | Aberta |
| P4 | Critério para liberar espaço no celular | Igual a P3; a Central informa a cobertura de backup em `/v1/status` | Fase 5 | Aberta |
| P5 | Duplicidade sem UUID comum (mesma família por duas equipes) | Sugerir candidatas (nome normalizado + distância GPS); permitir **vincular** sem fundir nem apagar | Fase 4 | Aberta |
| P6 | Reuso de aparelho em nova operação / equipe já vinculada a outra operação | "Encerrar operação" só com tudo confirmado; nova equipe | Fase 3 | Aberta |
| P7 | Distribuição do APK e `targetSdk` | Sideload; `targetSdk 37` + `ACCESS_LOCAL_NETWORK` | Fase 0 | Aberta |
| P8 | Hotspot do Windows sem internet e com a tampa fechada | Teste de bancada ([rede-em-campo.md](../operacao/rede-em-campo.md)) | Fase 0 | Aberta |
| P9 | Versões exatas (Compose BOM, Kotlin, AGP, Go, ML Kit, PrimeVue, ECharts, Leaflet, lib PDF) | Fixar as estáveis mais recentes no spike | Fase 0 | Aberta |

## Resolvidas
| # | Pendência | Decisão | Data |
|---|---|---|---|
| — | Política de conflitos | Sentido único + conflito manual (A9) | 2026-11-13 |
| — | Operações múltiplas | Entidade operação (A10) | 2026-11-13 |
| — | Catálogo de eletrodomésticos | Manter ar-condicionado e ventilador (A5) | 2026-11-13 |
| — | Faixas etárias | ECA 0–11 / 12–17 / 18+ (A6) | 2026-11-13 |
| — | Identidade após reinstalação | Restauração pela Central (A12) | 2026-11-13 |
| — | Acesso ao desktop | Sem login na v1 (A17) | 2026-11-13 |
| — | Segurança de transporte e pacotes | TLS + pinning; pacote sem cifra (A11) | 2026-11-13 |
| — | Volume e fotos | Pequeno; ~2560 px (A13/A14) | 2026-11-13 |
| — | Versões mínimas | Android 8+ / Windows 10 22H2+ (A15) | 2026-11-13 |
