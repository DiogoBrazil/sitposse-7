package br.gov.ro.pm.sitposse7.campo.sync

import android.app.Application
import android.os.Build
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import br.gov.ro.pm.sitposse7.campo.CampoApp
import br.gov.ro.pm.sitposse7.campo.rede.RedeWifi
import br.gov.ro.pm.sitposse7.dominio.LeitorQr
import br.gov.ro.pm.sitposse7.dominio.LeituraQr
import br.gov.ro.pm.sitposse7.dominio.PayloadQr
import br.gov.ro.pm.sitposse7.sync.FalhaCentral
import br.gov.ro.pm.sitposse7.sync.Progresso
import br.gov.ro.pm.sitposse7.sync.ResumoEnvio
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch

sealed interface EtapaSinc {
    data object Ocioso : EtapaSinc
    data object LendoQr : EtapaSinc
    data object ProcurandoRede : EtapaSinc
    /** O pedido foi enviado; a Central espera o operador confirmar. */
    data object AguardandoOperador : EtapaSinc
    data class Enviando(val progresso: Progresso) : EtapaSinc
    data class Concluido(val resumo: ResumoEnvio?) : EtapaSinc
    data class Erro(val mensagem: String, val ofereceQr: Boolean) : EtapaSinc
}

/** [6] Sincronizar: pareamento pelo QR e envio com recibo (RF-A11). Tudo iniciado pela pessoa, em primeiro plano. */
class SincronizarViewModel(app: Application) : AndroidViewModel(app) {
    private val campo = app as CampoApp
    private val _etapa = MutableStateFlow<EtapaSinc>(EtapaSinc.Ocioso)
    val etapa: StateFlow<EtapaSinc> = _etapa
    val pareamento = campo.banco.pareamentoDao().observar()

    private fun semWifi() = getApplication<Application>().getString(br.gov.ro.pm.sitposse7.campo.R.string.diag_sem_wifi)

    fun lerQr() { _etapa.value = EtapaSinc.LendoQr }

    fun cancelarQr() { _etapa.value = EtapaSinc.Ocioso }

    /** Chamado pelo analisador da câmera (qualquer thread). */
    fun qrDetectado(texto: String) {
        if (_etapa.value != EtapaSinc.LendoQr) return
        when (val l = LeitorQr.ler(texto)) {
            is LeituraQr.Valido -> parear(l.payload)
            is LeituraQr.Invalido -> Unit // continua lendo
        }
    }

    private fun parear(qr: PayloadQr) {
        _etapa.value = EtapaSinc.ProcurandoRede
        viewModelScope.launch {
            val rede = RedeWifi.obter(getApplication())
            if (rede == null) { _etapa.value = EtapaSinc.Erro(semWifi(), ofereceQr = false); return@launch }
            rede.use {
                try {
                    _etapa.value = EtapaSinc.AguardandoOperador
                    campo.pareador.parear(qr, "${Build.MANUFACTURER} ${Build.MODEL}".trim(), it.network.socketFactory)
                    enviarCom(it)
                } catch (f: FalhaCentral) {
                    _etapa.value = EtapaSinc.Erro(f.message ?: f.codigo, ofereceQr = true)
                }
            }
        }
    }

    fun sincronizar() {
        _etapa.value = EtapaSinc.ProcurandoRede
        viewModelScope.launch {
            val rede = RedeWifi.obter(getApplication())
            if (rede == null) { _etapa.value = EtapaSinc.Erro(semWifi(), ofereceQr = false); return@launch }
            rede.use { enviarCom(it) }
        }
    }

    private suspend fun enviarCom(rede: RedeWifi) {
        try {
            val api = campo.pareador.conectar(rede.network.socketFactory)
            api.use {
                _etapa.value = EtapaSinc.Enviando(Progresso.Conectando)
                val r = campo.sincronizador.sincronizar(it) { p -> _etapa.value = EtapaSinc.Enviando(p) }
                _etapa.value = EtapaSinc.Concluido(r)
            }
        } catch (f: FalhaCentral) {
            _etapa.value = EtapaSinc.Erro(f.message ?: f.codigo, ofereceQr = f.fatal)
        }
    }

    companion object {
        const val API_REDE_LOCAL = 37
        val precisaPermissaoRedeLocal get() = Build.VERSION.SDK_INT >= API_REDE_LOCAL
    }
}
