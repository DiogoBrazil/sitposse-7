# Rede local em campo (sem internet)

Status: **rascunho, a validar na Fase 0** (pendência P8). Sem roteador dedicado por ora (A16).

## Cenário A — Hotspot de um celular (caminho principal até P8 ser validada)
1. Um celular (de qualquer equipe ou dedicado) ativa o **Ponto de acesso / Roteador Wi‑Fi**. Funciona sem chip e sem internet.
2. O notebook da Central e os celulares das equipes conectam nessa rede.
3. Se o Android perguntar "Esta rede não tem acesso à internet. Manter conexão?", responder **Sim** (o app também força o uso dessa rede durante a sincronização).
4. Na Central: **Sincronização → Parear aparelho**. Os celulares leem o QR.
5. O celular que criou o hotspot também consegue sincronizar.

Pontos de atenção: consumo de bateria do celular-hotspot (manter na tomada ou power bank); alguns fabricantes isolam os clientes, o que precisa ser testado nos modelos da tropa (risco K5).

## Cenário B — Hotspot nativo do Windows (condicionado ao teste)
**Não há documentação oficial** de que o hotspot do Windows continue funcionando sem internet de origem. Relatos indicam que ele pode desligar quando a origem cai, e há uma opção de **economia de energia** que o desliga depois de alguns minutos sem clientes.

### Teste de bancada (Fase 0, notebook real)
| # | Procedimento | Resultado |
|---|---|---|
| 1 | Ligar o hotspot com internet; desconectar a origem; verificar por 30 min se continua ativo e se os celulares alcançam a Central | _a preencher_ |
| 2 | Repetir com "Economia de energia: desligar quando nenhum dispositivo estiver conectado" **desativada** | _a preencher_ |
| 3 | Instalar o **Adaptador de Loopback da Microsoft** e usá-lo como origem do hotspot, sem nenhuma internet | _a preencher_ |
| 4 | Plano de energia "Ao fechar a tampa: **Não fazer nada**" (bateria e tomada); fechar a tampa e sincronizar | _a preencher_ |
| 5 | Apenas desligar a tela (sem suspender) e sincronizar | _a preencher_ |
| 6 | Gerenciador de Dispositivos → adaptador Wi‑Fi → Gerenciamento de Energia → desmarcar "Permitir que o computador desligue este dispositivo"; repetir o teste 1 | _a preencher_ |
| 7 | Verificar o perfil da rede (Público/Privado) e se a regra de firewall da Central atende | _a preencher_ |

### O que já se sabe
- **Fechar a tampa suspende o notebook por padrão**: o servidor da Central e o hotspot param. Isso **só** é evitado configurando "Ao fechar a tampa: Não fazer nada".
- **Desligar só a tela** não interrompe a rede.
- Durante a recepção, a Central impede a suspensão **por inatividade** (`SetThreadExecutionState`), mas isso **não** impede a suspensão pela tampa.

### Decisão após o teste
- Se 1–2 ou 3 passarem: documentar o passo a passo e manter os dois cenários disponíveis.
- Se falharem: o cenário A passa a ser o caminho oficial; o cenário B fica documentado como "somente com internet de origem".

## Firewall
O instalador cria uma regra de entrada para o executável da Central, TCP na porta configurada, `RemoteAddress=LocalSubnet`, perfis **Privado e Público** (redes de hotspot costumam ser classificadas como Públicas). A tela de Sincronização mostra diagnóstico se não houver interface de rede ativa.

## Contingência
Sem rede: usar o **pacote USB** (Android → Exportar pacote → cabo USB ou pen drive OTG → Central → Importar pacote → recibo por QR).
