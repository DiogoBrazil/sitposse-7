package br.gov.ro.pm.sitposse7.campo.gps

import android.annotation.SuppressLint
import android.content.Context
import android.location.GnssStatus
import android.location.Location
import android.location.LocationListener
import android.location.LocationManager
import android.os.Build
import android.os.Handler
import android.os.Looper
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.callbackFlow

/** Uma leitura do GNSS (RN-50..55): só o receptor do aparelho, nunca internet. */
data class LeituraGnss(
    val lat: Double?,
    val lon: Double?,
    val precisaoM: Double?,
    val instante: Long?,
    val satelitesEmUso: Int,
    val satelitesVistos: Int,
)

object Gnss {
    fun disponivel(contexto: Context): Boolean =
        contexto.getSystemService(LocationManager::class.java)?.isProviderEnabled(LocationManager.GPS_PROVIDER) == true

    /** Leituras do GPS_PROVIDER + contagem de satélites. Exige ACCESS_FINE_LOCATION concedida. */
    @SuppressLint("MissingPermission")
    fun leituras(contexto: Context): Flow<LeituraGnss> = callbackFlow {
        val lm = contexto.getSystemService(LocationManager::class.java)
        val handler = Handler(Looper.getMainLooper())
        var ultima = LeituraGnss(null, null, null, null, 0, 0)
        val ouvinte = LocationListener { l: Location ->
            ultima = ultima.copy(lat = l.latitude, lon = l.longitude, precisaoM = if (l.hasAccuracy()) l.accuracy.toDouble() else null, instante = l.time)
            trySend(ultima)
        }
        val status = object : GnssStatus.Callback() {
            override fun onSatelliteStatusChanged(s: GnssStatus) {
                val usados = (0 until s.satelliteCount).count { s.usedInFix(it) }
                ultima = ultima.copy(satelitesEmUso = usados, satelitesVistos = s.satelliteCount)
                trySend(ultima)
            }
        }
        lm.requestLocationUpdates(LocationManager.GPS_PROVIDER, 1000L, 0f, ouvinte, Looper.getMainLooper())
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) lm.registerGnssStatusCallback(contexto.mainExecutor, status)
        else @Suppress("DEPRECATION") lm.registerGnssStatusCallback(status, handler)
        awaitClose {
            lm.removeUpdates(ouvinte)
            lm.unregisterGnssStatusCallback(status)
        }
    }
}
