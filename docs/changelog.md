# Changelog

Formato: data · área · descrição. Mudanças de **esquema** e **protocolo** sempre citam o ADR e a migração.

## 2026-11-14 (Fase 0) — reteste do instalador na VM
- **desktop** · Instalador NSIS: o texto do `netsh` deixa de ir para o log. Redirecionado, o `netsh` escreve em UTF‑8, e o `nsExec` o mostrava com acentos quebrados com ou sem `/OEM` (a correção de 13/11 partia de um diagnóstico errado). Agora `nsExec::Exec` + `Pop $0`, e o log usa mensagens próprias conforme o código de saída; se a regra de firewall não puder ser criada, o instalador avisa na tela. Build `01040f2b…6aa8f1f`, com o mesmo `sitposse7-central.exe` (`39c266d1…`). Sem mudança de esquema ou de protocolo.
- **docs** · `planning/fase-0.md` §Reteste do instalador: achados 1–4 confirmados como corrigidos (o 1 com o certificado antigo, só com `127.0.0.1`, reaproveitado); achado 6 novo (perfil do WebView2 fica em `%APPDATA%` após desinstalar); ponto "Windows limpo" recriado à mão, após exclusão por engano; teste do zero dispensado pelo usuário. `planning/continuacao.md`: estado e próximos passos (aprovação da Fase 0 e pedido de MVP para teste em campo).

## 2026-11-13 (Fase 0) — correções do teste do instalador e ícones (Central e app)
- **desktop** · `syncserver.CarregarOuCriarIdentidade`: ao reaproveitar o certificado salvo, os IPs anunciados (tela de diagnóstico e QR) passam a ser os atuais, e não o SAN do certificado (que traz `127.0.0.1` e pode trazer IPs de redes antigas). Teste novo: `TestIdentidadeReaproveitadaAnunciaSoIPsAtuais`. Pin e certificado inalterados; sem mudança de esquema ou de protocolo.
- **desktop** · Instalador NSIS: `nsExec::ExecToLog /OEM` nas chamadas ao `netsh` (acentos corretos no log) e remoção da pasta da empresa em `Program Files` quando ela fica vazia na desinstalação (`RMDir` sem `/r`).
- **desktop** · Ícone próprio da Central: `build/appicon.svg` (fonte), `build/appicon.png` e `build/windows/icon.ico`. Renderizado com o Edge headless já presente no Windows; nenhuma dependência nova.
- **mobile** · Ícone adaptativo do app (mesmo desenho da Central): `res/drawable/ic_launcher_background.xml`, `ic_launcher_foreground.xml`, `ic_launcher_monochrome.xml` e `res/mipmap-anydpi/ic_launcher{,_round}.xml`; `android:icon`/`android:roundIcon` no manifesto. Só vetores, sem dependência nova.
- **docs** · `fase-0.md` e `continuacao.md`: achados 1 a 4 marcados como corrigidos; reteste na VM pendente; `continuacao.md` com o roteiro da próxima sessão (reteste, ícone no celular, limpeza da ISO, aprovação da Fase 0) e o comando de regerar o ícone (§4).

## 2026-11-13 (Fase 0) — instalador testado no Windows 11
- **docs** · `planning/fase-0.md`: spike (d) concluído no Windows 11 (VM Hyper-V limpa, sem rede): instalação, Central abrindo sem rede, regra de firewall (`LocalSubnet`, TCP 38807, presa ao executável), desinstalação preservando os dados e reinstalação com o mesmo ID e pin. Nova seção §Teste do instalador com o ambiente (ativação online da avaliação antes do ponto de verificação) e cinco achados não bloqueantes, entre eles o `127.0.0.1` na tela de diagnóstico e possíveis IPs antigos no QR. Windows 10 22H2 não testado.
- **docs** · `planning/continuacao.md`: item 2 da Fase 0 feito; roteiro para repetir o teste na VM; achados em "Melhorias".
- Sem mudança de código, esquema ou protocolo.

## 2026-11-13 (Fase 0) — instalador gerado e A22
- **desktop** · Instalador NSIS gerado pelo WSL (`wails3 package GOOS=windows`, `makensis` 3.09), com o WebView2 Evergreen Standalone x64 (assinatura da Microsoft conferida). Sem mudança no script; o binário do WebView2 continua fora do git. Teste em VM limpa pendente.
- **docs** · A22 (matriz): a Fase 0 pode ser concluída com os testes de hotspot do celular e de Android 17/8/9 adiados; P7 segue aberta (prazo recomendado: Fase 3). `plano.md` (critério da Fase 0), `pendencias.md`, `continuacao.md` (§2 com o roteiro da VM; §4 com o comando do instalador) e `fase-0.md`.

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
