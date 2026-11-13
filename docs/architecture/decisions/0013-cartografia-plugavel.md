# ADR-0013 — Cartografia online com provedor plugável

- **Status:** Aprovado (escolha do provedor pendente: P1)
- **Data:** 2026-11-13

## Contexto
Mapas e cartas-imagem são online (RN-90). Os provedores diferem em custo, qualidade e **direito de imprimir ou exportar**:
- **Esri World Imagery:** os termos de *static maps* permitem relatórios, apresentações e obras governamentais com atribuição "Esri e licenciadores"; exige chave (ArcGIS Location Platform).
- **Brasil MAIS (PF/Senasp, Planet):** gratuito para órgãos de segurança aderentes; falta confirmar se a PMRO tem acesso e se há serviço WMS/XYZ consumível.
- **Google Map Tiles:** exige faturamento; impressão de satélite restrita; atribuição rígida.
- **Mapbox:** impressão exige contato comercial.

## Decisão
- Interface `ProvedorCartografico` (Go e TS): URL de tiles XYZ/WMTS, atribuição, zoom máximo, `permiteExportacao`, chave.
- Mapa interativo com Leaflet. A carta-imagem é montada em Go a partir dos tiles, **em memória, sem cache persistente**, só se `permiteExportacao`.
- Escala calculada pela resolução Web Mercator na latitude central; norte para cima.
- Nenhum provedor pago é contratado sem aprovação de custos e termos (P1).

## Fontes
- https://doc.esri.com/en/arcgis-online/latest/reference/static-maps.html
- https://developers.google.com/maps/documentation/tile/policies
- https://docs.mapbox.com/help/print/
