# Regras específicas do Android

As regras de negócio estão em [`docs/requirements/regras-de-negocio.md`](../../docs/requirements/regras-de-negocio.md). Aqui ficam apenas **comportamentos de UI e validação** do app.

| Tema | Comportamento |
|---|---|
| Nome da propriedade | Erro inline "Informe o nome da propriedade" só ao tentar salvar; foco vai ao campo |
| Contadores fixos | Stepper "– n +" com toque longo acelerado; campo aceita digitação com teclado numérico; vazio = 0; não permite negativo |
| Faixas etárias | Ajuda abaixo de Pessoas: "Crianças 0–11 · Jovens 12–17 · Adultos 18+" (RN-20) |
| Itens predefinidos | Lista de chips; tocar em item já adicionado abre a edição da quantidade (RN-32) |
| "Outros" | Nome obrigatório; aviso "Já existe 'X' nesta propriedade. Somar quantidades?" com opções Somar e Manter separado (RN-33) |
| Quantidade de item | ≥ 1; para zerar, o usuário remove (RN-34) |
| Área | Campo decimal (vírgula ou ponto) + seletor ha/alqueire paulista; mostra "= 25,41 ha" ao lado; até 3 casas; unidade obrigatória se houver valor |
| GPS | Ver `persistencia-offline.md`; aviso amarelo se a precisão passar de 15 m |
| Coordenada manual | Dois campos com teclado decimal com sinal; validação de faixa; aceita vírgula ou ponto |
| Exclusão | Diálogo "Excluir propriedade X? Ela continuará registrada como excluída e a exclusão será enviada à Central." |
| Remoção de foto | Diálogo de confirmação |
| Saída com alterações | "Você tem alterações não salvas" com Salvar, Descartar e Continuar editando |
