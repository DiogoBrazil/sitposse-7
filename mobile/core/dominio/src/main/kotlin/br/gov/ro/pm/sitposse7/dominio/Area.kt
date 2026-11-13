package br.gov.ro.pm.sitposse7.dominio

/**
 * Área informada pelo usuário, preservada como digitada (em milésimos da unidade).
 * Regras: docs/requirements/regras-de-negocio.md (RN-40..45) e ADR-0009.
 * Paridade com desktop/internal/dominio/area.go, verificada pelas fixtures de shared/test-fixtures/area.
 */
data class Area(val milesimos: Long, val unidade: UnidadeArea) {
    /** Decímetros quadrados inteiros: cálculo canônico e exato. */
    val dm2: Long get() = milesimos * unidade.dm2PorMilesimo
}

enum class UnidadeArea(val codigo: String, val dm2PorMilesimo: Long) {
    HECTARE("ha", 1_000),
    ALQUEIRE_PAULISTA("alq_paulista", 2_420); // 1 alq. paulista = 2,42 ha

    companion object {
        fun doCodigo(codigo: String): UnidadeArea? = entries.firstOrNull { it.codigo == codigo }
    }
}

/** Erros de validação; [codigo] é idêntico ao da fixture e ao do Go. */
enum class ErroArea(val codigo: String) {
    NEGATIVO("negativo"),
    MAIS_DE_3_CASAS("mais_de_3_casas"),
    NAO_NUMERICO("nao_numerico"),
    SEPARADOR_DE_MILHAR("separador_de_milhar_nao_suportado"),
    UNIDADE_OBRIGATORIA("unidade_obrigatoria"),
    UNIDADE_INVALIDA("unidade_invalida"),
}

sealed interface ResultadoArea {
    /** Campo vazio: **não** é zero (RN-45). */
    data object SemArea : ResultadoArea
    data class Valida(val area: Area) : ResultadoArea
    data class Invalida(val erro: ErroArea) : ResultadoArea
}

object Areas {
    /** Limite que garante somas em Long sem transbordar (igual ao Go). */
    private const val MAX_MILESIMOS = 1_000_000_000_000L

    /**
     * Interpreta o texto digitado. Aceita vírgula ou ponto decimal, até 3 casas.
     * [codigoUnidade] é o código salvo (`ha`, `alq_paulista`) ou null se não escolhida.
     */
    fun interpretar(entrada: String, codigoUnidade: String?): ResultadoArea {
        var s = entrada.trim()
        if (s.isEmpty()) return ResultadoArea.SemArea
        if (s.startsWith("-")) return ResultadoArea.Invalida(ErroArea.NEGATIVO)
        if ('.' in s && ',' in s) return ResultadoArea.Invalida(ErroArea.SEPARADOR_DE_MILHAR)
        s = s.replace(',', '.')
        val ponto = s.indexOf('.')
        val inteira = if (ponto < 0) s else s.substring(0, ponto)
        val frac = if (ponto < 0) "" else s.substring(ponto + 1)
        if (inteira.isEmpty() || !inteira.soDigitos() || (ponto >= 0 && (frac.isEmpty() || !frac.soDigitos()))) {
            return ResultadoArea.Invalida(ErroArea.NAO_NUMERICO)
        }
        if (frac.length > 3) return ResultadoArea.Invalida(ErroArea.MAIS_DE_3_CASAS)
        val semZeros = inteira.trimStart('0')
        if (semZeros.length > 12) return ResultadoArea.Invalida(ErroArea.NAO_NUMERICO)
        val milesimos = (semZeros + frac.padEnd(3, '0')).fold(0L) { acc, c -> acc * 10 + (c - '0') }
        if (milesimos > MAX_MILESIMOS) return ResultadoArea.Invalida(ErroArea.NAO_NUMERICO)
        if (codigoUnidade == null) return ResultadoArea.Invalida(ErroArea.UNIDADE_OBRIGATORIA)
        val unidade = UnidadeArea.doCodigo(codigoUnidade) ?: return ResultadoArea.Invalida(ErroArea.UNIDADE_INVALIDA)
        return ResultadoArea.Valida(Area(milesimos, unidade))
    }

    /** dm² → hectares com 2 casas (HALF_UP) e vírgula decimal. */
    fun formatarHa(dm2: Long): String {
        require(dm2 >= 0) { "área negativa" }
        val centiHa = (dm2 + 5_000) / 10_000
        return "%d,%02d".format(centiHa / 100, centiHa % 100)
    }

    data class Soma(val totalDm2: Long, val comArea: Int, val semArea: Int)

    /** Soma as áreas presentes; `null` (sem área) não vale zero e é contado à parte (RN-45). */
    fun somar(areas: List<Area?>): Soma {
        var total = 0L
        var com = 0
        var sem = 0
        for (a in areas) {
            if (a == null) sem++ else { total += a.dm2; com++ }
        }
        return Soma(total, com, sem)
    }

    private fun String.soDigitos() = all { it in '0'..'9' }
}
