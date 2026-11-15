# Testes — Central

Estratégia geral: [`docs/testing/estrategia.md`](../../docs/testing/estrategia.md). Específico da Central:

| Tipo | Ferramenta | Casos-chave |
|---|---|---|
| Unidade | `go test` (tabelas) | Área e normalização (fixtures `shared`), FiltroEstudo → SQL, tabela de decisão de conflitos |
| Banco | SQLite temporário | Migrações em sequência, PRAGMAs efetivos, CHECKs, visões vigentes |
| Importação | `go test` + fixtures `sync/` | Idempotência, rejeições, conflitos, recibo estável |
| Falhas | Subprocesso + checkpoints + proxy | Crash em cada checkpoint; queda de conexão em cada fase; disco cheio |
| Backup | `go test` | Backup → corromper → restaurar → comparar contagens e hashes |
| Estatísticas | Fixture `estatisticas/*.v2` gravada num banco migrado real (`internal/estatisticas`) | 23 casos, incluindo as 7 perguntas da §22-A.3, jornadas e outro estudo; excluídos nunca contam; coerência totais = Σ famílias = Σ equipes = Σ jornadas; filtro inválido; 900 propriedades (limite 300 ms) |
| Estudos e códigos | `internal/estudos`, `internal/codigos` | Fixture `codigos/codigos.v1.json` (a mesma do Kotlin); toda troca de um caractere detectada; colisão forçada sorteia outro; estudo/jornadas/equipes, nomes repetidos, duas jornadas na mesma data, exclusão só sem coletas |
| Destino da sincronização | `internal/nucleo/fluxo_test.go` (HTTPS real) | Aguardando antes de importar; código inexistente corrigido; código de outra equipe com destino alterado; cancelamento + reenvio com lote novo reaproveitando a foto; destino travado; decisão repetida; lote aguardando após reiniciar; coleta de outro aparelho |
| Frontend | Vitest (`npm test` em `frontend/`) | Ordenação e paginação (`tabela.test.ts`), filtros: datas locais, chips e drill-down (`filtro.test.ts`); 11 testes; Estatísticas sem rede. Revisão visual: `npm run demo` + capturas em 1366×768, 880 px e 520 px, claro e escuro |
| Cartografia | Provedor falso | Timeout, DNS, 401/403, 429, 5xx; só o mapa fica indisponível |
| Instalador | Manual (VM limpa) | Windows 10 22H2 e 11 sem internet; firewall; WebView2 |
| Cruzado com o Android | `cmd/central-teste` (Central completa, sem tela) chamado pelos testes JVM do `:core:sync` | Cria um estudo com as equipes Alfa e Bravo e informa os códigos. Opções `-confirmar=false`, `-espera=1s` e `-destino=auto|manual`; comandos na entrada padrão: `token`, `estado`, `pendentes`, `aceitar`, `recusar`, `aguardando`, `confirmar LOTE [CODIGO]`, `cancelar LOTE` (o teste faz o papel do operador). Ver [incidente de 14/11](../../mobile/docs/incidentes/2026-11-14-crash-pareamento.md) |
| Relatórios | `internal/relatorios` (`go test`) + Office real | PDF (páginas, fonte embutida, fichas), DOCX (XML bem formado, partes, brasão antes dos textos, cabeçalho repetido, divisórias), XLSX (abas, textos institucionais, tabelas, `SUBTOTAL`, números como número). Amostras: `SITPOSSE7_AMOSTRAS=<pasta> go test ./internal/relatorios -run Amostras`; validação no Word/Excel do Windows via PowerShell (COM) |
