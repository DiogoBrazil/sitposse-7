package br.gov.ro.pm.sitposse7.sync

import okhttp3.OkHttpClient
import java.util.concurrent.RejectedExecutionException

/**
 * Libera as conexões e as threads do OkHttp **sem fazer E/S de rede na thread que chama**.
 *
 * Fechar um socket TLS envia o close_notify. No Android, o Conscrypt faz isso passando pelo
 * BlockGuard e, na thread principal, lança `NetworkOnMainThreadException`; o OkHttp repassa
 * essa RuntimeException em `closeQuietly` (só ignora exceções verificadas). Como o
 * `close()` costuma rodar no fim de um `use { }` chamado do `viewModelScope` (thread
 * principal), o app caía logo depois de o operador confirmar o pareamento
 * (mobile/docs/incidentes/2026-11-14-crash-pareamento.md). Por isso o fechamento dos sockets
 * vai para a thread do próprio OkHttp, que roda a tarefa antes de encerrar.
 */
internal fun OkHttpClient.liberarForaDaThreadAtual() {
    val executor = dispatcher.executorService
    try {
        executor.execute { runCatching { connectionPool.evictAll() } }
    } catch (_: RejectedExecutionException) {
        // já liberado antes (close() chamado de novo)
    }
    executor.shutdown()
}
