package br.gov.ro.pm.sitposse7.campo.sync

import android.app.Application
import android.os.Build
import android.os.SystemClock
import android.util.Log
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import br.gov.ro.pm.sitposse7.campo.CampoApp
import br.gov.ro.pm.sitposse7.campo.R
import br.gov.ro.pm.sitposse7.campo.rede.RedeWifi
import br.gov.ro.pm.sitposse7.dominio.LeitorQr
import br.gov.ro.pm.sitposse7.dominio.LeituraQr
import br.gov.ro.pm.sitposse7.dominio.PayloadQr
import br.gov.ro.pm.sitposse7.sync.FalhaCentral
import br.gov.ro.pm.sitposse7.sync.Progresso
import br.gov.ro.pm.sitposse7.sync.ResumoEnvio
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import java.util.UUID

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

/**
 * Registro local (logcat) de uma tentativa, com um identificador curto para achar a mesma
 * tentativa na tela, no log e no relato da pessoa. Nunca registra token, credencial nem dados
 * de cadastro; nada sai do aparelho (AGENTS.md §6 e §8).
 */
private class Tentativa(val tipo: String) {
    val ref: String = UUID.randomUUID().toString().take(8)
    private val inicio = SystemClock.elapsedRealtime()
    private var ultima = inicio

    fun etapa(evento: String) {
        val agora = SystemClock.elapsedRealtime()
        Log.i(TAG, "[$tipo $ref] $evento (+${agora - ultima} ms, total ${agora - inicio} ms)")
        ultima = agora
    }

    fun falha(f: FalhaCentral) =
        Log.w(TAG, "[$tipo $ref] falha codigo=${f.codigo} fatal=${f.fatal} causa=${f.cause?.let { it::class.java.simpleName }}", f.cause)

    fun inesperada(e: Throwable) = Log.e(TAG, "[$tipo $ref] erro inesperado", e)

    companion object {
        const val TAG = "Sitposse7Sinc"
    }
}

/** [6] Sincronizar: pareamento pelo QR e envio com recibo (RF-A11). Tudo iniciado pela pessoa, em primeiro plano. */
class SincronizarViewModel(app: Application) : AndroidViewModel(app) {
    private val campo = app as CampoApp
    private val _etapa = MutableStateFlow<EtapaSinc>(EtapaSinc.Ocioso)
    val etapa: StateFlow<EtapaSinc> = _etapa
    val pareamento = campo.banco.pareamentoDao().observar()

    private fun texto(id: Int, vararg args: Any) = getApplication<Application>().getString(id, *args)

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

    /**
     * Roda uma tentativa com a rede Wi‑Fi do hotspot. Qualquer falha vira mensagem na tela com
     * nova tentativa: as previstas ([FalhaCentral]) com o texto delas, as demais com um texto
     * genérico e a referência do log. O app não fecha por erro de sincronização, e os cadastros
     * ficam no aparelho (só o recibo persistido os confirma).
     */
    private fun tentar(tipo: String, bloco: suspend (RedeWifi, Tentativa) -> Unit) {
        _etapa.value = EtapaSinc.ProcurandoRede
        val t = Tentativa(tipo)
        viewModelScope.launch {
            try {
                val rede = RedeWifi.obter(getApplication())
                if (rede == null) {
                    t.etapa("sem rede Wi‑Fi")
                    _etapa.value = EtapaSinc.Erro(texto(R.string.diag_sem_wifi), ofereceQr = false)
                    return@launch
                }
                t.etapa("rede Wi‑Fi obtida")
                rede.use { bloco(it, t) }
            } catch (c: CancellationException) {
                t.etapa("cancelada")
                throw c
            } catch (f: FalhaCentral) {
                t.falha(f)
                _etapa.value = EtapaSinc.Erro(f.message ?: f.codigo, ofereceQr = f.fatal)
            } catch (e: Exception) {
                t.inesperada(e)
                _etapa.value = EtapaSinc.Erro(texto(R.string.sinc_erro_inesperado, e::class.java.simpleName, t.ref), ofereceQr = true)
            }
        }
    }

    private fun parear(qr: PayloadQr) = tentar("pareamento") { rede, t ->
        t.etapa("QR lido: ${qr.ips.size} IP(s), porta ${qr.porta}")
        _etapa.value = EtapaSinc.AguardandoOperador
        campo.pareador.parear(qr, "${Build.MANUFACTURER} ${Build.MODEL}".trim(), rede.network.socketFactory)
        t.etapa("pareamento confirmado pela Central e gravado no aparelho")
        enviarCom(rede, t)
    }

    fun sincronizar() = tentar("envio") { rede, t -> enviarCom(rede, t) }

    private suspend fun enviarCom(rede: RedeWifi, t: Tentativa) {
        _etapa.value = EtapaSinc.Enviando(Progresso.Conectando)
        campo.pareador.conectar(rede.network.socketFactory).use { api ->
            t.etapa("conectado à Central")
            val r = campo.sincronizador.sincronizar(api) { p ->
                t.etapa("progresso: ${p::class.java.simpleName}")
                _etapa.value = EtapaSinc.Enviando(p)
            }
            t.etapa(if (r == null) "nada pendente" else "recibo gravado: ${r.propriedadesConfirmadas} propriedade(s), ${r.conflitos} conflito(s), ${r.rejeitados.size} rejeitado(s)")
            _etapa.value = EtapaSinc.Concluido(r)
        }
    }

    companion object {
        const val API_REDE_LOCAL = 37
        val precisaPermissaoRedeLocal get() = Build.VERSION.SDK_INT >= API_REDE_LOCAL
    }
}
