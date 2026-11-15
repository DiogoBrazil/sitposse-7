package br.gov.ro.pm.sitposse7.campo

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import br.gov.ro.pm.sitposse7.campo.ui.AppCampo
import br.gov.ro.pm.sitposse7.campo.ui.TemaCampo

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            TemaCampo { AppCampo() }
        }
    }
}
