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

**Planejamento aprovado em 13/11/2026; implementação não iniciada.** Próximo passo: Fase 0 (fundação e spikes de risco). Ver [`docs/planning/plano.md`](docs/planning/plano.md).

## Por onde começar

1. [`AGENTS.md`](AGENTS.md): regras globais para quem desenvolve, humano ou agente.
2. [`docs/README.md`](docs/README.md): índice da documentação.
3. [`docs/planning/matriz-decisoes.md`](docs/planning/matriz-decisoes.md): o que está aprovado, recomendado e pendente.
