# ADR-0023 — Estudo de situação, jornadas, equipes por jornada e códigos públicos

- **Status:** Proposto (plano aprovado pelo usuário em 15/11/2026, A28)
- **Data:** 2026-11-15
- **Substitui / Substituído por:** substitui o ADR-0012 (entidade operação) e a parte de equipe do ADR-0017 (§1, objeto `equipe` no lote e no pareamento)

## Contexto
No MVP a Central tem **uma única operação ativa**, que nunca pode ser encerrada, e a equipe é criada no celular (nome, responsável, área) e ligada à operação no primeiro pareamento. No uso real:
- um estudo de situação dura vários dias (jornadas), com equipes diferentes a cada dia;
- vários estudos independentes existem ao mesmo tempo;
- um celular passa por várias jornadas, e a equipe não deve ser digitada por extenso no campo.

## Decisão
1. **Hierarquia na Central:** `Estudo de Situação → Jornada → Equipe → Cadastros`. A **operação deixa de existir**; o Estudo ocupa o lugar dela em estatísticas, relatórios e cartas.
   - **Estudo:** UUID, nome obrigatório (não único) e **código público** único.
   - **Jornada:** UUID, estudo, **número sequencial no estudo** e **data civil** (`AAAA-MM-DD`). A data não é chave: pode haver duas jornadas na mesma data.
   - **Equipe:** UUID, jornada, nome livre (único dentro da jornada, comparando o nome normalizado) e **código público** único em toda a Central. Reaproveitar o nome de uma equipe de outra jornada (ou copiar a relação de equipes) **sempre gera um código novo**.
2. **Códigos públicos:**
   - Alfabeto Crockford Base32 (`0123456789ABCDEFGHJKMNPQRSTVWXYZ`). Na entrada: maiúsculas e minúsculas, espaços e hífens são ignorados, `O` vira `0` e `I`/`L` viram `1`.
   - **7 caracteres aleatórios** (gerador criptográfico) **+ 1 dígito verificador Luhn mod 32**. O dígito detecta toda troca de um caractere e quase toda inversão de dois vizinhos, o que permite ao celular recusar um código digitado errado **sem rede**.
   - Equipe: `EQ-XXXX-XXXX`; Estudo: `SITPOSSE7-XXXX-XXXX`. Guardados sem hífens (`EQ7K3M9QX2`) e exibidos com hífens.
   - `UNIQUE` no banco; colisão → gerar de novo (até 5 tentativas). Casos de referência em `shared/test-fixtures/codigos/codigos.v1.json`, usados por Go e Kotlin.
3. **Estudo em foco:** Painel, Propriedades, relatórios e cartas trabalham com **um estudo por vez**; jornada e equipe são filtros dentro dele. O consolidado do estudo soma as jornadas.
4. **Exclusão:** jornada e equipe só podem ser excluídas (tombstone) enquanto não houver coleta vinculada. Estudos não são excluídos nesta fase.

## Alternativas consideradas
- **Manter a operação e acrescentar jornadas:** a operação "uma ativa por vez" não permite estudos paralelos e não tem como ser encerrada; renomeá-la para estudo é mais claro.
- **Código hexadecimal sem dígito verificador** (`EQ-73B9D210`): o erro de digitação só seria notado na Central e poderia coincidir com outra equipe.
- **Quatro caracteres aleatórios:** risco de colisão e de acerto por engano alto demais.
- **Data da jornada como identificador:** impediria duas jornadas no mesmo dia.

## Consequências
- Migração da Central e do celular reescritas (ver ADR-0025); `FiltroEstudo`, relatórios e cartas trocam "Operação" e "Área identificada" por Estudo, Jornada e Equipe.
- Os campos `responsavel` e `identificacao_area` da equipe deixam de existir.
- A P6 (reuso do aparelho em outra operação) deixa de existir: o celular troca de coleta com outro código (ADR-0024).

## Fontes
- Pedido do usuário de 15/11/2026 (Partes I e V) e respostas às perguntas do planejamento.
- Crockford, *Base32 Encoding* (https://www.crockford.com/base32.html): alfabeto sem `I L O U` e decodificação tolerante.
- Algoritmo de Luhn mod N (https://en.wikipedia.org/wiki/Luhn_mod_N_algorithm): detecta toda troca de um caractere e quase toda inversão de vizinhos, para qualquer base par.
