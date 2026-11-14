package br.gov.ro.pm.sitposse7.campo.ui

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.saveable.listSaver
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import br.gov.ro.pm.sitposse7.campo.CampoApp
import br.gov.ro.pm.sitposse7.campo.diagnostico.DiagnosticoTela
import br.gov.ro.pm.sitposse7.campo.equipe.EquipeTela
import br.gov.ro.pm.sitposse7.campo.inicio.InicioTela
import br.gov.ro.pm.sitposse7.campo.propriedades.DetalheTela
import br.gov.ro.pm.sitposse7.campo.propriedades.FormularioTela
import br.gov.ro.pm.sitposse7.campo.propriedades.ListaTela
import br.gov.ro.pm.sitposse7.campo.sync.SincronizarTela
import java.util.UUID

/** Telas do app. Codificadas em texto para sobreviver à rotação e à morte do processo. */
sealed interface Tela {
    data object Inicio : Tela
    data object Lista : Tela
    data class Detalhe(val uuid: String) : Tela
    /** [rascunho] identifica o rascunho deste formulário (RF-A10). */
    data class Formulario(val uuid: String?, val rascunho: String) : Tela
    data object Sincronizar : Tela
    data object Equipe : Tela
    data object Diagnostico : Tela

    fun codificar(): String = when (this) {
        Inicio -> "inicio"; Lista -> "lista"; Sincronizar -> "sincronizar"; Equipe -> "equipe"; Diagnostico -> "diagnostico"
        is Detalhe -> "detalhe|$uuid"
        is Formulario -> "formulario|${uuid ?: ""}|$rascunho"
    }

    companion object {
        fun decodificar(s: String): Tela {
            val p = s.split('|')
            return when (p[0]) {
                "lista" -> Lista; "sincronizar" -> Sincronizar; "equipe" -> Equipe; "diagnostico" -> Diagnostico
                "detalhe" -> Detalhe(p[1])
                "formulario" -> Formulario(p[1].ifEmpty { null }, p[2])
                else -> Inicio
            }
        }

        fun novoCadastro() = Formulario(null, UUID.randomUUID().toString())
    }
}

class Navegador(private val pilha: MutableList<Tela>) {
    val atual: Tela get() = pilha.last()
    val podeVoltar: Boolean get() = pilha.size > 1
    fun ir(t: Tela) { pilha.add(t) }
    fun voltar() { if (pilha.size > 1) pilha.removeAt(pilha.lastIndex) }
    /** Troca a tela atual (ex.: depois de salvar, o formulário vira o detalhe). */
    fun substituir(t: Tela) { pilha[pilha.lastIndex] = t }
}

@Composable
fun AppCampo() {
    val app = LocalContext.current.applicationContext as CampoApp
    val equipe by app.repo.observarEquipe().collectAsState(initial = CARREGANDO)
    val pilha = rememberSaveable(saver = listSaver(save = { l -> l.map { it.codificar() } }, restore = { l -> mutableStateListOf(*l.map(Tela::decodificar).toTypedArray()) })) {
        mutableStateListOf<Tela>(Tela.Inicio)
    }
    val nav = Navegador(pilha)
    BackHandler(enabled = nav.podeVoltar) { nav.voltar() }

    when {
        equipe === CARREGANDO -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
        equipe == null -> EquipeTela(primeiraVez = true, aoConcluir = {})
        else -> when (val t = nav.atual) {
            Tela.Inicio -> InicioTela(nav)
            Tela.Lista -> ListaTela(nav)
            is Tela.Detalhe -> DetalheTela(t.uuid, nav)
            is Tela.Formulario -> FormularioTela(t, nav)
            Tela.Sincronizar -> SincronizarTela(nav)
            Tela.Equipe -> EquipeTela(primeiraVez = false, aoConcluir = { nav.voltar() }, aoVoltar = { nav.voltar() })
            Tela.Diagnostico -> DiagnosticoTela(aoVoltar = { nav.voltar() })
        }
    }
}

private val CARREGANDO = br.gov.ro.pm.sitposse7.dados.EquipeEntidade("", "", "", null, 0, 0, 0)
