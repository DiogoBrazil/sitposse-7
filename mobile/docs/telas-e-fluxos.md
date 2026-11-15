# Telas e fluxos — Android

Princípios: Material 3, **alto contraste ao sol** (tema claro como padrão, cores com contraste ≥ AA, opção de tema escuro), alvos ≥ 48 dp, ações primárias na **metade inferior** (uso com uma mão), poucos toques, feedback imediato.

## Mapa de telas (A28)
```
Abrir o app ──► nenhuma coleta em andamento ──► [1] Código da equipe ──► [2] Início
            ├─► uma coleta em andamento ─────────────────────────────────► [2] Início
            └─► várias ──► [1] Código da equipe (lista para escolher) ───► [2] Início
[2] Início ──► [3] Propriedades (lista) ──► [5] Detalhes ──► [4] Editar
         ├──► "+ Nova propriedade" ──► [4] Cadastro (7 etapas)
         ├──► [1b] Coletas (continuar, concluir, reabrir, corrigir código, nova coleta)
         ├──► [6] Sincronizar (QR → coletas → progresso → confirmação na Central → resultado)
         ├──► [7] Exportar pacote / Ler recibo (futuro)
         └──► Diagnóstico
Banco da versão de teste anterior ──► tela de bloqueio (reinstalar; nada é apagado; ADR-0025)
```

## [1] Código da equipe (RN-05, RN-07)
Campo único grande, monoespaçado, com prefixo `EQ-` fixo e oito caracteres editáveis (maiúsculas, sem autocorreção). O hífen após o quarto caractere é só visual; cursor e seleção são preservados ao apagar, substituir ou editar no meio. Aceita colar o corpo ou o código completo. O dígito verificador é conferido **no aparelho**, sem rede: "Confira os caracteres do código". Botão **Começar coleta**. Abaixo, as coletas em andamento com **Continuar**. Com várias coletas em andamento ao abrir o app, a tela começa pela lista "Escolha a coleta para continuar".
- Código de uma coleta em andamento → abre ela (não duplica).
- Código de uma coleta concluída → "Reabrir para continuar nela?".
- Código novo com outra coleta em andamento → "Concluir a coleta anterior?" [Concluir] [Manter em andamento].

## [1b] Coletas (RF-A02, RN-07, RN-08)
Cada coleta: código, "Em andamento"/"Concluída", início (e conclusão), cadastros e pendentes, destino confirmado pela Central ("Equipe Alfa · Jornada 1 (10/10/2026) · Fazenda Santa Maria") ou "Destino ainda não confirmado", e o estado do último envio (aguardando confirmação, cancelado com o motivo, erro). Ações: **Continuar**, **Concluir** (com resumo e aviso de pendentes), **Reabrir**, **Corrigir código** (só sem destino confirmado; mesmo campo e validação; o código anterior fica registrado), **Sincronizar**, **Nova coleta**.

## [2] Início
Cartão da coleta atual (código, início, destino quando confirmado, aviso de coleta concluída, pareado ou não) com **Trocar coleta**; números grandes **da coleta**: **Propriedades**, **Pendentes de envio**, **Confirmadas**; "Última sincronização confirmada" (fica amarelo após 24 h); rascunhos da coleta; FAB "+ Nova propriedade" (só em coleta em andamento); atalho "Sincronizar".
Ao abrir num dia diferente do início da coleta: "A coleta EQ-… começou em 10/10. Cada jornada costuma ter um código novo." [Continuar nela] [Informar outro código].

## [3] Lista
Filtro **Só esta coleta** (ligado por padrão); busca por nome da propriedade ou do responsável; filtros rápidos (Pendentes, Sem GPS, Sem foto, Erro); cada item mostra o nome, o responsável, ícones de GPS e fotos e o selo de estado:
`Salvo no aparelho` · `Pendente de envio` · `Enviando` · `Confirmado` · `Confirmado (em análise na Central)` · `Erro`.

## [4] Cadastro e edição em etapas (A28)
Cabeçalho "Etapa 3 de 7 · Eletrodomésticos e mobiliário" com barra segmentada; tocar nele abre a lista das etapas (com aviso nas que têm erro) para ir direto.
1. **Identificação e família**: nome da propriedade*, nome do responsável pela propriedade*, pessoas (faixas ECA).
2. **Veículos**: caminhões; automóveis, caminhonetes e utilitários; motocicletas e ciclomotores.
3. **Eletrodomésticos e mobiliário**: dois grupos recolhíveis (abertos), chips do catálogo + vários "Outros".
4. **Animais e área**: bovinos, equinos, aves (chips + "Outros") e área (valor + unidade + conversão).
5. **Localização e fotografias**: obter/digitar coordenada; miniaturas, câmera e remover.
6. **Observações relevantes**: texto livre multilinha, contador `n/4000`.
7. **Revisão**: cartões por categoria só com o que foi preenchido ("Nenhum informado" em uma linha), cada um com **Editar**; os erros aparecem no topo com a etapa de cada um.

Barra inferior: **Voltar** · **Próximo** (na etapa 6, **Revisar**; na 7, **Salvar cadastro**). A navegação nunca é bloqueada por campo vazio; só o salvamento exige os obrigatórios (RN-11). O botão Voltar do sistema volta uma etapa; a seta da barra superior sai (com a pergunta do rascunho). Após salvar, abre o Detalhe.

## [6] Sincronizar (ADR-0024)
1. Explica e pede `ACCESS_LOCAL_NETWORK` (API 37+) e a câmera para o QR. O pareamento é só do aparelho.
2. **Coletas a enviar**, com caixas de seleção (todas marcadas): código e cadastros pendentes.
3. Por coleta: "Coleta EQ-… (1 de 2)", "Enviando cadastros (12)", "Enviando fotos 5/18 (34 MB)", "A Central está conferindo as fotos", **"Aguardando o operador confirmar o destino no computador da Central…"** com **Parar de esperar** (o envio é reconciliado na próxima vez).
4. Resultado por coleta: confirmado ("EQ-…: 12 cadastro(s) confirmado(s) em Equipe Alfa · Jornada 1 (10/10/2026) · Fazenda Santa Maria", e se o código foi corrigido na Central), **cancelado** (motivo e **Corrigir o código**) ou ainda aguardando.
5. Erros sem jargão: "Não encontramos a Central nesta rede. Confira se o celular está no Wi‑Fi do notebook/hotspot." com botões **Tentar novamente** e **Usar pacote USB**.

## [7] Pacote USB
"Exportar pendentes" ou "Exportar backup completo" → seletor de local → "Pacote gravado e verificado". Depois: "Ler recibo da Central (QR)" ou "Abrir arquivo de recibo".

### Ajuste 0.2.1-mvp — localização
Na etapa Localização e fotografias, **Obter pelo aparelho** mantém a captura GNSS; **Digitar localização** abre a entrada manual sem pedir permissão de localização. Com coordenada existente, **Editar localização** abre latitude e longitude preenchidas (também para GNSS). Cancelar preserva o ponto; confirmar sem alterar preserva metadados; alterar números torna a origem manual, sem precisão, satélites ou instante GNSS (RN-52). Aceita ponto/vírgula decimal e só permite coordenadas finitas nas faixas RN-51.
