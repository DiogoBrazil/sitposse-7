# Importação e consolidação — Central

O núcleo único `internal/importacao` atende `PUT/POST /v1/lotes` (Wi‑Fi) e `.s7pkg` (USB).

## Passos
1. **Validar o envelope**: versões (`protocolo_versao`, `schema_versao`), dispositivo pareado e não revogado, equipe coerente com a instalação, `lote_uuid`.
2. **Idempotência do lote**: se `lotes_recebidos.status = importado`, devolver o recibo existente.
3. **Validar registros** contra o JSON Schema e as regras (CHECKs). Registro inválido → `rejeitado` com motivo; os demais seguem.
4. **Fotos**: conferir tamanho e SHA‑256 em staging; publicar com rename e fsync (ver [integridade](../../docs/synchronization/integridade.md)).
5. **Transação única**:
   - para cada cadastro, aplicar a tabela de decisão de [conflitos.md](../../docs/synchronization/conflitos.md);
   - gravar `aplicacoes`, `auditoria` (origem = equipe/instalação) e `conflitos`;
   - gravar `recebido_em`;
   - `lotes_recebidos.status = importado` + `recibo_json`.
6. Responder com o recibo (Wi‑Fi) ou gerar `.s7rcb` e QR (USB).

## Consolidação
- Visões vigentes reúnem todas as equipes da operação.
- `aliases_itens` agrupa nomes de "Outros" para estatística sem alterar o original.
- Possíveis duplicatas entre equipes (P5): sugestão por `nome_normalizado` semelhante + distância GPS; ação "Vincular como mesma família" registra a relação sem fundir nem apagar (detalhe a aprovar na Fase 4).

## Restauração de equipe
Ver [protocolo.md §7](../../docs/synchronization/protocolo.md).
