package br.gov.ro.pm.sitposse7.campo.sync

import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import androidx.lifecycle.viewmodel.compose.viewModel
import br.gov.ro.pm.sitposse7.campo.R
import br.gov.ro.pm.sitposse7.campo.diagnostico.LeitorQrCamera
import br.gov.ro.pm.sitposse7.campo.ui.AvisoCartao
import br.gov.ro.pm.sitposse7.campo.ui.BarraSuperior
import br.gov.ro.pm.sitposse7.campo.ui.Centralizado
import br.gov.ro.pm.sitposse7.campo.ui.Navegador
import br.gov.ro.pm.sitposse7.campo.ui.dataHoraLocal
import br.gov.ro.pm.sitposse7.sync.Progresso
import br.gov.ro.pm.sitposse7.sync.ResultadoColeta
import br.gov.ro.pm.sitposse7.campo.CampoApp
import br.gov.ro.pm.sitposse7.campo.coletas.DialogoCorrigirCodigo
import br.gov.ro.pm.sitposse7.campo.coletas.textoDestino
import br.gov.ro.pm.sitposse7.dominio.Codigos
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Row
import androidx.compose.material3.Checkbox
import androidx.compose.material3.TextButton
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.rememberCoroutineScope
import kotlinx.coroutines.launch

@Composable
fun SincronizarTela(nav: Navegador, vm: SincronizarViewModel = viewModel()) {
    val contexto = LocalContext.current
    val etapa by vm.etapa.collectAsState()
    val pareamento by vm.pareamento.collectAsState(initial = null)
    val paraEnviar by vm.paraEnviar.collectAsState()
    val selecionadas by vm.selecionadas.collectAsState()
    val resumos by vm.coletas.collectAsState(initial = emptyList())
    var acaoPendente by remember { mutableStateOf<(() -> Unit)?>(null) }
    var corrigir by remember { mutableStateOf<String?>(null) }
    var negado by remember { mutableStateOf<Int?>(null) }
    fun tem(p: String) = ContextCompat.checkSelfPermission(contexto, p) == PackageManager.PERMISSION_GRANTED
    val pedir = rememberLauncherForActivityResult(ActivityResultContracts.RequestMultiplePermissions()) { r ->
        if (r.values.all { it }) { negado = null; acaoPendente?.invoke() } else negado = R.string.sinc_permissoes_negadas
        acaoPendente = null
    }
    /** Pede câmera (para o QR) e rede local (Android 17+) antes de agir, com explicação na tela. */
    fun comPermissoes(camera: Boolean, acao: () -> Unit) {
        val faltam = buildList {
            if (camera && !tem(Manifest.permission.CAMERA)) add(Manifest.permission.CAMERA)
            if (SincronizarViewModel.precisaPermissaoRedeLocal && !tem(Manifest.permission.ACCESS_LOCAL_NETWORK)) add(Manifest.permission.ACCESS_LOCAL_NETWORK)
        }
        if (faltam.isEmpty()) acao() else { acaoPendente = acao; pedir.launch(faltam.toTypedArray()) }
    }

    Scaffold(topBar = { BarraSuperior(stringResource(R.string.sincronizar), aoVoltar = { nav.voltar() }) }) { pad ->
        Centralizado(640, Modifier.padding(pad).fillMaxSize()) {
            Column(Modifier.verticalScroll(rememberScrollState()).padding(20.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
                Card(Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.secondaryContainer)) {
                    Column(Modifier.padding(16.dp)) {
                        val p = pareamento
                        if (p == null) Text(stringResource(R.string.sinc_nao_pareado), style = MaterialTheme.typography.bodyLarge)
                        else {
                            Text(stringResource(R.string.sinc_pareado_com), style = MaterialTheme.typography.titleMedium)
                            Text(stringResource(R.string.sinc_pareado_em, dataHoraLocal(p.pareadoEm)), style = MaterialTheme.typography.bodyMedium)
                        }
                        Spacer(Modifier.height(8.dp))
                        Text(stringResource(R.string.sinc_explicacao), style = MaterialTheme.typography.bodyMedium)
                        if (SincronizarViewModel.precisaPermissaoRedeLocal) Text(stringResource(R.string.perm_rede_local_explicacao), style = MaterialTheme.typography.bodySmall)
                    }
                }
                negado?.let { AvisoCartao(stringResource(it)) }

                val ocioso = etapa is EtapaSinc.Ocioso || etapa is EtapaSinc.Concluido || etapa is EtapaSinc.Erro
                if (ocioso && paraEnviar.isNotEmpty()) {
                    Text(stringResource(R.string.sinc_coletas_titulo), style = MaterialTheme.typography.titleMedium)
                    paraEnviar.forEach { c ->
                        val marcada = selecionadas?.contains(c.uuid) ?: true
                        val pend = resumos.firstOrNull { it.uuid == c.uuid }?.pendentes ?: 0
                        Row(Modifier.fillMaxWidth().clickable { vm.alternar(c.uuid) }, verticalAlignment = Alignment.CenterVertically) {
                            Checkbox(marcada, { vm.alternar(c.uuid) })
                            Text(stringResource(R.string.sinc_coleta_linha, Codigos.exibir(c.codigoEquipe), pend), style = MaterialTheme.typography.bodyLarge)
                        }
                    }
                }

                when (val e = etapa) {
                    EtapaSinc.LendoQr -> {
                        Text(stringResource(R.string.diag_qr_aponte), style = MaterialTheme.typography.bodyLarge)
                        Box(Modifier.fillMaxWidth().aspectRatio(1f)) { LeitorQrCamera(aoLer = vm::qrDetectado, modifier = Modifier.fillMaxSize()) }
                        OutlinedButton(onClick = vm::cancelarQr, modifier = Modifier.fillMaxWidth()) { Text(stringResource(R.string.cancelar)) }
                    }
                    EtapaSinc.ProcurandoRede -> Andamento(stringResource(R.string.sinc_procurando))
                    EtapaSinc.AguardandoOperador -> Andamento(stringResource(R.string.sinc_aguardando_operador))
                    is EtapaSinc.Enviando -> {
                        if (e.codigo.isNotEmpty()) Text(stringResource(R.string.sinc_enviando_coleta, e.codigo, e.indice, e.total), style = MaterialTheme.typography.titleMedium)
                        when (val p = e.progresso) {
                            Progresso.Conectando -> Andamento(stringResource(R.string.sinc_conectando))
                            is Progresso.EnviandoCadastros -> Andamento(stringResource(R.string.sinc_enviando_cadastros, p.propriedades))
                            is Progresso.EnviandoFotos -> {
                                Text(stringResource(R.string.sinc_enviando_fotos, p.feitas, p.total, p.bytesEnviados / (1024 * 1024)), style = MaterialTheme.typography.titleMedium)
                                LinearProgressIndicator(progress = { if (p.total == 0) 1f else p.feitas.toFloat() / p.total }, modifier = Modifier.fillMaxWidth())
                            }
                            Progresso.ConferindoFotos -> Andamento(stringResource(R.string.sinc_conferindo_fotos))
                            Progresso.AguardandoConfirmacao -> {
                                Andamento(stringResource(R.string.sinc_aguardando_destino))
                                Text(stringResource(R.string.sinc_aguardando_destino_ajuda), style = MaterialTheme.typography.bodyMedium)
                                OutlinedButton(onClick = vm::pararDeEsperar, modifier = Modifier.fillMaxWidth()) { Text(stringResource(R.string.sinc_parar)) }
                            }
                        }
                    }
                    is EtapaSinc.Concluido -> {
                        if (e.resultados.isEmpty()) Text(stringResource(R.string.sinc_nada), style = MaterialTheme.typography.titleMedium)
                        e.resultados.forEach { (codigo, r) ->
                            when (r) {
                                is ResultadoColeta.Confirmado -> {
                                    val x = r.resumo
                                    Card(Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer)) {
                                        Column(Modifier.padding(16.dp)) {
                                            Icon(painterResource(R.drawable.ic_cloud_done), null)
                                            Text(stringResource(R.string.sinc_confirmado_destino, codigo, x.propriedadesConfirmadas, textoDestino(x.destino)), style = MaterialTheme.typography.titleMedium)
                                            if (x.destino.corrigido) Text(stringResource(R.string.coleta_destino_corrigido, Codigos.exibir(x.destino.codigoInformado)), style = MaterialTheme.typography.bodyMedium)
                                        }
                                    }
                                    if (x.rejeitados.isNotEmpty()) AvisoCartao(stringResource(R.string.sinc_rejeitados, x.rejeitados.joinToString("\n") { "• ${it.entidade}: ${it.motivo.orEmpty()}" }))
                                }
                                is ResultadoColeta.Cancelado -> AvisoCartao(stringResource(R.string.sinc_coleta_cancelada, codigo,
                                    stringResource(if (r.motivo == "codigo_inexistente") R.string.coleta_cancelada_codigo else R.string.coleta_cancelada_operador))) {
                                    TextButton(onClick = { corrigir = r.coleta }) { Text(stringResource(R.string.sinc_corrigir_codigo)) }
                                }
                                is ResultadoColeta.Aguardando -> AvisoCartao(stringResource(R.string.sinc_coleta_aguardando, codigo),
                                    cor = MaterialTheme.colorScheme.tertiaryContainer, corTexto = MaterialTheme.colorScheme.onTertiaryContainer)
                            }
                        }
                        Botoes(pareamento != null, { comPermissoes(false) { vm.sincronizar() } }, { comPermissoes(true) { vm.lerQr() } })
                    }
                    is EtapaSinc.Erro -> {
                        AvisoCartao(e.mensagem)
                        Botoes(pareamento != null, { comPermissoes(false) { vm.sincronizar() } }, { comPermissoes(true) { vm.lerQr() } })
                    }
                    EtapaSinc.Ocioso -> Botoes(pareamento != null, { comPermissoes(false) { vm.sincronizar() } }, { comPermissoes(true) { vm.lerQr() } })
                }
            }
        }
    }
    corrigir?.let { DialogoCorrecao(it, aoFechar = { corrigir = null }, vm) }
}

@Composable
private fun DialogoCorrecao(coleta: String, aoFechar: () -> Unit, vm: SincronizarViewModel) {
    val app = LocalContext.current.applicationContext as CampoApp
    val escopo = rememberCoroutineScope()
    var atual by remember { mutableStateOf<String?>(null) }
    LaunchedEffect(coleta) { atual = app.repo.coleta(coleta)?.codigoEquipe }
    val a = atual ?: return
    DialogoCorrigirCodigo(a, aoFechar) { novo, erro ->
        escopo.launch {
            runCatching { app.repo.corrigirCodigo(coleta, novo) }
                .onSuccess { aoFechar(); vm.atualizarLista() }
                .onFailure { erro(it.message ?: "") }
        }
    }
}

@Composable
private fun Andamento(texto: String) {
    Column(Modifier.fillMaxWidth().padding(vertical = 24.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(16.dp)) {
        CircularProgressIndicator()
        Text(texto, style = MaterialTheme.typography.titleMedium)
    }
}

@Composable
private fun Botoes(pareado: Boolean, aoEnviar: () -> Unit, aoLerQr: () -> Unit) {
    if (pareado) Button(onClick = aoEnviar, modifier = Modifier.fillMaxWidth().height(56.dp)) {
        Icon(painterResource(R.drawable.ic_upload), null); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.sinc_enviar))
    }
    OutlinedButton(onClick = aoLerQr, modifier = Modifier.fillMaxWidth().height(56.dp)) {
        Icon(painterResource(R.drawable.ic_qr), null); Spacer(Modifier.width(8.dp))
        Text(stringResource(if (pareado) R.string.sinc_parear_de_novo else R.string.sinc_parear))
    }
}
