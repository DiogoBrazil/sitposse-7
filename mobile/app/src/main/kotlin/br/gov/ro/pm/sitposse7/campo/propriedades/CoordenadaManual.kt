package br.gov.ro.pm.sitposse7.campo.propriedades

import br.gov.ro.pm.sitposse7.dominio.Coordenada

/** Editar números torna a origem manual; confirmar o mesmo ponto preserva a captura. */
internal fun coordenadaManual(latitude: String, longitude: String, inicial: Coordenada?): Coordenada? {
    val lat = latitude.trim().replace(',', '.').toDoubleOrNull() ?: return null
    val lon = longitude.trim().replace(',', '.').toDoubleOrNull() ?: return null
    if (!lat.isFinite() || !lon.isFinite() || lat !in -90.0..90.0 || lon !in -180.0..180.0) return null
    if (inicial != null && lat == inicial.lat && lon == inicial.lon) return inicial
    return Coordenada(lat, lon, "manual")
}
