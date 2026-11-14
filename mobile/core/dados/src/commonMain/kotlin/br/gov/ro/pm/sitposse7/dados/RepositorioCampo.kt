package br.gov.ro.pm.sitposse7.dados

import androidx.room3.withWriteTransaction
import kotlinx.coroutines.flow.Flow
import kotlin.uuid.ExperimentalUuidApi
import kotlin.uuid.Uuid

/** Violação de regra detectada na gravação; nada é gravado (a transação é desfeita). */
class ErroGravacao(mensagem: String) : Exception(mensagem)

data class GravacaoItem(
    val uuid: String?,
    val categoria: String,
    val codigo: String,
    val nomePersonalizado: String?,
    val nomeNormalizado: String?,
    val quantidade: Long,
)

data class GravacaoFoto(
    val uuid: String,
    val caminhoRelativo: String,
    val tamanhoBytes: Long,
    val sha256: String,
    val largura: Long,
    val altura: Long,
    val capturadaEm: Long,
)

/** O que o formulário quer gravar (já validado pelo domínio; RN-11..55). */
data class GravacaoPropriedade(
    val uuid: String?,
    val nome: String,
    val responsavel: String?,
    val quantidades: Map<String, Long>,
    val areaMilesimos: Long?,
    val areaUnidade: String?,
    val lat: Double?,
    val lon: Double?,
    val gpsOrigem: String?,
    val gpsPrecisaoM: Double?,
    val gpsCapturadoEm: Long?,
    val gpsSatelites: Long?,
    val itens: List<GravacaoItem>,
    val fotos: List<GravacaoFoto>,
)

data class PropriedadeCompleta(val propriedade: PropriedadeEntidade, val itens: List<ItemEntidade>, val fotos: List<FotoEntidade>)

/**
 * Único ponto de escrita dos dados de campo (mobile/AGENTS.md):
 * - toda escrita roda numa transação e incrementa `rev` de cada entidade alterada;
 * - exclusão é tombstone (`excluido_em`), nunca DELETE (AGENTS.md §3);
 * - foto só é registrada se o arquivo já estiver gravado ([arquivoExiste]);
 * - o chamador só mostra "Salvo no aparelho" depois do retorno (commit).
 */
@OptIn(ExperimentalUuidApi::class)
class RepositorioCampo(
    private val db: BancoCampo,
    private val agora: () -> Long,
    private val arquivoExiste: (caminhoRelativo: String) -> Boolean,
) {
    private fun novoUuid() = Uuid.random().toString()

    // ---------- equipe e instalação ----------

    fun observarEquipe(): Flow<EquipeEntidade?> = db.equipeDao().observar()

    suspend fun equipe(): EquipeEntidade? = db.equipeDao().obter()

    /** RN-01..03: cria a equipe e o UUID da instalação (aleatório; sem ID de hardware). */
    suspend fun configurarEquipe(nome: String, responsavel: String, area: String?): EquipeEntidade = db.withWriteTransaction {
        if (nome.isBlank() || responsavel.isBlank()) throw ErroGravacao("Nome da equipe e responsável são obrigatórios.")
        if (db.equipeDao().obter() != null) throw ErroGravacao("A equipe já está configurada neste aparelho.")
        val t = agora()
        val e = EquipeEntidade(novoUuid(), nome.trim(), responsavel.trim(), area?.trim()?.ifEmpty { null }, t, t, rev = 1)
        db.equipeDao().inserir(e)
        if (db.metaDao().ler(META_INSTALACAO) == null) db.metaDao().gravar(Meta(META_INSTALACAO, novoUuid(), t, t))
        e
    }

    /** RN-02: editar não muda o UUID. */
    suspend fun editarEquipe(nome: String, responsavel: String, area: String?): EquipeEntidade = db.withWriteTransaction {
        if (nome.isBlank() || responsavel.isBlank()) throw ErroGravacao("Nome da equipe e responsável são obrigatórios.")
        val atual = db.equipeDao().obter() ?: throw ErroGravacao("Equipe não configurada.")
        val nova = atual.copy(nome = nome.trim(), responsavel = responsavel.trim(), identificacaoArea = area?.trim()?.ifEmpty { null })
        if (nova == atual) return@withWriteTransaction atual
        val gravada = nova.copy(updatedAt = agora(), rev = atual.rev + 1)
        db.equipeDao().atualizar(gravada)
        gravada
    }

    suspend fun instalacaoUuid(): String = db.metaDao().ler(META_INSTALACAO)?.valor ?: throw ErroGravacao("Instalação não configurada.")

    // ---------- propriedades ----------

    fun observarPropriedades(): Flow<List<PropriedadeResumo>> = db.propriedadeDao().observarVigentes()

    fun observarContagens(): Flow<Contagens> = db.loteDao().observarContagens()

    fun observarUltimaConfirmacao(): Flow<Long?> = db.loteDao().observarUltimaConfirmacao()

    suspend fun carregar(uuid: String): PropriedadeCompleta? {
        val p = db.propriedadeDao().obter(uuid)?.takeIf { it.excluidoEm == null } ?: return null
        return PropriedadeCompleta(p, db.itemDao().vigentesDa(uuid), db.fotoDao().vigentesDa(uuid))
    }

    /**
     * Grava o cadastro completo numa transação e apaga o rascunho correspondente.
     * Devolve o UUID da propriedade. Só incrementa `rev` do que mudou.
     */
    suspend fun salvar(g: GravacaoPropriedade, rascunhoUuid: String? = null): String = db.withWriteTransaction {
        if (g.nome.isBlank()) throw ErroGravacao("Informe o nome da propriedade.")
        if ((g.areaMilesimos == null) != (g.areaUnidade == null)) throw ErroGravacao("Área e unidade devem vir juntas.")
        if ((g.lat == null) != (g.lon == null) || (g.lat == null) != (g.gpsOrigem == null)) throw ErroGravacao("Coordenada incompleta.")
        if (g.quantidades.values.any { it < 0 }) throw ErroGravacao("Quantidade negativa.")
        val equipe = db.equipeDao().obter() ?: throw ErroGravacao("Configure a equipe antes de cadastrar.")
        val t = agora()
        val q = { campo: String -> g.quantidades[campo] ?: 0L }

        val existente = g.uuid?.let { db.propriedadeDao().obter(it) }
        if (g.uuid != null && (existente == null || existente.excluidoEm != null)) throw ErroGravacao("Cadastro não encontrado.")
        val uuid = existente?.uuid ?: novoUuid()
        val base = existente ?: PropriedadeEntidade(uuid = uuid, equipeUuid = equipe.uuid, nomePropriedade = "", responsavelFamilia = null, createdAt = t, updatedAt = t, rev = 0)
        val desejada = base.copy(
            nomePropriedade = g.nome.trim(), responsavelFamilia = g.responsavel?.trim()?.ifEmpty { null },
            pessoasAdultos = q("pessoas_adultos"), pessoasJovens = q("pessoas_jovens"), pessoasCriancas = q("pessoas_criancas"),
            veicCaminhoes = q("veic_caminhoes"), veicAutomoveisUtilitarios = q("veic_automoveis_utilitarios"), veicMotosCiclomotores = q("veic_motos_ciclomotores"),
            bovAdultos = q("bov_adultos"), bovBezerros = q("bov_bezerros"), equAdultos = q("equ_adultos"), equPotros = q("equ_potros"),
            areaValorMilesimos = g.areaMilesimos, areaUnidade = g.areaUnidade,
            lat = g.lat, lon = g.lon, gpsOrigem = g.gpsOrigem, gpsPrecisaoM = g.gpsPrecisaoM, gpsCapturadoEm = g.gpsCapturadoEm, gpsSatelites = g.gpsSatelites,
        )
        when {
            existente == null -> db.propriedadeDao().inserir(desejada.copy(rev = 1))
            desejada != existente -> db.propriedadeDao().atualizar(desejada.copy(updatedAt = t, rev = existente.rev + 1))
        }

        // Itens (RN-31..34): novos, alterados e removidos (tombstone).
        val atuais = if (existente == null) emptyList() else db.itemDao().vigentesDa(uuid)
        val porUuid = atuais.associateBy { it.uuid }
        val predefinidos = mutableSetOf<String>()
        for (it in g.itens) {
            if (it.quantidade < 0) throw ErroGravacao("Quantidade de item negativa.")
            if (!it.codigo.endsWith(".outros") && !predefinidos.add(it.codigo)) throw ErroGravacao("Item repetido: ${it.codigo}.") // RN-32
            if (it.codigo.endsWith(".outros") == it.nomePersonalizado.isNullOrBlank()) throw ErroGravacao("Nome de 'Outros' inválido.")
        }
        val mantidos = mutableSetOf<String>()
        for (it in g.itens) {
            val atual = it.uuid?.let { u -> porUuid[u] }
            if (atual == null) {
                db.itemDao().inserir(ItemEntidade(novoUuid(), uuid, it.categoria, it.codigo, it.nomePersonalizado, it.nomeNormalizado, it.quantidade, t, t, rev = 1))
            } else {
                mantidos += atual.uuid
                val novo = atual.copy(quantidade = it.quantidade, nomePersonalizado = it.nomePersonalizado, nomeNormalizado = it.nomeNormalizado)
                if (novo != atual) db.itemDao().atualizar(novo.copy(updatedAt = t, rev = atual.rev + 1))
            }
        }
        atuais.filter { it.uuid !in mantidos }.forEach { db.itemDao().atualizar(it.copy(excluidoEm = t, updatedAt = t, rev = it.rev + 1)) }

        // Fotos (RN-60..62): novas exigem o arquivo já gravado; removidas viram tombstone.
        val fotosAtuais = if (existente == null) emptyList() else db.fotoDao().vigentesDa(uuid)
        val fotosQueFicam = g.fotos.map { it.uuid }.toSet()
        for (f in g.fotos) {
            val atual = db.fotoDao().obter(f.uuid)
            if (atual != null) {
                if (atual.propriedadeUuid != uuid || atual.excluidoEm != null) throw ErroGravacao("Foto ${f.uuid} não pertence a este cadastro.")
                continue // foto é imutável
            }
            if (!arquivoExiste(f.caminhoRelativo)) throw ErroGravacao("Arquivo da foto não encontrado; tire a foto de novo.")
            db.fotoDao().inserir(FotoEntidade(f.uuid, uuid, f.caminhoRelativo, "image/jpeg", f.tamanhoBytes, f.sha256, f.largura, f.altura, f.capturadaEm, t, t, rev = 1))
        }
        fotosAtuais.filter { it.uuid !in fotosQueFicam }.forEach { db.fotoDao().atualizar(it.copy(excluidoEm = t, updatedAt = t, rev = it.rev + 1)) }

        rascunhoUuid?.let { db.rascunhoDao().apagar(it) }
        uuid
    }

    /** RN-14: exclusão é tombstone do cadastro, dos itens e das fotos. */
    suspend fun excluir(uuid: String) = db.withWriteTransaction {
        val p = db.propriedadeDao().obter(uuid)?.takeIf { it.excluidoEm == null } ?: throw ErroGravacao("Cadastro não encontrado.")
        val t = agora()
        db.propriedadeDao().atualizar(p.copy(excluidoEm = t, updatedAt = t, rev = p.rev + 1))
        db.itemDao().vigentesDa(uuid).forEach { db.itemDao().atualizar(it.copy(excluidoEm = t, updatedAt = t, rev = it.rev + 1)) }
        db.fotoDao().vigentesDa(uuid).forEach { db.fotoDao().atualizar(it.copy(excluidoEm = t, updatedAt = t, rev = it.rev + 1)) }
    }

    // ---------- rascunhos (nunca sincronizados nem contados) ----------

    suspend fun gravarRascunho(uuid: String, propriedadeUuid: String?, json: String) =
        db.rascunhoDao().gravar(RascunhoEntidade(uuid, propriedadeUuid, json, agora()))

    suspend fun rascunhos(): List<RascunhoEntidade> = db.rascunhoDao().todos()

    suspend fun rascunho(uuid: String): RascunhoEntidade? = db.rascunhoDao().obter(uuid)

    suspend fun descartarRascunho(uuid: String) = db.rascunhoDao().apagar(uuid)

    /** Caminhos de foto referenciados por registros ou por rascunhos (para o reconciliador). */
    suspend fun caminhosFotosRegistradas(): Set<String> = db.fotoDao().todas().map { it.caminhoRelativo }.toSet()

    suspend fun fotosRegistradas(): List<FotoEntidade> = db.fotoDao().todas()

    companion object {
        const val META_INSTALACAO = "instalacao_uuid"
    }
}
