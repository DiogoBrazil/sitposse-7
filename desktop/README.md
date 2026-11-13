# SITPOSSE-7 Central (Windows)

App desktop de recepção, consolidação, consulta, correção, estatísticas, relatórios, backup e (online) mapas e cartas-imagem.

**Estado:** esqueleto e tela de diagnóstico da Fase 0 (banco, servidor HTTPS na porta 38807, QR de conexão).

## Compilar e testar
```bash
source ~/.config/sitposse7-env.sh
go test ./internal/... ./app/...       # a raiz (main do Wails) só compila para Windows
(cd frontend && npm ci)                # numa máquina nova
wails3 build GOOS=windows              # → bin/sitposse7-central.exe
```
Detalhes e armadilhas: [`docs/planning/continuacao.md`](../docs/planning/continuacao.md).

- Regras de plataforma: [AGENTS.md](AGENTS.md)
- Arquitetura: [docs/arquitetura.md](docs/arquitetura.md)
- Regras específicas: [docs/regras.md](docs/regras.md)
- Importação e consolidação: [docs/importacao-e-consolidacao.md](docs/importacao-e-consolidacao.md)
- Telas e fluxos: [docs/telas-e-fluxos.md](docs/telas-e-fluxos.md)
- Testes: [docs/testes.md](docs/testes.md)

Requisitos e regras de negócio: [`docs/`](../docs/README.md) na raiz.
