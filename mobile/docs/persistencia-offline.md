# Persistência offline e durabilidade — Android

## Banco
- Room 3 + `BundledSQLiteDriver`, arquivo no diretório privado do app.
- Na abertura: `PRAGMA journal_mode=WAL; PRAGMA synchronous=FULL; PRAGMA foreign_keys=ON;` (verificado por teste instrumentado com leitura dos PRAGMAs).
- Escrita sempre em transação (`withWriteTransaction`). O retorno com sucesso é o único gatilho de "Salvo no aparelho ✓".
- Erros (`SQLiteFullException`, I/O): mensagem "Não foi possível salvar: armazenamento cheio. Nada foi perdido do formulário." O estado do formulário permanece.
- Migrações: escritas à mão ou `AutoMigration`, testadas; **nunca destrutivas**.

## Rascunho
- Estado do formulário no `SavedStateHandle` (rotação e morte de processo) **e** na tabela `rascunhos` (gravação com intervalo de ~1 s), para interrupções longas.
- Banner: "Rascunho — ainda não salvo como cadastro". Ao abrir o app com rascunho existente: "Retomar rascunho de 'X'?".
- Rascunho é apagado só depois do commit do cadastro ou de descarte explícito.

## Fotos
1. Verificar espaço livre (mínimo de 200 MB; avisar abaixo disso).
2. CameraX `ImageCapture` → `files/fotos/tmp/{uuid}.capture.jpg`.
3. Ler a orientação EXIF; decodificar com `inSampleSize` para o lado maior ≥ 2560; escalar para ≤ 2560; JPEG 85; copiar `DateTimeOriginal`.
4. Gravar em `files/fotos/{uuid}.jpg.part`, calculando SHA‑256 em streaming; `FileOutputStream.fd.sync()`.
5. `rename` para `files/fotos/{uuid}.jpg`; apagar o arquivo de captura.
6. Transação: inserir `fotografias` (tamanho, sha256, dimensões) e incrementar `rev`.
7. **Reconciliador** (início do app): apaga `tmp/*` e `*.part` sem registro; arquivos finais sem registro ficam em quarentena; registro sem arquivo vira estado **"Arquivo ausente"** e é reportado, nunca apagado.
- Miniaturas: geradas sob demanda com amostragem (cache em memória ou em `cacheDir`, descartável).

## GNSS
- `LocationManager.requestLocationUpdates(GPS_PROVIDER)` + `GnssStatus.Callback` (satélites).
- A tela mostra a precisão atual e a melhor obtida, os satélites e o tempo. Para em ≤ 15 m ou em 90 s; opções Usar, Continuar tentando e Cancelar.
- Grava lat, lon, precisão, `gps_capturado_em` (horário da localização), satélites e `gps_origem=gnss`.
- GPS desligado: diálogo que leva às configurações de localização. Permissão negada: explicação + entrada manual.

## Backup e exportação
- **Pacote completo** `.s7pkg` gravado via SAF (`ACTION_CREATE_DOCUMENT`) em pen drive OTG, cartão ou pasta, com cópia em streaming e verificação de hash após gravar.
- A Central é o backup principal. Recomendar exportação diária quando não houver sincronização.
- `allowBackup=false`: nada vai à nuvem.

## Liberar espaço (pendência P3/P4)
Disponível só para fotos `confirmadas` cobertas por backup verificado da Central (informado em `/v1/status`). Ação explícita, com resumo "Liberar X MB de N fotos já guardadas na Central". Remove só o **arquivo**: o registro continua e passa a `arquivo_presente=0`.
