# ADR-0026 — Correções e exclusões na Central, recusa temporária e paginação

- **Status:** Aprovado pelo usuário ao autorizar a implementação do plano em 15/11/2026.
- **Data:** 2026-11-15
- **Altera:** A24 (consulta somente leitura) e ADR-0023 (estudos sem exclusão nesta fase); detalha a transição para ADR-0005/0006.

## Decisão
- Central permite editar todos os dados cadastrais da propriedade, itens e remoção de fotos; vínculos de estudo/jornada/equipe/coleta e UUIDs ficam fixos. Não acrescenta fotos.
- Nome de estudo/equipe pode ser editado; códigos não mudam. Exclusão isolada de equipe continua restrita à ausência de coleta vinculada.
- Exclusão do estudo ou da propriedade aplica tombstones em cascata, em transação com auditoria. Nunca apaga fotos, recibos, lotes importados ou coletas históricas. Cartas já geradas são histórico preservado. Lotes pendentes associados ao estudo são cancelados com decisão persistida; coletas com destino excluído não podem enviar lote novo para esse destino.
- A decisão sobre lote e a exclusão de estudo usam a mesma trava do importador; o destino também é revalidado dentro da transação que aplica o lote.
- **Escolha explícita do usuário:** a fila de conflitos continua adiada. Versões novas do celular de cadastros corrigidos/excluídos são recusadas com motivo; continuam pendentes no aparelho. A proteção abrange filhos (item/foto), mesmo quando o lote não traz a propriedade. Reenvios já aplicados continuam duplicados/idempotentes. Correções não voltam ao celular.
- Concorrência otimista por hash do documento persistido completo (incluindo revisões de propriedade, itens e fotos), carregado em transação de leitura. Salvar exige o mesmo token em transação de escrita, evitando sobrescrita de importação/edição concorrente. Auditoria preserva antes/depois; `registro_json` recebido do Android é mantido como base.
- Novo item criado na Central tem UUID próprio, origem explícita `central`, revisão inicial 1 e `rev_central=1`; não representa versão recebida de um aparelho. A propriedade fica protegida para evitar mescla silenciosa.
- Todas as tabelas de listagem têm cinco registros por página, após filtro/ordenação. Totais ficam fora da tabela e representam todo o conjunto filtrado. Exportações recebem filtros, nunca a página.
- Jornadas e equipes em tabelas separadas, seleção de jornada, ações por ícone com título e rótulo acessível. Modais nativos `dialog` centralizados com foco contido e retorno ao acionador; cancelamento de editor alterado pede descarte.

## Compatibilidade e verificação
Nenhuma mudança de esquema SQLite ou protocolo v2; sem dependência nova e sem reinício de banco. Os bindings Wails acrescentam operações de edição, exclusão e download de foto. Testes Go de fluxo real conferem recusa, auditoria, concorrência, cascata e rollback; JVM Kotlin × Go continua compatível. Frontend e navegador conferem cinco linhas, navegação, modais e responsividade. Windows real é validado pelo usuário com o instalador 0.7.0-mvp.
