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
import br.gov.ro.pm.sitposse7.dados.ColetaEntidade
import br.gov.ro.pm.sitposse7.dominio.Codigos
import br.gov.ro.pm.sitposse7.sync.Progresso
import br.gov.ro.pm.sitposse7.sync.ResultadoColeta
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
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
    /** Enviando a coleta [codigo] ([indice] de [total]). */
    data class Enviando(val codigo: String, val indice: Int, val total: Int, val progresso: Progresso) : EtapaSinc
    /** Desfecho de cada coleta enviada (vazio = nada pendente). */
    data class Concluido(val resultados: List<Pair<String, ResultadoColeta>>) : EtapaSinc
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

/**
 * [6] Sincronizar: pareamento pelo QR e envio das coletas com recibo (RF-A11, ADR-0024). Cada
 * coleta vai num lote e espera o operador confirmar o destino na Central. Tudo iniciado pela
 * pessoa, em primeiro plano.
 */
class SincronizarViewModel(app: Application) : AndroidViewModel(app) {
    private val campo = app as CampoApp
    private val _etapa = MutableStateFlow<EtapaSinc>(EtapaSinc.Ocioso)
    val etapa: StateFlow<EtapaSinc> = _etapa
    val pareamento = campo.banco.pareamentoDao().observar()
    val coletas = campo.repo.observarColetas()
    private val _paraEnviar = MutableStateFlow<List<ColetaEntidade>>(emptyList())
    /** Coletas com algo a enviar ou com envio em aberto (para a pessoa escolher). */
    val paraEnviar: StateFlow<List<ColetaEntidade>> = _paraEnviar
    private val _selecionadas = MutableStateFlow<Set<String>?>(null)
    /** null = todas as coletas a enviar. */
    val selecionadas: StateFlow<Set<String>?> = _selecionadas
    private var trabalho: Job? = null

    init { atualizarLista() }

    fun atualizarLista() = viewModelScope.launch { _paraEnviar.value = campo.sincronizador.coletasParaEnviar() }

    fun alternar(coleta: String) {
        val atual = _selecionadas.value ?: _paraEnviar.value.map { it.uuid }.toSet()
        _selecionadas.value = if (coleta in atual) atual - coleta else atual + coleta
    }

    /** Para de esperar o operador: o lote fica "aguardando" e é reconciliado na próxima vez. */
    fun pararDeEsperar() { trabalho?.cancel() }

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
        trabalho = viewModelScope.launch {
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
                _etapa.value = EtapaSinc.Ocioso
                atualizarLista()
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
        val todas = campo.sincronizador.coletasParaEnviar()
        val escolha = _selecionadas.value
        val coletas = if (escolha == null) todas else todas.filter { it.uuid in escolha }
        _etapa.value = EtapaSinc.Enviando("", 0, coletas.size, Progresso.Conectando)
        val resultados = mutableListOf<Pair<String, ResultadoColeta>>()
        campo.pareador.conectar(rede.network.socketFactory).use { api ->
            t.etapa("conectado à Central; ${coletas.size} coleta(s)")
            coletas.forEachIndexed { i, c ->
                val codigo = Codigos.exibir(c.codigoEquipe)
                val r = campo.sincronizador.sincronizarColeta(api, c.uuid, { p ->
                    t.etapa("coleta ${i + 1}: ${p::class.java.simpleName}")
                    _etapa.value = EtapaSinc.Enviando(codigo, i + 1, coletas.size, p)
                })
                t.etapa("coleta ${i + 1}: ${r?.let { it::class.java.simpleName } ?: "nada pendente"}")
                if (r != null) resultados += codigo to r
            }
        }
        _selecionadas.value = null
        atualizarLista()
        _etapa.value = EtapaSinc.Concluido(resultados)
    }

    companion object {
        const val API_REDE_LOCAL = 37
        val precisaPermissaoRedeLocal get() = Build.VERSION.SDK_INT >= API_REDE_LOCAL
    }
}
