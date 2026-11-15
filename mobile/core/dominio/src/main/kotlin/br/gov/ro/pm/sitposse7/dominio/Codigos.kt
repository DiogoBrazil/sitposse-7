package br.gov.ro.pm.sitposse7.dominio

/**
 * Códigos públicos de equipe e de estudo (RN-02, ADR-0023): alfabeto Crockford Base32,
 * 7 caracteres aleatórios e 1 dígito verificador Luhn mod 32. O celular só confere formato
 * e dígito (offline, RN-05); quem diz se o código existe é a Central, na sincronização.
 * Casos de referência: shared/test-fixtures/codigos/codigos.v1.json (os mesmos da Central).
 */
object Codigos {
    const val ALFABETO = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
    const val PREFIXO_EQUIPE = "EQ"
    const val PREFIXO_ESTUDO = "SITPOSSE7"
    const val TAMANHO_CORPO = 8

    enum class Erro(val mensagem: String) {
        VAZIO("Informe o código da equipe."),
        TAMANHO("O código tem 8 caracteres depois de \"EQ\"."),
        CARACTERE("O código tem um caractere que não existe nos códigos."),
        DIGITO("Confira os caracteres do código."),
    }

    sealed interface Resultado {
        data class Valido(val codigo: String) : Resultado
        data class Invalido(val erro: Erro) : Resultado
    }

    private fun indice(c: Char) = ALFABETO.indexOf(c)

    /** Dígito Luhn mod 32 de um corpo de 7 caracteres do alfabeto (null se tiver caractere fora). */
    fun digitoVerificador(corpo: String): Char? {
        var fator = 2
        var soma = 0
        for (i in corpo.indices.reversed()) {
            val v = indice(corpo[i]).takeIf { it >= 0 } ?: return null
            val a = fator * v
            fator = if (fator == 2) 1 else 2
            soma += a / 32 + a % 32
        }
        return ALFABETO[(32 - soma % 32) % 32]
    }

    private fun digitoConfere(corpo: String): Boolean {
        var fator = 1
        var soma = 0
        for (i in corpo.indices.reversed()) {
            val v = indice(corpo[i]).takeIf { it >= 0 } ?: return false
            val a = fator * v
            fator = if (fator == 1) 2 else 1
            soma += a / 32 + a % 32
        }
        return soma % 32 == 0
    }

    private fun separador(c: Char) = c.isWhitespace() || c == '-' || c == '.' || c == '_' || c == '/'

    /**
     * Converte o que foi digitado (com ou sem prefixo, hífens, espaços, minúsculas, O no lugar
     * de 0, I/L no lugar de 1) na forma guardada: prefixo + 8 caracteres.
     */
    fun normalizar(entrada: String, prefixo: String = PREFIXO_EQUIPE): Resultado {
        val s = entrada.uppercase().filterNot(::separador)
        if (s.isEmpty()) return Resultado.Invalido(Erro.VAZIO)
        val corpo = when {
            s.length == prefixo.length + TAMANHO_CORPO && s.startsWith(prefixo) -> s.substring(prefixo.length)
            s.length == TAMANHO_CORPO -> s
            else -> return Resultado.Invalido(Erro.TAMANHO)
        }
        val c = corpo.map {
            when (it) {
                'O' -> '0'
                'I', 'L' -> '1'
                else -> it
            }
        }.joinToString("")
        if (c.any { indice(it) < 0 }) return Resultado.Invalido(Erro.CARACTERE)
        if (!digitoConfere(c)) return Resultado.Invalido(Erro.DIGITO)
        return Resultado.Valido(prefixo + c)
    }

    /** Código guardado → forma de leitura: EQ-7K3M-9QX1. */
    fun exibir(codigo: String): String {
        if (codigo.length < TAMANHO_CORPO) return codigo
        val p = codigo.length - TAMANHO_CORPO
        val c = codigo.substring(p)
        return codigo.substring(0, p) + "-" + c.substring(0, 4) + "-" + c.substring(4)
    }

    /**
     * Formata enquanto a pessoa digita: maiúsculas, só caracteres de código, prefixo "EQ" e
     * hífens nos lugares certos (no máximo 8 caracteres depois do prefixo).
     */
    fun formatarDigitacao(texto: String): String {
        var s = texto.uppercase().filter { it.isLetterOrDigit() }
        if (s.startsWith(PREFIXO_EQUIPE)) s = s.substring(PREFIXO_EQUIPE.length)
        s = s.take(TAMANHO_CORPO)
        if (s.isEmpty()) return ""
        return PREFIXO_EQUIPE + "-" + if (s.length <= 4) s else s.substring(0, 4) + "-" + s.substring(4)
    }
}
