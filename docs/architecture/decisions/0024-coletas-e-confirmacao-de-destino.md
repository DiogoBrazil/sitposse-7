# ADR-0024 — Coletas no celular, protocolo v2 e confirmação do destino na Central

- **Status:** Proposto (plano aprovado pelo usuário em 15/11/2026, A28)
- **Data:** 2026-11-15
- **Substitui / Substituído por:** altera o ADR-0017 (§1, §2, §3 e a importação automática) e o ADR-0018 (§5, um lote aberto por vez)

## Contexto
Com o ADR-0023, o celular passa a identificar o que coleta só pelo **código da equipe**, digitado offline. Um erro de digitação pode gerar um código inexistente ou, por coincidência, o código de outra equipe. Hoje a Central importa o lote assim que as fotos chegam, sem conferência humana, e o lote é ligado à equipe do aparelho pareado.

## Decisão
1. **Coleta (celular):** conjunto de cadastros feitos com um código de equipe. Tem UUID (gerado no celular), código atual, código original, estado `em_andamento` ou `concluida` e, depois da primeira importação, o **destino confirmado** devolvido pela Central. Cada cadastro pertence a **uma** coleta para sempre (`propriedades.coleta_uuid`); o rascunho guarda a coleta em que começou.
   - Ao abrir o app: nenhuma coleta em andamento → pedir o código; uma → abrir direto; várias → listar para escolher.
   - Conclusão **manual**, com avisos: ao iniciar outro código ("Concluir a coleta anterior?") e ao abrir num dia diferente do início ("Cada jornada costuma ter um código novo").
   - Coleta concluída não aceita cadastros novos; os existentes podem ser editados ou excluídos e continuam sendo sincronizados; pode ser reaberta com confirmação.
   - Digitar o código de uma coleta que já existe reabre ou continua essa coleta, sem duplicá-la.
   - **Corrigir o código** só é possível sem destino confirmado e sem lote aberto (enviando, aguardando ou com erro recuperável ainda não reconciliado). O histórico fica em `coleta_codigos`. Nenhum dado dos cadastros muda.
2. **Pareamento só do aparelho:** o pedido deixa de levar equipe; QR e `status` deixam de levar operação. TLS, pin, token de uso único e confirmação do operador continuam como no ADR-0017.
3. **Um lote por coleta** (`protocolo_versao = 2`, `schema_versao = 2`). O lote leva o objeto `coleta` (UUID, código, timestamps, `rev`) no lugar de `equipe`. Várias coletas pendentes geram vários lotes, enviados em sequência e decididos **um a um**.
4. **Confirmação do destino:** `POST …/importar` só confere as fotos e põe o lote em **`aguardando_confirmacao`**; nada é aplicado. A Central mostra ao operador o destino resolvido pelo código (estudo, data da jornada, equipe, código) ou "Código de equipe não encontrado", com a quantidade de cadastros e fotos. Ações:
   - **Confirmar e importar**: aplica como hoje (SAVEPOINT por registro, recibo) e grava o destino da coleta.
   - **Alterar destino / Corrigir no Central**: escolhe Estudo → Jornada → Equipe, vê o código oficial e confirma. Preserva o código recebido.
   - **Cancelar sincronização**: lote `cancelado`, nada importado, nada apagado.
   - Cada decisão fica em `decisoes_destino` (somente acréscimo) e na auditoria.
5. **Destino travado:** depois da primeira importação, a coleta fica ligada à equipe para sempre (gatilho no banco). Os lotes seguintes da mesma coleta ainda pedem confirmação, mas sem "Alterar destino".
6. **Estado no celular:** o celular consulta `GET /v1/lotes/{id}` (a cada 2 s com a tela aberta, e sempre antes de qualquer outra ação sobre a coleta). `importado` → grava o recibo, `rev_confirmada` e o destino **na mesma transação**; `cancelado` → marca o lote como cancelado (terminal, nunca reenviado) e libera a correção do código; o próximo envio cria um lote novo. Nada é considerado concluído sem recibo.
7. **Recibo v2** traz `destino` (estudo, jornada, equipe, código oficial, código recebido, se foi corrigido) e entra no `recibo_sha256` como mais uma linha.
8. **Erros novos:** `409 lote_cancelado` (PUT ou importar em lote cancelado) e `409 ja_decidido` (decisão sobre lote já decidido). `outra_operacao` e `sem_operacao` deixam de existir.

## Alternativas consideradas
- **Validar o código no celular consultando a Central:** contraria o funcionamento offline.
- **Importar automaticamente quando o código existir:** um erro que coincide com outra equipe passaria em silêncio.
- **Pedido de confirmação em memória com espera longa (como o pareamento):** o operador pode demorar mais de 90 s, e o estado se perderia ao fechar a Central; o estado persistido no banco sobrevive a reinícios dos dois lados.
- **Permitir mudar o destino a cada envio:** uma coleta poderia acabar dividida entre equipes.

## Consequências
- `lotes_recebidos` ganha os estados `aguardando_confirmacao` e `cancelado`; nova tabela `coletas` na Central.
- O reconciliador da Central mantém por 7 dias os `.part` de lotes cancelados, reaproveitados no reenvio.
- O celular passa a ter um lote aberto **por coleta** e a usar o `GET` do lote para reconciliar.
- O `central-teste` ganha comandos para confirmar, alterar e cancelar destinos nos testes cruzados.

## Fontes
- Pedido do usuário de 15/11/2026 (Partes II e III) e respostas às perguntas do planejamento (destino travado; ciclo da coleta).
- `docs/synchronization/protocolo.md` e `integridade.md`.
