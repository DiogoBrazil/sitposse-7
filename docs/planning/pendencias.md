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
| P7 | Distribuição do APK e `targetSdk` | Sideload; `targetSdk 37` + `ACCESS_LOCAL_NETWORK` | Depois da Fase 0 (A22); recomendado até a Fase 3 (risco K3) | Parcial: app configurado assim (`fase-0.md`); falta o teste em aparelho com Android 17, **adiado pelo usuário em 13/11/2026** sem bloquear a Fase 0 (A22; ver `continuacao.md` §2) |
| P9 | Versões exatas (Compose BOM, Kotlin, AGP, Go, leitor de QR, PrimeVue, ECharts, Leaflet, lib PDF) | Fixar as estáveis mais recentes no spike | Fase 0 | Parcial: ver [fase-0.md](fase-0.md); PrimeVue/ECharts/Leaflet/PDF nas fases de uso |
| P11 | Mensagens de erro geradas no domínio e na sincronização (`ValidacaoCadastro`, `RepositorioCampo`, `ApiCentral`, `Sincronizador`) estão em português **no código** dos módulos `:core`, e não em `strings.xml` como pede o `mobile/AGENTS.md` (os módulos `:core` são Kotlin puro, sem recursos Android, e os testes JVM conferem essas mensagens) | Manter no código dos `:core` (pt-BR, uma língua só); `strings.xml` para todo texto das telas. Alternativa: códigos de erro nos `:core` e textos no `:app` (mais trabalho, mesma experiência) | Antes da Fase 7 | Aberta (registrada em 14/11 no MVP) |
| P12 | Celular pareado com **outra** Central (outro `central_id`) depois de já ter enviado dados. Hoje o novo pareamento substitui o anterior, e o que já tem recibo **não** é reenviado. A nova Central fica só com os cadastros novos (visto no teste de 14/11: 2 no notebook, 1 no PC) | Em campo há uma Central por operação. Ao ler o QR de uma Central diferente da pareada, o app **avisa** e pede confirmação, sem reenviar nada automaticamente. Juntar dados de duas Centrais fica para a exportação/importação entre Centrais (Fase 3 ou 4). Alternativa: ao trocar de Central, marcar tudo como pendente para essa Central (a Central é idempotente por `uuid`/`rev`), o que exige confirmação por Central no esquema do celular (ADR + migração) | Antes de usar mais de um computador como Central | Aberta. **14/11, usuário:** por enquanto, **um só computador** como Central; decidir depois. Alternativa sugerida pelo usuário: manter o comportamento atual e acrescentar **exportar os dados de uma Central e importar em outra** |
| P10 | `applicationId` / pacote definitivo do app | Provisório `br.gov.ro.pm.sitposse7.campo`. Trocar depois da primeira instalação em campo faz o Android tratá-lo como **outro app** (dados não migram), então decidir antes disso | Antes do 1º uso em campo (Fase 7) | Aberta |

## Resolvidas
| # | Pendência | Decisão | Data |
|---|---|---|---|
| P8 | Hotspot do Windows sem internet e com a tampa fechada | Não será usado assim: hotspot do Windows só com internet e computador ligado; sem teste de bancada (A21, ADR-0016) | 2026-11-13 |
| — | TLS 1.3 indisponível no Android 8/9 (conflito entre A15 e o transporte aprovado) | TLS 1.2 restrito como mínimo (A19, ADR-0014) | 2026-11-13 |
| — | ML Kit envia métricas de uso ao Google | Trocar por ZXing (A20, ADR-0015) | 2026-11-13 |
| — | Política de conflitos | Sentido único + conflito manual (A9) | 2026-11-13 |
| — | Operações múltiplas | Entidade operação (A10) | 2026-11-13 |
| — | Catálogo de eletrodomésticos | Manter ar-condicionado e ventilador (A5) | 2026-11-13 |
| — | Faixas etárias | ECA 0–11 / 12–17 / 18+ (A6) | 2026-11-13 |
| — | Identidade após reinstalação | Restauração pela Central (A12) | 2026-11-13 |
| — | Acesso ao desktop | Sem login na v1 (A17) | 2026-11-13 |
| — | Segurança de transporte e pacotes | TLS + pinning; pacote sem cifra (A11) | 2026-11-13 |
| — | Volume e fotos | Pequeno; ~2560 px (A13/A14) | 2026-11-13 |
| — | Versões mínimas | Android 8+ / Windows 10 22H2+ (A15) | 2026-11-13 |
