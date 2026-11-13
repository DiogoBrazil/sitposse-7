package br.gov.ro.pm.sitposse7.dominio

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.longOrNull
import java.io.File

/** Lê fixtures de shared/test-fixtures (caminho injetado pelo Gradle). */
object Fixtures {
    private val raiz = File(checkNotNull(System.getProperty("sitposse7.fixtures")) { "sitposse7.fixtures não definido" })

    fun ler(caminho: String): JsonObject =
        Json.parseToJsonElement(File(raiz, caminho).readText()) as JsonObject
}

fun JsonObject.texto(campo: String): String? =
    this[campo]?.takeUnless { it is JsonNull }?.jsonPrimitive?.content

fun JsonObject.inteiro(campo: String): Long? =
    this[campo]?.takeUnless { it is JsonNull }?.jsonPrimitive?.longOrNull

fun JsonElement.obj() = this as JsonObject
