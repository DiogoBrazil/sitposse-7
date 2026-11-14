# Fixtures de teste compartilhadas

Executadas pelos testes **Kotlin e Go**. Se um lado divergir, o teste falha.

| Arquivo | Regra | Formato |
|---|---|---|
| `area/conversao.json` | RN-40..45, ADR-0009 | `entrada` (texto digitado), `unidade`, `milesimos` esperado ou `erro`; `dm2`; `ha_exibicao` (2 casas, HALF_UP, vírgula) |
| `area/somas.json` | RN-45 | lista de propriedades e total esperado em dm²/ha, contagem "sem área" |
| `normalizacao/nomes.json` | RN-35 | `original` → `normalizado`, `valido` |
| `contratos/qr-pareamento.v1.json` | protocolo §2 | `validos` e `invalidos` (com `motivo`) do QR |
| `contratos/{lote,estado-lote,recibo}.v1.json` | protocolo §3–6, ADR-0017 | `validos` e `invalidos`; `recibo` traz também o `texto_canonico` e o `recibo_sha256` esperados |
| `contratos/pareamento.v1.json` | protocolo §2 | `pedidos_validos`, `respostas_validas`, `invalidos` |
| `catalogo/itens.v1.json` | RN-30 | catálogo inicial (códigos estáveis); o seed da Central e o do Android devem ser iguais |
| `tls/pin.json` + `tls/central-fixture.crt.pem` | ADR-0007 | certificado → pin SPKI esperado (calculado com OpenSSL; Go e Kotlin conferem) |

**Fórmulas de referência:**
- ha: `dm2 = milesimos × 1000`; alqueire paulista: `dm2 = milesimos × 2420`.
- Exibição em ha com 2 casas: `centi_ha = (dm2 + 5000) div 10000` (HALF_UP, inteiros ≥ 0).
- Normalização: trim → colapso de espaços em branco (inclui tab e quebra de linha) em um espaço → minúsculas → decomposição NFD → remoção de marcas combinantes (acentos, cedilha, til).
