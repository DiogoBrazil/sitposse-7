package br.gov.ro.pm.sitposse7.sync

import androidx.room3.withWriteTransaction
import br.gov.ro.pm.sitposse7.dados.BancoCampo
import br.gov.ro.pm.sitposse7.dados.EquipeEntidade
import br.gov.ro.pm.sitposse7.dados.FotoEntidade
import br.gov.ro.pm.sitposse7.dados.ItemEntidade
import br.gov.ro.pm.sitposse7.dados.LoteEnvioEntidade
import br.gov.ro.pm.sitposse7.dados.LoteItemEntidade
import br.gov.ro.pm.sitposse7.dados.PareamentoEntidade
import br.gov.ro.pm.sitposse7.dados.PropriedadeEntidade
import br.gov.ro.pm.sitposse7.dados.RepositorioCampo
import br.gov.ro.pm.sitposse7.dados.StatusLote
import br.gov.ro.pm.sitposse7.dominio.PayloadQr
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.builtins.serializer
import java.io.File
import java.io.RandomAccessFile
import java.util.UUID
import javax.net.SocketFactory

/** Etapas mostradas na tela de sincronização (mobile/docs/telas-e-fluxos.md [6]). */
sealed interface Progresso {
    data object Conectando : Progresso
    data class EnviandoCadastros(val propriedades: Int) : Progresso
    data class EnviandoFotos(val feitas: Int, val total: Int, val bytesEnviados: Long) : Progresso
    data object AguardandoImportacao : Progresso
}

/** Resultado de um envio confirmado (recibo gravado). */
data class ResumoEnvio(
    val lote: String,
    val propriedadesConfirmadas: Int,
    val aplicados: Int,
    val duplicados: Int,
    val conflitos: Int,
    val rejeitados: List<ResultadoReciboV1>,
)

/**
 * Envio de lotes à Central (protocolo.md §4–6), com as garantias do AGENTS.md §5:
 * - o lote é congelado em transação (JSON + lista de (entidade, uuid, rev)) e reenviado byte a byte;
 * - fotos em blocos de 1 MiB, retomando do offset que a Central informa;
 * - só é "confirmado" o que consta no recibo **persistido**, gravado numa transação;
 * - edições feitas durante o envio geram rev maior e ficam para o próximo lote.
 */
class Sincronizador(
    private val db: BancoCampo,
    private val agora: () -> Long,
    /** Arquivo de uma foto a partir do caminho relativo registrado. */
    private val arquivoDaFoto: (String) -> File,
) {
    private val repo = RepositorioCampo(db, agora) { true }

    /** Devolve o lote aberto (para retomar) ou congela um novo; null se não há nada pendente. */
    suspend fun preparar(): LoteEnvioEntidade? = db.withWriteTransaction {
        db.loteDao().aberto()?.let { return@withWriteTransaction it }
        val equipe = db.equipeDao().obter() ?: throw FalhaCentral("equipe", "Configure a equipe antes de sincronizar.", fatal = true)
        val props = db.propriedadeDao().pendentes()
        val itens = db.itemDao().pendentes()
        val fotos = db.fotoDao().pendentes()
        if (props.isEmpty() && itens.isEmpty() && fotos.isEmpty() && equipe.rev <= equipe.revConfirmada) return@withWriteTransaction null
        val t = agora()
        val uuid = UUID.randomUUID().toString()
        val lote = LoteV1(
            loteUuid = uuid, instalacaoUuid = repo.instalacaoUuid(), criadoEm = instante(t),
            equipe = equipe.v1(), propriedades = props.map { it.v1() }, itens = itens.map { it.v1() }, fotos = fotos.map { it.v1() },
        )
        val entidade = LoteEnvioEntidade(uuid, StatusLote.PREPARADO, "wifi", JsonContratos.encodeToString(LoteV1.serializer(), lote), t, t)
        db.loteDao().inserir(entidade)
        db.loteDao().inserirItens(
            listOf(LoteItemEntidade(uuid, "equipe", equipe.uuid, equipe.rev)) +
                props.map { LoteItemEntidade(uuid, "propriedade", it.uuid, it.rev) } +
                itens.map { LoteItemEntidade(uuid, "item", it.uuid, it.rev) } +
                fotos.map { LoteItemEntidade(uuid, "foto", it.uuid, it.rev) }
        )
        entidade
    }

    /**
     * Envia o lote aberto (ou um novo) e grava o recibo. Devolve null se não havia nada a enviar.
     * Em falha, o lote fica em erro (recuperável ou fatal) e é retomado na próxima vez; nada se perde.
     */
    suspend fun sincronizar(api: ApiCentral, progresso: (Progresso) -> Unit = {}): ResumoEnvio? {
        val lote = preparar() ?: return null
        db.loteDao().mudarStatus(lote.uuid, StatusLote.ENVIANDO, null, agora())
        try {
            val payload = JsonContratos.decodeFromString(LoteV1.serializer(), lote.payloadJson)
            progresso(Progresso.EnviandoCadastros(payload.propriedades.size))
            var estado = io { api.enviarLote(lote.uuid, lote.payloadJson) }
            var rodadas = 0
            while (estado.recibo == null) {
                if (++rodadas > 4) throw FalhaCentral("fotos", "A Central não conseguiu conferir as fotos. Tente de novo.", fatal = false)
                enviarFotos(api, estado.fotosFaltantes, payload.fotos.associateBy { it.uuid }, progresso)
                progresso(Progresso.AguardandoImportacao)
                estado = when (val r = io { api.importar(lote.uuid) }) {
                    is ApiCentral.ResultadoImportacao.Importado -> r.estado
                    is ApiCentral.ResultadoImportacao.FaltamFotos -> r.estado
                }
            }
            val recibo = estado.recibo!!
            if (!recibo.confere(lote.uuid)) throw FalhaCentral("recibo", "O recibo da Central veio corrompido. Tente de novo.", fatal = false)
            val resumo = gravarRecibo(lote.uuid, recibo)
            io { api.ack(lote.uuid) }
            return resumo
        } catch (f: FalhaCentral) {
            db.loteDao().mudarStatus(lote.uuid, if (f.fatal) StatusLote.ERRO_FATAL else StatusLote.ERRO_RECUPERAVEL, f.message, agora())
            throw f
        }
    }

    private suspend fun enviarFotos(api: ApiCentral, faltantes: List<FotoFaltanteV1>, fotos: Map<String, FotoV1>, progresso: (Progresso) -> Unit) {
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

    /** Grava o recibo e confirma as revs numa única transação (RN-74). */
    private suspend fun gravarRecibo(lote: String, recibo: ReciboV1): ResumoEnvio = db.withWriteTransaction {
        val json = JsonContratos.encodeToString(ReciboV1.serializer(), recibo)
        db.loteDao().confirmar(lote, json, agora())
        val itens = db.loteDao().itens(lote).associateBy { it.entidade to it.entidadeUuid }
        var props = 0
        for (r in recibo.resultados) {
            val item = itens[r.entidade to r.uuid] ?: continue // a Central só responde pelo que recebeu
            db.loteDao().registrarResultado(lote, r.entidade, r.uuid, r.resultado, r.motivo)
            if (r.resultado == "rejeitado") continue // continua pendente e aparece como erro
            val rev = minOf(r.rev, item.rev)
            when (r.entidade) {
                "equipe" -> db.equipeDao().confirmar(r.uuid, rev)
                "propriedade" -> { db.propriedadeDao().confirmar(r.uuid, rev); props++ }
                "item" -> db.itemDao().confirmar(r.uuid, rev)
                "foto" -> db.fotoDao().confirmar(r.uuid, rev)
            }
        }
        val por = recibo.resultados.groupBy { it.resultado }
        ResumoEnvio(lote, props, por["aplicado"].orEmpty().size, por["duplicado"].orEmpty().size, por["conflito"].orEmpty().size, por["rejeitado"].orEmpty())
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
        val equipe = repo.equipe() ?: throw FalhaCentral("equipe", "Configure a equipe antes de parear.", fatal = true)
        val instalacao = repo.instalacaoUuid()
        val (ip, api) = withContext(Dispatchers.IO) { encontrarCentral(qr.spkiSha256, qr.ips, qr.porta, socketFactory) }
        val resp = api.use {
            withContext(Dispatchers.IO) { it.parear(PedidoPareamentoV1(token = token, instalacaoUuid = instalacao, rotuloAparelho = rotulo, equipe = equipe.v1())) }
        }
        if (resp.centralId != qr.centralId) throw FalhaCentral("central", "A Central respondeu com outra identificação.", fatal = true)
        val p = PareamentoEntidade(
            centralId = resp.centralId, spkiSha256 = qr.spkiSha256,
            ipsJson = JsonContratos.encodeToString(ListSerializer(String.serializer()), listOf(ip) + qr.ips.filter { it != ip }),
            porta = qr.porta.toLong(), credencial = resp.credencial, operacaoUuid = resp.operacao.uuid, operacaoNome = resp.operacao.nome, pareadoEm = agora(),
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

internal fun EquipeEntidade.v1() = EquipeV1(uuid, nome, responsavel, identificacaoArea, instante(createdAt), instante(updatedAt), rev)

internal fun PropriedadeEntidade.v1() = PropriedadeV1(
    uuid, equipeUuid, nomePropriedade, responsavelFamilia,
    pessoasAdultos, pessoasJovens, pessoasCriancas, veicCaminhoes, veicAutomoveisUtilitarios, veicMotosCiclomotores,
    bovAdultos, bovBezerros, equAdultos, equPotros, areaValorMilesimos, areaUnidade,
    lat, lon, gpsOrigem, gpsPrecisaoM, instanteOuNulo(gpsCapturadoEm), gpsSatelites,
    instante(createdAt), instante(updatedAt), rev, instanteOuNulo(excluidoEm),
)

internal fun ItemEntidade.v1() = ItemV1(uuid, propriedadeUuid, categoria, codigoCatalogo, nomePersonalizado, quantidade, instante(createdAt), instante(updatedAt), rev, instanteOuNulo(excluidoEm))

internal fun FotoEntidade.v1() = FotoV1(uuid, propriedadeUuid, mime, tamanhoBytes, sha256, largura, altura, instante(capturadaEm), instante(createdAt), instante(updatedAt), rev, instanteOuNulo(excluidoEm))
