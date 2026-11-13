package br.gov.ro.pm.sitposse7.campo.diagnostico

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import br.gov.ro.pm.sitposse7.campo.rede.RedeWifi
import br.gov.ro.pm.sitposse7.dados.Pragmas
import br.gov.ro.pm.sitposse7.dados.abrirBancoAndroid
import br.gov.ro.pm.sitposse7.dados.pragmasEscrita
import br.gov.ro.pm.sitposse7.dados.pragmasLeitura
import br.gov.ro.pm.sitposse7.dominio.LeitorQr
import br.gov.ro.pm.sitposse7.dominio.LeituraQr
import br.gov.ro.pm.sitposse7.dominio.PayloadQr
import br.gov.ro.pm.sitposse7.sync.ClienteCentral
import br.gov.ro.pm.sitposse7.sync.ResultadoStatus
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

sealed interface EstadoBanco {
    data object Lendo : EstadoBanco
    data class Lido(val escrita: Pragmas, val leitura: Pragmas) : EstadoBanco
    data class Erro(val detalhe: String) : EstadoBanco
}

sealed interface EstadoConexao {
    data object Parado : EstadoConexao
    data object Conectando : EstadoConexao
    data object SemWifi : EstadoConexao
    data class Resultado(val ip: String, val resultado: ResultadoStatus, val ms: Long) : EstadoConexao
}

data class EstadoDiagnostico(
    val banco: EstadoBanco = EstadoBanco.Lendo,
    val lendoQr: Boolean = false,
    val qr: PayloadQr? = null,
    val qrInvalido: String? = null,
    val conexao: EstadoConexao = EstadoConexao.Parado,
)

class DiagnosticoViewModel(app: Application) : AndroidViewModel(app) {
    private val _estado = MutableStateFlow(EstadoDiagnostico())
    val estado: StateFlow<EstadoDiagnostico> = _estado.asStateFlow()

    init {
        viewModelScope.launch { lerBanco() }
    }

    private suspend fun lerBanco() {
        val r = withContext(Dispatchers.IO) {
            runCatching {
                val db = abrirBancoAndroid(getApplication())
                try {
                    EstadoBanco.Lido(db.pragmasEscrita(), db.pragmasLeitura())
                } finally {
                    db.close()
                }
            }.getOrElse { EstadoBanco.Erro("${it::class.simpleName}: ${it.message}") }
        }
        _estado.update { it.copy(banco = r) }
    }

    fun abrirLeitorQr() = _estado.update { it.copy(lendoQr = true, qrInvalido = null) }

    fun fecharLeitorQr() = _estado.update { it.copy(lendoQr = false) }

    /** Chamado pelo analisador da câmera (qualquer thread). */
    fun qrDetectado(texto: String) {
        when (val l = LeitorQr.ler(texto)) {
            is LeituraQr.Valido -> _estado.update {
                it.copy(lendoQr = false, qr = l.payload, qrInvalido = null, conexao = EstadoConexao.Parado)
            }
            is LeituraQr.Invalido -> _estado.update { it.copy(qrInvalido = l.motivo) }
        }
    }

    fun testarConexao() {
        val qr = _estado.value.qr ?: return
        if (_estado.value.conexao == EstadoConexao.Conectando) return
        _estado.update { it.copy(conexao = EstadoConexao.Conectando) }
        viewModelScope.launch {
            val r = RedeWifi.obter(getApplication())?.use { rede ->
                withContext(Dispatchers.IO) {
                    val cliente = ClienteCentral(qr.spkiSha256, rede.network.socketFactory)
                    try {
                        var ultimo: EstadoConexao = EstadoConexao.SemWifi
                        for (ip in qr.ips) {
                            val inicio = System.nanoTime()
                            val res = cliente.status(ip, qr.porta)
                            ultimo = EstadoConexao.Resultado(ip, res, (System.nanoTime() - inicio) / 1_000_000)
                            // Pin divergente é definitivo: não tentar outros IPs.
                            if (res is ResultadoStatus.Conectado || res is ResultadoStatus.PinDivergente) break
                        }
                        ultimo
                    } finally {
                        cliente.fechar()
                    }
                }
            } ?: EstadoConexao.SemWifi
            _estado.update { it.copy(conexao = r) }
        }
    }
}
