# Documentação do SITPOSSE-7

A documentação raiz contém **requisitos e contratos compartilhados**. As especificidades de plataforma ficam em `mobile/docs` e `desktop/docs`.

| Área | Documento | Conteúdo |
|---|---|---|
| Requisitos | [requirements/funcionais.md](requirements/funcionais.md) | O que cada produto faz |
| | [requirements/nao-funcionais.md](requirements/nao-funcionais.md) | Offline, durabilidade, desempenho, compatibilidade, UX |
| | [requirements/regras-de-negocio.md](requirements/regras-de-negocio.md) | **Fonte única** das regras de negócio |
| Arquitetura | [architecture/visao-geral.md](architecture/visao-geral.md) | Componentes, stack e fluxos |
| | [architecture/modelo-de-dados.md](architecture/modelo-de-dados.md) | Esquema SQLite conceitual |
| | [architecture/decisions/](architecture/decisions/README.md) | ADRs |
| Sincronização | [synchronization/protocolo.md](synchronization/protocolo.md) | Pareamento, lotes, recibos, pacote USB, restauração |
| | [synchronization/integridade.md](synchronization/integridade.md) | Hashes, staging, idempotência, recuperação |
| | [synchronization/conflitos.md](synchronization/conflitos.md) | Política de conflitos aprovada |
| Segurança | [security/seguranca-e-privacidade.md](security/seguranca-e-privacidade.md) | TLS, pinning, credenciais, privacidade, backup |
| Testes | [testing/estrategia.md](testing/estrategia.md) | Estratégia e critérios de aceite |
| Planejamento | [planning/continuacao.md](planning/continuacao.md) | **Comece por aqui ao retomar:** estado, próximos passos, ambiente, proteções |
| | [planning/plano.md](planning/plano.md) | Fases, entregas e estimativas |
| | [planning/fase-0.md](planning/fase-0.md) | Diário da Fase 0 (versões, spikes, achados, testes em aparelho) |
| | [planning/matriz-decisoes.md](planning/matriz-decisoes.md) | Aprovado / Recomendado / Pendente |
| | [planning/pendencias.md](planning/pendencias.md) | Decisões em aberto |
| | [planning/riscos.md](planning/riscos.md) | Riscos e mitigação |
| Operação | [operacao/rede-em-campo.md](operacao/rede-em-campo.md) | Como montar a rede local (hotspot do celular sem internet ou do Windows com internet) |
| Histórico | [changelog.md](changelog.md) | Mudanças de documentação, esquema e protocolo |
