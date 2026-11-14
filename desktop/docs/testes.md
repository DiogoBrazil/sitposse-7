# Testes — Central

Estratégia geral: [`docs/testing/estrategia.md`](../../docs/testing/estrategia.md). Específico da Central:

| Tipo | Ferramenta | Casos-chave |
|---|---|---|
| Unidade | `go test` (tabelas) | Área e normalização (fixtures `shared`), FiltroEstudo → SQL, tabela de decisão de conflitos |
| Banco | SQLite temporário | Migrações em sequência, PRAGMAs efetivos, CHECKs, visões vigentes |
| Importação | `go test` + fixtures `sync/` | Idempotência, rejeições, conflitos, recibo estável |
| Falhas | Subprocesso + checkpoints + proxy | Crash em cada checkpoint; queda de conexão em cada fase; disco cheio |
| Backup | `go test` | Backup → corromper → restaurar → comparar contagens e hashes |
| Estatísticas | Fixture `estatisticas/` | Todos os cruzamentos do prompt §18; benchmark < 300 ms no volume 3× |
| Frontend | Vitest | Filtros, formatação; Estatísticas sem rede |
| Cartografia | Provedor falso | Timeout, DNS, 401/403, 429, 5xx; só o mapa fica indisponível |
| Instalador | Manual (VM limpa) | Windows 10 22H2 e 11 sem internet; firewall; WebView2 |
| Cruzado com o Android | `cmd/central-teste` (Central completa, sem tela) chamado pelos testes JVM do `:core:sync` | Opções `-confirmar=false` e `-espera=1s`; comandos na entrada padrão: `token`, `estado`, `pendentes`, `aceitar`, `recusar` (o teste faz o papel do operador). Ver [incidente de 14/11](../../mobile/docs/incidentes/2026-11-14-crash-pareamento.md) |
