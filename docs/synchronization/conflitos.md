# Conflitos

Status: **Aprovado** (RN-70 a RN-73, ADR-0005, ADR-0006).

## Política
- Sincronização em **sentido único** (Android → Central).
- A Central pode **corrigir** cadastros. A correção **não volta** ao aparelho.
- **Conflito** = o Android envia nova versão (edição ou exclusão) de um cadastro que a Central corrigiu depois da última versão Android recebida.
- **Unidade:** o cadastro (propriedade + itens + metadados de fotos).
- **Resolução manual** na Central, campo a campo. Não há mescla automática nem sobrescrita silenciosa.

## Tabela de decisão (por cadastro, ao aplicar `rev = R`)
| Situação na Central | Resultado |
|---|---|
| Cadastro inexistente | `aplicado` (criação) |
| `R ≤ rev_origem` | `duplicado` (sem efeito) |
| `R > rev_origem`, sem correção pendente | `aplicado` |
| `R > rev_origem`, com correção desde `rev_origem` | `conflito` |
| Excluído pela Central e Android envia edição | `conflito` (não ressuscita) |
| Corrigido pela Central e Android envia exclusão | `conflito` |
| Excluído pela Central e Android envia exclusão | `aplicado` (convergente) |

## Enquanto o conflito está aberto
- A Central **continua exibindo a versão corrigida** e marca o selo "Conflito pendente". As estatísticas usam essa versão.
- A versão Android completa fica em `conflitos.snapshot_android_json`. **Fotos novas são publicadas normalmente** (arquivos nunca se perdem) e vinculadas ao cadastro na resolução.
- `rev_origem` **não** avança até a resolução. Novos envios do Android para o mesmo cadastro **atualizam** o snapshot do conflito aberto (mantendo o histórico), sem abrir conflitos duplicados.
- No Android, o cadastro aparece como "Confirmado (em análise na Central)".

## Tela de resolução (Central)
- Três colunas: **Base** (última versão Android antes da correção), **Central** e **Android**; destaque nos campos que diferem.
- Por campo, o operador escolhe Central, Android ou um valor editado. Itens e fotos aparecem como listas com marcação de adicionado, removido ou alterado.
- Atalhos: "Manter tudo da Central" e "Aceitar tudo do Android".
- Ao concluir: transação que aplica o resultado, define `rev_origem = R`, desliga `corrigido_desde_rev_origem` se o resultado igualar a versão Android (ou mantém ligado se ainda houver correção) e registra a decisão completa na auditoria.
