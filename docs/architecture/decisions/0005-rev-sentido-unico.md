# ADR-0005 — Versionamento por `rev` e sincronização em sentido único

- **Status:** Aprovado
- **Data:** 2026-11-13

## Contexto
Os relógios dos aparelhos divergem, e `updated_at` não serve para detectar mudanças. O usuário aprovou a sincronização **somente Android → Central** (RN-70).

## Decisão
- Cada entidade sincronizável tem `rev` inteiro, incrementado **somente pelo Android** na mesma transação da alteração, e `rev_confirmada`.
- Pendente = `rev > rev_confirmada`. Não há outbox separado: o estado pendente é derivado, sem crescimento ilimitado.
- A Central guarda `rev_origem` (última rev aplicada) e aplica `(uuid, rev)` uma única vez (idempotência).
- Tombstones têm `rev` própria e são sincronizados como qualquer alteração.
- Correções da Central incrementam `rev_central` e ligam `corrigido_desde_rev_origem`.

## Alternativas consideradas
- **Log de eventos/outbox:** mais flexível para replicação bidirecional, mas maior e desnecessário em sentido único.
- **Vetores de versão:** necessários só com múltiplos escritores replicando entre si.
- **`updated_at`:** rejeitado (relógios).

## Consequências
- Simples e robusto. Se um dia houver sincronização bidirecional, será preciso um novo ADR.
