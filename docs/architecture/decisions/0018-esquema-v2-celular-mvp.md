# ADR-0018 — Esquema v2 do banco do celular e sincronização no `:core:sync` (MVP)

- **Status:** Proposto (implementado no MVP, A24; aguarda aprovação do usuário)
- **Data:** 2026-11-14
- **Substitui / Substituído por:** —

## Contexto
O MVP (A24) precisa do cadastro completo offline e do envio seguro à Central. O `modelo-de-dados.md` descreve as tabelas, mas o Room 3 tem limites: não declara `CHECK` nem índice parcial, e valida o esquema real contra as entidades a cada abertura e migração. Também faltava decidir onde fica a lógica de envio, que precisa do banco e da rede e deve ser testável na JVM contra a Central Go real.

## Decisão
1. **Banco v2** (`BancoCampo`, migração `MIGRACAO_1_2`, só `CREATE TABLE/INDEX`): `equipe`, `propriedades`, `itens_propriedade`, `fotografias`, `rascunhos`, `lotes_envio` (com o **JSON congelado do lote**), `lote_itens` (com o resultado do recibo), `pareamento`. O UUID da instalação fica na tabela `meta` (`instalacao_uuid`). A migração usa exatamente o SQL exportado pelo Room em `schemas/.../2.json` e é testada com o `MigrationTestHelper` (`room3-testing`, só testes).
2. **Sem `CHECK` nem índice parcial no Android:** as regras (RN-11..55, RN-32) ficam no domínio (`ValidacaoCadastro`) e no `RepositorioCampo`, ambos testados na JVM, e a Central confere tudo de novo na importação (ADR-0017).
3. **Catálogo de itens no código** (`Catalogo`, `:core:dominio`), sem tabela no celular; um teste confere com `shared/test-fixtures/catalogo/itens.v1.json` (o mesmo que a Central semeia).
4. **Fotos do formulário:** o arquivo é gravado primeiro (pasta privada, `.part` + fsync + rename); o registro entra na **mesma transação** que salva o cadastro. O repositório recusa registrar foto cujo arquivo não exista.
5. **`:core:sync` depende de `:core:dados`** e concentra `ApiCentral`, `Sincronizador` e `Pareador`: o mesmo código roda no app (variante Android do `:core:dados`) e nos testes JVM contra o `central-teste`. Só um lote aberto por vez; ele é retomado até ter recibo.
6. **Confirmação (RN-74):** o recibo é conferido (`recibo_sha256` e `lote_uuid`) e gravado numa transação que faz `rev_confirmada = max(rev_confirmada, rev)` para `aplicado`, `duplicado` e `conflito`; `rejeitado` continua pendente e aparece como erro.

## Alternativas consideradas
- **CHECKs criados à mão na migração:** o Room criaria as tabelas sem eles numa instalação nova; os dois caminhos divergiriam.
- **Sincronização no módulo `:app`:** só testável com Android; perderia o teste cruzado na JVM.

## Consequências
- Toda versão nova do banco exige migração gerada a partir do esquema exportado e teste com `MigrationTestHelper`.
- O teste `SincronizacaoGoTest` cobre os cenários 1, 4, 5, 6 e 8 com o banco Room real e a Central Go real.

## Fontes
- Room 3: `withWriteTransaction`, `Migration.migrate(SQLiteConnection)` como `suspend` (https://developer.android.com/training/data-storage/room/migration-2-to-3); `MigrationTestHelper` (variante JVM em `androidx.room3:room3-testing:3.0.3`).
