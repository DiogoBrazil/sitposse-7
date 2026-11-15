package br.gov.ro.pm.sitposse7.campo.propriedades

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.provider.Settings
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Tab
import androidx.compose.material3.PrimaryTabRow
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import br.gov.ro.pm.sitposse7.campo.R
import br.gov.ro.pm.sitposse7.campo.gps.Gnss
import br.gov.ro.pm.sitposse7.campo.gps.LeituraGnss
import br.gov.ro.pm.sitposse7.dominio.Coordenada
import br.gov.ro.pm.sitposse7.dominio.ValidacaoCadastro
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.takeWhile

private const val LIMITE_S = 90

/** Obter localização (RF-A07, RN-50..55): GNSS sem internet, com progresso; ou digitação manual. */
@Composable
fun GpsDialogo(aoUsar: (Coordenada) -> Unit, aoFechar: () -> Unit) {
    val contexto = LocalContext.current
    var aba by rememberSaveable { mutableIntStateOf(0) }
    var permitido by remember { mutableStateOf(ContextCompat.checkSelfPermission(contexto, Manifest.permission.ACCESS_FINE_LOCATION) == PackageManager.PERMISSION_GRANTED) }
    var negado by remember { mutableStateOf(false) }
    val pedir = rememberLauncherForActivityResult(ActivityResultContracts.RequestMultiplePermissions()) { r ->
        permitido = r[Manifest.permission.ACCESS_FINE_LOCATION] == true
        negado = !permitido
        if (!permitido) aba = 1
    }
    var ligado by remember { mutableStateOf(Gnss.disponivel(contexto)) }
    var atual by remember { mutableStateOf<LeituraGnss?>(null) }
    var melhor by remember { mutableStateOf<LeituraGnss?>(null) }
    var segundos by remember { mutableLongStateOf(0) }
    var tentativa by remember { mutableIntStateOf(0) }
    var lat by rememberSaveable { mutableStateOf("") }
    var lon by rememberSaveable { mutableStateOf("") }

    LaunchedEffect(Unit) { if (!permitido) pedir.launch(arrayOf(Manifest.permission.ACCESS_FINE_LOCATION, Manifest.permission.ACCESS_COARSE_LOCATION)) }
    LaunchedEffect(permitido, aba, tentativa) {
        while (aba == 0 && !ligado) { delay(1000); ligado = Gnss.disponivel(contexto) }
    }
    LaunchedEffect(permitido, ligado, aba, tentativa) {
        if (!permitido || !ligado || aba != 0) return@LaunchedEffect
        val inicio = System.currentTimeMillis()
        segundos = 0
        Gnss.leituras(contexto).takeWhile { segundos < LIMITE_S && (melhor?.precisaoM ?: Double.MAX_VALUE) > ValidacaoCadastro.META_PRECISAO_M }.collect { l ->
            segundos = (System.currentTimeMillis() - inicio) / 1000
            atual = l
            if (l.lat != null && (melhor == null || (l.precisaoM ?: 999.0) <= (melhor?.precisaoM ?: 999.0))) melhor = l
        }
    }

    val m = melhor
    AlertDialog(
        onDismissRequest = aoFechar,
        title = { Text(stringResource(R.string.gps_titulo)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                PrimaryTabRow(selectedTabIndex = aba) {
                    Tab(aba == 0, { aba = 0 }, text = { Text(stringResource(R.string.gps_aba_gnss)) })
                    Tab(aba == 1, { aba = 1 }, text = { Text(stringResource(R.string.gps_aba_manual)) })
                }
                if (aba == 0) when {
                    !permitido -> Text(stringResource(if (negado) R.string.gps_permissao_negada else R.string.gps_permissao))
                    !ligado -> {
                        Text(stringResource(R.string.gps_desligado))
                        TextButton(onClick = { contexto.startActivity(Intent(Settings.ACTION_LOCATION_SOURCE_SETTINGS)) }) { Text(stringResource(R.string.gps_abrir_config)) }
                    }
                    else -> {
                        LinearProgressIndicator(progress = { (segundos.toFloat() / LIMITE_S).coerceIn(0f, 1f) }, modifier = Modifier.fillMaxWidth())
                        Text(stringResource(R.string.gps_satelites, atual?.satelitesEmUso ?: 0, atual?.satelitesVistos ?: 0))
                        Text(atual?.precisaoM?.let { stringResource(R.string.gps_precisao_atual, it) } ?: stringResource(R.string.gps_procurando))
                        m?.let {
                            Text(stringResource(R.string.gps_melhor, it.precisaoM ?: 0.0), style = MaterialTheme.typography.titleMedium)
                            if ((it.precisaoM ?: 999.0) > ValidacaoCadastro.META_PRECISAO_M) Text(stringResource(R.string.gps_acima_meta), color = MaterialTheme.colorScheme.tertiary)
                        }
                        if (segundos >= LIMITE_S || (m?.precisaoM ?: 999.0) <= ValidacaoCadastro.META_PRECISAO_M) Text(stringResource(R.string.gps_parou, segundos))
                    }
                } else {
                    Text(stringResource(R.string.gps_manual_ajuda), style = MaterialTheme.typography.bodySmall)
                    val dec = KeyboardOptions(keyboardType = KeyboardType.Text)
                    OutlinedTextField(lat, { lat = it.take(14) }, label = { Text(stringResource(R.string.gps_latitude)) }, singleLine = true, keyboardOptions = dec)
                    OutlinedTextField(lon, { lon = it.take(15) }, label = { Text(stringResource(R.string.gps_longitude)) }, singleLine = true, keyboardOptions = dec)
                }
            }
        },
        confirmButton = {
            Row {
                if (aba == 0) {
                    TextButton(onClick = { tentativa++ }, enabled = permitido && ligado) { Text(stringResource(R.string.gps_continuar)) }
                    TextButton(enabled = m?.lat != null, onClick = {
                        m?.let { aoUsar(Coordenada(it.lat!!, it.lon!!, "gnss", it.precisaoM, it.instante, it.satelitesEmUso.toLong())) }
                    }) { Text(stringResource(R.string.gps_usar)) }
                } else {
                    val la = lat.replace(',', '.').trim().toDoubleOrNull()
                    val lo = lon.replace(',', '.').trim().toDoubleOrNull()
                    TextButton(enabled = la != null && lo != null && la in -90.0..90.0 && lo in -180.0..180.0, onClick = {
                        aoUsar(Coordenada(la!!, lo!!, "manual")) // RN-52: manual não tem precisão
                    }) { Text(stringResource(R.string.gps_usar)) }
                }
            }
        },
        dismissButton = { TextButton(onClick = aoFechar) { Text(stringResource(R.string.cancelar)) } },
    )
}
