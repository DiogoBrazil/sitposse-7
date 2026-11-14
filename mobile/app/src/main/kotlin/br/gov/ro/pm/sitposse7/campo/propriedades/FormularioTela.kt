package br.gov.ro.pm.sitposse7.campo.propriedades

import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.AssistChip
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import br.gov.ro.pm.sitposse7.campo.CampoApp
import br.gov.ro.pm.sitposse7.campo.R
import br.gov.ro.pm.sitposse7.campo.fotos.CameraFoto
import br.gov.ro.pm.sitposse7.campo.ui.AvisoCartao
import br.gov.ro.pm.sitposse7.campo.ui.BarraSuperior
import br.gov.ro.pm.sitposse7.campo.ui.CampoQuantidade
import br.gov.ro.pm.sitposse7.campo.ui.Centralizado
import br.gov.ro.pm.sitposse7.campo.ui.Navegador
import br.gov.ro.pm.sitposse7.campo.ui.Secao
import br.gov.ro.pm.sitposse7.campo.ui.Tela
import br.gov.ro.pm.sitposse7.dominio.Areas
import br.gov.ro.pm.sitposse7.dominio.CadastroPropriedade
import br.gov.ro.pm.sitposse7.dominio.Catalogo
import br.gov.ro.pm.sitposse7.dominio.Categoria
import br.gov.ro.pm.sitposse7.dominio.ItemCadastro
import br.gov.ro.pm.sitposse7.dominio.Quantitativo
import br.gov.ro.pm.sitposse7.dominio.ResultadoArea
import br.gov.ro.pm.sitposse7.dominio.ValidacaoCadastro

/** [4] Cadastro e edição por seções recolhíveis, com barra inferior "Salvar". */
@Composable
fun FormularioTela(tela: Tela.Formulario, nav: Navegador) {
    val app = LocalContext.current.applicationContext as CampoApp
    val vm: FormularioViewModel = viewModel(key = tela.rascunho, factory = viewModelFactory { initializer { FormularioViewModel(app, tela.uuid, tela.rascunho) } })
    val e by vm.estado.collectAsState()
    var camera by rememberSaveable { mutableStateOf(false) }
    var gps by rememberSaveable { mutableStateOf(false) }
    var confirmarSaida by rememberSaveable { mutableStateOf(false) }
    var removerFoto by rememberSaveable { mutableStateOf<String?>(null) }
    var semEspaco by rememberSaveable { mutableStateOf(false) }

    LaunchedEffect(e.salvoComo) { e.salvoComo?.let { nav.substituir(Tela.Detalhe(it)) } }
    val sair = { if (e.modificado) confirmarSaida = true else nav.voltar() }
    BackHandler(enabled = !camera) { sair() }

    val contexto = LocalContext.current
    val permCamera = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { ok -> if (ok) camera = true }
    val abrirCamera = {
        when {
            !vm.espacoParaFoto() -> semEspaco = true
            ContextCompat.checkSelfPermission(contexto, Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED -> camera = true
            else -> permCamera.launch(Manifest.permission.CAMERA)
        }
    }

    if (camera) {
        CameraFoto(vm.pastaCaptura(), aoCapturar = { camera = false; vm.adicionarFoto(it) }, aoCancelar = { camera = false })
        return
    }

    val c = e.cadastro
    fun erro(campo: String) = e.erros.firstOrNull { it.campo == campo }?.mensagem

    Scaffold(
        topBar = { BarraSuperior(stringResource(if (tela.uuid == null) R.string.form_titulo_novo else R.string.form_titulo_editar), aoVoltar = sair) },
        bottomBar = {
            Surface(tonalElevation = 3.dp, shadowElevation = 8.dp) {
                Row(Modifier.fillMaxWidth().navigationBarsPadding().padding(12.dp), horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically) {
                    if (e.erros.isNotEmpty()) Text(stringResource(R.string.form_corrija, e.erros.size), color = MaterialTheme.colorScheme.error, modifier = Modifier.weight(1f))
                    else Spacer(Modifier.weight(1f))
                    Button(onClick = vm::salvar, enabled = e.carregado && !e.salvando && !e.processandoFoto, modifier = Modifier.height(56.dp).width(180.dp)) {
                        if (e.salvando) CircularProgressIndicator(Modifier.size(24.dp), color = MaterialTheme.colorScheme.onPrimary)
                        else { Icon(painterResource(R.drawable.ic_check), null); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.salvar)) }
                    }
                }
            }
        },
    ) { pad ->
        if (!e.carregado) { Box(Modifier.padding(pad).fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }; return@Scaffold }
        Centralizado(760, Modifier.padding(pad).fillMaxSize()) {
            Column(Modifier.verticalScroll(rememberScrollState()).imePadding().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                if (e.modificado) Surface(color = MaterialTheme.colorScheme.tertiaryContainer, shape = MaterialTheme.shapes.small) {
                    Text(stringResource(R.string.form_rascunho), Modifier.fillMaxWidth().padding(10.dp), style = MaterialTheme.typography.bodyMedium)
                }
                e.falha?.let { AvisoCartao(stringResource(R.string.form_falha, it)) }

                Secao(stringResource(R.string.secao_identificacao)) {
                    val cap = KeyboardOptions(capitalization = KeyboardCapitalization.Words)
                    OutlinedTextField(c.nome, { v -> vm.alterar { it.copy(nome = v.take(200)) } }, Modifier.fillMaxWidth(), label = { Text(stringResource(R.string.campo_nome_obrigatorio)) },
                        singleLine = true, keyboardOptions = cap, isError = erro("nome") != null, supportingText = erro("nome")?.let { { Text(it) } })
                    OutlinedTextField(c.responsavel, { v -> vm.alterar { it.copy(responsavel = v.take(200)) } }, Modifier.fillMaxWidth(), label = { Text(stringResource(R.string.campo_responsavel)) },
                        singleLine = true, keyboardOptions = cap)
                }

                Quantitativo.entries.groupBy { it.grupo }.forEach { (grupo, qs) ->
                    Secao(grupo, resumo = qs.sumOf { c.quantidade(it) }.toString(), aberta = grupo == "Pessoas") {
                        qs.forEach { q -> CampoQuantidade(q.rotulo, c.quantidade(q), { v -> vm.alterar { it.copy(quantidades = it.quantidades + (q.campo to v)) } }, q.ajuda, erro = erro(q.campo)) }
                    }
                }

                Categoria.entries.forEach { cat -> SecaoItens(cat, c, vm, e.erros.associate { it.campo to it.mensagem }) }

                Secao(stringResource(R.string.secao_area), resumo = resumoArea(c)) {
                    OutlinedTextField(c.areaTexto, { v -> vm.alterar { it.copy(areaTexto = v.take(20)) } }, Modifier.fillMaxWidth(), label = { Text(stringResource(R.string.area_valor)) },
                        singleLine = true, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal), isError = erro("area") != null,
                        supportingText = { Text(erro("area") ?: stringResource(R.string.area_ajuda)) })
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        FilterChip(c.areaUnidade == "ha", { vm.alterar { it.copy(areaUnidade = "ha") } }, label = { Text(stringResource(R.string.unidade_ha)) })
                        FilterChip(c.areaUnidade == "alq_paulista", { vm.alterar { it.copy(areaUnidade = "alq_paulista") } }, label = { Text(stringResource(R.string.unidade_alq)) })
                    }
                    (Areas.interpretar(c.areaTexto, c.areaUnidade) as? ResultadoArea.Valida)?.let {
                        Text(stringResource(R.string.area_conversao, Areas.formatarHa(it.area.dm2)), style = MaterialTheme.typography.bodyLarge)
                    }
                }

                Secao(stringResource(R.string.secao_localizacao), resumo = if (c.coordenada != null) stringResource(R.string.ok) else "—") {
                    c.coordenada?.let { g ->
                        Text("%.6f, %.6f".format(g.lat, g.lon), style = MaterialTheme.typography.titleMedium)
                        Text(
                            if (g.origem == "manual") stringResource(R.string.gps_manual) else stringResource(R.string.gps_precisao_satelites, g.precisaoM ?: 0.0, g.satelites ?: 0),
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    } ?: Text(stringResource(R.string.gps_nao_informado))
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Button(onClick = { gps = true }, modifier = Modifier.height(48.dp)) { Icon(painterResource(R.drawable.ic_place), null); Spacer(Modifier.width(6.dp)); Text(stringResource(R.string.gps_obter)) }
                        if (c.coordenada != null) TextButton(onClick = { vm.alterar { it.copy(coordenada = null) } }) { Text(stringResource(R.string.remover)) }
                    }
                    erro("coordenada")?.let { Text(it, color = MaterialTheme.colorScheme.error) }
                }

                Secao(stringResource(R.string.secao_fotos), resumo = c.fotos.size.toString()) {
                    c.fotos.chunked(3).forEach { linha ->
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            linha.forEach { f ->
                                Box(Modifier.weight(1f)) {
                                    Miniatura(vm.arquivo(f.caminhoRelativo), Modifier.fillMaxWidth())
                                    IconButton(onClick = { removerFoto = f.uuid }, modifier = Modifier.align(Alignment.TopEnd)) {
                                        Surface(shape = MaterialTheme.shapes.small, color = MaterialTheme.colorScheme.surface.copy(alpha = 0.85f)) {
                                            Icon(painterResource(R.drawable.ic_delete), stringResource(R.string.foto_remover), Modifier.padding(4.dp))
                                        }
                                    }
                                }
                            }
                            repeat(3 - linha.size) { Spacer(Modifier.weight(1f)) }
                        }
                    }
                    OutlinedButton(onClick = abrirCamera, enabled = !e.processandoFoto, modifier = Modifier.fillMaxWidth().height(56.dp)) {
                        if (e.processandoFoto) { CircularProgressIndicator(Modifier.size(20.dp)); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.foto_processando)) }
                        else { Icon(painterResource(R.drawable.ic_camera), null); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.foto_tirar)) }
                    }
                }
                Spacer(Modifier.height(24.dp))
            }
        }
    }

    if (gps) GpsDialogo(aoUsar = { coord -> vm.alterar { it.copy(coordenada = coord) }; gps = false }, aoFechar = { gps = false })

    removerFoto?.let { id ->
        AlertDialog(
            onDismissRequest = { removerFoto = null },
            title = { Text(stringResource(R.string.foto_remover)) }, text = { Text(stringResource(R.string.foto_remover_texto)) },
            confirmButton = { TextButton(onClick = { vm.alterar { it.copy(fotos = it.fotos.filter { f -> f.uuid != id }) }; removerFoto = null }) { Text(stringResource(R.string.remover)) } },
            dismissButton = { TextButton(onClick = { removerFoto = null }) { Text(stringResource(R.string.cancelar)) } },
        )
    }
    if (confirmarSaida) AlertDialog(
        onDismissRequest = { confirmarSaida = false },
        title = { Text(stringResource(R.string.form_sair_titulo)) }, text = { Text(stringResource(R.string.form_sair_texto)) },
        confirmButton = { TextButton(onClick = { confirmarSaida = false; nav.voltar() }) { Text(stringResource(R.string.form_manter_rascunho)) } },
        dismissButton = { TextButton(onClick = { confirmarSaida = false; vm.descartar { nav.voltar() } }) { Text(stringResource(R.string.descartar)) } },
    )
    if (semEspaco) AlertDialog(
        onDismissRequest = { semEspaco = false }, text = { Text(stringResource(R.string.foto_sem_espaco)) },
        confirmButton = { TextButton(onClick = { semEspaco = false }) { Text(stringResource(R.string.ok)) } },
    )
}

private fun resumoArea(c: CadastroPropriedade): String =
    (Areas.interpretar(c.areaTexto, c.areaUnidade) as? ResultadoArea.Valida)?.let { "${Areas.formatarHa(it.area.dm2)} ha" } ?: "—"

/** Itens dinâmicos de uma categoria (RN-30..34): chips do catálogo + vários "Outros". */
@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun SecaoItens(cat: Categoria, c: CadastroPropriedade, vm: FormularioViewModel, erros: Map<String, String>) {
    val indices = c.itens.indices.filter { Catalogo.item(c.itens[it].codigo)?.categoria == cat }
    var novoOutros by rememberSaveable(cat) { mutableStateOf<String?>(null) }
    var somarCom by rememberSaveable(cat) { mutableStateOf<Int?>(null) }
    Secao(cat.titulo, resumo = indices.sumOf { c.itens[it].quantidade }.toString(), aberta = false) {
        FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Catalogo.daCategoria(cat).forEach { item ->
                val presente = c.itens.any { it.codigo == item.codigo }
                if (item.outros) AssistChip(onClick = { novoOutros = "" }, label = { Text(stringResource(R.string.itens_outros_adicionar)) }, leadingIcon = { Icon(painterResource(R.drawable.ic_add), null) })
                else FilterChip(presente, onClick = {
                    // RN-32: adicionar de novo abre a quantidade existente (o chip só liga/desliga).
                    if (presente) vm.alterar { x -> x.copy(itens = x.itens.filter { it.codigo != item.codigo }) }
                    else vm.alterar { x -> x.copy(itens = x.itens + ItemCadastro(codigo = item.codigo)) }
                }, label = { Text(item.nome) })
            }
        }
        indices.forEach { i ->
            val it0 = c.itens[i]
            val nome = Catalogo.item(it0.codigo)?.let { if (it.outros) it0.nomePersonalizado.orEmpty() else it.nome }.orEmpty()
            Row(verticalAlignment = Alignment.CenterVertically) {
                Box(Modifier.weight(1f)) {
                    CampoQuantidade(nome, it0.quantidade, { v -> vm.alterar { x -> x.copy(itens = x.itens.mapIndexed { j, y -> if (j == i) y.copy(quantidade = v) else y }) } }, minimo = 1, erro = erros["item.$i"])
                }
                IconButton(onClick = { vm.alterar { x -> x.copy(itens = x.itens.filterIndexed { j, _ -> j != i }) } }) { Icon(painterResource(R.drawable.ic_close), stringResource(R.string.remover)) }
            }
        }
    }
    novoOutros?.let { texto ->
        val codigo = "${cat.prefixo}.outros"
        AlertDialog(
            onDismissRequest = { novoOutros = null },
            title = { Text(stringResource(R.string.itens_outros_titulo, cat.titulo)) },
            text = { OutlinedTextField(texto, { novoOutros = it.take(200) }, label = { Text(stringResource(R.string.itens_outros_nome)) }, singleLine = true) },
            confirmButton = {
                TextButton(enabled = texto.isNotBlank(), onClick = {
                    val repetido = ValidacaoCadastro.outrosRepetido(c.itens, codigo, texto)
                    if (repetido != null) somarCom = repetido // RN-33: avisa e oferece somar
                    else { vm.alterar { x -> x.copy(itens = x.itens + ItemCadastro(codigo = codigo, nomePersonalizado = texto.trim())) }; novoOutros = null }
                }) { Text(stringResource(R.string.adicionar)) }
            },
            dismissButton = { TextButton(onClick = { novoOutros = null }) { Text(stringResource(R.string.cancelar)) } },
        )
    }
    somarCom?.let { idx ->
        AlertDialog(
            onDismissRequest = { somarCom = null },
            text = { Text(stringResource(R.string.itens_outros_repetido, c.itens.getOrNull(idx)?.nomePersonalizado.orEmpty())) },
            confirmButton = { TextButton(onClick = { vm.alterar { x -> x.copy(itens = x.itens.mapIndexed { j, y -> if (j == idx) y.copy(quantidade = y.quantidade + 1) else y }) }; somarCom = null; novoOutros = null }) { Text(stringResource(R.string.itens_somar)) } },
            dismissButton = { TextButton(onClick = { val t = novoOutros.orEmpty().trim(); vm.alterar { x -> x.copy(itens = x.itens + ItemCadastro(codigo = "${cat.prefixo}.outros", nomePersonalizado = t)) }; somarCom = null; novoOutros = null }) { Text(stringResource(R.string.itens_manter_separado)) } },
        )
    }
}
