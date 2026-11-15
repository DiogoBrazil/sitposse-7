package br.gov.ro.pm.sitposse7.dominio

import kotlinx.serialization.Serializable

/** Campos de quantidade fixa (RN-20..24), com o nome da coluna/JSON. */
enum class Quantitativo(val campo: String, val grupo: String, val rotulo: String, val ajuda: String? = null) {
    PESSOAS_ADULTOS("pessoas_adultos", "Pessoas", "Adultos", "18 anos ou mais"),
    PESSOAS_JOVENS("pessoas_jovens", "Pessoas", "Jovens", "12 a 17 anos"),
    PESSOAS_CRIANCAS("pessoas_criancas", "Pessoas", "Crianças", "0 a 11 anos"),
    VEIC_CAMINHOES("veic_caminhoes", "Veículos", "Caminhões"),
    VEIC_AUTOMOVEIS("veic_automoveis_utilitarios", "Veículos", "Automóveis, caminhonetes e utilitários"),
    VEIC_MOTOS("veic_motos_ciclomotores", "Veículos", "Motocicletas e ciclomotores"),
    BOV_ADULTOS("bov_adultos", "Bovinos", "Bovinos adultos"),
    BOV_BEZERROS("bov_bezerros", "Bovinos", "Bezerros e bezerras"),
    EQU_ADULTOS("equ_adultos", "Equinos", "Equinos adultos"),
    EQU_POTROS("equ_potros", "Equinos", "Potros"),
}

/** Coordenada coletada (RN-50..55). Manual não tem precisão (RN-52). */
@Serializable
data class Coordenada(
    val lat: Double,
    val lon: Double,
    val origem: String, // "gnss" | "manual"
    val precisaoM: Double? = null,
    val capturadoEm: Long? = null,
    val satelites: Long? = null,
)

@Serializable
data class ItemCadastro(
    /** null = item novo ainda não salvo. */
    val uuid: String? = null,
    val codigo: String,
    /** Só para "Outros" (RN-33); texto original, nunca alterado. */
    val nomePersonalizado: String? = null,
    val quantidade: Long = 1,
)

/** Foto já gravada em disco (arquivo antes do registro). */
@Serializable
data class FotoCadastro(
    val uuid: String,
    val caminhoRelativo: String,
    val tamanhoBytes: Long,
    val sha256: String,
    val largura: Long,
    val altura: Long,
    val capturadaEm: Long,
)

/** Estado completo do formulário; também é o conteúdo do rascunho (RF-A10). */
@Serializable
data class CadastroPropriedade(
    val uuid: String? = null,
    val nome: String = "",
    val responsavel: String = "",
    val quantidades: Map<String, Long> = emptyMap(),
    val areaTexto: String = "",
    val areaUnidade: String? = null,
    val coordenada: Coordenada? = null,
    val itens: List<ItemCadastro> = emptyList(),
    val fotos: List<FotoCadastro> = emptyList(),
    /** Observações relevantes (RN-15): texto livre opcional, multilinha. */
    val observacoes: String = "",
) {
    fun quantidade(q: Quantitativo): Long = quantidades[q.campo] ?: 0
}

/** Erro de validação ligado a um campo do formulário. */
data class ErroCampo(val campo: String, val mensagem: String)

/** Milésimos → texto editável com vírgula, sem zeros à direita (12500 → "12,5"). */
fun textoArea(milesimos: Long): String {
    val inteira = milesimos / 1000
    val frac = (milesimos % 1000).toString().padStart(3, '0').trimEnd('0')
    return if (frac.isEmpty()) inteira.toString() else "$inteira,$frac"
}

/** Conta caracteres Unicode (code points), como a Central e o JSON Schema (maxLength). */
fun contarCaracteres(s: String): Int = s.codePointCount(0, s.length)

/** Observação como vai para o banco: quebras de linha uniformes; só espaços → null (RN-15). */
fun observacaoParaGravar(s: String): String? = s.replace("\r\n", "\n").replace('\r', '\n').takeUnless { it.isBlank() }

object ValidacaoCadastro {
    const val MAX_TEXTO = 200
    const val MAX_OBSERVACOES = 4000
    const val MAX_QUANTIDADE = 1_000_000L
    const val META_PRECISAO_M = 15.0

    /** Valida o formulário; lista vazia = pode salvar. */
    fun validar(c: CadastroPropriedade): List<ErroCampo> {
        val erros = mutableListOf<ErroCampo>()
        if (c.nome.isBlank()) erros += ErroCampo("nome", "Informe o nome da propriedade.")
        if (c.nome.length > MAX_TEXTO) erros += ErroCampo("nome", "Use no máximo $MAX_TEXTO caracteres.")
        if (c.responsavel.isBlank()) erros += ErroCampo("responsavel", "Informe o nome do responsável pela propriedade.") // RN-11 (A28)
        if (c.responsavel.length > MAX_TEXTO) erros += ErroCampo("responsavel", "Use no máximo $MAX_TEXTO caracteres.")
        if (contarCaracteres(c.observacoes) > MAX_OBSERVACOES) erros += ErroCampo("observacoes", "Use no máximo $MAX_OBSERVACOES caracteres.") // RN-15
        Quantitativo.entries.forEach { q ->
            val v = c.quantidade(q)
            if (v < 0 || v > MAX_QUANTIDADE) erros += ErroCampo(q.campo, "Quantidade inválida.")
        }
        when (val a = Areas.interpretar(c.areaTexto, c.areaUnidade)) {
            is ResultadoArea.Invalida -> erros += ErroCampo("area", mensagemArea(a.erro))
            else -> Unit
        }
        c.coordenada?.let { g ->
            if (g.lat !in -90.0..90.0 || g.lon !in -180.0..180.0) erros += ErroCampo("coordenada", "Coordenada fora da faixa válida.")
            if (g.origem != "gnss" && g.origem != "manual") erros += ErroCampo("coordenada", "Origem da coordenada inválida.")
            if (g.origem == "manual" && g.precisaoM != null) erros += ErroCampo("coordenada", "Coordenada manual não tem precisão.")
        }
        val predefinidos = mutableSetOf<String>()
        c.itens.forEachIndexed { i, item ->
            val cat = Catalogo.item(item.codigo)
            if (cat == null) {
                erros += ErroCampo("item.$i", "Item desconhecido.")
                return@forEachIndexed
            }
            if (item.quantidade < 1 || item.quantidade > MAX_QUANTIDADE) erros += ErroCampo("item.$i", "Quantidade deve ser 1 ou mais. Para zerar, remova o item.")
            if (cat.outros) {
                val n = item.nomePersonalizado
                if (n.isNullOrBlank()) erros += ErroCampo("item.$i", "Informe o nome do item.")
                else if (n.length > MAX_TEXTO) erros += ErroCampo("item.$i", "Use no máximo $MAX_TEXTO caracteres.")
            } else if (!predefinidos.add(item.codigo)) {
                erros += ErroCampo("item.$i", "${cat.nome} já foi adicionado; edite a quantidade.") // RN-32
            }
        }
        return erros
    }

    /**
     * RN-33: "Outros" com o mesmo nome normalizado na mesma categoria não é bloqueado,
     * mas o app avisa e oferece somar. Devolve o índice do item já existente, se houver.
     */
    fun outrosRepetido(itens: List<ItemCadastro>, codigo: String, nome: String, ignorarIndice: Int = -1): Int? {
        val alvo = Normalizacao.normalizarNome(nome) ?: return null
        return itens.indices.firstOrNull { i ->
            i != ignorarIndice && itens[i].codigo == codigo && itens[i].nomePersonalizado?.let { Normalizacao.normalizarNome(it) } == alvo
        }
    }

    fun mensagemArea(e: ErroArea): String = when (e) {
        ErroArea.NEGATIVO -> "A área não pode ser negativa."
        ErroArea.MAIS_DE_3_CASAS -> "Use no máximo 3 casas decimais."
        ErroArea.NAO_NUMERICO -> "Digite só números, com vírgula para os decimais."
        ErroArea.SEPARADOR_DE_MILHAR -> "Não use ponto de milhar; use só a vírgula decimal."
        ErroArea.UNIDADE_OBRIGATORIA -> "Escolha a unidade (hectares ou alqueire paulista)."
        ErroArea.UNIDADE_INVALIDA -> "Unidade inválida."
    }
}
