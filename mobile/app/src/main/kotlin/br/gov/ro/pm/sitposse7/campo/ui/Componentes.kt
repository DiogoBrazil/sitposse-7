package br.gov.ro.pm.sitposse7.campo.ui

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledTonalIconButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import br.gov.ro.pm.sitposse7.campo.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BarraSuperior(titulo: String, aoVoltar: (() -> Unit)? = null, acoes: @Composable RowScope.() -> Unit = {}) {
    TopAppBar(
        title = { Text(titulo, maxLines = 1) },
        navigationIcon = {
            if (aoVoltar != null) IconButton(onClick = aoVoltar) { Icon(painterResource(R.drawable.ic_back), stringResource(R.string.voltar)) }
        },
        actions = acoes,
        colors = TopAppBarDefaults.topAppBarColors(
            containerColor = MaterialTheme.colorScheme.primary,
            titleContentColor = MaterialTheme.colorScheme.onPrimary,
            navigationIconContentColor = MaterialTheme.colorScheme.onPrimary,
            actionIconContentColor = MaterialTheme.colorScheme.onPrimary,
        ),
    )
}

/** Seção recolhível do formulário, com resumo no cabeçalho (ex.: "Veículos · 3"). */
@Composable
fun Secao(titulo: String, resumo: String? = null, aberta: Boolean = true, conteudo: @Composable () -> Unit) {
    var aberto by rememberSaveable(titulo) { mutableStateOf(aberta) }
    Card(Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface), elevation = CardDefaults.cardElevation(1.dp)) {
        Row(
            Modifier.fillMaxWidth().clickable { aberto = !aberto }.padding(horizontal = 16.dp, vertical = 14.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(titulo, style = MaterialTheme.typography.titleMedium, modifier = Modifier.weight(1f))
            if (resumo != null) Text(resumo, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Spacer(Modifier.width(8.dp))
            Icon(painterResource(if (aberto) R.drawable.ic_collapse else R.drawable.ic_expand), null)
        }
        AnimatedVisibility(aberto) {
            Column(Modifier.padding(start = 16.dp, end = 16.dp, bottom = 16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) { conteudo() }
        }
    }
}

/** Quantidade inteira ≥ [minimo] com botões −/+ (alvos ≥ 48 dp) e digitação. */
@Composable
fun CampoQuantidade(rotulo: String, valor: Long, aoMudar: (Long) -> Unit, ajuda: String? = null, minimo: Long = 0, erro: String? = null) {
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Column(Modifier.weight(1f)) {
            Text(rotulo, style = MaterialTheme.typography.bodyLarge)
            if (ajuda != null) Text(ajuda, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            if (erro != null) Text(erro, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error)
        }
        FilledTonalIconButton(onClick = { if (valor > minimo) aoMudar(valor - 1) }, enabled = valor > minimo, modifier = Modifier.size(48.dp)) {
            Icon(painterResource(R.drawable.ic_remove), stringResource(R.string.diminuir))
        }
        OutlinedTextField(
            value = valor.toString(),
            onValueChange = { t -> val d = t.filter(Char::isDigit).take(7); aoMudar(if (d.isEmpty()) minimo else d.toLong()) },
            modifier = Modifier.width(88.dp).padding(horizontal = 6.dp),
            singleLine = true,
            textStyle = MaterialTheme.typography.titleMedium.copy(textAlign = TextAlign.Center),
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
        )
        FilledTonalIconButton(onClick = { aoMudar(valor + 1) }, modifier = Modifier.size(48.dp)) {
            Icon(painterResource(R.drawable.ic_add), stringResource(R.string.aumentar))
        }
    }
}

/** Selo de estado (persistência e sincronização). */
@Composable
fun Selo(texto: String, cor: Color, corTexto: Color) {
    Surface(color = cor, contentColor = corTexto, shape = MaterialTheme.shapes.small) {
        Text(texto, style = MaterialTheme.typography.labelMedium, modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp))
    }
}

@Composable
fun AvisoCartao(texto: String, cor: Color = MaterialTheme.colorScheme.errorContainer, corTexto: Color = MaterialTheme.colorScheme.onErrorContainer, acoes: @Composable RowScope.() -> Unit = {}) {
    Card(Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = cor, contentColor = corTexto)) {
        Column(Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(painterResource(R.drawable.ic_warning), null)
                Spacer(Modifier.width(12.dp))
                Text(texto, style = MaterialTheme.typography.bodyLarge)
            }
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End, content = acoes)
        }
    }
}

/** Formata epoch ms em horário local (ADR-0010: exibição sempre local). */
fun dataHoraLocal(ms: Long): String =
    java.text.DateFormat.getDateTimeInstance(java.text.DateFormat.SHORT, java.text.DateFormat.SHORT).format(java.util.Date(ms))
