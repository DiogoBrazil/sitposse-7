package br.gov.ro.pm.sitposse7.campo.propriedades

import android.graphics.Bitmap
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import br.gov.ro.pm.sitposse7.campo.CampoApp
import br.gov.ro.pm.sitposse7.campo.R
import br.gov.ro.pm.sitposse7.campo.fotos.ProcessamentoFoto
import br.gov.ro.pm.sitposse7.campo.ui.BarraSuperior
import br.gov.ro.pm.sitposse7.campo.ui.Centralizado
import br.gov.ro.pm.sitposse7.campo.ui.Navegador
import br.gov.ro.pm.sitposse7.campo.ui.Secao
import br.gov.ro.pm.sitposse7.campo.ui.Tela
import br.gov.ro.pm.sitposse7.campo.ui.dataHoraLocal
import br.gov.ro.pm.sitposse7.dados.PropriedadeCompleta
import br.gov.ro.pm.sitposse7.dominio.Area
import br.gov.ro.pm.sitposse7.dominio.Areas
import br.gov.ro.pm.sitposse7.dominio.Catalogo
import br.gov.ro.pm.sitposse7.dominio.Categoria
import br.gov.ro.pm.sitposse7.dominio.Quantitativo
import br.gov.ro.pm.sitposse7.dominio.UnidadeArea
import br.gov.ro.pm.sitposse7.dominio.textoArea
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.util.UUID

/** [5] Detalhes do cadastro, com editar e excluir (tombstone). */
@Composable
fun DetalheTela(uuid: String, nav: Navegador) {
    val app = LocalContext.current.applicationContext as CampoApp
    val escopo = rememberCoroutineScope()
    var dados by remember { mutableStateOf<PropriedadeCompleta?>(null) }
    var carregou by remember { mutableStateOf(false) }
    var confirmarExclusao by remember { mutableStateOf(false) }
    LaunchedEffect(uuid) { dados = app.repo.carregar(uuid); carregou = true }

    Scaffold(topBar = {
        BarraSuperior(dados?.propriedade?.nomePropriedade.orEmpty(), aoVoltar = { nav.voltar() }) {
            if (dados != null) {
                IconButton(onClick = { nav.ir(Tela.Formulario(uuid, UUID.randomUUID().toString())) }) { Icon(painterResource(R.drawable.ic_edit), stringResource(R.string.editar)) }
                IconButton(onClick = { confirmarExclusao = true }) { Icon(painterResource(R.drawable.ic_delete), stringResource(R.string.excluir)) }
            }
        }
    }) { pad ->
        val d = dados
        if (d == null) {
            if (carregou) Text(stringResource(R.string.detalhe_nao_encontrado), Modifier.padding(pad).padding(24.dp))
            return@Scaffold
        }
        val p = d.propriedade
        Centralizado(840, Modifier.padding(pad).fillMaxSize()) {
            Column(Modifier.verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Secao(stringResource(R.string.secao_identificacao)) {
                    Linha(stringResource(R.string.campo_nome), p.nomePropriedade)
                    Linha(stringResource(R.string.campo_responsavel), p.responsavelFamilia ?: "—")
                    Linha(stringResource(R.string.detalhe_atualizado), dataHoraLocal(p.updatedAt))
                }
                val valores = mapOf(
                    "pessoas_adultos" to p.pessoasAdultos, "pessoas_jovens" to p.pessoasJovens, "pessoas_criancas" to p.pessoasCriancas,
                    "veic_caminhoes" to p.veicCaminhoes, "veic_automoveis_utilitarios" to p.veicAutomoveisUtilitarios, "veic_motos_ciclomotores" to p.veicMotosCiclomotores,
                    "bov_adultos" to p.bovAdultos, "bov_bezerros" to p.bovBezerros, "equ_adultos" to p.equAdultos, "equ_potros" to p.equPotros,
                )
                Quantitativo.entries.groupBy { it.grupo }.forEach { (grupo, qs) ->
                    Secao(grupo, resumo = qs.sumOf { valores[it.campo] ?: 0 }.toString(), aberta = false) {
                        qs.forEach { q -> Linha(q.rotulo, (valores[q.campo] ?: 0).toString()) }
                    }
                }
                Categoria.entries.forEach { cat ->
                    val itens = d.itens.filter { it.categoria == cat.codigo }
                    Secao(cat.titulo, resumo = itens.sumOf { it.quantidade }.toString(), aberta = false) {
                        if (itens.isEmpty()) Text(stringResource(R.string.nenhum_item))
                        itens.forEach { Linha(it.nomePersonalizado ?: Catalogo.item(it.codigoCatalogo)?.nome ?: it.codigoCatalogo, it.quantidade.toString()) }
                    }
                }
                Secao(stringResource(R.string.secao_area)) {
                    val m = p.areaValorMilesimos
                    val u = p.areaUnidade?.let(UnidadeArea::doCodigo)
                    if (m == null || u == null) Text(stringResource(R.string.area_nao_informada))
                    else Linha(stringResource(if (u == UnidadeArea.HECTARE) R.string.unidade_ha else R.string.unidade_alq), "${textoArea(m)} · ${Areas.formatarHa(Area(m, u).dm2)} ha")
                }
                Secao(stringResource(R.string.secao_localizacao)) {
                    if (p.lat == null || p.lon == null) Text(stringResource(R.string.gps_nao_informado))
                    else {
                        Linha(stringResource(R.string.gps_coordenadas), "%.6f, %.6f".format(p.lat, p.lon))
                        Linha(stringResource(R.string.gps_origem), stringResource(if (p.gpsOrigem == "manual") R.string.gps_manual else R.string.gps_gnss))
                        p.gpsPrecisaoM?.let { Linha(stringResource(R.string.gps_precisao), "%.0f m".format(it)) }
                    }
                }
                Secao(stringResource(R.string.secao_fotos), resumo = d.fotos.size.toString()) {
                    if (d.fotos.isEmpty()) Text(stringResource(R.string.sem_fotos))
                    d.fotos.chunked(3).forEach { linha ->
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            linha.forEach { f -> Miniatura(app.fotos.arquivo(f.caminhoRelativo), Modifier.weight(1f)) }
                            repeat(3 - linha.size) { androidx.compose.foundation.layout.Spacer(Modifier.weight(1f)) }
                        }
                    }
                }
            }
        }
    }

    if (confirmarExclusao) AlertDialog(
        onDismissRequest = { confirmarExclusao = false },
        title = { Text(stringResource(R.string.excluir_titulo)) },
        text = { Text(stringResource(R.string.excluir_texto)) },
        confirmButton = { TextButton(onClick = { escopo.launch { app.repo.excluir(uuid); confirmarExclusao = false; nav.voltar() } }) { Text(stringResource(R.string.excluir)) } },
        dismissButton = { TextButton(onClick = { confirmarExclusao = false }) { Text(stringResource(R.string.cancelar)) } },
    )
}

@Composable
fun Linha(rotulo: String, valor: String) {
    Row(Modifier.fillMaxWidth()) {
        Text(rotulo, Modifier.weight(1f), style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
        androidx.compose.foundation.layout.Spacer(Modifier.width(12.dp))
        Text(valor, style = MaterialTheme.typography.bodyLarge)
    }
}

/** Miniatura gerada com amostragem fora da main thread. */
@Composable
fun Miniatura(arquivo: java.io.File, modifier: Modifier = Modifier) {
    val bmp by produceState<Bitmap?>(null, arquivo) { value = withContext(Dispatchers.IO) { ProcessamentoFoto.miniatura(arquivo) } }
    val b = bmp
    if (b != null) Image(b.asImageBitmap(), null, modifier.aspectRatio(1f), contentScale = ContentScale.Crop)
    else androidx.compose.foundation.layout.Box(modifier.aspectRatio(1f))
}
