# Riscos

| # | Risco | Prob. | Impacto | Mitigação | Dono/Fase |
|---|---|---|---|---|---|
| K1 | Wails v3 estável há poucas semanas (26/10/2026) | Média | Médio | Núcleo Go independente (ADR-0003); spike; plano B Wails v2.17 | Fase 0 |
| K2 | Internet da origem cai durante o uso do hotspot do Windows e o hotspot desliga | Média | Baixo | A21/ADR-0016: hotspot do Windows só com internet e computador ligado; sem internet, hotspot do celular ou pacote USB; a queda só interrompe a sincronização (nada é confirmado sem recibo, reenvio não duplica); guia operacional | Fase 7 |
| K3 | Android 17 bloqueia a rede local para `targetSdk 37` | Certa | Alto | `ACCESS_LOCAL_NETWORK` com explicação prévia; testes em Android 17 | Fase 0/3 |
| K4 | Android prefere dados móveis em Wi‑Fi sem internet | Alta | Alto | `requestNetwork(TRANSPORT_WIFI)` + `socketFactory` da rede | Fase 0 |
| K5 | Isolamento de clientes no hotspot de alguns celulares | Média | Médio | Testar marcas da tropa; alternativa: o aparelho que cria o hotspot sincroniza primeiro, ou pacote USB | Fase 0/7 |
| K6 | Windows 10 fora do suporte geral desde 10/2025 | Média | Baixo/Médio | WebView2 embutido; recomendar Windows 11 | Fase 0/7 |
| K7 | Perda do celular antes de sincronizar | Média | Alto | Tela inicial mostra "há X horas sem enviar"; rotina diária de sincronização ou pacote USB | Fase 1/3 |
| K8 | Termos do provedor vetam a exportação da carta | Média | Médio | P1 com análise; carta condicionada a `permiteExportacao` | Fase 6 |
| K9 | Relógios divergentes | Alta | Baixo | `rev` em vez de horário; `recebido_em` separado | Arquitetura |
| K10 | Firewall do Windows bloqueia a recepção | Alta | Alto | Regra criada no instalador; diagnóstico na tela | Fase 0/2 |
| K11 | Pacotes USB sem cifra expostos | Média | Médio | Orientação operacional; cifra como evolução | Operação |
| K12 | Celulares com pouca RAM ou armazenamento | Média | Médio | Fotos reduzidas; verificação de espaço; decodificação por amostragem | Fase 1 |
