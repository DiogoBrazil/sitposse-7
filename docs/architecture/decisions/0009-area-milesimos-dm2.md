# ADR-0009 — Área em milésimos e cálculo em dm²

- **Status:** Aprovado
- **Data:** 2026-11-13

## Contexto
Ponto flutuante gera erros de arredondamento em somas, e Kotlin e Go precisam produzir resultados idênticos. 1 alqueire paulista = 2,42 ha (aprovado).

## Decisão
- Armazenar `area_valor_milesimos` (INTEGER), que é o valor digitado × 1000 (até 3 casas), e `area_unidade` (`ha`|`alq_paulista`).
- Converter para **dm² inteiros**: ha → ×1.000; alq. paulista → ×2.420 (por milésimo).
- Somar em inteiros (Int64) e arredondar **só na exibição** (ha com 2 casas, HALF_UP).
- Entrada com mais de 3 casas é rejeitada na UI, com mensagem.
- Casos de referência: `shared/test-fixtures/area/conversao.json`, executados em Kotlin e Go.

## Consequências
- Exato para qualquer soma realista (Int64 comporta ~9,2 × 10¹⁸ dm²).
