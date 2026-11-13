# Telas e fluxos — Android

Princípios: Material 3, **alto contraste ao sol** (tema claro como padrão, cores com contraste ≥ AA, opção de tema escuro), alvos ≥ 48 dp, ações primárias na **metade inferior** (uso com uma mão), poucos toques, feedback imediato.

## Mapa de telas
```
Primeira abertura ──► [1] Configurar equipe ──► [2] Início
[2] Início ──► [3] Propriedades (lista) ──► [5] Detalhes ──► [4] Editar
         ├──► "+ Nova propriedade" ──► [4] Cadastro
         ├──► [6] Sincronizar (QR → progresso → resultado)
         ├──► [7] Exportar pacote / Ler recibo
         └──► [8] Configurações (equipe, backup, liberar espaço, sobre)
```

## [1] Configurar equipe
Nome da equipe*, Responsável*, Identificação da área (opcional), botão "Começar". Texto: "Os dados ficam neste aparelho e funcionam sem internet."

## [2] Início
Cartão da equipe (nome, operação ou "não pareada"); números grandes: **Propriedades**, **Pendentes de envio**, **Confirmadas**; "Última sincronização confirmada: há 3 h" (fica amarelo após 24 h); FAB "+ Nova propriedade"; atalho "Sincronizar".

## [3] Lista
Busca por nome da propriedade ou do responsável; filtros rápidos (Pendentes, Sem GPS, Sem foto, Erro); cada item mostra o nome, o responsável, ícones de GPS e fotos e o selo de estado:
`Salvo no aparelho` · `Pendente de envio` · `Enviando` · `Confirmado` · `Confirmado (em análise na Central)` · `Erro`.

## [4] Cadastro e edição
Seções recolhíveis com resumo no cabeçalho (p. ex. "Veículos · 3"):
1. Identificação (nome*, responsável pela família)
2. Pessoas (com faixas ECA) — total
3. Veículos — total
4. Eletrodomésticos (chips + "Outros")
5. Móveis
6. Bovinos — total
7. Equinos — total
8. Aves (chips + "Outros")
9. Área (valor + unidade + conversão)
10. Localização (Obter localização / Digitar)
11. Fotos (grade de miniaturas + câmera)

Barra inferior fixa: **Salvar**. Após salvar: snackbar "Salvo no aparelho ✓" e volta para Detalhes.

## [6] Sincronizar
1. Explica e pede `ACCESS_LOCAL_NETWORK` (API 37+) e a câmera para o QR.
2. Leitura do QR → "Conectando à Central X…".
3. Progresso por fase: "Enviando cadastros (12)", "Enviando fotos 5/18 (34 MB)", "Aguardando a Central importar".
4. Resultado: "18 cadastros confirmados · 1 em análise na Central · 0 erros".
5. Erros sem jargão: "Não encontramos a Central nesta rede. Confira se o celular está no Wi‑Fi do notebook/hotspot." com botões **Tentar novamente** e **Usar pacote USB**.

## [7] Pacote USB
"Exportar pendentes" ou "Exportar backup completo" → seletor de local → "Pacote gravado e verificado". Depois: "Ler recibo da Central (QR)" ou "Abrir arquivo de recibo".
