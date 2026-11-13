# SITPOSSE-7

Sistema de estudo de situação para reintegração de posse do **7º Batalhão de Polícia Militar — PMRO**.

Coleta, em campo e **sem internet**, informações sobre propriedades rurais (famílias, pessoas, veículos, bens, animais, área, coordenadas e fotografias) e as consolida em uma central Windows para planejamento operacional.

> O sistema **não** executa reintegrações, **não** certifica titularidade, área ou limites fundiários e **não** substitui documentação oficial.

## Produtos

| Produto | Plataforma | Pasta | Função |
|---|---|---|---|
| **SITPOSSE-7 Campo** | Android | [`mobile/`](mobile/README.md) | Cadastro e coleta offline, GPS/GNSS, fotos, envio à Central |
| **SITPOSSE-7 Central** | Windows | [`desktop/`](desktop/README.md) | Recepção e consolidação, consulta e edição, estatísticas, relatórios, mapas online |
| Contratos compartilhados | — | [`shared/`](shared/README.md) | Esquemas, contratos de payload e fixtures de teste comuns |

## Estado atual

**Fase 0 (fundação e spikes de risco) em andamento.** Esqueletos da Central e do app Android prontos; primeira conexão real celular → Central validada em 13/11/2026. Faltam os testes de bancada (hotspot do celular, Android 17, instalador); o hotspot do Windows só é usado com internet (ADR-0016). Ver [`docs/planning/continuacao.md`](docs/planning/continuacao.md).

## Por onde começar

1. [`docs/planning/continuacao.md`](docs/planning/continuacao.md): onde parou e como continuar.
1. [`AGENTS.md`](AGENTS.md): regras globais para quem desenvolve, humano ou agente.
2. [`docs/README.md`](docs/README.md): índice da documentação.
3. [`docs/planning/matriz-decisoes.md`](docs/planning/matriz-decisoes.md): o que está aprovado, recomendado e pendente.
