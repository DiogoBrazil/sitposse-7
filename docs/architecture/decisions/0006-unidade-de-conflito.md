# ADR-0006 — Unidade de conflito = cadastro

- **Status:** Aprovado
- **Data:** 2026-11-13

## Decisão
O conflito é detectado por **cadastro**: a propriedade, seus itens e os metadados das fotos. Se a Central corrigiu qualquer parte do cadastro depois da última `rev_origem` recebida e o Android envia nova versão de qualquer parte, abre-se **um** conflito para o cadastro, com base, versão Android e versão Central. A resolução é manual e campo a campo (ver `docs/synchronization/conflitos.md`).

## Alternativas consideradas
- **Conflito por linha (entidade):** mais granular, mas fragmenta a análise de uma mesma família em vários conflitos.
- **Mescla automática de campos disjuntos:** o usuário escolheu a opção sem mescla automática.

## Consequências
- Fotos novas enviadas pelo Android são **sempre preservadas**: o arquivo e o registro entram no conflito e nada é descartado.
