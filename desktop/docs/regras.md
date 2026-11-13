# Regras específicas da Central

Regras de negócio: [`docs/requirements/regras-de-negocio.md`](../../docs/requirements/regras-de-negocio.md). Aqui ficam só os comportamentos da Central.

| Tema | Comportamento |
|---|---|
| Correção de cadastro | Toda edição pelo operador marca `corrigido_desde_rev_origem=1`, incrementa `rev_central` e grava auditoria |
| Exclusão na Central | Tombstone com auditoria; o cadastro some das estatísticas e aparece em "Excluídos" |
| Criação na Central | Permitida (p. ex. informação recebida por outro meio); `equipe` = "Central"; origem registrada |
| Conflito | Selo "Conflito pendente" na lista e nos detalhes; contador no menu |
| Operação ativa | Só uma operação ativa recebe pareamentos; consultas permitem escolher qualquer operação |
| Nome do operador | Obrigatório nas Configurações antes da primeira alteração (vai para a auditoria) |
| Mapa offline | Só o painel do mapa mostra a mensagem RN-91; o resto continua funcionando |
