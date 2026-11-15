# Regras específicas da Central

Regras de negócio: [`docs/requirements/regras-de-negocio.md`](../../docs/requirements/regras-de-negocio.md). Aqui ficam só os comportamentos da Central.

| Tema | Comportamento |
|---|---|
| Correção de cadastro | Toda edição pelo operador marca `corrigido_desde_rev_origem=1`, incrementa `rev_central` e grava auditoria |
| Exclusão na Central | Tombstone com auditoria; o cadastro some das estatísticas e aparece em "Excluídos" |
| Criação na Central | Permitida no futuro (p. ex. informação recebida por outro meio); origem registrada |
| Conflito | Selo "Conflito pendente" na lista e nos detalhes; contador no menu |
| Estudo em foco | Painel, Propriedades, relatórios e cartas mostram um estudo por vez (A28); a escolha fica guardada em `meta.estudo_em_foco`. A recepção aceita lotes de qualquer estudo |
| Destino da sincronização | Todo lote recebido espera o operador confirmar, alterar ou cancelar o destino (RN-09); a coleta importada fica presa à equipe |
| Banco de geração anterior | A Central abre só com a tela de bloqueio e não altera o banco (ADR-0025) |
| Nome do operador | Obrigatório nas Configurações antes da primeira alteração (vai para a auditoria) |
| Mapa offline | Só o painel do mapa mostra a mensagem RN-91; o resto continua funcionando |
