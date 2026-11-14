package br.gov.ro.pm.sitposse7.dados

import androidx.room3.Dao
import androidx.room3.Insert
import androidx.room3.OnConflictStrategy
import androidx.room3.Query
import androidx.room3.Update
import androidx.room3.Upsert
import kotlinx.coroutines.flow.Flow

@Dao
interface EquipeDao {
    @Query("SELECT * FROM equipe LIMIT 1")
    suspend fun obter(): EquipeEntidade?

    @Query("SELECT * FROM equipe LIMIT 1")
    fun observar(): Flow<EquipeEntidade?>

    @Insert(onConflict = OnConflictStrategy.ABORT)
    suspend fun inserir(e: EquipeEntidade)

    @Update
    suspend fun atualizar(e: EquipeEntidade)

    @Query("UPDATE equipe SET rev_confirmada = max(rev_confirmada, :rev) WHERE uuid = :uuid")
    suspend fun confirmar(uuid: String, rev: Long)
}

/** Linha da lista de propriedades (RF-A09). */
data class PropriedadeResumo(
    val uuid: String,
    val nome_propriedade: String,
    val responsavel_familia: String?,
    val tem_gps: Boolean,
    val fotos: Int,
    val pendente: Boolean,
    val rejeitada: Boolean,
    val updated_at: Long,
)

@Dao
interface PropriedadeDao {
    @Query("SELECT * FROM propriedades WHERE uuid = :uuid")
    suspend fun obter(uuid: String): PropriedadeEntidade?

    @Query(
        """SELECT p.uuid, p.nome_propriedade, p.responsavel_familia, p.lat IS NOT NULL AS tem_gps,
        (SELECT count(*) FROM fotografias f WHERE f.propriedade_uuid = p.uuid AND f.excluido_em IS NULL) AS fotos,
        (p.rev > p.rev_confirmada
          OR EXISTS (SELECT 1 FROM itens_propriedade i WHERE i.propriedade_uuid = p.uuid AND i.rev > i.rev_confirmada)
          OR EXISTS (SELECT 1 FROM fotografias f WHERE f.propriedade_uuid = p.uuid AND f.rev > f.rev_confirmada)) AS pendente,
        EXISTS (SELECT 1 FROM lote_itens li JOIN lotes_envio l ON l.uuid = li.lote_uuid
          WHERE li.entidade = 'propriedade' AND li.entidade_uuid = p.uuid AND li.rev = p.rev AND li.resultado = 'rejeitado') AS rejeitada,
        p.updated_at
        FROM propriedades p WHERE p.excluido_em IS NULL ORDER BY p.nome_propriedade COLLATE NOCASE, p.uuid"""
    )
    fun observarVigentes(): Flow<List<PropriedadeResumo>>

    @Insert(onConflict = OnConflictStrategy.ABORT)
    suspend fun inserir(p: PropriedadeEntidade)

    @Update
    suspend fun atualizar(p: PropriedadeEntidade)

    @Query("SELECT * FROM propriedades WHERE rev > rev_confirmada ORDER BY created_at, uuid")
    suspend fun pendentes(): List<PropriedadeEntidade>

    @Query("UPDATE propriedades SET rev_confirmada = max(rev_confirmada, :rev) WHERE uuid = :uuid")
    suspend fun confirmar(uuid: String, rev: Long)

    @Query("SELECT count(*) FROM propriedades WHERE excluido_em IS NULL")
    fun observarTotal(): Flow<Int>
}

@Dao
interface ItemDao {
    @Query("SELECT * FROM itens_propriedade WHERE propriedade_uuid = :propriedade AND excluido_em IS NULL ORDER BY categoria, codigo_catalogo, created_at")
    suspend fun vigentesDa(propriedade: String): List<ItemEntidade>

    @Insert(onConflict = OnConflictStrategy.ABORT)
    suspend fun inserir(i: ItemEntidade)

    @Update
    suspend fun atualizar(i: ItemEntidade)

    @Query("SELECT * FROM itens_propriedade WHERE rev > rev_confirmada ORDER BY created_at, uuid")
    suspend fun pendentes(): List<ItemEntidade>

    @Query("UPDATE itens_propriedade SET rev_confirmada = max(rev_confirmada, :rev) WHERE uuid = :uuid")
    suspend fun confirmar(uuid: String, rev: Long)
}

@Dao
interface FotoDao {
    @Query("SELECT * FROM fotografias WHERE propriedade_uuid = :propriedade AND excluido_em IS NULL ORDER BY capturada_em, uuid")
    suspend fun vigentesDa(propriedade: String): List<FotoEntidade>

    @Query("SELECT * FROM fotografias WHERE uuid = :uuid")
    suspend fun obter(uuid: String): FotoEntidade?

    @Query("SELECT * FROM fotografias")
    suspend fun todas(): List<FotoEntidade>

    @Insert(onConflict = OnConflictStrategy.ABORT)
    suspend fun inserir(f: FotoEntidade)

    @Update
    suspend fun atualizar(f: FotoEntidade)

    @Query("SELECT * FROM fotografias WHERE rev > rev_confirmada ORDER BY created_at, uuid")
    suspend fun pendentes(): List<FotoEntidade>

    @Query("UPDATE fotografias SET rev_confirmada = max(rev_confirmada, :rev) WHERE uuid = :uuid")
    suspend fun confirmar(uuid: String, rev: Long)
}

@Dao
interface RascunhoDao {
    @Upsert
    suspend fun gravar(r: RascunhoEntidade)

    @Query("SELECT * FROM rascunhos WHERE uuid = :uuid")
    suspend fun obter(uuid: String): RascunhoEntidade?

    @Query("SELECT * FROM rascunhos ORDER BY atualizado_em DESC")
    suspend fun todos(): List<RascunhoEntidade>

    /** Rascunho não é entidade de negócio: apagá-lo após salvar ou descartar é permitido (R5). */
    @Query("DELETE FROM rascunhos WHERE uuid = :uuid")
    suspend fun apagar(uuid: String)
}

/** Contagens da tela inicial. */
data class Contagens(val propriedades: Int, val pendentes: Int, val confirmadas: Int)

@Dao
interface LoteDao {
    @Insert(onConflict = OnConflictStrategy.ABORT)
    suspend fun inserir(l: LoteEnvioEntidade)

    @Insert(onConflict = OnConflictStrategy.ABORT)
    suspend fun inserirItens(itens: List<LoteItemEntidade>)

    @Query("SELECT * FROM lotes_envio WHERE status IN ('preparado', 'enviando', 'erro_recuperavel', 'erro_fatal') ORDER BY criado_em LIMIT 1")
    suspend fun aberto(): LoteEnvioEntidade?

    @Query("SELECT * FROM lotes_envio WHERE uuid = :uuid")
    suspend fun obter(uuid: String): LoteEnvioEntidade?

    @Query("SELECT * FROM lote_itens WHERE lote_uuid = :lote")
    suspend fun itens(lote: String): List<LoteItemEntidade>

    @Query("UPDATE lotes_envio SET status = :status, erro = :erro, atualizado_em = :agora WHERE uuid = :uuid")
    suspend fun mudarStatus(uuid: String, status: String, erro: String?, agora: Long)

    @Query("UPDATE lotes_envio SET status = 'confirmado', recibo_json = :recibo, erro = NULL, atualizado_em = :agora WHERE uuid = :uuid")
    suspend fun confirmar(uuid: String, recibo: String, agora: Long)

    @Query("UPDATE lote_itens SET resultado = :resultado, motivo = :motivo WHERE lote_uuid = :lote AND entidade = :entidade AND entidade_uuid = :uuid")
    suspend fun registrarResultado(lote: String, entidade: String, uuid: String, resultado: String, motivo: String?)

    @Query("SELECT max(atualizado_em) FROM lotes_envio WHERE status = 'confirmado'")
    fun observarUltimaConfirmacao(): Flow<Long?>

    @Query(
        """SELECT
        (SELECT count(*) FROM propriedades WHERE excluido_em IS NULL) AS propriedades,
        (SELECT count(*) FROM propriedades p WHERE p.rev > p.rev_confirmada
            OR EXISTS (SELECT 1 FROM itens_propriedade i WHERE i.propriedade_uuid = p.uuid AND i.rev > i.rev_confirmada)
            OR EXISTS (SELECT 1 FROM fotografias f WHERE f.propriedade_uuid = p.uuid AND f.rev > f.rev_confirmada)) AS pendentes,
        (SELECT count(*) FROM propriedades p WHERE p.excluido_em IS NULL AND p.rev_confirmada > 0 AND p.rev = p.rev_confirmada
            AND NOT EXISTS (SELECT 1 FROM itens_propriedade i WHERE i.propriedade_uuid = p.uuid AND i.rev > i.rev_confirmada)
            AND NOT EXISTS (SELECT 1 FROM fotografias f WHERE f.propriedade_uuid = p.uuid AND f.rev > f.rev_confirmada)) AS confirmadas"""
    )
    fun observarContagens(): Flow<Contagens>
}

@Dao
interface PareamentoDao {
    @Query("SELECT * FROM pareamento LIMIT 1")
    suspend fun obter(): PareamentoEntidade?

    @Query("SELECT * FROM pareamento LIMIT 1")
    fun observar(): Flow<PareamentoEntidade?>

    /** Um aparelho fica pareado com uma Central por vez; o pareamento novo substitui o anterior. */
    @Query("DELETE FROM pareamento")
    suspend fun limpar()

    @Insert(onConflict = OnConflictStrategy.ABORT)
    suspend fun inserir(p: PareamentoEntidade)
}
