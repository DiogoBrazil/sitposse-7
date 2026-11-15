package br.gov.ro.pm.sitposse7.campo.inicio

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.layout.Box
import androidx.compose.material3.AlertDialog
import androidx.compose.ui.text.font.FontFamily
import br.gov.ro.pm.sitposse7.campo.CampoApp
import br.gov.ro.pm.sitposse7.campo.R
import br.gov.ro.pm.sitposse7.campo.coletas.comecouEmOutroDia
import br.gov.ro.pm.sitposse7.campo.coletas.dataLocal
import br.gov.ro.pm.sitposse7.campo.coletas.destinoDe
import br.gov.ro.pm.sitposse7.campo.coletas.textoDestino
import br.gov.ro.pm.sitposse7.dados.StatusColeta
import br.gov.ro.pm.sitposse7.dominio.Codigos
import br.gov.ro.pm.sitposse7.campo.ui.AvisoCartao
import br.gov.ro.pm.sitposse7.campo.ui.BarraSuperior
import br.gov.ro.pm.sitposse7.campo.ui.Centralizado
import br.gov.ro.pm.sitposse7.campo.ui.Largura
import br.gov.ro.pm.sitposse7.campo.ui.Navegador
import br.gov.ro.pm.sitposse7.campo.ui.Tela
import br.gov.ro.pm.sitposse7.campo.ui.dataHoraLocal
import br.gov.ro.pm.sitposse7.campo.ui.larguraAtual
import br.gov.ro.pm.sitposse7.dados.Contagens
import br.gov.ro.pm.sitposse7.dados.RascunhoEntidade
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/** [2] Início: coleta atual, números grandes, última sincronização, rascunhos e atalhos. */
@Composable
fun InicioTela(nav: Navegador, aoSemColeta: () -> Unit) {
    val app = LocalContext.current.applicationContext as CampoApp
    val escopo = rememberCoroutineScope()
    var coletaUuid by remember { mutableStateOf<String?>(null) }
    LaunchedEffect(Unit) {
        val atual = app.repo.coletaAtual()?.takeIf { app.repo.coleta(it) != null }
        if (atual == null) aoSemColeta() else coletaUuid = atual
    }
    val uuid = coletaUuid ?: run { Box(Modifier.fillMaxSize()); return }
    val coleta by app.repo.observarColeta(uuid).collectAsState(initial = null)
    val pareamento by app.banco.pareamentoDao().observar().collectAsState(initial = null)
    val c by app.repo.observarContagens(uuid).collectAsState(initial = Contagens(0, 0, 0))
    val ultima by app.repo.observarUltimaConfirmacao().collectAsState(initial = null)
    val ausentes by app.fotosAusentes.collectAsState()
    var rascunhos by remember { mutableStateOf<List<RascunhoEntidade>>(emptyList()) }
    LaunchedEffect(uuid) { rascunhos = app.repo.rascunhos(uuid) }
    val emAndamento = coleta?.status == StatusColeta.EM_ANDAMENTO
    // RN-07: aviso ao abrir o app num dia diferente do início da coleta.
    var outroDia by remember { mutableStateOf(false) }
    LaunchedEffect(coleta?.uuid) {
        val col = coleta ?: return@LaunchedEffect
        if (!app.avisoOutroDiaRespondido && col.status == StatusColeta.EM_ANDAMENTO && comecouEmOutroDia(col.iniciadaEm)) outroDia = true
    }

    Scaffold(
        topBar = {
            BarraSuperior(stringResource(R.string.app_nome)) {
                IconButton(onClick = { nav.ir(Tela.Coletas) }) { Icon(painterResource(R.drawable.ic_group), stringResource(R.string.coletas_titulo)) }
                IconButton(onClick = { nav.ir(Tela.Diagnostico) }) { Icon(painterResource(R.drawable.ic_settings_info), stringResource(R.string.diag_titulo)) }
            }
        },
        floatingActionButton = {
            if (emAndamento) ExtendedFloatingActionButton(
                onClick = { nav.ir(Tela.novoCadastro(uuid)) },
                icon = { Icon(painterResource(R.drawable.ic_add), null) },
                text = { Text(stringResource(R.string.nova_propriedade)) },
            )
        },
    ) { pad ->
        Centralizado(840, Modifier.padding(pad).fillMaxSize()) {
            Column(Modifier.verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
                Card(Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer)) {
                    Column(Modifier.padding(16.dp)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Column(Modifier.weight(1f)) {
                                Text(stringResource(R.string.coleta_codigo, Codigos.exibir(coleta?.codigoEquipe.orEmpty())), style = MaterialTheme.typography.titleLarge, fontFamily = FontFamily.Monospace)
                                coleta?.let { Text(stringResource(R.string.coleta_iniciada, dataLocal(it.iniciadaEm)), style = MaterialTheme.typography.bodyMedium) }
                            }
                            TextButton(onClick = { nav.ir(Tela.Coletas) }) { Text(stringResource(R.string.coleta_trocar)) }
                        }
                        destinoDe(coleta?.destinoJson)?.let { Text(textoDestino(it), style = MaterialTheme.typography.bodyMedium) }
                        if (!emAndamento && coleta != null) Text(stringResource(R.string.coleta_concluida_aviso), style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.SemiBold)
                        Spacer(Modifier.height(6.dp))
                        Text(
                            stringResource(if (pareamento != null) R.string.inicio_pareado else R.string.inicio_nao_pareada),
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    }
                }

                if (ausentes.isNotEmpty()) AvisoCartao(stringResource(R.string.inicio_fotos_ausentes, ausentes.size))

                rascunhos.forEach { r ->
                    val nome = remember(r.json) { runCatching { Json.parseToJsonElement(r.json).jsonObject["nome"]?.jsonPrimitive?.content }.getOrNull().orEmpty() }
                    AvisoCartao(
                        stringResource(R.string.inicio_rascunho, nome.ifBlank { stringResource(R.string.sem_nome) }),
                        cor = MaterialTheme.colorScheme.tertiaryContainer, corTexto = MaterialTheme.colorScheme.onTertiaryContainer,
                    ) {
                        TextButton(onClick = { escopo.launch { app.repo.descartarRascunho(r.uuid); rascunhos = app.repo.rascunhos(uuid) } }) { Text(stringResource(R.string.descartar)) }
                        TextButton(onClick = { nav.ir(Tela.Formulario(r.propriedadeUuid, r.uuid, r.coletaUuid)) }) { Text(stringResource(R.string.retomar)) }
                    }
                }

                val numeros = @Composable { m: Modifier ->
                    Numero(stringResource(R.string.inicio_propriedades), c.propriedades, MaterialTheme.colorScheme.secondaryContainer, m)
                    Numero(stringResource(R.string.inicio_pendentes), c.pendentes, if (c.pendentes > 0) MaterialTheme.colorScheme.tertiaryContainer else MaterialTheme.colorScheme.surfaceVariant, m)
                    Numero(stringResource(R.string.inicio_confirmadas), c.confirmadas, MaterialTheme.colorScheme.primaryContainer, m)
                }
                if (larguraAtual() == Largura.COMPACTA) {
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) { numeros(Modifier.weight(1f)) }
                } else {
                    Row(horizontalArrangement = Arrangement.spacedBy(16.dp)) { numeros(Modifier.weight(1f)) }
                }

                val atrasada = ultima?.let { System.currentTimeMillis() - it > 24 * 3600_000L } ?: (c.pendentes > 0)
                Text(
                    ultima?.let { stringResource(R.string.inicio_ultima_sinc, dataHoraLocal(it)) } ?: stringResource(R.string.inicio_nunca_sinc),
                    color = if (atrasada) Color(0xFF8D4E00) else MaterialTheme.colorScheme.onSurfaceVariant,
                    style = MaterialTheme.typography.bodyMedium,
                )

                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    OutlinedButton(onClick = { nav.ir(Tela.Lista) }, modifier = Modifier.weight(1f).height(56.dp)) {
                        Icon(painterResource(R.drawable.ic_search), null); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.inicio_ver_lista))
                    }
                    Button(onClick = { nav.ir(Tela.Sincronizar) }, modifier = Modifier.weight(1f).height(56.dp)) {
                        Icon(painterResource(R.drawable.ic_sync), null); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.sincronizar))
                    }
                }
                Spacer(Modifier.height(72.dp)) // espaço para o botão flutuante
            }
        }
    }

    if (outroDia) coleta?.let { col ->
        AlertDialog(
            onDismissRequest = {},
            title = { Text(stringResource(R.string.coleta_outro_dia_titulo)) },
            text = { Text(stringResource(R.string.coleta_outro_dia_texto, Codigos.exibir(col.codigoEquipe), dataLocal(col.iniciadaEm))) },
            confirmButton = { TextButton(onClick = { outroDia = false; app.avisoOutroDiaRespondido = true }) { Text(stringResource(R.string.coleta_continuar_nela)) } },
            dismissButton = { TextButton(onClick = { outroDia = false; app.avisoOutroDiaRespondido = true; nav.ir(Tela.Codigo) }) { Text(stringResource(R.string.codigo_outro)) } },
        )
    }
}

@Composable
private fun Numero(rotulo: String, valor: Int, cor: Color, modifier: Modifier) {
    Card(modifier, colors = CardDefaults.cardColors(containerColor = cor)) {
        Column(Modifier.fillMaxWidth().padding(vertical = 16.dp, horizontal = 8.dp), horizontalAlignment = Alignment.CenterHorizontally) {
            Text(valor.toString(), style = MaterialTheme.typography.displaySmall, fontWeight = FontWeight.Bold)
            Text(rotulo, style = MaterialTheme.typography.labelLarge, maxLines = 1)
        }
    }
}
