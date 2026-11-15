package br.gov.ro.pm.sitposse7.campo.coletas

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import br.gov.ro.pm.sitposse7.campo.CampoApp
import br.gov.ro.pm.sitposse7.campo.R
import br.gov.ro.pm.sitposse7.campo.ui.BarraSuperior
import br.gov.ro.pm.sitposse7.campo.ui.Centralizado
import br.gov.ro.pm.sitposse7.campo.ui.Navegador
import br.gov.ro.pm.sitposse7.campo.ui.Tela
import br.gov.ro.pm.sitposse7.dados.ColetaEntidade
import br.gov.ro.pm.sitposse7.dados.ColetaResumo
import br.gov.ro.pm.sitposse7.dados.InicioColeta
import br.gov.ro.pm.sitposse7.dados.StatusColeta
import br.gov.ro.pm.sitposse7.dominio.Codigos
import kotlinx.coroutines.launch

/**
 * Código da equipe (RF-A01, RN-05, RN-07): fluxo principal ao abrir sem coleta em andamento e
 * ao começar outra jornada. Lista as coletas em andamento para continuar com um toque.
 */
@Composable
fun CodigoTela(aoEscolher: () -> Unit, aoVoltar: (() -> Unit)?, nav: Navegador) {
    val app = LocalContext.current.applicationContext as CampoApp
    val escopo = rememberCoroutineScope()
    val coletas by app.repo.observarColetas().collectAsState(initial = emptyList())
    val emAndamento = coletas.filter { it.status == StatusColeta.EM_ANDAMENTO }
    var texto by rememberSaveable { mutableStateOf("") }
    var erro by rememberSaveable { mutableStateOf<String?>(null) }
    var reabrir by rememberSaveable { mutableStateOf<String?>(null) } // uuid da coleta concluída com o mesmo código
    var concluirAnteriores by rememberSaveable { mutableStateOf<String?>(null) } // uuid da coleta nova

    fun escolher(uuid: String) = escopo.launch {
        app.repo.definirColetaAtual(uuid)
        app.avisoOutroDiaRespondido = true // escolheu agora: não perguntar de novo
        aoEscolher()
    }

    fun comecar() {
        val (codigo, msg) = validarCodigo(texto)
        erro = msg
        if (codigo == null) return
        escopo.launch {
            runCatching { app.repo.iniciarColeta(codigo) }
                .onFailure { erro = it.message }
                .onSuccess { r ->
                    when {
                        r is InicioColeta.Existente && r.coleta.status == StatusColeta.CONCLUIDA -> reabrir = r.coleta.uuid
                        r is InicioColeta.Existente -> escolher(r.coleta.uuid)
                        app.repo.coletasEmAndamento().any { it.uuid != r.coleta.uuid } -> concluirAnteriores = r.coleta.uuid
                        else -> escolher(r.coleta.uuid)
                    }
                }
        }
    }

    Scaffold(topBar = {
        BarraSuperior(stringResource(R.string.codigo_titulo), aoVoltar) {
            IconButton(onClick = { nav.ir(Tela.Coletas) }) { Icon(painterResource(R.drawable.ic_group), stringResource(R.string.coletas_titulo)) }
            IconButton(onClick = { nav.ir(Tela.Diagnostico) }) { Icon(painterResource(R.drawable.ic_settings_info), stringResource(R.string.diag_titulo)) }
        }
    }) { pad ->
        Centralizado(560, Modifier.padding(pad).fillMaxSize()) {
            Column(Modifier.verticalScroll(rememberScrollState()).imePadding().padding(20.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
                if (emAndamento.size > 1 && aoVoltar == null) {
                    Text(stringResource(R.string.codigo_escolher), style = MaterialTheme.typography.titleLarge)
                    emAndamento.forEach { c -> CartaoColetaCurto(c) { escolher(c.uuid) } }
                    Spacer(Modifier.height(8.dp))
                    Text(stringResource(R.string.codigo_outro), style = MaterialTheme.typography.titleMedium)
                }
                Text(stringResource(R.string.codigo_explicacao), style = MaterialTheme.typography.bodyLarge)
                CampoCodigo(texto, { texto = it; erro = null }, erro, ::comecar, Modifier.fillMaxWidth())
                Button(onClick = ::comecar, enabled = texto.isNotBlank(), modifier = Modifier.fillMaxWidth().height(56.dp)) {
                    Text(stringResource(R.string.codigo_comecar))
                }
                if (emAndamento.isNotEmpty() && !(emAndamento.size > 1 && aoVoltar == null)) {
                    Spacer(Modifier.height(8.dp))
                    Text(stringResource(R.string.codigo_em_andamento), style = MaterialTheme.typography.titleMedium)
                    emAndamento.forEach { c -> CartaoColetaCurto(c) { escolher(c.uuid) } }
                }
                if (coletas.any { it.status == StatusColeta.CONCLUIDA && it.temPendencia }) {
                    OutlinedButton(onClick = { nav.ir(Tela.Sincronizar) }, modifier = Modifier.fillMaxWidth()) { Text(stringResource(R.string.sincronizar)) }
                }
            }
        }
    }

    reabrir?.let { uuid ->
        val c = coletas.firstOrNull { it.uuid == uuid }
        AlertDialog(
            onDismissRequest = { reabrir = null },
            title = { Text(stringResource(R.string.coleta_reabrir)) },
            text = { Text(stringResource(R.string.coleta_ja_existe_concluida, Codigos.exibir(c?.codigo_equipe.orEmpty()), c?.concluida_em?.let(::dataLocal).orEmpty())) },
            confirmButton = { TextButton(onClick = { reabrir = null; escopo.launch { app.repo.reabrirColeta(uuid); escolher(uuid) } }) { Text(stringResource(R.string.coleta_reabrir)) } },
            dismissButton = { TextButton(onClick = { reabrir = null }) { Text(stringResource(R.string.cancelar)) } },
        )
    }
    concluirAnteriores?.let { nova ->
        val outras = emAndamento.filter { it.uuid != nova }
        AlertDialog(
            onDismissRequest = { concluirAnteriores = null; escolher(nova) },
            title = { Text(stringResource(R.string.coleta_concluir_anteriores)) },
            text = { Text(stringResource(R.string.coleta_concluir_anteriores_texto, outras.joinToString { Codigos.exibir(it.codigo_equipe) })) },
            confirmButton = {
                TextButton(onClick = {
                    concluirAnteriores = null
                    escopo.launch { outras.forEach { app.repo.concluirColeta(it.uuid) }; escolher(nova) }
                }) { Text(stringResource(R.string.coleta_concluir)) }
            },
            dismissButton = { TextButton(onClick = { concluirAnteriores = null; escolher(nova) }) { Text(stringResource(R.string.coleta_manter)) } },
        )
    }
}

@Composable
private fun CartaoColetaCurto(c: ColetaResumo, aoContinuar: () -> Unit) {
    Card(Modifier.fillMaxWidth()) {
        Row(Modifier.padding(16.dp), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(Codigos.exibir(c.codigo_equipe), style = MaterialTheme.typography.titleMedium, fontFamily = FontFamily.Monospace)
                Text(stringResource(R.string.coleta_iniciada, dataLocal(c.iniciada_em)), style = MaterialTheme.typography.bodyMedium)
                Text(stringResource(R.string.coleta_cadastros, c.cadastros, c.pendentes), style = MaterialTheme.typography.bodyMedium)
                destinoDe(c.destino_json)?.let { Text(textoDestino(it), style = MaterialTheme.typography.bodySmall) }
            }
            Button(onClick = aoContinuar) { Text(stringResource(R.string.continuar)) }
        }
    }
}

/** Coleta atual (para o Início e o formulário). */
suspend fun coletaAtual(app: CampoApp): ColetaEntidade? = app.repo.coletaAtual()?.let { app.repo.coleta(it) }
