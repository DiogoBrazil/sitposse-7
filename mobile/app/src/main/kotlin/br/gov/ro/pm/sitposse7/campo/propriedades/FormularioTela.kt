package br.gov.ro.pm.sitposse7.campo.propriedades

import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.AssistChip
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LifecycleEventEffect
import androidx.lifecycle.createSavedStateHandle
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
import br.gov.ro.pm.sitposse7.dominio.ErroCampo
import br.gov.ro.pm.sitposse7.dominio.ItemCadastro
import br.gov.ro.pm.sitposse7.dominio.Quantitativo
import br.gov.ro.pm.sitposse7.dominio.ResultadoArea
import br.gov.ro.pm.sitposse7.dominio.ValidacaoCadastro
import br.gov.ro.pm.sitposse7.dominio.contarCaracteres

/**
 * [4] Cadastro e edição em 7 etapas (A28): Identificação e família · Veículos · Eletrodomésticos e
 * mobiliário · Animais e área · Localização e fotografias · Observações relevantes · Revisão.
 * A navegação nunca é bloqueada por campo vazio; só o salvamento exige os obrigatórios (RN-11).
 */
@Composable
fun FormularioTela(tela: Tela.Formulario, nav: Navegador) {
    val app = LocalContext.current.applicationContext as CampoApp
    val vm: FormularioViewModel = viewModel(key = tela.rascunho, factory = viewModelFactory {
        initializer { FormularioViewModel(app, createSavedStateHandle(), tela.uuid, tela.rascunho, tela.coleta) }
    })
    val e by vm.estado.collectAsState()
    val etapa by vm.etapa.collectAsState()
    var camera by rememberSaveable { mutableStateOf(false) }
    var gps by rememberSaveable { mutableStateOf(false) }
    var gpsManual by rememberSaveable { mutableStateOf(false) }
    var confirmarSaida by rememberSaveable { mutableStateOf(false) }
    var removerFoto by rememberSaveable { mutableStateOf<String?>(null) }
    var semEspaco by rememberSaveable { mutableStateOf(false) }
    var menuEtapas by rememberSaveable { mutableStateOf(false) }

    // App indo para o segundo plano (ou câmera/ligação por cima): grava o rascunho já.
    LifecycleEventEffect(Lifecycle.Event.ON_STOP) { vm.gravarAgora() }
    LaunchedEffect(e.salvoComo) { e.salvoComo?.let { nav.substituir(Tela.Detalhe(it)) } }
    val sair = { if (e.modificado) confirmarSaida = true else nav.voltar() }
    BackHandler(enabled = !camera) { if (etapa.ordinal > 0) vm.anterior() else sair() }

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
    val ultima = Etapa.entries.last()

    Scaffold(
        topBar = { BarraSuperior(stringResource(if (tela.uuid == null) R.string.form_titulo_novo else R.string.form_titulo_editar), aoVoltar = sair) },
        bottomBar = {
            Surface(tonalElevation = 3.dp, shadowElevation = 8.dp) {
                Row(Modifier.fillMaxWidth().navigationBarsPadding().padding(12.dp), horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically) {
                    if (etapa.ordinal > 0) OutlinedButton(onClick = vm::anterior, modifier = Modifier.height(56.dp).weight(1f)) {
                        Icon(painterResource(R.drawable.ic_back), null); Spacer(Modifier.width(6.dp)); Text(stringResource(R.string.voltar))
                    } else Spacer(Modifier.weight(1f))
                    if (etapa == ultima) {
                        Button(onClick = vm::salvar, enabled = e.carregado && !e.salvando && !e.processandoFoto, modifier = Modifier.height(56.dp).weight(1.3f)) {
                            if (e.salvando) CircularProgressIndicator(Modifier.size(24.dp), color = MaterialTheme.colorScheme.onPrimary)
                            else { Icon(painterResource(R.drawable.ic_check), null); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.form_salvar_cadastro)) }
                        }
                    } else {
                        Button(onClick = vm::proxima, enabled = e.carregado, modifier = Modifier.height(56.dp).weight(1.3f)) {
                            Text(stringResource(if (etapa.ordinal == ultima.ordinal - 1) R.string.form_revisar else R.string.form_proximo))
                            Spacer(Modifier.width(6.dp)); Icon(painterResource(R.drawable.ic_forward), null)
                        }
                    }
                }
            }
        },
    ) { pad ->
        if (!e.carregado) { Box(Modifier.padding(pad).fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }; return@Scaffold }
        Centralizado(760, Modifier.padding(pad).fillMaxSize()) {
            Column(Modifier.fillMaxSize()) {
                Progresso(etapa, aoAbrir = { menuEtapas = true })
                Column(Modifier.weight(1f).verticalScroll(rememberScrollState()).imePadding().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    if (e.modificado) Surface(color = MaterialTheme.colorScheme.tertiaryContainer, shape = MaterialTheme.shapes.small) {
                        Text(stringResource(R.string.form_rascunho), Modifier.fillMaxWidth().padding(10.dp), style = MaterialTheme.typography.bodyMedium)
                    }
                    e.falha?.let { AvisoCartao(stringResource(R.string.form_falha, it)) }
                    when (etapa) {
                        Etapa.IDENTIFICACAO -> EtapaIdentificacao(c, vm, ::erro)
                        Etapa.VEICULOS -> Quantidades(c, vm, ::erro, Quantitativo.entries.filter { it.grupo == "Veículos" })
                        Etapa.BENS -> {
                            SecaoItens(Categoria.ELETRODOMESTICO, c, vm, e.erros.associate { it.campo to it.mensagem }, aberta = true)
                            SecaoItens(Categoria.MOVEL, c, vm, e.erros.associate { it.campo to it.mensagem }, aberta = true)
                        }
                        Etapa.ANIMAIS_AREA -> EtapaAnimaisArea(c, vm, ::erro)
                        Etapa.LOCAL_FOTOS -> EtapaLocalFotos(c, vm, ::erro, e.processandoFoto, abrirGps = { gpsManual = false; gps = true }, abrirManual = { gpsManual = true; gps = true }, abrirCamera = abrirCamera, remover = { removerFoto = it })
                        Etapa.OBSERVACOES -> EtapaObservacoes(c, vm, ::erro)
                        Etapa.REVISAO -> Revisao(c, e.erros, vm)
                    }
                    Spacer(Modifier.height(24.dp))
                }
            }
        }
    }

    if (menuEtapas) AlertDialog(
        onDismissRequest = { menuEtapas = false },
        title = { Text(stringResource(R.string.form_etapas)) },
        text = {
            Column {
                Etapa.entries.forEach { x ->
                    val comErro = e.erros.any { Etapa.doCampo(it.campo, c) == x }
                    Row(Modifier.fillMaxWidth().clickable { vm.irPara(x); menuEtapas = false }.padding(vertical = 12.dp), verticalAlignment = Alignment.CenterVertically) {
                        Text("${x.ordinal + 1}.", Modifier.width(32.dp), fontWeight = FontWeight.SemiBold)
                        Text(stringResource(x.titulo), Modifier.weight(1f), fontWeight = if (x == etapa) FontWeight.Bold else FontWeight.Normal)
                        if (comErro) Icon(painterResource(R.drawable.ic_warning), null, tint = MaterialTheme.colorScheme.error)
                    }
                }
            }
        },
        confirmButton = { TextButton(onClick = { menuEtapas = false }) { Text(stringResource(R.string.cancelar)) } },
    )
    if (gps) GpsDialogo(inicial = e.cadastro.coordenada, manual = gpsManual, aoUsar = { coord -> vm.alterar { it.copy(coordenada = coord) }; gps = false }, aoFechar = { gps = false })
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

/** Cabeçalho "Etapa 3 de 7 · Nome" com barra segmentada; tocar abre a lista de etapas. */
@Composable
private fun Progresso(etapa: Etapa, aoAbrir: () -> Unit) {
    val total = Etapa.entries.size
    val rotulo = stringResource(R.string.form_etapa, etapa.ordinal + 1, total)
    val nome = stringResource(etapa.titulo)
    Column(Modifier.fillMaxWidth().clickable(onClick = aoAbrir).padding(horizontal = 16.dp, vertical = 10.dp).semantics { contentDescription = "$rotulo, $nome" }) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(rotulo, style = MaterialTheme.typography.labelLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Text(" · $nome", style = MaterialTheme.typography.titleMedium, modifier = Modifier.weight(1f), maxLines = 1)
            Icon(painterResource(R.drawable.ic_expand), stringResource(R.string.form_etapas))
        }
        Spacer(Modifier.height(8.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
            Etapa.entries.forEach { x ->
                Box(Modifier.weight(1f).height(6.dp).clip(RoundedCornerShape(3.dp))
                    .background(if (x.ordinal <= etapa.ordinal) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surfaceVariant))
            }
        }
    }
    HorizontalDivider()
}

@Composable
private fun Subtitulo(texto: String) = Text(texto, style = MaterialTheme.typography.titleMedium, color = MaterialTheme.colorScheme.primary)

@Composable
private fun EtapaIdentificacao(c: CadastroPropriedade, vm: FormularioViewModel, erro: (String) -> String?) {
    val cap = KeyboardOptions(capitalization = KeyboardCapitalization.Words)
    OutlinedTextField(c.nome, { v -> vm.alterar { it.copy(nome = v.take(200)) } }, Modifier.fillMaxWidth(), label = { Text(stringResource(R.string.campo_nome_obrigatorio)) },
        singleLine = true, keyboardOptions = cap, isError = erro("nome") != null, supportingText = erro("nome")?.let { { Text(it) } })
    OutlinedTextField(c.responsavel, { v -> vm.alterar { it.copy(responsavel = v.take(200)) } }, Modifier.fillMaxWidth(), label = { Text(stringResource(R.string.campo_responsavel)) },
        singleLine = true, keyboardOptions = cap, isError = erro("responsavel") != null, supportingText = erro("responsavel")?.let { { Text(it) } })
    Spacer(Modifier.height(4.dp))
    Subtitulo(stringResource(R.string.secao_pessoas))
    Quantidades(c, vm, erro, Quantitativo.entries.filter { it.grupo == "Pessoas" })
}

@Composable
private fun Quantidades(c: CadastroPropriedade, vm: FormularioViewModel, erro: (String) -> String?, qs: List<Quantitativo>) {
    qs.forEach { q -> CampoQuantidade(q.rotulo, c.quantidade(q), { v -> vm.alterar { it.copy(quantidades = it.quantidades + (q.campo to v)) } }, q.ajuda, erro = erro(q.campo)) }
}

@Composable
private fun EtapaAnimaisArea(c: CadastroPropriedade, vm: FormularioViewModel, erro: (String) -> String?) {
    Subtitulo(stringResource(R.string.secao_bovinos))
    Quantidades(c, vm, erro, Quantitativo.entries.filter { it.grupo == "Bovinos" })
    Subtitulo(stringResource(R.string.secao_equinos))
    Quantidades(c, vm, erro, Quantitativo.entries.filter { it.grupo == "Equinos" })
    SecaoItens(Categoria.AVE, c, vm, emptyMap(), aberta = true)
    Subtitulo(stringResource(R.string.secao_area))
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

@Composable
private fun EtapaLocalFotos(
    c: CadastroPropriedade, vm: FormularioViewModel, erro: (String) -> String?, processandoFoto: Boolean,
    abrirGps: () -> Unit, abrirManual: () -> Unit, abrirCamera: () -> Unit, remover: (String) -> Unit,
) {
    Subtitulo(stringResource(R.string.secao_localizacao))
    c.coordenada?.let { g ->
        Text("%.6f, %.6f".format(g.lat, g.lon), style = MaterialTheme.typography.titleMedium)
        Text(
            if (g.origem == "manual") stringResource(R.string.gps_manual) else stringResource(R.string.gps_precisao_satelites, g.precisaoM ?: 0.0, g.satelites ?: 0),
            style = MaterialTheme.typography.bodyMedium,
        )
    } ?: Text(stringResource(R.string.gps_nao_informado))
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Button(onClick = abrirGps, modifier = Modifier.height(48.dp)) { Icon(painterResource(R.drawable.ic_place), null); Spacer(Modifier.width(6.dp)); Text(stringResource(R.string.gps_obter)) }
    }
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        OutlinedButton(onClick = abrirManual, modifier = Modifier.height(48.dp)) {
            Text(stringResource(if (c.coordenada == null) R.string.gps_digitar else R.string.gps_editar))
        }
        if (c.coordenada != null) TextButton(onClick = { vm.alterar { it.copy(coordenada = null) } }) { Text(stringResource(R.string.remover)) }
    }
    erro("coordenada")?.let { Text(it, color = MaterialTheme.colorScheme.error) }
    Spacer(Modifier.height(8.dp))
    Subtitulo(stringResource(R.string.secao_fotos) + " (${c.fotos.size})")
    c.fotos.chunked(3).forEach { linha ->
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            linha.forEach { f ->
                Box(Modifier.weight(1f)) {
                    Miniatura(vm.arquivo(f.caminhoRelativo), Modifier.fillMaxWidth())
                    IconButton(onClick = { remover(f.uuid) }, modifier = Modifier.align(Alignment.TopEnd)) {
                        Surface(shape = MaterialTheme.shapes.small, color = MaterialTheme.colorScheme.surface.copy(alpha = 0.85f)) {
                            Icon(painterResource(R.drawable.ic_delete), stringResource(R.string.foto_remover), Modifier.padding(4.dp))
                        }
                    }
                }
            }
            repeat(3 - linha.size) { Spacer(Modifier.weight(1f)) }
        }
    }
    OutlinedButton(onClick = abrirCamera, enabled = !processandoFoto, modifier = Modifier.fillMaxWidth().height(56.dp)) {
        if (processandoFoto) { CircularProgressIndicator(Modifier.size(20.dp)); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.foto_processando)) }
        else { Icon(painterResource(R.drawable.ic_camera), null); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.foto_tirar)) }
    }
}

@Composable
private fun EtapaObservacoes(c: CadastroPropriedade, vm: FormularioViewModel, erro: (String) -> String?) {
    Text(stringResource(R.string.observacoes_ajuda), style = MaterialTheme.typography.bodyLarge)
    val n = contarCaracteres(c.observacoes)
    OutlinedTextField(
        c.observacoes,
        { v -> vm.alterar { it.copy(observacoes = v) } },
        Modifier.fillMaxWidth(),
        label = { Text(stringResource(R.string.observacoes_campo)) },
        minLines = 6, maxLines = 14,
        keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences),
        isError = erro("observacoes") != null || n > ValidacaoCadastro.MAX_OBSERVACOES,
        supportingText = {
            Row(Modifier.fillMaxWidth()) {
                Text(erro("observacoes").orEmpty(), Modifier.weight(1f))
                Text(stringResource(R.string.observacoes_contador, n, ValidacaoCadastro.MAX_OBSERVACOES))
            }
        },
    )
}

/** Revisão: resumo por categoria, só com o que foi preenchido, cada um com "Editar" (A28). */
@Composable
private fun Revisao(c: CadastroPropriedade, erros: List<ErroCampo>, vm: FormularioViewModel) {
    if (erros.isNotEmpty()) {
        Card(Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.errorContainer, contentColor = MaterialTheme.colorScheme.onErrorContainer)) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(stringResource(R.string.form_obrigatorios_faltando), style = MaterialTheme.typography.titleMedium)
                erros.forEach { er ->
                    Row(Modifier.fillMaxWidth().clickable { vm.irPara(Etapa.doCampo(er.campo, c)) }.padding(vertical = 4.dp)) {
                        Text("• ${er.mensagem}", Modifier.weight(1f))
                        Text(stringResource(Etapa.doCampo(er.campo, c).titulo), style = MaterialTheme.typography.labelLarge)
                    }
                }
            }
        }
    }
    fun soma(vararg q: Quantitativo) = q.sumOf { c.quantidade(it) }
    fun linhasQ(grupo: String) = Quantitativo.entries.filter { it.grupo == grupo && c.quantidade(it) > 0 }.map { "${it.rotulo}: ${c.quantidade(it)}" }
    fun linhasItens(cat: Categoria) = c.itens.filter { Catalogo.item(it.codigo)?.categoria == cat }.map { i ->
        val nome = Catalogo.item(i.codigo)?.let { if (it.outros) i.nomePersonalizado.orEmpty() else it.nome }.orEmpty()
        "$nome: ${i.quantidade}"
    }
    val nenhum = stringResource(R.string.revisao_nenhum)

    CartaoRevisao(stringResource(Etapa.IDENTIFICACAO.titulo), { vm.irPara(Etapa.IDENTIFICACAO) }, listOf(
        c.nome.ifBlank { "— " + stringResource(R.string.campo_nome_obrigatorio) },
        c.responsavel.ifBlank { "— " + stringResource(R.string.campo_responsavel) },
        stringResource(R.string.revisao_pessoas, soma(Quantitativo.PESSOAS_ADULTOS, Quantitativo.PESSOAS_JOVENS, Quantitativo.PESSOAS_CRIANCAS).toInt(),
            c.quantidade(Quantitativo.PESSOAS_ADULTOS).toInt(), c.quantidade(Quantitativo.PESSOAS_JOVENS).toInt(), c.quantidade(Quantitativo.PESSOAS_CRIANCAS).toInt()),
    ))
    CartaoRevisao(stringResource(Etapa.VEICULOS.titulo), { vm.irPara(Etapa.VEICULOS) }, linhasQ("Veículos").ifEmpty { listOf(nenhum) })
    CartaoRevisao(stringResource(Etapa.BENS.titulo), { vm.irPara(Etapa.BENS) },
        (linhasItens(Categoria.ELETRODOMESTICO) + linhasItens(Categoria.MOVEL)).ifEmpty { listOf(nenhum) })
    val area = (Areas.interpretar(c.areaTexto, c.areaUnidade) as? ResultadoArea.Valida)?.let {
        "${stringResource(R.string.secao_area)}: ${c.areaTexto} ${stringResource(if (c.areaUnidade == "ha") R.string.unidade_ha else R.string.unidade_alq)} (${Areas.formatarHa(it.area.dm2)} ha)"
    } ?: stringResource(R.string.area_nao_informada)
    CartaoRevisao(stringResource(Etapa.ANIMAIS_AREA.titulo), { vm.irPara(Etapa.ANIMAIS_AREA) },
        (linhasQ("Bovinos") + linhasQ("Equinos") + linhasItens(Categoria.AVE)).ifEmpty { listOf(nenhum) } + area)
    CartaoRevisao(stringResource(Etapa.LOCAL_FOTOS.titulo), { vm.irPara(Etapa.LOCAL_FOTOS) }, listOf(
        c.coordenada?.let { "%.6f, %.6f".format(it.lat, it.lon) } ?: stringResource(R.string.gps_nao_informado),
        stringResource(R.string.revisao_fotos, c.fotos.size),
    ))
    CartaoRevisao(stringResource(Etapa.OBSERVACOES.titulo), { vm.irPara(Etapa.OBSERVACOES) }, listOf(c.observacoes.trim().ifBlank { stringResource(R.string.revisao_sem_observacoes) }))
}

@Composable
private fun CartaoRevisao(titulo: String, aoEditar: () -> Unit, linhas: List<String>) {
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(start = 16.dp, end = 8.dp, top = 8.dp, bottom = 12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(titulo, style = MaterialTheme.typography.titleMedium, modifier = Modifier.weight(1f))
                TextButton(onClick = aoEditar) { Text(stringResource(R.string.editar)) }
            }
            linhas.forEach { Text(it, style = MaterialTheme.typography.bodyLarge) }
        }
    }
}

/** Itens dinâmicos de uma categoria (RN-30..34): chips do catálogo + vários "Outros". */
@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun SecaoItens(cat: Categoria, c: CadastroPropriedade, vm: FormularioViewModel, erros: Map<String, String>, aberta: Boolean = true) {
    val indices = c.itens.indices.filter { Catalogo.item(c.itens[it].codigo)?.categoria == cat }
    var novoOutros by rememberSaveable(cat) { mutableStateOf<String?>(null) }
    var somarCom by rememberSaveable(cat) { mutableStateOf<Int?>(null) }
    Secao(cat.titulo, resumo = indices.sumOf { c.itens[it].quantidade }.toString(), aberta = aberta) {
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
