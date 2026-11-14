package br.gov.ro.pm.sitposse7.campo.fotos

import java.io.File
import java.io.FileOutputStream
import java.io.IOException
import java.io.OutputStream
import java.security.MessageDigest
import java.util.UUID

/** Arquivo de foto já gravado de forma durável (pronto para ser registrado no banco). */
data class FotoGravada(val uuid: String, val caminhoRelativo: String, val tamanhoBytes: Long, val sha256: String)

/**
 * Fotos na pasta privada do app (mobile/docs/persistencia-offline.md §Fotos):
 * grava em `{uuid}.jpg.part` calculando o SHA-256 em streaming, faz fsync, renomeia para
 * `{aa}/{uuid}.jpg` e só então o registro pode entrar no banco (AGENTS.md §3).
 * Só usa java.io: testável na JVM.
 */
class ArmazemFotos(val raiz: File) {
    val tmp: File get() = File(raiz, "tmp")

    fun arquivo(caminhoRelativo: String): File = File(raiz, caminhoRelativo)

    fun existe(caminhoRelativo: String): Boolean = arquivo(caminhoRelativo).isFile

    /** Espaço livre mínimo antes de fotografar (200 MB). */
    fun espacoSuficiente(): Boolean = raiz.apply { mkdirs() }.usableSpace >= 200L * 1024 * 1024

    /** Grava os bytes produzidos por [escrever] de forma durável. Em qualquer falha, nada fica registrado. */
    fun gravar(escrever: (OutputStream) -> Unit): FotoGravada {
        val uuid = UUID.randomUUID().toString()
        val rel = "${uuid.take(2)}/$uuid.jpg"
        val final = arquivo(rel)
        final.parentFile?.mkdirs()
        val parte = File(final.parentFile, "$uuid.jpg.part")
        val md = MessageDigest.getInstance("SHA-256")
        try {
            FileOutputStream(parte).use { fos ->
                val saida = object : OutputStream() {
                    override fun write(b: Int) { fos.write(b); md.update(b.toByte()) }
                    override fun write(b: ByteArray, off: Int, len: Int) { fos.write(b, off, len); md.update(b, off, len) }
                }
                escrever(saida)
                fos.flush()
                fos.fd.sync()
            }
            if (!parte.renameTo(final)) throw IOException("não foi possível publicar a foto")
        } catch (t: Throwable) {
            parte.delete()
            throw t
        }
        return FotoGravada(uuid, rel, final.length(), md.digest().joinToString("") { "%02x".format(it) })
    }

    /**
     * Reconciliador (início do app): apaga `tmp/` e `.part`; arquivos finais que nem registro
     * nem rascunho referenciam vão para `quarentena/` (nunca são apagados aqui). Devolve os
     * registros vigentes cujo arquivo sumiu — defeito grave a mostrar, nunca apagar.
     */
    fun reconciliar(referenciados: Set<String>, vigentes: Map<String, String>): List<String> {
        tmp.deleteRecursively()
        val quarentena = File(raiz, "quarentena")
        raiz.listFiles()?.filter { it.isDirectory && it.name.length == 2 }?.forEach { pasta ->
            pasta.listFiles()?.forEach { f ->
                val rel = "${pasta.name}/${f.name}"
                when {
                    f.name.endsWith(".part") -> f.delete()
                    f.name.endsWith(".jpg") && rel !in referenciados -> {
                        quarentena.mkdirs()
                        f.renameTo(File(quarentena, f.name))
                    }
                }
            }
        }
        return vigentes.filterValues { !existe(it) }.keys.toList()
    }
}
