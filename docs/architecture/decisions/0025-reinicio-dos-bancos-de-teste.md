# ADR-0025 — Reinício único dos bancos de teste (geração 2 do esquema)

- **Status:** Aprovado pelo usuário em 15/11/2026, após implementação e teste em bancada (A28)
- **Data:** 2026-11-15
- **Substitui / Substituído por:** altera o ADR-0017 (§8, migração 0002 da Central) e o ADR-0018 (§1, banco v2 do celular)

## Contexto
O ADR-0023 e o ADR-0024 mudam a estrutura central dos dois bancos (operação e equipe somem; coleta, estudo e jornada entram). Os bancos existentes têm **só dados de teste** (A24), e o usuário autorizou removê-los nesta fase. Uma migração que convertesse os dados antigos exigiria inventar estudos e jornadas que não existem. Ao mesmo tempo, o AGENTS.md §3 proíbe migração destrutiva e recriação silenciosa de banco.

## Decisão
1. **Nova linha de base, uma única vez:**
   - Central: `0001_meta.sql` mantida; `0002` reescrita com o esquema novo e o marcador `meta.esquema_geracao = '2'`.
   - Celular: Room versão **3**, sem migração a partir das versões 1 e 2; esquemas `1.json` e `2.json` ficam como histórico.
2. **Guarda contra banco antigo** (nunca apaga nada):
   - Central: banco com `user_version ≥ 2` e sem o marcador → não abre, não altera e mostra a tela de bloqueio com instruções.
   - Celular: arquivo com `user_version` 1 ou 2 → o Room não é aberto e o app mostra "Dados de teste da versão anterior: desinstale e instale o app novamente".
3. **Reinício controlado, feito à mão e com confirmação do usuário na hora:**
   - Central fechada; os arquivos `sitposse7.db`, `-wal`, `-shm` e as pastas `fotos` e `staging` são **movidos** (não apagados) para `mvp-antigo-AAAAMMDD-HHMMSS` dentro de `%LOCALAPPDATA%\SITPOSSE-7`. `tls`, `cartografia` e as credenciais do Copernicus ficam.
   - Celular: desinstalar o app (apaga só os dados dele) e instalar o novo APK.
4. **Daqui em diante, migrações voltam a ser não destrutivas e testadas** (`0003…` na Central; `MIGRACAO_3_4…` no celular). Esta autorização vale só para esta fase e não vira comportamento do instalador nem do app.

## Alternativas consideradas
- **Migração completa dos dados de teste:** exigiria criar estudos e jornadas fictícios; custo sem benefício.
- **Migrações novas por cima das antigas (sem reinício):** deixariam tabelas mortas (`operacoes`, `equipe`) e reconstruções de tabela com NOT NULL difíceis de testar.
- **Trocar o nome do arquivo do banco:** deixaria o banco antigo esquecido no aparelho e na Central.

## Consequências
- Os aparelhos precisam ser pareados de novo.
- O teste `TestMigraBancoDaFase0SemPerderDados` continua valendo (0001 → 0002 nova); novos testes conferem que a guarda recusa o banco antigo sem alterá-lo.
- O procedimento fica em `docs/operacao/teste-de-campo-mvp.md`.

## Fontes
- Pedido do usuário de 15/11/2026 (Parte VI); AGENTS.md §3.
