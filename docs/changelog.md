# Changelog

Formato: data · área · descrição. Mudanças de **esquema** e **protocolo** sempre citam o ADR e a migração.

## 2026-11-13 (Fase 0) — testes adiados
- **docs** · A pedido do usuário, o teste no hotspot do celular (cenário A) e o teste em Android 17 (P7), além de Android 8/9, ficam para depois: seção "Adiados" em `planning/continuacao.md` §2; notas em `fase-0.md` e `pendencias.md` (P7). O critério da Fase 0 não muda; o fechamento depende de fazê-los ou de o usuário aprovar levá-los adiante.

## 2026-11-13 (Fase 0) — hotspot do Windows só com internet (A21, ADR-0016)
- **docs** · Decisão do usuário: o hotspot do Windows é usado **somente com internet na origem** e com o computador (PC ou notebook) **ligado**; sem internet, hotspot do celular ou pacote USB. ADR-0016 substitui a condição "hotspot do Windows condicionado ao teste" da A16; matriz com A21.
- **docs** · P8 encerrada sem teste de bancada (`pendencias.md`); spike (b) cancelado (`plano.md`, `fase-0.md`); `operacao/rede-em-campo.md` reescrito no cenário B (passo a passo com internet, tampa aberta, economia de energia desativada, queda de internet durante o uso); risco K2 reavaliado (`riscos.md`); `continuacao.md` sem o item P8; `README.md`.
- Sem mudança de código, esquema ou protocolo. A Central continua sem depender de internet em nenhuma função (A2): a internet é requisito operacional apenas do cenário B.

## 2026-11-13 (Fase 0) — segunda máquina e cartão "Banco de dados" no aparelho
- **docs** · `planning/fase-0.md`: spike (c) concluído no aparelho (moto g56, Android 16: `wal · synchronous=2 · foreign_keys=1`, SQLite 3.50.1); nova conexão pelo hotspot do Windows em TLSv1.3, 92 ms. Achado: o celular alcançou a Central pelo IP da origem (Ethernet), encaminhado pelo compartilhamento de internet do Windows. Ambiente da 2ª máquina registrado.
- **docs** · `planning/continuacao.md`: item 1 da Fase 0 concluído; P8 com o passo "Atualizar e reler o QR depois de desligar a origem"; armadilhas novas (`--no-metrics` só no binário `android`, ausência de `unzip`, chave de debug por máquina).
- Sem mudança de código, esquema ou protocolo. Bateria do §4 rodada na 2ª máquina: 22 testes Kotlin (0 pulados) e pacotes Go passando.

## 2026-11-13 (Fase 0) — passagem de bastão
- **docs** · `planning/continuacao.md`: estado, pendências da Fase 0 em ordem, preparo de máquina nova, armadilhas, comandos de teste e proteções contra regressão. Links no `AGENTS.md`, `README.md` e `docs/README.md`; READMEs de `desktop/` e `mobile/` com o estado atual.

## 2026-11-13 (Fase 0, em andamento) — Android e testes cruzados
- **mobile** · Projeto Gradle 9.9.1 (AGP 9.5.1, Kotlin 2.4.21): `:core:dominio` (área, normalização e leitura do QR com as fixtures `shared`), `:core:dados` (KMP android+jvm, Room 3.0.3 + `BundledSQLiteDriver` + `DriverDuravel`, banco v1 só com `meta`, esquema exportado em `core/dados/schemas`), `:core:sync` (OkHttp com pinning SPKI, `GET /v1/status`), `:app` (tela de diagnóstico: PRAGMAs, leitura do QR, conexão pela rede Wi‑Fi). `allowBackup=false`, `dataExtractionRules` excluindo tudo, `targetSdk 37` com `ACCESS_LOCAL_NETWORK`.
- **mobile** · Achado: o Room 3 rebaixa `synchronous` para NORMAL em WAL e deixa `foreign_keys` desligado. Corrigido pelo `DriverDuravel` (ADR-0002 mantido; ver `mobile/docs/persistencia-offline.md`).
- **protocolo** · TLS 1.2 restrito (ECDHE-ECDSA + AES-GCM/ChaCha20) aceito como mínimo, para Android 8/9; TLS 1.3 continua preferido (ADR-0014). Sem mudança de `protocolo_versao` (o formato das mensagens não muda).
- **desktop** · `syncserver` com TLS 1.2 restrito e testes (`tls_test.go`, `pin_fixture_test.go`); `cmd/central-teste` (servidor sem interface para os testes cruzados do Android).
- **shared** · Fixture `tls/pin.json` + `tls/central-fixture.crt.pem` (pin calculado com OpenSSL; Go e Kotlin conferem).
- **dependências** · ZXing core 3.5.4 no lugar do ML Kit, que traz telemetria (ADR-0015). Novas: OkHttp 5.5.1, kotlinx.serialization 1.13.0, JUnit 6.1.3 (só testes). Todas fixadas em `mobile/gradle/libs.versions.toml`.
- **protocolo** · Porta padrão da Central passa de 47807 para **38807**: a 47807 cai na faixa que a rede espelhada do WSL reserva no Windows (44620–48715). Regra de firewall do instalador e fixtures atualizadas; teste `porta_test.go`.
- **docs** · ADRs 0014 e 0015; matriz (A19, A20); pendências (P7 parcial, P10 nova; duas resolvidas); textos de transporte e stack atualizados.

## 2026-11-13 (Fase 0, em andamento)
- **desktop** · Projeto Wails v3.1.0 + Vue 3 + TS (só Windows). Núcleo Go: `dominio` (área, normalização), `banco` (migração 0001_meta; PRAGMAs verificados), `syncserver` (identidade TLS, pin SPKI, `GET /v1/status`, QR). Tela de diagnóstico. Instalador NSIS com WebView2 offline e regra de firewall.
- **shared** · `schemas/qr-pareamento.v1.schema.json`, `schemas/status.v1.schema.json`, fixture `contratos/qr-pareamento.v1.json`.
- **protocolo** · Porta padrão 47807; pin = SHA-256 do SPKI em base64url sem padding (ADR-0007).
- **docs** · Diário da Fase 0 (`planning/fase-0.md`) com versões fixadas e achados.

## 2026-11-13
- **docs** · Planejamento aprovado. Criada a estrutura do monorepositório e da documentação: requisitos, regras de negócio (fonte única), arquitetura, modelo de dados, protocolo de sincronização v1, integridade, conflitos, segurança, testes, plano, matriz de decisões, pendências, riscos e rede em campo.
- **docs** · ADRs 0001–0013 aprovados.
- **shared** · Fixtures iniciais: `area/conversao.json` (RN-44) e `normalizacao/nomes.json` (RN-35).
- Nenhum código de aplicativo implementado.
