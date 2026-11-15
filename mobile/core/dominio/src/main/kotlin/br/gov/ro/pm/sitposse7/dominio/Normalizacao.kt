package br.gov.ro.pm.sitposse7.dominio

import java.text.Normalizer

/**
 * Chave de agrupamento de itens "Outros" (RN-35). O texto original nunca é alterado.
 * Paridade com desktop/internal/dominio/normalizacao.go (fixture normalizacao/nomes.json).
 */
object Normalizacao {
    private val espacos = Regex("\\s+")
    private val marcas = Regex("\\p{Mn}+")

    /** Retorna o nome normalizado ou `null` quando fica vazio (inválido). */
    fun normalizarNome(original: String): String? {
        val s = original.trim().replace(espacos, " ").lowercase()
        val n = marcas.replace(Normalizer.normalize(s, Normalizer.Form.NFD), "")
        return n.ifEmpty { null }
    }
}
