# Importação e consolidação — Central

O núcleo único `internal/importacao` atende `PUT/POST /v1/lotes` (Wi‑Fi) e `.s7pkg` (USB).

## Passos (protocolo v2, ADR-0024)
1. **Validar o envelope** (`PUT /v1/lotes`): versões 2, dispositivo pareado e não revogado, `lote_uuid`, coleta (código no formato com dígito verificador) pertencente a esta instalação. A coleta é registrada em `coletas` (só dados de recepção) e o lote fica `recebendo`.
2. **Idempotência do lote**: mesmo `lote_uuid` e bytes → estado atual (recibo, se importado); bytes diferentes → `lote_divergente`; lote `cancelado` → `lote_cancelado`.
3. **Fotos** (`POST …/importar`): conferir tamanho e SHA‑256 em staging. Tudo certo → lote `aguardando_confirmacao`; nada é aplicado.
4. **Decisão do operador** (`Importador.Decidir`, modal "Confirmar destino da sincronização", `Importador.Aguardando` lista os lotes):
   - destino sugerido = equipe do código recebido (`estudos.DestinoPorCodigo`); código inexistente → sem sugestão;
   - coleta já importada → destino travado (só a mesma equipe é aceita);
   - cancelar → `cancelado` + `decisoes_destino`; nada importado nem apagado.
5. **Confirmar ou alterar**: conferir de novo as fotos (staging perdido → volta a `recebendo`); publicar com rename e fsync (ver [integridade](../../docs/synchronization/integridade.md)); **transação única**:
   - gravar o destino da coleta (gatilho impede trocá-lo depois) e a decisão (`confirmar` ou `alterar`, com o código recebido e o oficial);
   - validar e aplicar cada registro com `internal/contratos` num `SAVEPOINT` (inválido → `rejeitado`), seguindo a tabela de [conflitos.md](../../docs/synchronization/conflitos.md); propriedades gravam `coleta_uuid` e a `equipe_uuid` de destino;
   - gravar `aplicacoes`, `auditoria` (origem = `coleta:<uuid>`), `recebido_em`;
   - `lotes_recebidos.status = importado` + `recibo_json` (com `destino`).
6. O celular obtém o recibo por `GET /v1/lotes/{id}` (Wi‑Fi) ou `.s7rcb` e QR (USB, futuro).

## Consolidação
- Visões vigentes trazem equipe, jornada e estudo; telas e relatórios trabalham com um estudo em foco (A28).
- `aliases_itens` agrupa nomes de "Outros" para estatística sem alterar o original.
- Possíveis duplicatas entre equipes (P5): sugestão por `nome_normalizado` semelhante + distância GPS; ação "Vincular como mesma família" registra a relação sem fundir nem apagar (detalhe a aprovar na Fase 4).

## Restauração de cadastros
Ver [protocolo.md §7](../../docs/synchronization/protocolo.md).

## Central 0.7.0-mvp (A29)
Implementação e critérios em [ADR-0026](../../docs/architecture/decisions/0026-correcoes-e-exclusoes-central.md). Tabelas com 5 registros/página e totais externos. Editor por seções e exclusão lógica em cascata; fotos/recibos/cartas preservados. Novas versões de cadastros corrigidos são recusadas temporariamente, inclusive itens/fotos isolados. Nenhum dado é alterado no celular. Testes de fluxo em `internal/nucleo/correcoes_test.go`; validar ações/modais, download e relatórios no Windows.
