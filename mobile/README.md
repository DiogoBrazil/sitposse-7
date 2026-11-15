# SITPOSSE-7 Campo (Android)

App de coleta **100% offline**: equipe, propriedades/famílias, quantitativos, itens, área, GNSS, fotos e envio à Central (Wi‑Fi local ou pacote USB).

**Estado:** esqueleto e app de diagnóstico da Fase 0.

## Compilar e testar (WSL/Linux)
```bash
source ~/.config/sitposse7-env.sh     # JDK 21, Go, ANDROID_HOME
./gradlew test :core:dados:jvmTest :app:testDebugUnitTest   # testes JVM (inclui o teste cruzado com o servidor Go)
./gradlew :app:assembleDebug          # APK em app/build/outputs/apk/debug/
```
Preparo do ambiente, roteiro de teste no aparelho e armadilhas: [`docs/planning/continuacao.md`](../docs/planning/continuacao.md).

O teste cruzado (`ClienteCentralGoTest`) compila `desktop/cmd/central-teste` e **falha** se o Go não estiver no `PATH` (ou em `SITPOSSE7_GO`).

- Regras de plataforma: [AGENTS.md](AGENTS.md)
- Arquitetura: [docs/arquitetura.md](docs/arquitetura.md)
- Regras específicas de UI e validação: [docs/regras.md](docs/regras.md)
- Persistência e durabilidade: [docs/persistencia-offline.md](docs/persistencia-offline.md)
- Telas e fluxos: [docs/telas-e-fluxos.md](docs/telas-e-fluxos.md)
- Testes: [docs/testes.md](docs/testes.md)

Requisitos e regras de negócio: [`docs/`](../docs/README.md) na raiz.
