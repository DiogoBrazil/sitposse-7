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

@Composable
fun SincronizarTela(nav: Navegador, vm: SincronizarViewModel = viewModel()) {
    val contexto = LocalContext.current
    val etapa by vm.etapa.collectAsState()
    val pareamento by vm.pareamento.collectAsState(initial = null)
    var acaoPendente by remember { mutableStateOf<(() -> Unit)?>(null) }
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
                            Text(stringResource(R.string.sinc_pareado_com, p.operacaoNome), style = MaterialTheme.typography.titleMedium)
                            Text(stringResource(R.string.sinc_pareado_em, dataHoraLocal(p.pareadoEm)), style = MaterialTheme.typography.bodyMedium)
                        }
                        Spacer(Modifier.height(8.dp))
                        Text(stringResource(R.string.sinc_explicacao), style = MaterialTheme.typography.bodyMedium)
                        if (SincronizarViewModel.precisaPermissaoRedeLocal) Text(stringResource(R.string.perm_rede_local_explicacao), style = MaterialTheme.typography.bodySmall)
                    }
                }
                negado?.let { AvisoCartao(stringResource(it)) }

                when (val e = etapa) {
                    EtapaSinc.LendoQr -> {
                        Text(stringResource(R.string.diag_qr_aponte), style = MaterialTheme.typography.bodyLarge)
                        Box(Modifier.fillMaxWidth().aspectRatio(1f)) { LeitorQrCamera(aoLer = vm::qrDetectado, modifier = Modifier.fillMaxSize()) }
                        OutlinedButton(onClick = vm::cancelarQr, modifier = Modifier.fillMaxWidth()) { Text(stringResource(R.string.cancelar)) }
                    }
                    EtapaSinc.ProcurandoRede -> Andamento(stringResource(R.string.sinc_procurando))
                    EtapaSinc.AguardandoOperador -> Andamento(stringResource(R.string.sinc_aguardando_operador))
                    is EtapaSinc.Enviando -> when (val p = e.progresso) {
                        Progresso.Conectando -> Andamento(stringResource(R.string.sinc_conectando))
                        is Progresso.EnviandoCadastros -> Andamento(stringResource(R.string.sinc_enviando_cadastros, p.propriedades))
                        is Progresso.EnviandoFotos -> {
                            Text(stringResource(R.string.sinc_enviando_fotos, p.feitas, p.total, p.bytesEnviados / (1024 * 1024)), style = MaterialTheme.typography.titleMedium)
                            LinearProgressIndicator(progress = { if (p.total == 0) 1f else p.feitas.toFloat() / p.total }, modifier = Modifier.fillMaxWidth())
                        }
                        Progresso.AguardandoImportacao -> Andamento(stringResource(R.string.sinc_aguardando_importacao))
                    }
                    is EtapaSinc.Concluido -> {
                        val r = e.resumo
                        if (r == null) Text(stringResource(R.string.sinc_nada), style = MaterialTheme.typography.titleMedium)
                        else {
                            Card(Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer)) {
                                Column(Modifier.padding(16.dp)) {
                                    Icon(painterResource(R.drawable.ic_cloud_done), null)
                                    Text(stringResource(R.string.sinc_resultado, r.propriedadesConfirmadas, r.conflitos, r.rejeitados.size), style = MaterialTheme.typography.titleMedium)
                                }
                            }
                            if (r.rejeitados.isNotEmpty()) AvisoCartao(stringResource(R.string.sinc_rejeitados, r.rejeitados.joinToString("\n") { "• ${it.entidade}: ${it.motivo.orEmpty()}" }))
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
