package br.gov.ro.pm.sitposse7.sync

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.io.BufferedReader
import java.io.File
import java.nio.file.Files
import java.util.concurrent.TimeUnit

/** Processo `central-teste` (desktop/cmd/central-teste) em execução: a Central real, sem tela. */
class CentralGo private constructor(
    private val processo: Process,
    private val saida: BufferedReader,
    val porta: Int,
    val pin: String,
    val centralId: String,
    val token: String,
) : AutoCloseable {
    private fun comando(c: String): JsonObject {
        processo.outputStream.write("$c\n".toByteArray())
        processo.outputStream.flush()
        return Json.parseToJsonElement(checkNotNull(saida.readLine()) { "central-teste encerrou" }).jsonObject
    }

    /** Novo token de pareamento (uso único). */
    fun novoToken(): String = comando("token").getValue("token").jsonPrimitive.content

    /** Pedidos de pareamento esperando o operador (só com `-confirmar=false`). */
    fun pendentes(): Int = comando("pendentes").getValue("pendentes").jsonPrimitive.int

    /** Faz o papel do operador: aceita ou recusa os pedidos pendentes; devolve quantos decidiu. */
    fun decidir(aceitar: Boolean): Int = comando(if (aceitar) "aceitar" else "recusar").getValue("decididos").jsonPrimitive.int

    /** Mata o processo sem encerramento limpo (computador desligado ou Central fechada à força). */
    fun derrubar() {
        processo.destroyForcibly()
        processo.waitFor(10, TimeUnit.SECONDS)
    }

    /** Contagens das visões vigentes na Central. */
    fun estado(): Map<String, Int> = comando("estado").mapValues { it.value.jsonPrimitive.int }

    override fun close() {
        runCatching { processo.outputStream.close() } // o servidor encerra ao fechar a entrada padrão
        if (!processo.waitFor(10, TimeUnit.SECONDS)) processo.destroyForcibly()
    }

    companion object {
        private val binario: File by lazy {
            val desktop = File(checkNotNull(System.getProperty("sitposse7.desktop")))
            val saida = Files.createTempDirectory("central-teste").toFile().resolve("central-teste")
            val go = System.getenv("SITPOSSE7_GO") ?: "go"
            val p = ProcessBuilder(go, "build", "-o", saida.absolutePath, "./cmd/central-teste")
                .directory(desktop).redirectErrorStream(true).start()
            val log = p.inputStream.bufferedReader().readText()
            check(p.waitFor(300, TimeUnit.SECONDS) && p.exitValue() == 0) { "go build falhou:\n$log" }
            saida
        }

        /** [argumentos]: opções do central-teste, por exemplo `-confirmar=false` e `-espera=2s`. */
        fun iniciar(vararg argumentos: String): CentralGo {
            val p = ProcessBuilder(listOf(binario.absolutePath) + argumentos).redirectError(ProcessBuilder.Redirect.INHERIT).start()
            val r = p.inputStream.bufferedReader()
            val o = Json.parseToJsonElement(checkNotNull(r.readLine()) { "central-teste não respondeu" }).jsonObject
            return CentralGo(
                p, r, o.getValue("porta").jsonPrimitive.int, o.getValue("pin").jsonPrimitive.content,
                o.getValue("central_id").jsonPrimitive.content, o.getValue("token").jsonPrimitive.content,
            )
        }
    }
}
