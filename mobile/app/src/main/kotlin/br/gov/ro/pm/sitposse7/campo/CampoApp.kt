package br.gov.ro.pm.sitposse7.campo

import android.app.Application
import br.gov.ro.pm.sitposse7.campo.fotos.ArmazemFotos
import br.gov.ro.pm.sitposse7.dados.BancoCampo
import br.gov.ro.pm.sitposse7.dados.BancoGeracaoAnterior
import br.gov.ro.pm.sitposse7.dados.RepositorioCampo
import br.gov.ro.pm.sitposse7.dados.abrirBancoAndroid
import br.gov.ro.pm.sitposse7.sync.Pareador
import br.gov.ro.pm.sitposse7.sync.Sincronizador
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import java.io.File

/** Dependências do app (injeção manual, mobile/docs/arquitetura.md). */
class CampoApp : Application() {
    private val abertura: Result<BancoCampo> by lazy {
        runCatching { abrirBancoAndroid(this) }.onFailure { if (it !is BancoGeracaoAnterior) throw it }
    }

    /** Banco de uma geração anterior (MVP): o app só mostra a tela de bloqueio (ADR-0025). */
    val bloqueio: String? get() = abertura.exceptionOrNull()?.message

    val banco: BancoCampo get() = abertura.getOrThrow()

    /** A coleta já foi escolhida neste processo (RN-07: a escolha é refeita só ao abrir o app). */
    @Volatile var coletaEscolhidaNestaSessao = false

    /** O aviso de "coleta de outro dia" já foi respondido neste processo. */
    @Volatile var avisoOutroDiaRespondido = false
    val fotos: ArmazemFotos by lazy { ArmazemFotos(File(filesDir, "fotos")) }
    private val relogio = { System.currentTimeMillis() }
    val repo: RepositorioCampo by lazy { RepositorioCampo(banco, relogio) { fotos.existe(it) } }
    val sincronizador: Sincronizador by lazy { Sincronizador(banco, relogio) { fotos.arquivo(it) } }
    val pareador: Pareador by lazy { Pareador(banco, relogio) }
    val escopo = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    private val _fotosAusentes = MutableStateFlow<List<String>>(emptyList())
    /** Registros de foto sem arquivo (defeito grave, mostrado na tela inicial). */
    val fotosAusentes: StateFlow<List<String>> = _fotosAusentes

    override fun onCreate() {
        super.onCreate()
        escopo.launch {
            if (bloqueio != null) return@launch
            runCatching {
                val registradas = repo.fotosRegistradas()
                val rascunhos = repo.rascunhos().flatMap { r -> Regex("\"caminhoRelativo\":\"([^\"]+)\"").findAll(r.json).map { it.groupValues[1] } }
                val referenciados = registradas.map { it.caminhoRelativo }.toSet() + rascunhos
                val vigentes = registradas.filter { it.excluidoEm == null }.associate { it.uuid to it.caminhoRelativo }
                _fotosAusentes.value = fotos.reconciliar(referenciados, vigentes)
            }
        }
    }
}
