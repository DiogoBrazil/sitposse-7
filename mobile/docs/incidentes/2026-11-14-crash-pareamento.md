# Incidente: app fecha logo depois de o operador confirmar o pareamento (14/11/2026)

**Situação:** corrigido no app `0.1.1-mvp` (versionCode 3) e **validado no aparelho em 14/11** (moto g56). Ver "Resultado no aparelho" no fim.
**Afetado:** só o app Android. A Central (Go) concluía o pareamento corretamente.
**Dados:** nenhum cadastro perdido. Os cadastros só são confirmados com recibo persistido, e nenhum lote chegou a ser enviado.

## Sintoma
Teste de bancada do MVP (moto g56, Android 16, `0.1.0-mvp`), em dois computadores (PC desktop e notebook), com o celular no hotspot do Windows:
1. O app lê o QR, e a Central mostra o pedido ao operador.
2. O operador clica em **Permitir**, e o app fecha na hora. Na repetição, o Android mostra "SITPOSSE-7 Campo apresenta falhas continuamente".
3. Ao ler o mesmo QR de novo, a Central responde "QR já usado". Isso está correto: a Central tinha concluído o pareamento e consumido o token.

No PC desktop, os reinícios do computador ocorreram na mesma hora, mas não têm relação com o app. São erros de hardware do processador (WHEA 18, Machine Check Exception) que já aconteciam desde setembro. No notebook, o app caiu sem nenhum reinício.

## Causa raiz
`ApiCentral.close()` (e `ClienteCentral.fechar()`) chamavam `connectionPool.evictAll()` **na thread que chamou**. Em `Pareador.parear`, o `api.use { }` em volta da chamada de pareamento termina no contexto do `viewModelScope`, que é a **thread principal**.

1. A conexão TLS usada no `POST /v1/pareamento` volta ao pool quando a Central responde, ou seja, quando o operador confirma.
2. `evictAll()` fecha esse socket na thread principal. O Conscrypt do Android envia o `close_notify` passando por `Platform.blockGuardOnNetwork()`, e o StrictMode padrão lança `NetworkOnMainThreadException`.
3. O OkHttp 5.5.1 **repassa** a exceção: `closeQuietly(Socket)` só ignora exceções verificadas (conferido no bytecode de `okhttp3.internal._UtilJvmKt`).
4. O `ViewModel` só tratava `FalhaCentral`, então a exceção derrubou o app **antes** de o pareamento ser gravado no celular. Na Central, o pareamento e o token consumido já estavam gravados.

O mesmo defeito derrubaria o app no fim de todo envio (`api.use { }` em `enviarCom`), depois de o recibo ter sido gravado.

Por que não apareceu antes: na Fase 0, o teste de conexão no aparelho usava o `ClienteCentral` com `fechar()` dentro de `withContext(Dispatchers.IO)`. Os testes JVM rodam sem StrictMode.

### Evidências
- **Reprodução na JVM:** o teste `FechamentoConexaoTest` usa uma fábrica de sockets que imita o StrictMode (fechar na thread principal lança `RuntimeException`). Antes da correção, falhou com a pilha `Pareador.parear` → `ApiCentral.close` → `ConnectionPool.evictAll` → `_UtilJvmKt.closeQuietly` → `Socket.close`. Isso acontece depois da resposta da Central e antes do `withWriteTransaction` que grava o pareamento.
- **Pilha equivalente já documentada** para o OkHttp embutido no Android: `NetworkOnMainThreadException` em `com.android.org.conscrypt...shutdownAndFreeSslNative` ao fechar uma conexão na thread principal ([firebase-android-sdk#6935](https://github.com/firebase/firebase-android-sdk/issues/6935)).
- **Banco da Central do PC (investigação anterior):** pareamentos gravados às 18:08:43 e às 18:10:00, nenhum lote recebido. O banco passou na verificação de integridade.
- **Falta o logcat do aparelho** para confirmar a pilha no próprio Android (ver "Validação no aparelho").

## Correção (mínima)
- `:core:sync` `Conexoes.kt`: `OkHttpClient.liberarForaDaThreadAtual()` entrega o `evictAll()` à thread do próprio OkHttp (`dispatcher.executorService`) e depois encerra o executor. `ApiCentral.close()` e `ClienteCentral.fechar()` usam essa função, e podem ser chamados em qualquer thread.
- **Defesa:** `SincronizarViewModel.tentar()` transforma **qualquer** erro inesperado (exceto cancelamento) em mensagem na tela, com a classe do erro e uma referência curta (`ref. a1b2c3d4`), e oferece nova tentativa. O app não fecha mais por falha de sincronização.
- `Sincronizador.sincronizar`: um erro não previsto deixa o lote em `erro_recuperavel`, com a causa, em vez de "enviando". O lote continua aberto e é reenviado byte a byte (idempotente).
- **Registro local (logcat, tag `Sitposse7Sinc`):** cada tentativa tem uma referência de 8 caracteres e registra as etapas com a duração: rede Wi‑Fi, QR lido (quantidade de IPs e porta), pareamento confirmado e gravado, conectado, progresso, recibo, falha (`codigo`, `fatal`, classe da causa). Não registra token, credencial nem dado de cadastro, e nada sai do aparelho.
- Sem mudança de esquema, de protocolo ou de dependência. A Central não mudou.

## Lado da Central (investigado, sem defeito)
`pareamento.Servico.Pedir` espera a decisão num canal (com `select`, prazo de 90 s e `ctx.Done()`). `efetivar` confere o token de novo **dentro** da transação, consome o token, grava a equipe e o dispositivo (hash da credencial) e a auditoria, e só depois responde `200` com a credencial. A confirmação não é só visual. O mapa de pedidos pendentes é protegido por `mutex`, e o canal tem buffer de 1 com envio não bloqueante (decisão em dobro é ignorada).

**Comportamento esperado, não é defeito:** se o celular perde a resposta depois da confirmação, o token já foi usado, e é preciso gerar um novo QR. O novo pareamento substitui a credencial do aparelho (`ON CONFLICT (instalacao_uuid)`).

## Testes
- `FechamentoConexaoTest` (2): pareamento + envio, e cliente de diagnóstico, fechando na thread "principal" simulada. **Falhavam antes da correção.**
- `PareamentoGoTest` (4), com a Central Go real **sem** confirmação automática (`central-teste -confirmar=false`, comandos `pendentes`/`aceitar`/`recusar`, `-espera`):
  - recusado (`recusado`, fatal) e depois aceito com o mesmo QR;
  - QR repetido (`token_invalido`);
  - operador não confirma a tempo (`tempo_esgotado`);
  - Central derrubada durante a espera (`rede`, recuperável);
  - Central fora do ar antes do pedido (`rede`).

  Em todos: nada gravado como pareado e o cadastro continua pendente.
- Já existentes: TLS 1.3/1.2, pin divergente e resposta inválida (`ClienteCentralGoTest`); queda de rede, reenvio e ACK perdido (`SincronizacaoGoTest`).
- Total Android: **48** (eram 42). Go: `go vet` e `go test ./internal/... ./app/... ./cmd/...` OK.

**Não cobertos por teste automático:**
- celular desconectado **depois** da aprovação (exige cortar a rede no meio da resposta);
- resposta HTTP ou JSON inesperada no pareamento (exige um servidor falso com o certificado pinado);
- nova conexão depois de reiniciar a Central (o `central-teste` usa porta aleatória).

Pelo código, os três caem em `FalhaCentral` ou na nova defesa do `ViewModel`.

## Validação no aparelho
1. Instalar `Downloads\sitposse7-teste\sitposse7-campo-debug.apk` (`f4a7fd54…`, `0.1.1-mvp`) **por cima** do atual. A assinatura é a mesma (`ab61d27f…`), então os cadastros ficam. **Não desinstale.**
2. Em Informações do app, conferir a versão `0.1.1-mvp`.
3. Celular no hotspot do computador da Central → Sincronizar → Ler QR de pareamento → **Permitir** na Central.
4. Esperado: "Conectando…" → "Enviando cadastros (2)…" → resultado com 2 confirmados. Na Central, o lote aparece em Sincronização, e as propriedades no painel.
5. Se aparecer uma mensagem de erro, anotar o texto e a `ref.`. Com a depuração por Wi‑Fi, o log sai com `adb logcat -s Sitposse7Sinc:V AndroidRuntime:E`. O crash antigo fica em `adb shell dumpsys dropbox --print data_app_crash`, que sobrevive a reinícios.

## Resultado no aparelho (14/11, 19:21–19:29)
APK `0.1.1-mvp` instalado por cima, sem perder dados.
- **Notebook como Central** (operação "Levantamento C-25"): pareou depois do "Permitir", sem fechar o app, e enviou os 2 cadastros antigos. Resultado no app: "2 cadastro(s) confirmado(s) · 0 em análise · 0 erro(s)".
- **PC desktop como Central** (outra operação, "Levantamento LC-25"): um 3º cadastro, novo pareamento e envio OK. Painel da Central: 1 propriedade, 1 lote importado às 19:28, 1 equipe com 1 aparelho. No app: 3 propriedades, 0 pendentes, 3 confirmadas.
- **Duas Centrais, dados divididos:** cada Central ficou só com o que recebeu (2 no notebook, 1 no PC), porque o celular não reenvia o que já tem recibo. É o comportamento atual, mas o que fazer ao parear com **outra** Central não estava decidido: ver P12 em `docs/planning/pendencias.md`.
- O logcat não foi colhido. A causa ficou confirmada pela reprodução na JVM e pelo fim do crash no aparelho com a correção.
