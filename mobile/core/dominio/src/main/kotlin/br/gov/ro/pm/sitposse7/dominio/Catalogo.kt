package br.gov.ro.pm.sitposse7.dominio

/** Categoria de item dinâmico (RN-30). */
enum class Categoria(val codigo: String, val prefixo: String, val titulo: String) {
    ELETRODOMESTICO("eletrodomestico", "eletro", "Eletrodomésticos"),
    MOVEL("movel", "movel", "Móveis"),
    AVE("ave", "ave", "Aves de criação");

    companion object {
        fun doCodigo(c: String): Categoria? = entries.firstOrNull { it.codigo == c }
    }
}

data class ItemCatalogo(val codigo: String, val categoria: Categoria, val nome: String, val ordem: Int) {
    val outros: Boolean get() = codigo.endsWith(".outros")
}

/**
 * Catálogo inicial (RN-30), igual a shared/test-fixtures/catalogo/itens.v1.json e ao seed
 * da Central (teste CatalogoTest). Códigos estáveis: nunca reutilizar nem renomear.
 */
object Catalogo {
    const val VERSAO = 1

    val itens: List<ItemCatalogo> = listOf(
        ItemCatalogo("eletro.freezer", Categoria.ELETRODOMESTICO, "Freezer", 1),
        ItemCatalogo("eletro.geladeira", Categoria.ELETRODOMESTICO, "Geladeira", 2),
        ItemCatalogo("eletro.fogao", Categoria.ELETRODOMESTICO, "Fogão", 3),
        ItemCatalogo("eletro.maquina_lavar", Categoria.ELETRODOMESTICO, "Máquina de lavar", 4),
        ItemCatalogo("eletro.ar_condicionado", Categoria.ELETRODOMESTICO, "Ar-condicionado", 5),
        ItemCatalogo("eletro.ventilador", Categoria.ELETRODOMESTICO, "Ventilador", 6),
        ItemCatalogo("eletro.outros", Categoria.ELETRODOMESTICO, "Outros", 7),
        ItemCatalogo("movel.sofa", Categoria.MOVEL, "Sofá", 1),
        ItemCatalogo("movel.mesa_jantar", Categoria.MOVEL, "Mesa de jantar", 2),
        ItemCatalogo("movel.armario", Categoria.MOVEL, "Armário", 3),
        ItemCatalogo("movel.estante", Categoria.MOVEL, "Estante", 4),
        ItemCatalogo("movel.guarda_roupa", Categoria.MOVEL, "Guarda-roupa", 5),
        ItemCatalogo("movel.outros", Categoria.MOVEL, "Outros", 6),
        ItemCatalogo("ave.galinha", Categoria.AVE, "Galinha", 1),
        ItemCatalogo("ave.ganso", Categoria.AVE, "Ganso", 2),
        ItemCatalogo("ave.pato", Categoria.AVE, "Pato", 3),
        ItemCatalogo("ave.outros", Categoria.AVE, "Outros", 4),
    )

    private val porCodigo = itens.associateBy { it.codigo }

    fun item(codigo: String): ItemCatalogo? = porCodigo[codigo]

    fun daCategoria(c: Categoria): List<ItemCatalogo> = itens.filter { it.categoria == c }.sortedBy { it.ordem }
}
