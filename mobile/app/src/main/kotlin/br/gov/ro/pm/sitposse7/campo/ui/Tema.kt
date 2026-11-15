package br.gov.ro.pm.sitposse7.campo.ui

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

// Alto contraste ao sol (mobile/docs/telas-e-fluxos.md): claro por padrão, cores AA.
private val Claro = lightColorScheme(
    primary = Color(0xFF1B5E20), onPrimary = Color.White,
    primaryContainer = Color(0xFFC8E6C9), onPrimaryContainer = Color(0xFF00210B),
    secondary = Color(0xFF0D47A1), onSecondary = Color.White,
    secondaryContainer = Color(0xFFD6E3FF), onSecondaryContainer = Color(0xFF001B3E),
    tertiary = Color(0xFF8D4E00), onTertiary = Color.White,
    tertiaryContainer = Color(0xFFFFDCBE), onTertiaryContainer = Color(0xFF2D1600),
    error = Color(0xFFB3261E), errorContainer = Color(0xFFF9DEDC), onErrorContainer = Color(0xFF410E0B),
    background = Color(0xFFF7F9F4), onBackground = Color(0xFF111411),
    surface = Color.White, onSurface = Color(0xFF111411),
    surfaceVariant = Color(0xFFE1E6DE), onSurfaceVariant = Color(0xFF3A4237),
    outline = Color(0xFF6B7468),
)
private val Escuro = darkColorScheme(
    primary = Color(0xFF8BD68E), onPrimary = Color(0xFF00390F),
    primaryContainer = Color(0xFF1B5E20), onPrimaryContainer = Color(0xFFC8E6C9),
    secondary = Color(0xFFAAC7FF), secondaryContainer = Color(0xFF0D47A1),
    tertiary = Color(0xFFFFB870), tertiaryContainer = Color(0xFF6A3B00),
)

private val Tipos = Typography().run {
    copy(
        titleLarge = titleLarge.copy(fontWeight = FontWeight.SemiBold),
        titleMedium = titleMedium.copy(fontWeight = FontWeight.SemiBold),
        bodyLarge = bodyLarge.copy(fontSize = 17.sp, lineHeight = 24.sp),
        labelLarge = labelLarge.copy(fontSize = 16.sp, fontWeight = FontWeight.SemiBold),
    )
}

@Composable
fun TemaCampo(conteudo: @Composable () -> Unit) {
    MaterialTheme(colorScheme = if (isSystemInDarkTheme()) Escuro else Claro, typography = Tipos, content = conteudo)
}

/** Classe de largura da janela (celular, celular grande/dobrável, tablet). */
enum class Largura { COMPACTA, MEDIA, EXPANDIDA }

@Composable
fun larguraAtual(): Largura {
    val w = LocalConfiguration.current.screenWidthDp
    return when {
        w < 600 -> Largura.COMPACTA
        w < 840 -> Largura.MEDIA
        else -> Largura.EXPANDIDA
    }
}

/** Centraliza o conteúdo e limita a largura em telas grandes (formulários legíveis). */
@Composable
fun Centralizado(larguraMaxima: Int = 720, modifier: Modifier = Modifier, conteudo: @Composable BoxScope.() -> Unit) {
    Box(modifier.fillMaxWidth(), contentAlignment = Alignment.TopCenter) {
        Box(Modifier.widthIn(max = larguraMaxima.dp).fillMaxWidth(), content = conteudo)
    }
}
