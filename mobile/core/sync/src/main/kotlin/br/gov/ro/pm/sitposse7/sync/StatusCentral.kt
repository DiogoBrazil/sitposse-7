package br.gov.ro.pm.sitposse7.sync

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** Resposta de GET /v1/status (shared/schemas/status.v2.schema.json). */
@Serializable
data class StatusCentral(
    @SerialName("protocolo_versao") val protocoloVersao: Int,
    @SerialName("protocolo_versao_min") val protocoloVersaoMin: Int,
    @SerialName("protocolo_versao_max") val protocoloVersaoMax: Int,
    @SerialName("schema_versao_min") val schemaVersaoMin: Int,
    @SerialName("schema_versao_max") val schemaVersaoMax: Int,
    @SerialName("central_id") val centralId: String,
    val horario: String,
)
