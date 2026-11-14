# ADR-0017 — Contratos v1 de sincronização e esquema 0002 da Central (MVP)

- **Status:** Proposto (implementado no MVP, A24; aguarda aprovação do usuário)
- **Data:** 2026-11-14
- **Substitui / Substituído por:** — (detalha o protocolo aprovado em `protocolo.md`, sem mudar suas decisões)

## Contexto
O MVP (A24) precisa da recepção real de lotes na Central. O `protocolo.md` (aprovado com o plano) define endpoints, máquina de estados, recibo e idempotência, mas deixava em aberto: o formato exato dos payloads, os códigos de erro, como o pedido de pareamento espera o operador, como calcular o `recibo_sha256`, como detectar um lote reenviado com conteúdo diferente e como validar os registros na Central sem embutir os schemas (que ficam fora do módulo Go).

## Decisão
1. **Contratos v1** em `shared/schemas`: `comum`, `pareamento` (pedido e resposta), `lote`, `estado-lote` e `recibo`, com fixtures válidas e inválidas em `shared/test-fixtures/contratos` e o catálogo em `shared/test-fixtures/catalogo/itens.v1.json`. UUID v4 em minúsculas; instantes ISO 8601 UTC com ms; registros com todas as chaves presentes (`null` explícito); tombstone = `excluido_em` preenchido. O `nome_normalizado` dos "Outros" **não** trafega: cada lado calcula pela RN-35.
2. **Pareamento por espera longa:** `POST /v1/pareamento` só responde depois que o operador confirma ou recusa na tela da Central, ou após **90 s** (`tempo_esgotado`). O token do QR vale 10 min, é de uso único e só é marcado como usado na confirmação, dentro da transação que grava o aparelho.
3. **Erros:** corpo `{"codigo","erro"}` com códigos estáveis (`invalido` 400, `nao_autorizado` 401, `proibido`/`recusado` 403, `nao_encontrado` 404, `lote_divergente`/`outra_operacao`/`tempo_esgotado`/`sem_operacao` 409, `token_invalido` 410, `versao_incompativel` 422). **Não se usa 408**, porque o OkHttp repete sozinho requisições que recebem 408. `PUT /v1/fotos` com offset errado responde 409 com `offset` (bytes já recebidos). `POST …/importar` com fotos faltando ou hash divergente responde **409 com o estado do lote** (`fotos_faltantes`), sem aplicar nada.
4. **Idempotência do lote:** a Central guarda o SHA‑256 dos bytes do manifesto. O mesmo `lote_uuid` com bytes idênticos é aceito quantas vezes vier (e, se já importado, devolve o mesmo recibo); com bytes diferentes, `409 lote_divergente`. Por isso o Android **congela o JSON do lote** ao prepará-lo e reenvia exatamente esses bytes.
5. **`recibo_sha256`** = SHA‑256 do texto `lote_uuid\ncentral_id\nimportado_em\n` seguido de uma linha `entidade|uuid|rev|resultado|motivo\n` por resultado, na ordem do recibo (fixture com o hash esperado).
6. **Aplicação:** cada registro num `SAVEPOINT` (um registro inválido vira `rejeitado` sem derrubar o lote); exclusões de itens antes das inclusões (índice único da RN-32); `rev > rev_origem` → `aplicado`, senão `duplicado`. Como a Central não corrige cadastros no MVP, não há conflito; se um dia houver correção sem a fila de conflitos, o registro é **rejeitado**, nunca sobrescrito.
7. **Validação na Central em Go** (`internal/contratos`), não com biblioteca de JSON Schema em produção. Os testes validam todas as fixtures **com o schema e com o código Go**, e falham se os dois discordarem.
8. **Esquema 0002 da Central** (`internal/banco/migracoes/0002_mvp_recepcao.sql`), só com tabelas novas: o modelo de `modelo-de-dados.md` mais `registro_json` em cada entidade (último payload aplicado, usado como "antes" da auditoria), `manifesto_sha256` e `ack_em` em `lotes_recebidos`, `fotos_esperadas` (fotos que um lote em recepção aguarda), no máximo **uma operação ativa** e gatilhos que tornam a `auditoria` somente acréscimo. As tabelas `conflitos`, `aliases_itens` e `backups` entram nas fases delas.

## Alternativas consideradas
- **Pareamento com consulta periódica** (`202` + `GET` do pedido): mais estados no app; a espera longa é suficiente para o volume (poucos aparelhos).
- **Validar com JSON Schema em produção:** exigiria copiar os schemas para dentro do módulo Go ou um passo de build; a dupla checagem nos testes dá a mesma garantia.
- **Hash do recibo sobre JSON canônico:** a canonicalização de JSON difere entre Go e Kotlin; o texto por linhas é trivial nos dois lados.

## Consequências
- `protocolo_versao` e `schema_versao` continuam **1**: é a primeira versão implementada.
- O Android precisa congelar o JSON de cada lote (`lotes_envio`) para reenviar bytes idênticos.
- Testes de ponta a ponta em Go (`internal/nucleo/fluxo_test.go`) cobrem os cenários 1, 4, 5, 6, 7 e 8 de `testing/estrategia.md` §4; o cliente Kotlin repetirá os mesmos cenários contra o `central-teste`.

## Fontes
- `docs/synchronization/protocolo.md`, `integridade.md`; `docs/architecture/modelo-de-dados.md`.
- OkHttp 5, `RequestBody.isOneShot`: por padrão o OkHttp retransmite o corpo em conexão reaproveitada que caiu, em **HTTP 408**, em 401/407 com `Authenticator`, em 503 com `Retry-After: 0` e em 421 (https://square.github.io/okhttp/5.x/okhttp/okhttp3/-request-body/is-one-shot.html). Por isso todo endpoint é idempotente e o `PUT /v1/fotos` confere o offset.
