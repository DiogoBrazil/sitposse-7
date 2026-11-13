# Rede local em campo

Status: **cenário B validado com internet na origem** (13/11/2026); **cenário A a validar na Fase 0**. Sem roteador dedicado por ora (A16). A sincronização usa só a rede local, nunca a internet (ADR-0007).

## Cenário A — Hotspot de um celular (funciona sem internet)
1. Um celular (de qualquer equipe ou dedicado) ativa o **Ponto de acesso / Roteador Wi‑Fi**. Funciona sem chip e sem internet.
2. O computador da Central e os celulares das equipes conectam nessa rede.
3. Se o Android perguntar "Esta rede não tem acesso à internet. Manter conexão?", responder **Sim** (o app também força o uso dessa rede durante a sincronização).
4. Na Central: **Sincronização → Parear aparelho**. Os celulares leem o QR.
5. O celular que criou o hotspot também consegue sincronizar.

Pontos de atenção: consumo de bateria do celular-hotspot (manter na tomada ou power bank); alguns fabricantes isolam os clientes, o que precisa ser testado nos modelos da tropa (risco K5).

## Cenário B — Hotspot nativo do Windows (só com internet na origem)
Decisão A21 / [ADR-0016](../architecture/decisions/0016-hotspot-windows-com-internet.md): o hotspot do Windows é usado **somente quando o computador da Central (PC ou notebook) tem internet** e **fica ligado** durante a sincronização. Não há documentação oficial de que ele continue ativo sem internet de origem, e esse modo **não é usado**.

1. Conectar o computador à internet (Wi‑Fi, Ethernet ou outra conexão).
2. Configurações → Rede e Internet → **Hotspot móvel**: escolher a conexão de origem, **desativar a "Economia de energia"** e ligar o hotspot.
3. Os celulares conectam ao hotspot (o próprio Windows mostra um QR de Wi‑Fi para a câmera do celular).
4. Abrir a Central e, se o hotspot foi ligado depois, clicar em **Atualizar** antes de mostrar o QR de pareamento.
5. Manter o computador **ligado e acordado** até o fim da sincronização.

### O que já se sabe
- Validado em aparelho real em 13/11/2026: notebook com origem Wi‑Fi (TLSv1.3, 108 ms) e PC desktop com origem Ethernet (TLSv1.3, 92 ms). Ver `planning/fase-0.md`.
- Com a origem ativa, o celular pode alcançar a Central pelo IP da **origem** (ex.: `192.168.0.24`) em vez do IP do hotspot (`192.168.137.1`): o compartilhamento de internet do Windows encaminha entre as interfaces. É normal.
- **Fechar a tampa suspende o notebook por padrão**: o servidor da Central e o hotspot param. Manter a tampa aberta (ou configurar "Ao fechar a tampa: Não fazer nada").
- Durante a recepção, a Central impede a suspensão **por inatividade** (`SetThreadExecutionState`), mas isso **não** impede a suspensão pela tampa.
- Se a internet cair durante o uso, o Windows pode desligar o hotspot. A sincronização em andamento não é confirmada (sem recibo) e pode ser repetida sem duplicar. Tentar de novo quando a internet voltar, ou passar para o cenário A ou para o pacote USB.

## Firewall
O instalador cria uma regra de entrada para o executável da Central, TCP na porta configurada, `RemoteAddress=LocalSubnet`, perfis **Privado e Público** (redes de hotspot costumam ser classificadas como Públicas). A tela de Sincronização mostra diagnóstico se não houver interface de rede ativa.

## Contingência
Sem rede: usar o **pacote USB** (Android → Exportar pacote → cabo USB ou pen drive OTG → Central → Importar pacote → recibo por QR).
