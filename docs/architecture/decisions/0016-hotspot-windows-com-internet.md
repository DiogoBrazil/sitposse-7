# ADR-0016 — Hotspot do Windows só com internet de origem

- **Status:** Aprovado
- **Data:** 2026-11-13
- **Substitui / Substituído por:** substitui, na A16, a condição "hotspot nativo do Windows condicionado ao teste" e encerra a pendência P8

## Contexto
A A16 aprovou dois cenários de rede em campo: o hotspot de um celular (cenário A) e o hotspot nativo do Windows (cenário B). O cenário B ficou **condicionado** a um teste de bancada (P8), porque não há documentação oficial de que o hotspot do Windows continue ativo quando a origem perde a internet, nem com a tampa do notebook fechada. O roteiro tinha 7 testes (`rede-em-campo.md`): origem desligada por 30 min, Adaptador de Loopback como origem, tampa fechada, tela apagada etc.

Em 13/11/2026 o hotspot do Windows **com** internet na origem já foi validado duas vezes em aparelho real (moto g56, Android 16): num notebook com origem Wi‑Fi (TLSv1.3, 108 ms) e num PC desktop com origem Ethernet (TLSv1.3, 92 ms).

O usuário decidiu (13/11/2026) que, na operação real, o computador da Central (PC ou notebook com Windows) **terá internet e ficará ligado** quando for usado como hotspot. Não há necessidade operacional de manter o hotspot do Windows sem internet nem com a tampa fechada.

## Decisão
- **Cenário B (hotspot do Windows):** usado **somente com internet na origem** (Wi‑Fi, Ethernet ou outra conexão do computador), com o computador **ligado e acordado** (tampa aberta ou plano de energia que não suspenda) e com a "Economia de energia" do hotspot **desativada**.
- **Sem internet na origem:** usar o **cenário A** (hotspot de um celular, que funciona sem chip e sem internet) ou o **pacote USB** (ADR-0008).
- **P8 resolvida sem teste de bancada.** Os testes de origem desligada, Adaptador de Loopback, tampa fechada e tela apagada saem do escopo da Fase 0. O spike (b) é cancelado.
- **Nada muda no software.** A Central continua sem depender de internet em nenhuma função (A2, RN-90/91): a internet é um requisito **operacional** de um dos modos de rede, não do programa. A sincronização segue só pela rede local, com TLS e pinning (ADR-0007), e recusa clientes de fora das faixas privadas.

## Alternativas consideradas
- **Manter o teste de bancada completo (P8):** custo de tempo na Fase 0 para validar um modo de operação que não será usado.
- **Adaptador de Loopback como origem falsa:** contorno sem suporte oficial, que poderia parar de funcionar em atualizações do Windows.
- **Roteador dedicado:** fora do escopo por ora (A16).

## Consequências
- A Fase 0 fica menor: sai o spike (b), e o critério de aceite passa a ser "P7 fechada; P8 resolvida pela A21".
- **Risco residual:** se a internet da origem cair durante o uso, o Windows pode desligar o hotspot. O efeito é o mesmo de qualquer queda de rede (a sincronização não termina, nada é confirmado sem recibo e o reenvio não duplica, AGENTS.md §5). Orientação: tentar de novo quando a internet voltar, ou usar o cenário A ou o pacote USB.
- No cenário B, o celular pode alcançar a Central pelo IP da **origem** (achado de 13/11), porque o compartilhamento de internet do Windows encaminha entre as interfaces. Isso é normal com a origem ativa.
- O manual de operação (Fase 7) precisa dizer: "hotspot do Windows só com internet; computador ligado; economia de energia desativada".
- A Central ainda impede a suspensão **por inatividade** durante a recepção (`SetThreadExecutionState`), mas isso não impede a suspensão pela tampa. Manter o computador ligado é uma regra operacional.

## Fontes
- Testes em aparelho real de 13/11/2026: `docs/planning/fase-0.md` §Testes em aparelho real.
- Roteiro anterior: `docs/operacao/rede-em-campo.md` (versão até 13/11/2026).
