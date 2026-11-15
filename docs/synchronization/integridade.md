# Integridade e recuperação

## Invariantes (verificadas por testes após cada cenário)
1. Nenhum registro de foto vigente aponta para arquivo inexistente ou com SHA‑256 diferente.
2. Cada `(entidade_uuid, rev)` é aplicado **no máximo uma vez** na Central.
3. Um lote importado tem **exatamente um** recibo persistido, e a consulta repetida devolve o mesmo recibo.
4. Contagens por equipe na Central = contagens vigentes confirmadas no Android, descontados conflitos e rejeições.
5. Nenhum tombstone "ressuscita" sem passar pela fila de conflitos.
6. Estatísticas = agregação de `v_*_vigentes`.
7. Nenhum lote é aplicado sem uma decisão do operador em `decisoes_destino` (ADR-0024).
8. Uma coleta com destino confirmado nunca muda de equipe; um lote `cancelado` nunca é aplicado nem reaberto.
9. Uma correção de destino ou de código atinge só a coleta escolhida.

## Fotos na Central: staging e publicação
`SQLite` não faz transação de sistema de arquivos. A ordem é sempre **arquivo antes do registro**:
1. Recepção em `dados/staging/{uuid}.part`, com append e fsync a cada bloco.
2. Na importação: verificar tamanho e SHA‑256.
3. `rename` atômico para `dados/fotos/{2 primeiros hex do uuid}/{uuid}.jpg` (mesmo volume) e fsync do diretório.
4. Transação SQLite com o registro.
5. **Reconciliador** na inicialização e periodicamente:
   - remove `.part` de lotes já importados ou com mais de 7 dias sem atividade; mantém por até 7 dias os de lotes `aguardando_confirmacao` e `cancelado` (reaproveitados no reenvio);
   - lista arquivos publicados **sem** registro (queda entre os passos 3 e 4) e os remove só depois de confirmar que nenhum lote pendente os referencia;
   - registros sem arquivo são **alarme**, nunca apagados.

## Fotos no Android
Ver `mobile/docs/persistencia-offline.md`. A ordem é a mesma (arquivo com fsync e rename, depois o banco) e há reconciliador na inicialização.

## Pontos de falha simulados (testes)
| Ponto | Comportamento esperado |
|---|---|
| Queda durante `PUT /v1/lotes` | Reenvio idempotente |
| Queda no meio de uma foto | Retoma pelo offset |
| Hash divergente | Apenas aquela foto é reenviada |
| Crash após publicar arquivos e antes da transação | Arquivos órfãos removidos com segurança; o reenvio reimporta |
| Crash após o commit e antes da resposta | `GET /v1/lotes/{id}` devolve o recibo |
| Central fechada com lote aguardando confirmação | Ao abrir, o modal reaparece; o celular continua vendo `aguardando_confirmacao` |
| Operador decide com o celular fora da rede | O celular obtém o recibo (ou `cancelado`) pelo `GET` antes de qualquer outra ação |
| Correção do código no celular com lote em trânsito | Bloqueada até o `GET` reconciliar o lote |
| Crash do Android após receber o recibo e antes de gravá-lo | Nova consulta devolve o mesmo recibo |
| Disco cheio na Central | Lote em erro, nada parcial publicado, mensagem clara |
| Disco cheio no Android | Salvamento falha com mensagem; formulário preservado |
| Notebook desligado durante a importação | Ao religar: a transação ou foi aplicada por completo ou não foi; o reconciliador limpa o resto |
