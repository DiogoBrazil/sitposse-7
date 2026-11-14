package br.gov.ro.pm.sitposse7.campo.propriedades

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.horizontalScroll
import androidx.compose.material3.Card
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import br.gov.ro.pm.sitposse7.campo.CampoApp
import br.gov.ro.pm.sitposse7.campo.R
import br.gov.ro.pm.sitposse7.campo.ui.BarraSuperior
import br.gov.ro.pm.sitposse7.campo.ui.Navegador
import br.gov.ro.pm.sitposse7.campo.ui.Selo
import br.gov.ro.pm.sitposse7.campo.ui.Tela
import br.gov.ro.pm.sitposse7.dados.PropriedadeResumo
import br.gov.ro.pm.sitposse7.dominio.Normalizacao

private enum class Filtro(val rotulo: Int) { PENDENTES(R.string.filtro_pendentes), SEM_GPS(R.string.filtro_sem_gps), SEM_FOTO(R.string.filtro_sem_foto), ERRO(R.string.filtro_erro) }

/** [3] Lista pesquisável com estados (RF-A09). */
@Composable
fun ListaTela(nav: Navegador) {
    val app = LocalContext.current.applicationContext as CampoApp
    val todas by app.repo.observarPropriedades().collectAsState(initial = emptyList())
    var busca by rememberSaveable { mutableStateOf("") }
    var filtro by rememberSaveable { mutableStateOf<Filtro?>(null) }
    val termo = Normalizacao.normalizarNome(busca)
    val visiveis = remember(todas, termo, filtro) {
        todas.filter { p ->
            (termo == null || listOfNotNull(p.nome_propriedade, p.responsavel_familia).any { Normalizacao.normalizarNome(it)?.contains(termo) == true }) &&
                when (filtro) {
                    null -> true
                    Filtro.PENDENTES -> p.pendente
                    Filtro.SEM_GPS -> !p.tem_gps
                    Filtro.SEM_FOTO -> p.fotos == 0
                    Filtro.ERRO -> p.rejeitada
                }
        }
    }
    Scaffold(
        topBar = { BarraSuperior(stringResource(R.string.lista_titulo, todas.size), aoVoltar = { nav.voltar() }) },
        floatingActionButton = { FloatingActionButton(onClick = { nav.ir(Tela.novoCadastro()) }) { Icon(painterResource(R.drawable.ic_add), stringResource(R.string.nova_propriedade)) } },
    ) { pad ->
        Column(Modifier.padding(pad).fillMaxSize()) {
            OutlinedTextField(
                busca, { busca = it }, Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
                placeholder = { Text(stringResource(R.string.lista_buscar)) }, leadingIcon = { Icon(painterResource(R.drawable.ic_search), null) }, singleLine = true,
            )
            Row(Modifier.horizontalScroll(rememberScrollState()).padding(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Filtro.entries.forEach { f -> FilterChip(filtro == f, { filtro = if (filtro == f) null else f }, label = { Text(stringResource(f.rotulo)) }) }
            }
            if (visiveis.isEmpty()) {
                Text(stringResource(if (todas.isEmpty()) R.string.lista_vazia else R.string.lista_nenhum_resultado), Modifier.padding(24.dp), style = MaterialTheme.typography.bodyLarge)
            }
            LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 320.dp),
                contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 8.dp, bottom = 96.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp), verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                items(visiveis, key = { it.uuid }) { p -> ItemLista(p) { nav.ir(Tela.Detalhe(p.uuid)) } }
            }
        }
    }
}

@Composable
private fun ItemLista(p: PropriedadeResumo, aoClicar: () -> Unit) {
    Card(Modifier.fillMaxWidth().clickable(onClick = aoClicar)) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(p.nome_propriedade, style = MaterialTheme.typography.titleMedium)
            p.responsavel_familia?.let { Text(it, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant) }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                SeloEstado(p)
                Icon(painterResource(R.drawable.ic_place), stringResource(if (p.tem_gps) R.string.tem_gps else R.string.sem_gps), tint = if (p.tem_gps) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outline)
                Icon(painterResource(R.drawable.ic_camera), null, tint = if (p.fotos > 0) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outline)
                Text(p.fotos.toString(), style = MaterialTheme.typography.labelLarge)
            }
        }
    }
}

/** Estados de persistência e sincronização (mobile/docs/telas-e-fluxos.md [3]). */
@Composable
fun SeloEstado(p: PropriedadeResumo) = when {
    p.rejeitada -> Selo(stringResource(R.string.estado_erro), MaterialTheme.colorScheme.errorContainer, MaterialTheme.colorScheme.onErrorContainer)
    p.pendente -> Selo(stringResource(R.string.estado_pendente), MaterialTheme.colorScheme.tertiaryContainer, MaterialTheme.colorScheme.onTertiaryContainer)
    else -> Selo(stringResource(R.string.estado_confirmado), MaterialTheme.colorScheme.primaryContainer, MaterialTheme.colorScheme.onPrimaryContainer)
}

