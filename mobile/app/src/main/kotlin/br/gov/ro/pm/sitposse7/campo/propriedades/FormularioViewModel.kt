package br.gov.ro.pm.sitposse7.campo.propriedades

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import br.gov.ro.pm.sitposse7.campo.CampoApp
import br.gov.ro.pm.sitposse7.campo.fotos.ProcessamentoFoto
import br.gov.ro.pm.sitposse7.dados.GravacaoFoto
import br.gov.ro.pm.sitposse7.dados.GravacaoItem
import br.gov.ro.pm.sitposse7.dados.GravacaoPropriedade
import br.gov.ro.pm.sitposse7.dominio.Areas
import br.gov.ro.pm.sitposse7.dominio.CadastroPropriedade
import br.gov.ro.pm.sitposse7.dominio.Catalogo
import br.gov.ro.pm.sitposse7.dominio.Coordenada
import br.gov.ro.pm.sitposse7.dominio.ErroCampo
import br.gov.ro.pm.sitposse7.dominio.FotoCadastro
import br.gov.ro.pm.sitposse7.dominio.ItemCadastro
import br.gov.ro.pm.sitposse7.dominio.Normalizacao
import br.gov.ro.pm.sitposse7.dominio.ResultadoArea
import br.gov.ro.pm.sitposse7.dominio.ValidacaoCadastro
import br.gov.ro.pm.sitposse7.dominio.textoArea
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import java.io.File

data class EstadoFormulario(
    val carregado: Boolean = false,
    val cadastro: CadastroPropriedade = CadastroPropriedade(),
    val erros: List<ErroCampo> = emptyList(),
    val salvando: Boolean = false,
    val processandoFoto: Boolean = false,
    /** Mensagem de falha (o formulário continua preenchido). */
    val falha: String? = null,
    /** UUID gravado: só existe depois do commit (AGENTS.md §3). */
    val salvoComo: String? = null,
    /** Já foi modificado nesta sessão (para o banner de rascunho). */
    val modificado: Boolean = false,
)

/**
 * Formulário de cadastro (RF-A03..A10). O estado completo também é o rascunho, gravado
 * na tabela `rascunhos` cerca de 1 s após cada mudança (sobrevive à morte do processo).
 */
class FormularioViewModel(app: Application, private val uuidPropriedade: String?, private val rascunhoUuid: String) : AndroidViewModel(app) {
    private val campo = app as CampoApp
    private val json = Json { ignoreUnknownKeys = true }
    private val _estado = MutableStateFlow(EstadoFormulario())
    val estado: StateFlow<EstadoFormulario> = _estado
    private var gravacaoRascunho: Job? = null

    init {
        viewModelScope.launch {
            val doRascunho = campo.repo.rascunho(rascunhoUuid)?.let { runCatching { json.decodeFromString(CadastroPropriedade.serializer(), it.json) }.getOrNull() }
            val inicial = doRascunho ?: uuidPropriedade?.let { carregarExistente(it) } ?: CadastroPropriedade()
            _estado.update { it.copy(carregado = true, cadastro = inicial, modificado = doRascunho != null) }
        }
    }

    private suspend fun carregarExistente(uuid: String): CadastroPropriedade? {
        val c = campo.repo.carregar(uuid) ?: return null
        val p = c.propriedade
        return CadastroPropriedade(
            uuid = p.uuid, nome = p.nomePropriedade, responsavel = p.responsavelFamilia.orEmpty(),
            quantidades = mapOf(
                "pessoas_adultos" to p.pessoasAdultos, "pessoas_jovens" to p.pessoasJovens, "pessoas_criancas" to p.pessoasCriancas,
                "veic_caminhoes" to p.veicCaminhoes, "veic_automoveis_utilitarios" to p.veicAutomoveisUtilitarios, "veic_motos_ciclomotores" to p.veicMotosCiclomotores,
                "bov_adultos" to p.bovAdultos, "bov_bezerros" to p.bovBezerros, "equ_adultos" to p.equAdultos, "equ_potros" to p.equPotros,
            ),
            areaTexto = p.areaValorMilesimos?.let(::textoArea).orEmpty(), areaUnidade = p.areaUnidade,
            coordenada = if (p.lat != null && p.lon != null) Coordenada(p.lat!!, p.lon!!, p.gpsOrigem ?: "manual", p.gpsPrecisaoM, p.gpsCapturadoEm, p.gpsSatelites) else null,
            itens = c.itens.map { ItemCadastro(it.uuid, it.codigoCatalogo, it.nomePersonalizado, it.quantidade) },
            fotos = c.fotos.map { FotoCadastro(it.uuid, it.caminhoRelativo, it.tamanhoBytes, it.sha256, it.largura, it.altura, it.capturadaEm) },
        )
    }

    /** Toda edição passa por aqui: atualiza o estado e agenda o rascunho. */
    fun alterar(f: (CadastroPropriedade) -> CadastroPropriedade) {
        _estado.update { e ->
            val novo = f(e.cadastro)
            e.copy(cadastro = novo, modificado = true, falha = null, erros = if (e.erros.isEmpty()) e.erros else ValidacaoCadastro.validar(novo))
        }
        agendarRascunho()
    }

    private fun agendarRascunho() {
        gravacaoRascunho?.cancel()
        gravacaoRascunho = viewModelScope.launch {
            delay(1000)
            gravarRascunhoAgora()
        }
    }

    private suspend fun gravarRascunhoAgora() {
        val c = _estado.value.cadastro
        if (_estado.value.salvoComo != null) return
        runCatching { campo.repo.gravarRascunho(rascunhoUuid, c.uuid, json.encodeToString(CadastroPropriedade.serializer(), c)) }
    }

    fun salvar() {
        val c = _estado.value.cadastro
        val erros = ValidacaoCadastro.validar(c)
        if (erros.isNotEmpty()) {
            _estado.update { it.copy(erros = erros) }
            return
        }
        _estado.update { it.copy(salvando = true, erros = emptyList(), falha = null) }
        viewModelScope.launch {
            gravacaoRascunho?.cancel()
            val area = Areas.interpretar(c.areaTexto, c.areaUnidade) as? ResultadoArea.Valida
            val g = GravacaoPropriedade(
                uuid = c.uuid, nome = c.nome, responsavel = c.responsavel, quantidades = c.quantidades,
                areaMilesimos = area?.area?.milesimos, areaUnidade = area?.area?.unidade?.codigo,
                lat = c.coordenada?.lat, lon = c.coordenada?.lon, gpsOrigem = c.coordenada?.origem,
                gpsPrecisaoM = c.coordenada?.precisaoM, gpsCapturadoEm = c.coordenada?.capturadoEm, gpsSatelites = c.coordenada?.satelites,
                itens = c.itens.map { i ->
                    val cat = Catalogo.item(i.codigo)!!
                    GravacaoItem(i.uuid, cat.categoria.codigo, i.codigo, i.nomePersonalizado?.takeIf { cat.outros }, i.nomePersonalizado?.takeIf { cat.outros }?.let(Normalizacao::normalizarNome), i.quantidade)
                },
                fotos = c.fotos.map { GravacaoFoto(it.uuid, it.caminhoRelativo, it.tamanhoBytes, it.sha256, it.largura, it.altura, it.capturadaEm) },
            )
            runCatching { campo.repo.salvar(g, rascunhoUuid) }
                .onSuccess { uuid -> _estado.update { it.copy(salvando = false, salvoComo = uuid) } }
                .onFailure { t ->
                    gravarRascunhoAgora() // o formulário nunca se perde
                    _estado.update { it.copy(salvando = false, falha = t.message ?: t::class.simpleName) }
                }
        }
    }

    fun descartar(aoTerminar: () -> Unit) {
        gravacaoRascunho?.cancel()
        _estado.update { it.copy(modificado = false) }
        viewModelScope.launch { campo.repo.descartarRascunho(rascunhoUuid); aoTerminar() }
    }

    /** Processa a captura (redução + gravação durável) e anexa ao formulário. */
    fun adicionarFoto(captura: File) {
        _estado.update { it.copy(processandoFoto = true) }
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { ProcessamentoFoto.processar(captura, campo.fotos, System.currentTimeMillis()) } }
                .onSuccess { f ->
                    alterar { c -> c.copy(fotos = c.fotos + FotoCadastro(f.gravada.uuid, f.gravada.caminhoRelativo, f.gravada.tamanhoBytes, f.gravada.sha256, f.largura.toLong(), f.altura.toLong(), f.capturadaEm)) }
                    gravacaoRascunho?.cancel(); gravarRascunhoAgora() // a foto já está no disco: guardar a referência logo
                }
                .onFailure { t -> _estado.update { it.copy(falha = t.message) } }
            _estado.update { it.copy(processandoFoto = false) }
        }
    }

    fun pastaCaptura(): File = campo.fotos.tmp

    fun espacoParaFoto(): Boolean = campo.fotos.espacoSuficiente()

    fun arquivo(caminho: String): File = campo.fotos.arquivo(caminho)

    override fun onCleared() {
        // Garante o rascunho mesmo se a tela fechar antes do intervalo de 1 s.
        if (_estado.value.modificado && _estado.value.salvoComo == null) campo.escopo.launch { gravarRascunhoAgora() }
        super.onCleared()
    }
}
