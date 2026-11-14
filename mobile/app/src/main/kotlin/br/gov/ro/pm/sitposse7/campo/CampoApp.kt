package br.gov.ro.pm.sitposse7.campo

import android.app.Application
import br.gov.ro.pm.sitposse7.campo.fotos.ArmazemFotos
import br.gov.ro.pm.sitposse7.dados.BancoCampo
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
    val banco: BancoCampo by lazy { abrirBancoAndroid(this) }
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
