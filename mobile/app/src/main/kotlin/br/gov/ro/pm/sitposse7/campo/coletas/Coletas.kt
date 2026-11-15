package br.gov.ro.pm.sitposse7.campo.coletas

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.text.input.TextFieldState
import androidx.compose.foundation.text.input.TextFieldLineLimits
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.snapshotFlow
import kotlinx.coroutines.flow.drop
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import br.gov.ro.pm.sitposse7.campo.R
import br.gov.ro.pm.sitposse7.dominio.Codigos
import br.gov.ro.pm.sitposse7.sync.DestinoV2
import br.gov.ro.pm.sitposse7.sync.JsonContratos
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.format.DateTimeFormatter

/** Destino confirmado pela Central (gravado na coleta com o recibo). */
fun destinoDe(json: String?): DestinoV2? = json?.let { runCatching { JsonContratos.decodeFromString(DestinoV2.serializer(), it) }.getOrNull() }

/** Data local de um instante (ADR-0010: exibição sempre local). */
fun dataLocal(ms: Long): String = DateTimeFormatter.ofPattern("dd/MM/yyyy").format(Instant.ofEpochMilli(ms).atZone(ZoneId.systemDefault()))

/** "2026-10-10" → "10/10/2026". */
fun dataBR(iso: String): String = runCatching { DateTimeFormatter.ofPattern("dd/MM/yyyy").format(LocalDate.parse(iso)) }.getOrDefault(iso)

/** A coleta começou num dia (local) diferente de hoje (aviso da RN-07). */
fun comecouEmOutroDia(iniciadaEm: Long, agora: Long = System.currentTimeMillis()): Boolean {
    val z = ZoneId.systemDefault()
    return Instant.ofEpochMilli(iniciadaEm).atZone(z).toLocalDate() != Instant.ofEpochMilli(agora).atZone(z).toLocalDate()
}

@Composable
fun textoDestino(d: DestinoV2): String = stringResource(R.string.coleta_destino, d.equipeNome, d.jornadaNumero.toInt(), dataBR(d.jornadaData), d.estudoNome)

/**
 * Campo do código da equipe: grande, monoespaçado, formata EQ-XXXX-XXXX enquanto digita e
 * mostra o erro do dígito verificador (RN-02), sem consultar a Central (RN-05).
 */
@Composable
fun CampoCodigo(estado: TextFieldState, aoEditar: () -> Unit, erro: String?, aoConfirmar: () -> Unit, modifier: Modifier = Modifier) {
    LaunchedEffect(estado) { snapshotFlow { estado.text.toString() }.drop(1).collect { aoEditar() } }
    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
        OutlinedTextField(
            state = estado,
            inputTransformation = EntradaCodigo,
            outputTransformation = MascaraCodigo,
            leadingIcon = { Text(Codigos.PREFIXO_EQUIPE + "-", fontFamily = FontFamily.Monospace) },
            modifier = modifier,
            label = { Text(stringResource(R.string.codigo_campo)) },
            placeholder = { Text(stringResource(R.string.codigo_corpo_exemplo)) },
            lineLimits = TextFieldLineLimits.SingleLine,
            isError = erro != null,
            supportingText = erro?.let { { Text(it) } },
            textStyle = MaterialTheme.typography.headlineSmall.copy(fontFamily = FontFamily.Monospace, letterSpacing = 2.sp),
            keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Characters, autoCorrectEnabled = false,
                keyboardType = KeyboardType.Ascii, imeAction = ImeAction.Done),
            onKeyboardAction = { aoConfirmar() },
        )
    }
}

/** Valida o texto digitado; devolve o código normalizado ou a mensagem de erro. */
fun validarCodigo(texto: String): Pair<String?, String?> = when (val r = Codigos.normalizar(texto)) {
    is Codigos.Resultado.Valido -> r.codigo to null
    is Codigos.Resultado.Invalido -> null to r.erro.mensagem
}
