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
    val responsavel: String,
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
    /** Observações relevantes já normalizadas (null = vazio; RN-15). */
    val observacoes: String? = null,
)

/** Resultado de informar um código de equipe (RN-07). */
sealed interface InicioColeta {
    val coleta: ColetaEntidade

    /** Coleta nova, em andamento. */
    data class Nova(override val coleta: ColetaEntidade) : InicioColeta

    /** Já existe coleta com este código (em andamento ou concluída): nenhuma foi criada. */
    data class Existente(override val coleta: ColetaEntidade) : InicioColeta
}

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

    // ---------- instalação e coletas (RN-03, RN-05, RN-07, RN-08) ----------

    /** UUID aleatório da instalação (criado na primeira vez; sem ID de hardware, RN-03). */
    suspend fun garantirInstalacao(): String = db.withWriteTransaction {
        db.metaDao().ler(META_INSTALACAO)?.valor ?: novoUuid().also {
            val t = agora()
            db.metaDao().gravar(Meta(META_INSTALACAO, it, t, t))
        }
    }

    suspend fun instalacaoUuid(): String = db.metaDao().ler(META_INSTALACAO)?.valor ?: throw ErroGravacao("Instalação não configurada.")

    fun observarColetas(): Flow<List<ColetaResumo>> = db.coletaDao().observarResumos()

    fun observarColeta(uuid: String): Flow<ColetaEntidade?> = db.coletaDao().observar(uuid)

    suspend fun coleta(uuid: String): ColetaEntidade? = db.coletaDao().obter(uuid)

    suspend fun coletasEmAndamento(): List<ColetaEntidade> = db.coletaDao().emAndamento()

    suspend fun historicoCodigos(coleta: String): List<ColetaCodigoEntidade> = db.coletaDao().historico(coleta)

    /** Coleta escolhida nesta sessão de uso (a tela decide quando perguntar; RN-07). */
    suspend fun coletaAtual(): String? = db.metaDao().ler(META_COLETA_ATUAL)?.valor?.ifEmpty { null }

    suspend fun definirColetaAtual(uuid: String?) {
        val t = agora()
        db.metaDao().gravar(Meta(META_COLETA_ATUAL, uuid ?: "", t, t))
    }

    private fun conferirCodigo(codigo: String) {
        if (!REGEX_CODIGO.matches(codigo)) throw ErroGravacao("Código de equipe inválido.")
    }

    /**
     * Informar um código (já normalizado e com o dígito conferido pelo domínio): se já existe
     * coleta com ele, devolve essa coleta (a tela oferece continuar ou reabrir), sem duplicar;
     * senão cria uma coleta nova, em andamento.
     */
    suspend fun iniciarColeta(codigo: String): InicioColeta = db.withWriteTransaction {
        conferirCodigo(codigo)
        db.coletaDao().porCodigo(codigo).firstOrNull()?.let { return@withWriteTransaction InicioColeta.Existente(it) }
        if (db.metaDao().ler(META_INSTALACAO) == null) db.metaDao().gravar(Meta(META_INSTALACAO, novoUuid(), agora(), agora()))
        val t = agora()
        val c = ColetaEntidade(novoUuid(), codigo, codigo, StatusColeta.EM_ANDAMENTO, iniciadaEm = t, createdAt = t, updatedAt = t, rev = 1)
        db.coletaDao().inserir(c)
        InicioColeta.Nova(c)
    }

    /** Concluir: a coleta deixa de aceitar cadastros novos (RN-07). Estado local, sem rev. */
    suspend fun concluirColeta(uuid: String): ColetaEntidade = mudarStatus(uuid, StatusColeta.CONCLUIDA)

    /** Reabrir uma coleta concluída por engano (RN-07). */
    suspend fun reabrirColeta(uuid: String): ColetaEntidade = mudarStatus(uuid, StatusColeta.EM_ANDAMENTO)

    private suspend fun mudarStatus(uuid: String, status: String): ColetaEntidade = db.withWriteTransaction {
        val c = db.coletaDao().obter(uuid) ?: throw ErroGravacao("Coleta não encontrada.")
        if (c.status == status) return@withWriteTransaction c
        val t = agora()
        val nova = c.copy(status = status, concluidaEm = if (status == StatusColeta.CONCLUIDA) t else null, updatedAt = t)
        db.coletaDao().atualizar(nova)
        nova
    }

    /**
     * Corrige o código de uma coleta (RN-08): só sem destino confirmado e sem envio aberto.
     * Um lote só preparado (nunca saiu do aparelho) é descartado; os demais exigem sincronizar
     * antes, para reconciliar com a Central. Nenhum dado dos cadastros muda.
     */
    suspend fun corrigirCodigo(uuid: String, novoCodigo: String): ColetaEntidade = db.withWriteTransaction {
        conferirCodigo(novoCodigo)
        val c = db.coletaDao().obter(uuid) ?: throw ErroGravacao("Coleta não encontrada.")
        if (c.destinoConfirmadoEm != null) throw ErroGravacao("Esta coleta já foi importada pela Central; o código não pode mais ser alterado.")
        if (novoCodigo == c.codigoEquipe) return@withWriteTransaction c
        val t = agora()
        db.loteDao().abertoDa(uuid)?.let { l ->
            if (l.status != StatusLote.PREPARADO) {
                throw ErroGravacao("Há um envio desta coleta em andamento. Sincronize de novo para concluir ou cancelar o envio antes de corrigir o código.")
            }
            db.loteDao().mudarStatus(l.uuid, StatusLote.DESCARTADO, "Descartado ao corrigir o código da equipe.", t)
        }
        db.coletaDao().registrarCodigo(ColetaCodigoEntidade(novoUuid(), uuid, c.codigoEquipe, novoCodigo, t))
        val nova = c.copy(codigoEquipe = novoCodigo, updatedAt = t, rev = c.rev + 1)
        db.coletaDao().atualizar(nova)
        nova
    }

    // ---------- propriedades ----------

    /** Propriedades vigentes de uma coleta (null = de todas). */
    fun observarPropriedades(coleta: String? = null): Flow<List<PropriedadeResumo>> = db.propriedadeDao().observarVigentes(coleta)

    fun observarContagens(coleta: String? = null): Flow<Contagens> = db.loteDao().observarContagens(coleta)

    fun observarUltimaConfirmacao(): Flow<Long?> = db.loteDao().observarUltimaConfirmacao()

    suspend fun carregar(uuid: String): PropriedadeCompleta? {
        val p = db.propriedadeDao().obter(uuid)?.takeIf { it.excluidoEm == null } ?: return null
        return PropriedadeCompleta(p, db.itemDao().vigentesDa(uuid), db.fotoDao().vigentesDa(uuid))
    }

    /**
     * Grava o cadastro completo numa transação e apaga o rascunho correspondente.
     * Cadastro novo exige [coletaUuid] em andamento (RN-07); um cadastro existente continua na
     * coleta dele, mesmo concluída. Devolve o UUID da propriedade. Só incrementa `rev` do que mudou.
     */
    suspend fun salvar(g: GravacaoPropriedade, coletaUuid: String, rascunhoUuid: String? = null): String = db.withWriteTransaction {
        if (g.nome.isBlank()) throw ErroGravacao("Informe o nome da propriedade.")
        if (g.responsavel.isBlank()) throw ErroGravacao("Informe o nome do responsável pela propriedade.")
        if (g.observacoes != null && (g.observacoes.isBlank() || g.observacoes.codePointCount(0, g.observacoes.length) > MAX_OBSERVACOES)) {
            throw ErroGravacao("Observações relevantes com mais de $MAX_OBSERVACOES caracteres.")
        }
        if ((g.areaMilesimos == null) != (g.areaUnidade == null)) throw ErroGravacao("Área e unidade devem vir juntas.")
        if ((g.lat == null) != (g.lon == null) || (g.lat == null) != (g.gpsOrigem == null)) throw ErroGravacao("Coordenada incompleta.")
        if (g.quantidades.values.any { it < 0 }) throw ErroGravacao("Quantidade negativa.")
        val t = agora()
        val q = { campo: String -> g.quantidades[campo] ?: 0L }

        val existente = g.uuid?.let { db.propriedadeDao().obter(it) }
        if (g.uuid != null && (existente == null || existente.excluidoEm != null)) throw ErroGravacao("Cadastro não encontrado.")
        if (existente == null) {
            val coleta = db.coletaDao().obter(coletaUuid) ?: throw ErroGravacao("Informe o código da equipe antes de cadastrar.")
            if (coleta.status != StatusColeta.EM_ANDAMENTO) throw ErroGravacao("Esta coleta foi concluída e não aceita cadastros novos. Reabra a coleta ou informe outro código.")
        }
        val uuid = existente?.uuid ?: novoUuid()
        val base = existente ?: PropriedadeEntidade(uuid = uuid, coletaUuid = coletaUuid, nomePropriedade = "", responsavelFamilia = "", createdAt = t, updatedAt = t, rev = 0)
        val desejada = base.copy(
            nomePropriedade = g.nome.trim(), responsavelFamilia = g.responsavel.trim(), observacoesRelevantes = g.observacoes,
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

    suspend fun gravarRascunho(uuid: String, coletaUuid: String, propriedadeUuid: String?, json: String) =
        db.rascunhoDao().gravar(RascunhoEntidade(uuid, coletaUuid, propriedadeUuid, json, agora()))

    /** Rascunhos de uma coleta (null = de todas). */
    suspend fun rascunhos(coleta: String? = null): List<RascunhoEntidade> =
        if (coleta == null) db.rascunhoDao().todos() else db.rascunhoDao().daColeta(coleta)

    suspend fun rascunho(uuid: String): RascunhoEntidade? = db.rascunhoDao().obter(uuid)

    suspend fun descartarRascunho(uuid: String) = db.rascunhoDao().apagar(uuid)

    /** Caminhos de foto referenciados por registros ou por rascunhos (para o reconciliador). */
    suspend fun caminhosFotosRegistradas(): Set<String> = db.fotoDao().todas().map { it.caminhoRelativo }.toSet()

    suspend fun fotosRegistradas(): List<FotoEntidade> = db.fotoDao().todas()

    companion object {
        const val META_INSTALACAO = "instalacao_uuid"
        const val META_COLETA_ATUAL = "coleta_atual"
        const val MAX_OBSERVACOES = 4000

        /** Formato normalizado do código de equipe (o dígito é conferido no domínio, `Codigos`). */
        val REGEX_CODIGO = Regex("^EQ[0-9A-HJKMNP-TV-Z]{8}$")
    }
}
