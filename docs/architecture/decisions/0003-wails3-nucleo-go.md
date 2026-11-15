# ADR-0003 — Wails v3 com núcleo Go independente

- **Status:** Aprovado (validar no spike da Fase 0)
- **Data:** 2026-11-13

## Contexto
A Central é um app desktop Windows em Go + Vue 3 + TS. O Wails v3.0.0 virou estável em 26/10/2026, e a v3.1.0 (01/11/2026) é a "Latest". Por ser muito recente, há risco de regressões.

## Decisão
- Usar **Wails v3.1.x** como casca de UI.
- **Todo o núcleo** (domínio, banco, servidor de sync, importação, estatísticas, backup, relatórios, cartografia) fica em pacotes Go **sem importar Wails**, testáveis com `go test`.
- Serviços expostos ao frontend por *bindings* finos.
- Distribuição por instalador NSIS com WebView2 embutido (instalação offline) e regra de firewall.

## Alternativas consideradas
- **Wails v2.17**: maduro; fica como **plano B**. Pela arquitetura acima, a troca afeta só a casca.
- **Tauri (Rust)**: foge da stack Go sugerida.
- **Electron**: pesado.
- **Fyne/Gio (UI nativa Go)**: ecossistema de tabelas, gráficos e mapas mais pobre.

## Consequências
- O spike da Fase 0 precisa validar no Windows 10 22H2 e 11: build, bindings, instalador, WebView2 offline e o servidor HTTPS no mesmo processo.
- As versões exatas são fixadas em `desktop/go.mod` e `package.json`.

## Fontes
- https://wails.io/blog/wails-v3-released/
- https://github.com/wailsapp/wails/releases
- https://wails.io/docs/guides/windows
