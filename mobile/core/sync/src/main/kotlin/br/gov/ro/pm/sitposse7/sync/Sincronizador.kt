package br.gov.ro.pm.sitposse7.sync

import androidx.room3.withWriteTransaction
import br.gov.ro.pm.sitposse7.dados.BancoCampo
import br.gov.ro.pm.sitposse7.dados.ColetaEntidade
import br.gov.ro.pm.sitposse7.dados.FotoEntidade
import br.gov.ro.pm.sitposse7.dados.ItemEntidade
import br.gov.ro.pm.sitposse7.dados.LoteEnvioEntidade
import br.gov.ro.pm.sitposse7.dados.LoteItemEntidade
import br.gov.ro.pm.sitposse7.dados.PareamentoEntidade
import br.gov.ro.pm.sitposse7.dados.PropriedadeEntidade
import br.gov.ro.pm.sitposse7.dados.RepositorioCampo
import br.gov.ro.pm.sitposse7.dados.StatusLote
import br.gov.ro.pm.sitposse7.dominio.PayloadQr
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.builtins.serializer
import java.io.File
import java.io.RandomAccessFile
import java.util.UUID
import javax.net.SocketFactory
import kotlin.time.Duration
import kotlin.time.Duration.Companion.minutes
import kotlin.time.Duration.Companion.seconds
import kotlin.time.TimeSource

/** Etapas mostradas na tela de sincronização (mobile/docs/telas-e-fluxos.md). */
sealed interface Progresso {
    data object Conectando : Progresso
    data class EnviandoCadastros(val propriedades: Int) : Progresso
    data class EnviandoFotos(val feitas: Int, val total: Int, val bytesEnviados: Long) : Progresso
    data object ConferindoFotos : Progresso
    /** A Central recebeu tudo; o operador precisa confirmar o destino no computador (ADR-0024). */
    data object AguardandoConfirmacao : Progresso
}

/** Resultado de um envio confirmado (recibo gravado). */
data class ResumoEnvio(
    val lote: String,
    val propriedadesConfirmadas: Int,
    val aplicados: Int,
    val duplicados: Int,
    val conflitos: Int,
    val rejeitados: List<ResultadoReciboV2>,
    val destino: DestinoV2,
)

/** Desfecho do envio de uma coleta. */
sealed interface ResultadoColeta {
    val coleta: String

    /** Recibo gravado: cadastros confirmados e destino da coleta registrado. */
    data class Confirmado(override val coleta: String, val resumo: ResumoEnvio) : ResultadoColeta

    /** O operador cancelou na Central; nada foi importado nem apagado (o envio pode ser refeito). */
    data class Cancelado(override val coleta: String, val lote: String, val motivo: String?) : ResultadoColeta

    /** A Central recebeu tudo e o operador ainda não decidiu (a tela foi fechada ou o tempo de espera acabou). */
    data class Aguardando(override val coleta: String, val lote: String) : ResultadoColeta
}

/**
 * Envio das coletas à Central (protocolo.md §4–6, ADR-0024), com as garantias do AGENTS.md §5:
 * - um lote por coleta, congelado em transação (JSON + lista de (entidade, uuid, rev)) e reenviado byte a byte;
 * - fotos em blocos de 1 MiB, retomando do offset que a Central informa;
 * - depois das fotos, o lote espera o operador confirmar o destino; o estado é sempre perguntado à
 *   Central (PUT idempotente e GET), nunca presumido;
 * - só é "confirmado" o que consta no recibo **persistido**, gravado numa transação com o destino;
 * - lote cancelado na Central é terminal: os registros continuam pendentes para um lote novo;
 * - edições feitas durante o envio geram rev maior e ficam para o próximo lote.
 */
class Sincronizador(
    private val db: BancoCampo,
    private val agora: () -> Long,
    /** Arquivo de uma foto a partir do caminho relativo registrado. */
    private val arquivoDaFoto: (String) -> File,
) {
    private val repo = RepositorioCampo(db, agora) { true }

    /** Coletas com algo a enviar, ou com lote aberto (a reconciliar), na ordem em que começaram. */
    suspend fun coletasParaEnviar(): List<ColetaEntidade> {
        val comLote = db.loteDao().abertos().map { it.coletaUuid }.toSet()
        val pend = db.coletaDao().comPendencias()
        val extras = comLote.filter { u -> pend.none { it.uuid == u } }.mapNotNull { db.coletaDao().obter(it) }
        return (pend + extras).sortedWith(compareBy({ it.iniciadaEm }, { it.uuid }))
    }

    /** Devolve o lote aberto da coleta (para retomar) ou congela um novo; null se não há nada pendente. */
    suspend fun preparar(coletaUuid: String): LoteEnvioEntidade? = db.withWriteTransaction {
        db.loteDao().abertoDa(coletaUuid)?.let { return@withWriteTransaction it }
        val coleta = db.coletaDao().obter(coletaUuid) ?: throw FalhaCentral("coleta", "Coleta não encontrada.", fatal = true)
        val props = db.propriedadeDao().pendentesDa(coletaUuid)
        val itens = db.itemDao().pendentesDa(coletaUuid)
        val fotos = db.fotoDao().pendentesDa(coletaUuid)
        if (props.isEmpty() && itens.isEmpty() && fotos.isEmpty() && coleta.rev <= coleta.revConfirmada) return@withWriteTransaction null
        val t = agora()
        val uuid = UUID.randomUUID().toString()
        val lote = LoteV2(
            loteUuid = uuid, instalacaoUuid = repo.garantirInstalacao(), criadoEm = instante(t),
            coleta = coleta.v2(), propriedades = props.map { it.v2() }, itens = itens.map { it.v2() }, fotos = fotos.map { it.v2() },
        )
        val entidade = LoteEnvioEntidade(uuid, coletaUuid, StatusLote.PREPARADO, "wifi", JsonContratos.encodeToString(LoteV2.serializer(), lote), t, t)
        db.loteDao().inserir(entidade)
        db.loteDao().inserirItens(
            listOf(LoteItemEntidade(uuid, "coleta", coleta.uuid, coleta.rev)) +
                props.map { LoteItemEntidade(uuid, "propriedade", it.uuid, it.rev) } +
                itens.map { LoteItemEntidade(uuid, "item", it.uuid, it.rev) } +
                fotos.map { LoteItemEntidade(uuid, "foto", it.uuid, it.rev) }
        )
        entidade
    }

    /**
     * Envia a coleta (retomando o lote aberto, se houver) e acompanha a decisão do operador por
     * até [espera], consultando a Central a cada [intervalo]. Devolve null se não havia nada a enviar.
     * Em falha, o lote fica em erro (recuperável ou fatal) e é retomado na próxima vez; nada se perde.
     */
    suspend fun sincronizarColeta(
        api: ApiCentral,
        coletaUuid: String,
        progresso: (Progresso) -> Unit = {},
        espera: Duration = 15.minutes,
        intervalo: Duration = 2.seconds,
    ): ResultadoColeta? {
        val lote = preparar(coletaUuid) ?: return null
        if (lote.status == StatusLote.PREPARADO || lote.status == StatusLote.ERRO_RECUPERAVEL || lote.status == StatusLote.ERRO_FATAL) {
            db.loteDao().mudarStatus(lote.uuid, StatusLote.ENVIANDO, null, agora())
        }
        try {
            val payload = JsonContratos.decodeFromString(LoteV2.serializer(), lote.payloadJson)
            progresso(Progresso.EnviandoCadastros(payload.propriedades.size))
            // PUT idempotente: também é a reconciliação (devolve o estado atual na Central).
            var estado = try {
                io { api.enviarLote(lote.uuid, lote.payloadJson) }
            } catch (f: FalhaCentral) {
                if (f.codigo != "lote_cancelado") throw f
                io { api.estadoLote(lote.uuid) } ?: throw f
            }
            var rodadasFotos = 0
            val limite = TimeSource.Monotonic.markNow() + espera
            while (true) {
                when (estado.status) {
                    StatusCentralLote.IMPORTADO -> {
                        val recibo = estado.recibo ?: throw FalhaCentral("recibo", "A Central não enviou o recibo. Tente de novo.", fatal = false)
                        if (!recibo.confere(lote.uuid)) throw FalhaCentral("recibo", "O recibo da Central veio corrompido. Tente de novo.", fatal = false)
                        val resumo = gravarRecibo(lote.uuid, coletaUuid, recibo)
                        io { api.ack(lote.uuid) }
                        return ResultadoColeta.Confirmado(coletaUuid, resumo)
                    }
                    StatusCentralLote.CANCELADO -> {
                        db.loteDao().mudarStatus(lote.uuid, StatusLote.CANCELADO, mensagemCancelamento(estado.motivo), agora())
                        return ResultadoColeta.Cancelado(coletaUuid, lote.uuid, estado.motivo)
                    }
                    StatusCentralLote.AGUARDANDO -> {
                        db.loteDao().mudarStatus(lote.uuid, StatusLote.AGUARDANDO, null, agora())
                        progresso(Progresso.AguardandoConfirmacao)
                        if (limite.hasPassedNow()) return ResultadoColeta.Aguardando(coletaUuid, lote.uuid)
                        delay(intervalo)
                        estado = io { api.estadoLote(lote.uuid) }
                            ?: throw FalhaCentral("lote", "A Central não encontrou este envio. Sincronize de novo.", fatal = false)
                    }
                    else -> { // recebendo: faltam fotos (ou a Central pediu de novo)
                        if (++rodadasFotos > 4) throw FalhaCentral("fotos", "A Central não conseguiu conferir as fotos. Tente de novo.", fatal = false)
                        db.loteDao().mudarStatus(lote.uuid, StatusLote.ENVIANDO, null, agora())
                        enviarFotos(api, estado.fotosFaltantes, payload.fotos.associateBy { it.uuid }, progresso)
                        progresso(Progresso.ConferindoFotos)
                        estado = when (val r = io { api.importar(lote.uuid) }) {
                            is ApiCentral.ResultadoImportacao.Recebido -> r.estado
                            is ApiCentral.ResultadoImportacao.FaltamFotos -> r.estado
                        }
                    }
                }
            }
        } catch (f: FalhaCentral) {
            if (f.codigo == "lote_cancelado") {
                db.loteDao().mudarStatus(lote.uuid, StatusLote.CANCELADO, mensagemCancelamento(null), agora())
                return ResultadoColeta.Cancelado(coletaUuid, lote.uuid, null)
            }
            db.loteDao().mudarStatus(lote.uuid, if (f.fatal) StatusLote.ERRO_FATAL else StatusLote.ERRO_RECUPERAVEL, f.message, agora())
            throw f
        } catch (c: CancellationException) {
            throw c // o lote fica como está ("enviando" ou "aguardando") e é reconciliado na próxima vez
        } catch (e: Exception) {
            // Falha não prevista: o lote continua aberto para nova tentativa, com a causa registrada.
            runCatching { db.loteDao().mudarStatus(lote.uuid, StatusLote.ERRO_RECUPERAVEL, "Erro inesperado: ${e::class.java.simpleName}", agora()) }
            throw e
        }
    }

    private fun mensagemCancelamento(motivo: String?) = when (motivo) {
        "codigo_inexistente" -> "A Central não encontrou o código da equipe. Corrija o código e sincronize de novo."
        else -> "O envio foi cancelado no computador da Central. Confira o código da equipe e sincronize de novo."
    }

    private suspend fun enviarFotos(api: ApiCentral, faltantes: List<FotoFaltanteV2>, fotos: Map<String, FotoV2>, progresso: (Progresso) -> Unit) {
        var enviados = 0L
        faltantes.forEachIndexed { i, falt ->
            progresso(Progresso.EnviandoFotos(i, faltantes.size, enviados))
            val meta = fotos[falt.uuid] ?: throw FalhaCentral("fotos", "A Central pediu uma foto que não está no lote.", fatal = false)
            val caminho = db.fotoDao().obter(falt.uuid)?.caminhoRelativo
                ?: throw FalhaCentral("foto_ausente", "Registro da foto não encontrado no aparelho.", fatal = true)
            val arquivo = arquivoDaFoto(caminho)
            if (!arquivo.isFile || arquivo.length() != meta.tamanhoBytes) {
                throw FalhaCentral("foto_ausente", "Arquivo de foto ausente ou alterado no aparelho (${falt.uuid}).", fatal = true)
            }
            enviados += io { enviarArquivo(api, falt.uuid, arquivo, falt.offset) }
        }
        progresso(Progresso.EnviandoFotos(faltantes.size, faltantes.size, enviados))
    }

    private fun enviarArquivo(api: ApiCentral, foto: String, arquivo: File, inicio: Long): Long {
        val tamanho = arquivo.length()
        val buf = ByteArray(BLOCO)
        var offset = inicio
        var enviados = 0L
        RandomAccessFile(arquivo, "r").use { raf ->
            var trocas = 0
            while (offset < tamanho) {
                raf.seek(offset)
                val n = raf.read(buf, 0, minOf(BLOCO.toLong(), tamanho - offset).toInt())
                if (n <= 0) throw FalhaCentral("foto_ausente", "Falha ao ler a foto no aparelho.", fatal = true)
                when (val r = api.enviarBloco(foto, offset, buf, n)) {
                    is ApiCentral.ResultadoBloco.Aceito -> { enviados += r.offset - offset; offset = r.offset }
                    is ApiCentral.ResultadoBloco.OffsetDiferente -> {
                        if (r.offset > tamanho || ++trocas > 3) throw FalhaCentral("fotos", "Divergência no envio da foto. Tente de novo.", fatal = false)
                        offset = r.offset // a Central já tem até aqui: continuar dela
                    }
                }
            }
        }
        return enviados
    }

    /** Grava o recibo, confirma as revs e registra o destino da coleta numa única transação (RN-74, RN-09). */
    private suspend fun gravarRecibo(lote: String, coleta: String, recibo: ReciboV2): ResumoEnvio = db.withWriteTransaction {
        val json = JsonContratos.encodeToString(ReciboV2.serializer(), recibo)
        val t = agora()
        db.loteDao().confirmar(lote, json, t)
        db.coletaDao().gravarDestino(coleta, JsonContratos.encodeToString(DestinoV2.serializer(), recibo.destino), t)
        val itens = db.loteDao().itens(lote).associateBy { it.entidade to it.entidadeUuid }
        var props = 0
        for (r in recibo.resultados) {
            val item = itens[r.entidade to r.uuid] ?: continue // a Central só responde pelo que recebeu
            db.loteDao().registrarResultado(lote, r.entidade, r.uuid, r.resultado, r.motivo)
            if (r.resultado == "rejeitado") continue // continua pendente e aparece como erro
            val rev = minOf(r.rev, item.rev)
            when (r.entidade) {
                "coleta" -> db.coletaDao().confirmar(r.uuid, rev)
                "propriedade" -> { db.propriedadeDao().confirmar(r.uuid, rev); props++ }
                "item" -> db.itemDao().confirmar(r.uuid, rev)
                "foto" -> db.fotoDao().confirmar(r.uuid, rev)
            }
        }
        val por = recibo.resultados.groupBy { it.resultado }
        ResumoEnvio(lote, props, por["aplicado"].orEmpty().size, por["duplicado"].orEmpty().size, por["conflito"].orEmpty().size, por["rejeitado"].orEmpty(), recibo.destino)
    }

    private suspend fun <T> io(bloco: () -> T): T = withContext(Dispatchers.IO) { bloco() }

    companion object {
        const val BLOCO = 1 shl 20 // 1 MiB (protocolo.md §3)
    }
}

/** Pareamento pelo QR da Central (protocolo.md §2). */
class Pareador(private val db: BancoCampo, private val agora: () -> Long) {
    private val repo = RepositorioCampo(db, agora) { true }

    /**
     * Encontra a Central por um dos IPs do QR e pede o pareamento; a Central responde
     * depois que o operador confirmar. Grava o pareamento (substituindo o anterior).
     */
    suspend fun parear(qr: PayloadQr, rotulo: String?, socketFactory: SocketFactory? = null): PareamentoEntidade {
        val token = qr.tokenPareamento
            ?: throw FalhaCentral("qr", "Este QR não é de pareamento. Na Central, abra Sincronização → Parear aparelho.", fatal = true)
        val instalacao = repo.garantirInstalacao()
        val (ip, api) = withContext(Dispatchers.IO) { encontrarCentral(qr.spkiSha256, qr.ips, qr.porta, socketFactory) }
        val resp = api.use {
            withContext(Dispatchers.IO) { it.parear(PedidoPareamentoV2(token = token, instalacaoUuid = instalacao, rotuloAparelho = rotulo)) }
        }
        if (resp.centralId != qr.centralId) throw FalhaCentral("central", "A Central respondeu com outra identificação.", fatal = true)
        val p = PareamentoEntidade(
            centralId = resp.centralId, spkiSha256 = qr.spkiSha256,
            ipsJson = JsonContratos.encodeToString(ListSerializer(String.serializer()), listOf(ip) + qr.ips.filter { it != ip }),
            porta = qr.porta.toLong(), credencial = resp.credencial, pareadoEm = agora(),
        )
        db.withWriteTransaction {
            db.pareamentoDao().limpar()
            db.pareamentoDao().inserir(p)
        }
        return p
    }

    /** Abre a API com o pareamento gravado, no primeiro IP que responder. */
    suspend fun conectar(socketFactory: SocketFactory? = null): ApiCentral {
        val p = db.pareamentoDao().obter() ?: throw FalhaCentral("nao_pareado", "Aparelho ainda não pareado. Leia o QR de pareamento da Central.", fatal = true)
        val ips = JsonContratos.decodeFromString(ListSerializer(String.serializer()), p.ipsJson)
        val (_, api) = withContext(Dispatchers.IO) { encontrarCentral(p.spkiSha256, ips, p.porta.toInt(), socketFactory) }
        api.credencial = p.credencial
        return api
    }
}

/** Tenta os IPs em ordem e devolve o primeiro em que a Central (com o pin certo) responde. */
internal fun encontrarCentral(pin: String, ips: List<String>, porta: Int, socketFactory: SocketFactory?): Pair<String, ApiCentral> {
    if (ips.isEmpty()) throw FalhaCentral("sem_ip", "A Central não informou nenhum endereço de rede. Ligue o hotspot no computador e gere o QR de novo.", fatal = true)
    var ultima: FalhaCentral? = null
    for (ip in ips) {
        val api = ApiCentral(pin, ip, porta, socketFactory)
        try {
            api.status()
            return ip to api
        } catch (f: FalhaCentral) {
            api.close()
            if (f.codigo == "pin") throw f
            ultima = f
        }
    }
    throw FalhaCentral("rede", "Não encontramos a Central nesta rede. Confira se o celular está no Wi‑Fi do computador da Central.", fatal = false, ultima)
}

// ---------- entidades → contratos ----------

internal fun ColetaEntidade.v2() = ColetaV2(uuid, codigoEquipe, codigoOriginal, instante(createdAt), instante(updatedAt), rev)

internal fun PropriedadeEntidade.v2() = PropriedadeV2(
    uuid, coletaUuid, nomePropriedade, responsavelFamilia, observacoesRelevantes,
    pessoasAdultos, pessoasJovens, pessoasCriancas, veicCaminhoes, veicAutomoveisUtilitarios, veicMotosCiclomotores,
    bovAdultos, bovBezerros, equAdultos, equPotros, areaValorMilesimos, areaUnidade,
    lat, lon, gpsOrigem, gpsPrecisaoM, instanteOuNulo(gpsCapturadoEm), gpsSatelites,
    instante(createdAt), instante(updatedAt), rev, instanteOuNulo(excluidoEm),
)

internal fun ItemEntidade.v2() = ItemV2(uuid, propriedadeUuid, categoria, codigoCatalogo, nomePersonalizado, quantidade, instante(createdAt), instante(updatedAt), rev, instanteOuNulo(excluidoEm))

internal fun FotoEntidade.v2() = FotoV2(uuid, propriedadeUuid, mime, tamanhoBytes, sha256, largura, altura, instante(capturadaEm), instante(createdAt), instante(updatedAt), rev, instanteOuNulo(excluidoEm))
