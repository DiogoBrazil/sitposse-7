package br.gov.ro.pm.sitposse7.campo.ui

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.listSaver
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import br.gov.ro.pm.sitposse7.campo.CampoApp
import br.gov.ro.pm.sitposse7.campo.R
import br.gov.ro.pm.sitposse7.campo.coletas.CodigoTela
import br.gov.ro.pm.sitposse7.campo.coletas.ColetasTela
import br.gov.ro.pm.sitposse7.campo.diagnostico.DiagnosticoTela
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
    /** [rascunho] identifica o rascunho deste formulário (RF-A10); [coleta] é a coleta dona (RN-05). */
    data class Formulario(val uuid: String?, val rascunho: String, val coleta: String) : Tela
    data object Sincronizar : Tela
    /** Informar um código de equipe (nova coleta ou continuar uma existente). */
    data object Codigo : Tela
    data object Coletas : Tela
    data object Diagnostico : Tela

    fun codificar(): String = when (this) {
        Inicio -> "inicio"; Lista -> "lista"; Sincronizar -> "sincronizar"; Codigo -> "codigo"; Coletas -> "coletas"; Diagnostico -> "diagnostico"
        is Detalhe -> "detalhe|$uuid"
        is Formulario -> "formulario|${uuid ?: ""}|$rascunho|$coleta"
    }

    companion object {
        fun decodificar(s: String): Tela {
            val p = s.split('|')
            return when (p[0]) {
                "lista" -> Lista; "sincronizar" -> Sincronizar; "codigo" -> Codigo; "coletas" -> Coletas; "diagnostico" -> Diagnostico
                "detalhe" -> Detalhe(p[1])
                "formulario" -> if (p.size >= 4) Formulario(p[1].ifEmpty { null }, p[2], p[3]) else Inicio
                else -> Inicio
            }
        }

        fun novoCadastro(coleta: String) = Formulario(null, UUID.randomUUID().toString(), coleta)
    }
}

class Navegador(private val pilha: MutableList<Tela>) {
    val atual: Tela get() = pilha.last()
    val podeVoltar: Boolean get() = pilha.size > 1
    fun ir(t: Tela) { pilha.add(t) }
    fun voltar() { if (pilha.size > 1) pilha.removeAt(pilha.lastIndex) }
    /** Troca a tela atual (ex.: depois de salvar, o formulário vira o detalhe). */
    fun substituir(t: Tela) { pilha[pilha.lastIndex] = t }
    /** Volta para o Início, sem histórico (ex.: depois de escolher a coleta). */
    fun inicio() { pilha.clear(); pilha.add(Tela.Inicio) }
}

/** Estado da escolha da coleta ao abrir o app (RN-07). */
private enum class Abertura { CARREGANDO, PEDIR_CODIGO, PRONTO }

@Composable
fun AppCampo() {
    val app = LocalContext.current.applicationContext as CampoApp
    app.bloqueio?.let { BloqueioTela(it); return }

    val pilha = rememberSaveable(saver = listSaver(save = { l -> l.map { it.codificar() } }, restore = { l -> mutableStateListOf(*l.map(Tela::decodificar).toTypedArray()) })) {
        mutableStateListOf<Tela>(Tela.Inicio)
    }
    val nav = Navegador(pilha)
    BackHandler(enabled = nav.podeVoltar) { nav.voltar() }

    // Ao abrir o app (processo novo): nenhuma coleta em andamento → pedir o código; uma → abrir
    // direto; várias → listar para escolher. Rotação e volta do segundo plano não perguntam de novo.
    var abertura by rememberSaveable { mutableStateOf(Abertura.CARREGANDO) }
    LaunchedEffect(Unit) {
        if (!app.coletaEscolhidaNestaSessao) {
            val em = app.repo.coletasEmAndamento()
            app.repo.definirColetaAtual(em.singleOrNull()?.uuid)
            app.coletaEscolhidaNestaSessao = true
            abertura = if (em.size == 1) Abertura.PRONTO else Abertura.PEDIR_CODIGO
        } else {
            abertura = if (app.repo.coletaAtual() != null) Abertura.PRONTO else Abertura.PEDIR_CODIGO
        }
    }
    val pronto = { abertura = Abertura.PRONTO; nav.inicio() }

    when (abertura) {
        Abertura.CARREGANDO -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
        Abertura.PEDIR_CODIGO -> when (nav.atual) {
            Tela.Diagnostico -> DiagnosticoTela(aoVoltar = { nav.voltar() })
            Tela.Sincronizar -> SincronizarTela(nav)
            Tela.Coletas -> ColetasTela(nav, aoEscolher = pronto)
            else -> CodigoTela(aoEscolher = pronto, aoVoltar = null, nav = nav)
        }
        Abertura.PRONTO -> when (val t = nav.atual) {
            Tela.Inicio -> InicioTela(nav, aoSemColeta = { abertura = Abertura.PEDIR_CODIGO })
            Tela.Lista -> ListaTela(nav)
            is Tela.Detalhe -> DetalheTela(t.uuid, nav)
            is Tela.Formulario -> FormularioTela(t, nav)
            Tela.Sincronizar -> SincronizarTela(nav)
            Tela.Codigo -> CodigoTela(aoEscolher = pronto, aoVoltar = { nav.voltar() }, nav = nav)
            Tela.Coletas -> ColetasTela(nav, aoEscolher = pronto)
            Tela.Diagnostico -> DiagnosticoTela(aoVoltar = { nav.voltar() })
        }
    }
}

/** Banco da geração do MVP: nada é aberto nem apagado; a pessoa reinstala o app (ADR-0025). */
@Composable
private fun BloqueioTela(mensagem: String) {
    Scaffold(topBar = { BarraSuperior(stringResource(R.string.bloqueio_titulo)) }) { pad ->
        Column(Modifier.padding(pad).padding(24.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
            AvisoCartao(mensagem)
            Text(stringResource(R.string.bloqueio_acao), style = MaterialTheme.typography.bodyLarge)
        }
    }
}
