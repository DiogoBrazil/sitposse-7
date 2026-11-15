package br.gov.ro.pm.sitposse7.sync

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.io.BufferedReader
import java.io.File
import java.nio.file.Files
import java.util.concurrent.TimeUnit

/** Equipe do estudo de teste criado pelo central-teste (jornada 1). */
data class EquipeTeste(val uuid: String, val nome: String, val codigo: String)

/** Lote esperando o operador decidir o destino. */
data class LoteAguardando(val lote: String, val codigo: String, val travado: Boolean, val encontrado: Boolean)

/** Processo `central-teste` (desktop/cmd/central-teste) em execução: a Central real, sem tela. */
class CentralGo private constructor(
    private val processo: Process,
    private val saida: BufferedReader,
    val porta: Int,
    val pin: String,
    val centralId: String,
    val token: String,
    /** Equipes "Alfa" e "Bravo" do estudo de teste, com os códigos oficiais. */
    val equipes: List<EquipeTeste>,
) : AutoCloseable {
    val alfa get() = equipes.first { it.nome == "Alfa" }
    val bravo get() = equipes.first { it.nome == "Bravo" }
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

    /** Lotes esperando a decisão do operador (com `-destino=manual`, ou de código inexistente). */
    fun aguardando(): List<LoteAguardando> = comando("aguardando").getValue("aguardando").jsonArray.map {
        val o = it.jsonObject
        LoteAguardando(o.getValue("lote").jsonPrimitive.content, o.getValue("codigo").jsonPrimitive.content,
            o.getValue("travado").jsonPrimitive.boolean, o.getValue("encontrado").jsonPrimitive.boolean)
    }

    /** Faz o papel do operador: importa o lote para a equipe do [codigo] (padrão: a sugerida). Devolve o status ou o erro. */
    fun confirmar(lote: String, codigo: String? = null): String =
        comando(listOfNotNull("confirmar", lote, codigo).joinToString(" ")).let { (it["status"] ?: it.getValue("erro")).jsonPrimitive.content }

    /** Faz o papel do operador: cancela a sincronização do lote. */
    fun cancelar(lote: String): String = comando("cancelar $lote").let { (it["status"] ?: it.getValue("erro")).jsonPrimitive.content }

    /** Espera até haver [n] lotes aguardando (ou falha em 10 s). */
    fun esperarAguardando(n: Int = 1): List<LoteAguardando> {
        val limite = System.currentTimeMillis() + 10_000
        while (true) {
            val a = aguardando()
            if (a.size >= n) return a
            check(System.currentTimeMillis() < limite) { "nenhum lote chegou para o operador decidir" }
            Thread.sleep(20)
        }
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

        /** [argumentos]: opções do central-teste, por exemplo `-confirmar=false`, `-espera=2s` e `-destino=manual`. */
        fun iniciar(vararg argumentos: String): CentralGo {
            val p = ProcessBuilder(listOf(binario.absolutePath) + argumentos).redirectError(ProcessBuilder.Redirect.INHERIT).start()
            val r = p.inputStream.bufferedReader()
            val o = Json.parseToJsonElement(checkNotNull(r.readLine()) { "central-teste não respondeu" }).jsonObject
            return CentralGo(
                p, r, o.getValue("porta").jsonPrimitive.int, o.getValue("pin").jsonPrimitive.content,
                o.getValue("central_id").jsonPrimitive.content, o.getValue("token").jsonPrimitive.content,
                o.getValue("equipes").jsonArray.map { e ->
                    val x = e.jsonObject
                    EquipeTeste(x.getValue("uuid").jsonPrimitive.content, x.getValue("nome").jsonPrimitive.content, x.getValue("codigo").jsonPrimitive.content)
                },
            )
        }
    }
}
