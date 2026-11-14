package br.gov.ro.pm.sitposse7.campo.equipe

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.unit.dp
import br.gov.ro.pm.sitposse7.campo.CampoApp
import br.gov.ro.pm.sitposse7.campo.R
import br.gov.ro.pm.sitposse7.campo.ui.BarraSuperior
import br.gov.ro.pm.sitposse7.campo.ui.Centralizado
import kotlinx.coroutines.launch

/** [1] Configurar equipe (RF-A01) e editar (RF-A02). */
@Composable
fun EquipeTela(primeiraVez: Boolean, aoConcluir: () -> Unit, aoVoltar: (() -> Unit)? = null) {
    val app = LocalContext.current.applicationContext as CampoApp
    val escopo = rememberCoroutineScope()
    var nome by rememberSaveable { mutableStateOf("") }
    var responsavel by rememberSaveable { mutableStateOf("") }
    var area by rememberSaveable { mutableStateOf("") }
    var carregado by rememberSaveable { mutableStateOf(primeiraVez) }
    var erro by rememberSaveable { mutableStateOf<String?>(null) }
    var tentou by rememberSaveable { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        if (!carregado) app.repo.equipe()?.let { nome = it.nome; responsavel = it.responsavel; area = it.identificacaoArea.orEmpty() }
        carregado = true
    }

    Scaffold(topBar = { BarraSuperior(stringResource(if (primeiraVez) R.string.equipe_titulo_nova else R.string.equipe_titulo), aoVoltar) }) { pad ->
        Centralizado(560, Modifier.padding(pad).fillMaxSize()) {
            Column(Modifier.verticalScroll(rememberScrollState()).imePadding().padding(20.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
                if (primeiraVez) Text(stringResource(R.string.equipe_explicacao), style = MaterialTheme.typography.bodyLarge)
                val cap = KeyboardOptions(capitalization = KeyboardCapitalization.Words)
                OutlinedTextField(nome, { nome = it.take(200) }, label = { Text(stringResource(R.string.equipe_nome)) }, singleLine = true, keyboardOptions = cap,
                    isError = tentou && nome.isBlank(), supportingText = { if (tentou && nome.isBlank()) Text(stringResource(R.string.obrigatorio)) }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(responsavel, { responsavel = it.take(200) }, label = { Text(stringResource(R.string.equipe_responsavel)) }, singleLine = true, keyboardOptions = cap,
                    isError = tentou && responsavel.isBlank(), supportingText = { if (tentou && responsavel.isBlank()) Text(stringResource(R.string.obrigatorio)) }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(area, { area = it.take(200) }, label = { Text(stringResource(R.string.equipe_area)) }, singleLine = true, keyboardOptions = cap,
                    supportingText = { Text(stringResource(R.string.equipe_area_ajuda)) }, modifier = Modifier.fillMaxWidth())
                erro?.let { Text(it, color = MaterialTheme.colorScheme.error) }
                Button(
                    onClick = {
                        tentou = true
                        if (nome.isBlank() || responsavel.isBlank()) return@Button
                        escopo.launch {
                            runCatching {
                                if (primeiraVez) app.repo.configurarEquipe(nome, responsavel, area) else app.repo.editarEquipe(nome, responsavel, area)
                            }.onSuccess { aoConcluir() }.onFailure { erro = it.message }
                        }
                    },
                    modifier = Modifier.fillMaxWidth().height(56.dp),
                ) { Text(stringResource(if (primeiraVez) R.string.equipe_comecar else R.string.salvar)) }
            }
        }
    }
}
