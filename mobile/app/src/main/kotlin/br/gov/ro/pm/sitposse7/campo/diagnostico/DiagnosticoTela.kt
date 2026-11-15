package br.gov.ro.pm.sitposse7.campo.diagnostico

import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import br.gov.ro.pm.sitposse7.campo.R
import br.gov.ro.pm.sitposse7.dados.Pragmas
import br.gov.ro.pm.sitposse7.sync.ResultadoStatus

private const val API_REDE_LOCAL = 37 // Android 17

@Composable
fun DiagnosticoTela(aoVoltar: () -> Unit = {}, vm: DiagnosticoViewModel = viewModel()) {
    val estado by vm.estado.collectAsStateWithLifecycle()
    val contexto = LocalContext.current
    var aviso by remember { mutableStateOf<Int?>(null) }

    val pedirCamera = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { ok ->
        if (ok) vm.abrirLeitorQr() else aviso = R.string.perm_camera_negada
    }
    val pedirRedeLocal = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { ok ->
        if (ok) vm.testarConexao() else aviso = R.string.perm_rede_local_negada
    }
    fun temPermissao(p: String) = ContextCompat.checkSelfPermission(contexto, p) == PackageManager.PERMISSION_GRANTED

    Scaffold(topBar = { br.gov.ro.pm.sitposse7.campo.ui.BarraSuperior(stringResource(R.string.diag_titulo), aoVoltar) }) { pad ->
        Column(
            Modifier.fillMaxSize().padding(pad).padding(16.dp).verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(stringResource(R.string.diag_titulo), style = MaterialTheme.typography.headlineSmall)
            Text(stringResource(R.string.diag_aparelho, Build.VERSION.RELEASE, Build.VERSION.SDK_INT))

            Secao(stringResource(R.string.diag_banco_titulo)) { Banco(estado.banco) }

            Secao(stringResource(R.string.diag_central_titulo)) {
                if (estado.lendoQr) {
                    Text(stringResource(R.string.diag_qr_aponte))
                    LeitorQrCamera(vm::qrDetectado, Modifier.fillMaxWidth().aspectRatio(1f))
                    estado.qrInvalido?.let { Text(stringResource(R.string.diag_qr_invalido, it), color = MaterialTheme.colorScheme.error) }
                    OutlinedButton(onClick = vm::fecharLeitorQr) { Text(stringResource(R.string.diag_cancelar)) }
                } else {
                    estado.qr?.let { qr ->
                        val ips = qr.ips.joinToString().ifEmpty { stringResource(R.string.diag_qr_sem_ips) }
                        Text(stringResource(R.string.diag_qr_lido, qr.centralId.take(8), ips, qr.porta))
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        OutlinedButton(onClick = {
                            if (temPermissao(Manifest.permission.CAMERA)) vm.abrirLeitorQr() else pedirCamera.launch(Manifest.permission.CAMERA)
                        }) { Text(stringResource(R.string.diag_ler_qr)) }
                        Button(
                            enabled = estado.qr != null && estado.conexao != EstadoConexao.Conectando,
                            onClick = {
                                aviso = null
                                if (Build.VERSION.SDK_INT >= API_REDE_LOCAL && !temPermissao(Manifest.permission.ACCESS_LOCAL_NETWORK)) {
                                    aviso = R.string.perm_rede_local_explicacao
                                    pedirRedeLocal.launch(Manifest.permission.ACCESS_LOCAL_NETWORK)
                                } else {
                                    vm.testarConexao()
                                }
                            },
                        ) { Text(stringResource(R.string.diag_testar)) }
                    }
                    Conexao(estado.conexao)
                }
                aviso?.let { Text(stringResource(it)) }
            }
        }
    }
}

@Composable
private fun Secao(titulo: String, conteudo: @Composable () -> Unit) {
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(titulo, style = MaterialTheme.typography.titleMedium)
            conteudo()
        }
    }
}

@Composable
private fun Banco(b: EstadoBanco) {
    when (b) {
        EstadoBanco.Lendo -> Text(stringResource(R.string.diag_banco_lendo))
        is EstadoBanco.Erro -> Text(stringResource(R.string.diag_banco_erro, b.detalhe), color = MaterialTheme.colorScheme.error)
        is EstadoBanco.Lido -> {
            LinhaPragmas(R.string.diag_banco_escrita, b.escrita)
            LinhaPragmas(R.string.diag_banco_leitura, b.leitura)
            Text(stringResource(R.string.diag_banco_versao, b.escrita.sqliteVersao))
            val ok = b.escrita.conforme && b.leitura.conforme
            Text(
                stringResource(if (ok) R.string.diag_banco_ok else R.string.diag_banco_falha),
                color = if (ok) Color(0xFF1B5E20) else MaterialTheme.colorScheme.error,
            )
        }
    }
}

@Composable
private fun LinhaPragmas(rotulo: Int, p: Pragmas) =
    Text(stringResource(rotulo, p.journalMode, p.synchronous, p.foreignKeys), style = MaterialTheme.typography.bodySmall)

@Composable
private fun Conexao(c: EstadoConexao) {
    when (c) {
        EstadoConexao.Parado -> Unit
        EstadoConexao.Conectando -> Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            CircularProgressIndicator(Modifier.padding(2.dp))
            Text(stringResource(R.string.diag_conectando))
        }
        EstadoConexao.SemWifi -> Text(stringResource(R.string.diag_sem_wifi), color = MaterialTheme.colorScheme.error)
        is EstadoConexao.Resultado -> when (val r = c.resultado) {
            is ResultadoStatus.Conectado -> Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(stringResource(R.string.diag_conectado, c.ip, r.versaoTls, c.ms), color = Color(0xFF1B5E20))
                Text(stringResource(R.string.diag_conectado_horario, r.status.horario))
                if (!r.compativel) Text(stringResource(R.string.diag_incompativel), color = MaterialTheme.colorScheme.error)
            }
            ResultadoStatus.PinDivergente -> Text(stringResource(R.string.diag_pin_divergente), color = MaterialTheme.colorScheme.error)
            is ResultadoStatus.SemConexao -> Text(stringResource(R.string.diag_sem_conexao, c.ip, r.detalhe), color = MaterialTheme.colorScheme.error)
            is ResultadoStatus.RespostaInvalida -> Text(stringResource(R.string.diag_resposta_invalida, c.ip, r.detalhe), color = MaterialTheme.colorScheme.error)
        }
    }
}
