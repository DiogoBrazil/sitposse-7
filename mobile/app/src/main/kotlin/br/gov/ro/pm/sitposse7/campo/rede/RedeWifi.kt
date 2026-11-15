package br.gov.ro.pm.sitposse7.campo.rede

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.withTimeoutOrNull

/**
 * Rede Wi‑Fi para falar com a Central, mesmo sem internet (mobile/docs/arquitetura.md).
 * Sem exigir NET_CAPABILITY_INTERNET: o hotspot da Central não tem saída para a internet,
 * e o Android poderia preferir os dados móveis como rede padrão.
 */
class RedeWifi private constructor(
    private val cm: ConnectivityManager,
    private val callback: ConnectivityManager.NetworkCallback,
    val network: Network,
) : AutoCloseable {
    override fun close() {
        runCatching { cm.unregisterNetworkCallback(callback) }
    }

    companion object {
        suspend fun obter(contexto: Context, tempoMs: Long = 10_000): RedeWifi? {
            val cm = contexto.getSystemService(ConnectivityManager::class.java)
            val pedido = NetworkRequest.Builder()
                .addTransportType(NetworkCapabilities.TRANSPORT_WIFI)
                .removeCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
                .build()
            val disponivel = CompletableDeferred<Network>()
            val callback = object : ConnectivityManager.NetworkCallback() {
                override fun onAvailable(network: Network) {
                    disponivel.complete(network)
                }
            }
            cm.requestNetwork(pedido, callback)
            val rede = withTimeoutOrNull(tempoMs) { disponivel.await() }
            if (rede == null) {
                runCatching { cm.unregisterNetworkCallback(callback) }
                return null
            }
            return RedeWifi(cm, callback, rede)
        }
    }
}
