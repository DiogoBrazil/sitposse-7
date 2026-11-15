package br.gov.ro.pm.sitposse7.campo.coletas

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.Icon
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
import br.gov.ro.pm.sitposse7.campo.ui.AvisoCartao
import br.gov.ro.pm.sitposse7.campo.ui.BarraSuperior
import br.gov.ro.pm.sitposse7.campo.ui.Centralizado
import br.gov.ro.pm.sitposse7.campo.ui.Navegador
import br.gov.ro.pm.sitposse7.campo.ui.Selo
import br.gov.ro.pm.sitposse7.campo.ui.Tela
import br.gov.ro.pm.sitposse7.dados.ColetaResumo
import br.gov.ro.pm.sitposse7.dados.StatusColeta
import br.gov.ro.pm.sitposse7.dados.StatusLote
import br.gov.ro.pm.sitposse7.dominio.Codigos
import kotlinx.coroutines.launch

/** Coletas do aparelho (RF-A02): continuar, concluir, reabrir e corrigir o código (RN-07, RN-08). */
@Composable
fun ColetasTela(nav: Navegador, aoEscolher: () -> Unit) {
    val app = LocalContext.current.applicationContext as CampoApp
    val escopo = rememberCoroutineScope()
    val coletas by app.repo.observarColetas().collectAsState(initial = emptyList())
    var concluir by rememberSaveable { mutableStateOf<String?>(null) }
    var reabrir by rememberSaveable { mutableStateOf<String?>(null) }
    var corrigir by rememberSaveable { mutableStateOf<String?>(null) }
    var falha by rememberSaveable { mutableStateOf<String?>(null) }

    Scaffold(
        topBar = { BarraSuperior(stringResource(R.string.coletas_titulo), aoVoltar = { nav.voltar() }) },
        floatingActionButton = {
            ExtendedFloatingActionButton(onClick = { nav.ir(Tela.Codigo) }, icon = { Icon(painterResource(R.drawable.ic_add), null) }, text = { Text(stringResource(R.string.coletas_nova)) })
        },
    ) { pad ->
        Centralizado(760, Modifier.padding(pad).fillMaxSize()) {
            LazyColumn(Modifier.fillMaxSize(), contentPadding = androidx.compose.foundation.layout.PaddingValues(16.dp, 16.dp, 16.dp, 96.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                falha?.let { item { AvisoCartao(it) } }
                if (coletas.isEmpty()) item { Text(stringResource(R.string.coletas_vazia), style = MaterialTheme.typography.bodyLarge) }
                items(coletas, key = { it.uuid }) { c ->
                    CartaoColeta(
                        c,
                        aoContinuar = { escopo.launch { app.repo.definirColetaAtual(c.uuid); app.avisoOutroDiaRespondido = true; aoEscolher() } },
                        aoConcluir = { concluir = c.uuid },
                        aoReabrir = { reabrir = c.uuid },
                        aoCorrigir = { corrigir = c.uuid },
                        aoSincronizar = { nav.ir(Tela.Sincronizar) },
                    )
                }
            }
        }
    }

    concluir?.let { uuid ->
        val c = coletas.firstOrNull { it.uuid == uuid } ?: return@let
        AlertDialog(
            onDismissRequest = { concluir = null },
            title = { Text(stringResource(R.string.coleta_concluir)) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(stringResource(R.string.coleta_concluir_texto, Codigos.exibir(c.codigo_equipe), c.cadastros))
                    if (c.pendentes > 0) Text(stringResource(R.string.coleta_concluir_pendentes, c.pendentes), color = MaterialTheme.colorScheme.tertiary)
                }
            },
            confirmButton = { TextButton(onClick = { concluir = null; escopo.launch { app.repo.concluirColeta(uuid) } }) { Text(stringResource(R.string.coleta_concluir)) } },
            dismissButton = { TextButton(onClick = { concluir = null }) { Text(stringResource(R.string.cancelar)) } },
        )
    }
    reabrir?.let { uuid ->
        val c = coletas.firstOrNull { it.uuid == uuid } ?: return@let
        AlertDialog(
            onDismissRequest = { reabrir = null },
            title = { Text(stringResource(R.string.coleta_reabrir)) },
            text = { Text(stringResource(R.string.coleta_reabrir_texto, Codigos.exibir(c.codigo_equipe))) },
            confirmButton = { TextButton(onClick = { reabrir = null; escopo.launch { app.repo.reabrirColeta(uuid) } }) { Text(stringResource(R.string.coleta_reabrir)) } },
            dismissButton = { TextButton(onClick = { reabrir = null }) { Text(stringResource(R.string.cancelar)) } },
        )
    }
    corrigir?.let { uuid ->
        val c = coletas.firstOrNull { it.uuid == uuid } ?: return@let
        DialogoCorrigirCodigo(c.codigo_equipe, aoFechar = { corrigir = null }) { novo, erro ->
            escopo.launch {
                runCatching { app.repo.corrigirCodigo(uuid, novo) }
                    .onSuccess { corrigir = null; falha = null }
                    .onFailure { erro(it.message ?: "") }
            }
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun CartaoColeta(c: ColetaResumo, aoContinuar: () -> Unit, aoConcluir: () -> Unit, aoReabrir: () -> Unit, aoCorrigir: () -> Unit, aoSincronizar: () -> Unit) {
    val emAndamento = c.status == StatusColeta.EM_ANDAMENTO
    val destino = destinoDe(c.destino_json)
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(Codigos.exibir(c.codigo_equipe), style = MaterialTheme.typography.titleLarge, fontFamily = FontFamily.Monospace, modifier = Modifier.weight(1f))
                if (emAndamento) Selo(stringResource(R.string.coleta_em_andamento), MaterialTheme.colorScheme.primaryContainer, MaterialTheme.colorScheme.onPrimaryContainer)
                else Selo(stringResource(R.string.coleta_concluida), MaterialTheme.colorScheme.surfaceVariant, MaterialTheme.colorScheme.onSurfaceVariant)
            }
            Text(stringResource(R.string.coleta_iniciada, dataLocal(c.iniciada_em)) + (c.concluida_em?.let { " · " + stringResource(R.string.coleta_concluida_em, dataLocal(it)) } ?: ""),
                style = MaterialTheme.typography.bodyMedium)
            if (c.codigo_original != c.codigo_equipe) Text(stringResource(R.string.coleta_corrigido_de, Codigos.exibir(c.codigo_original)), style = MaterialTheme.typography.bodySmall)
            Text(stringResource(R.string.coleta_cadastros, c.cadastros, c.pendentes), style = MaterialTheme.typography.bodyMedium)
            Text(destino?.let { textoDestino(it) } ?: stringResource(R.string.coleta_sem_destino), style = MaterialTheme.typography.bodyMedium,
                color = if (destino != null) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant)
            destino?.takeIf { it.corrigido }?.let { Text(stringResource(R.string.coleta_destino_corrigido, Codigos.exibir(it.codigoInformado)), style = MaterialTheme.typography.bodySmall) }
            when (c.ultimo_lote_status) {
                StatusLote.AGUARDANDO -> Text(stringResource(R.string.coleta_aguardando), color = MaterialTheme.colorScheme.tertiary, style = MaterialTheme.typography.bodyMedium)
                StatusLote.CANCELADO -> AvisoCartao(stringResource(R.string.coleta_cancelada, c.ultimo_lote_erro.orEmpty()))
                StatusLote.ERRO_RECUPERAVEL, StatusLote.ERRO_FATAL -> c.ultimo_lote_erro?.let { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodyMedium) }
            }
            FlowRow(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp, Alignment.End)) {
                if (destino == null) TextButton(onClick = aoCorrigir) { Text(stringResource(R.string.coleta_corrigir)) }
                if (emAndamento) TextButton(onClick = aoConcluir) { Text(stringResource(R.string.coleta_concluir)) }
                else TextButton(onClick = aoReabrir) { Text(stringResource(R.string.coleta_reabrir)) }
                if (c.temPendencia) OutlinedButton(onClick = aoSincronizar) { Text(stringResource(R.string.sincronizar)) }
                if (emAndamento) Button(onClick = aoContinuar) { Text(stringResource(R.string.continuar)) }
            }
        }
    }
}

/** Correção do código de uma coleta (RN-08): mesmo campo e validação do início. */
@Composable
fun DialogoCorrigirCodigo(atual: String, aoFechar: () -> Unit, aoConfirmar: (String, (String) -> Unit) -> Unit) {
    var texto by rememberSaveable { mutableStateOf(Codigos.exibir(atual)) }
    var erro by rememberSaveable { mutableStateOf<String?>(null) }
    val confirmar = {
        val (codigo, msg) = validarCodigo(texto)
        erro = msg
        if (codigo != null) aoConfirmar(codigo) { erro = it }
    }
    AlertDialog(
        onDismissRequest = aoFechar,
        title = { Text(stringResource(R.string.coleta_corrigir)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(stringResource(R.string.coleta_codigo_atual, Codigos.exibir(atual)))
                Text(stringResource(R.string.coleta_corrigir_texto), style = MaterialTheme.typography.bodyMedium)
                CampoCodigo(texto, { texto = it; erro = null }, erro, confirmar, Modifier.fillMaxWidth())
            }
        },
        confirmButton = { TextButton(onClick = confirmar) { Text(stringResource(R.string.ok)) } },
        dismissButton = { TextButton(onClick = aoFechar) { Text(stringResource(R.string.cancelar)) } },
    )
}
