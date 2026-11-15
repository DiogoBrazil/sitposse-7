<script setup lang="ts">
/**
 * Painel de estatísticas (Fase 4, A25; docs/requirements/estatisticas.md §22-A).
 * Um único cálculo no Go (ConsultaService.Estatisticas) alimenta cartões, gráficos e tabelas,
 * sempre com os mesmos filtros (RN-81). Todo número abre as famílias que o compõem (RN-82).
 */
import { computed, onActivated, onDeactivated, onMounted, onUnmounted, ref, watch } from 'vue'
import { ConsultaService as Consulta } from '../../bindings/sitposse7/desktop/app'
import type { Resumo } from '../../bindings/sitposse7/desktop/internal/consultas/models'
import { estudoAtual, estudoFoco } from '../estudo'
import type { Faixa, TipoItem } from '../../bindings/sitposse7/desktop/internal/estatisticas/models'
import BarraFiltros from '../componentes/BarraFiltros.vue'
import CartaoIndicador from '../componentes/CartaoIndicador.vue'
import Detalhamento from '../componentes/Detalhamento.vue'
import ExportarRelatorio from '../componentes/ExportarRelatorio.vue'
import Grafico from '../componentes/Grafico.vue'
import TabelaDados from '../componentes/TabelaDados.vue'
import { alturaBarras, barrasHorizontais } from '../graficos'
import { chips, comExtra, type Filtro, filtroVazio, rotuloCampo } from '../filtro'
import { CONJUNTOS, dominioDoCampo, extraDoIndicador, type Familia, type Grupo, type LinhaIndicador, type Resultado, v } from '../painel'
import { haNumero, mensagem, num, plural, quando } from '../formato'
import type { Tela } from '../navegacao'
import type { Coluna } from '../tabela'

const emit = defineEmits<{ ir: [tela: Tela]; abrir: [uuid: string] }>()
const filtro = ref<Filtro>(filtroVazio(estudoFoco.value))
// Trocar o estudo em foco recomeça os filtros dentro do novo estudo.
watch(estudoFoco, (e) => { if (filtro.value.estudo !== e) filtro.value = filtroVazio(e) })
const r = ref<Resultado | null>(null)
const resumo = ref<Resumo | null>(null)
const erro = ref('')

async function carregar() {
  try {
    const [a, b] = await Promise.all([Consulta.Estatisticas(filtro.value), Consulta.Resumo(filtro.value.estudo)])
    r.value = a; resumo.value = b; erro.value = ''
  } catch (e) { erro.value = mensagem(e) }
}
let atraso: number | undefined
watch(filtro, () => { window.clearTimeout(atraso); atraso = window.setTimeout(carregar, 120) }, { deep: true })
// Novos lotes chegam durante a sincronização: recalcula a cada 10 s com o Painel na tela.
let timer: number | undefined
const iniciar = () => { carregar(); window.clearInterval(timer); timer = window.setInterval(carregar, 10_000) }
const parar = () => window.clearInterval(timer)
onMounted(iniciar); onActivated(iniciar); onDeactivated(parar); onUnmounted(parar)

const t = computed(() => r.value?.totais)
const campos = computed(() => r.value?.campos ?? [])
const opcoes = computed(() => ({ equipes: r.value?.equipes ?? [], jornadas: r.value?.jornadas ?? [] }))
const rotuloDe = (chave: string) => campos.value.find((c) => c.chave === chave)?.rotulo ?? chave

// ---------- exportação ----------
const exportando = ref(false)
const criterios = computed(() => chips(filtro.value, { ...opcoes.value, campos: campos.value }).map((c) => c.texto))

// ---------- drill-down ----------
const detalhe = ref<{ titulo: string; filtro: Filtro; campo: string | null } | null>(null)
function detalhar(chave: string, rotulo: string, extra: Partial<Filtro> = {}) {
  const ehQuantidade = campos.value.some((c) => c.chave === chave)
  detalhe.value = { titulo: rotulo, filtro: comExtra(comExtra(filtro.value, extraDoIndicador(chave)), extra), campo: ehQuantidade ? chave : null }
}
function aplicarDetalhe(f: Filtro) { filtro.value = f; detalhe.value = null }

// ---------- cartões ----------
const itensDe = (cat: string) => (r.value?.itens ?? []).filter((i) => i.campo.categoria === cat && i.total > 0).sort((a, b) => b.total - a.total)
function linhasItens(cat: string): LinhaIndicador[] {
  return itensDe(cat).slice(0, 4).map((i) => ({ chave: i.campo.chave, rotulo: i.campo.rotulo + (i.campo.outros ? ' *' : ''), valor: i.total }))
}
function rodapeItens(cat: string): string | undefined {
  const lista = itensDe(cat)
  const partes: string[] = []
  if (lista.length > 4) partes.push(`+ ${lista.length - 4} tipo(s) em “Itens por tipo”`)
  if (lista.slice(0, 4).some((i) => i.campo.outros)) partes.push('* digitado em “Outros”')
  return partes.length ? partes.join(' · ') : undefined
}
const lin = (chave: string, rotulo: string): LinhaIndicador => ({ chave, rotulo, valor: v(t.value, chave) })

// ---------- análises ----------
type Aba = 'grupos' | 'familias' | 'itens' | 'tamanho'
const aba = ref<Aba>('grupos')
const agrupar = ref<'equipe' | 'jornada'>('equipe')
const conjunto = ref('resumo')
const indicador = ref('pessoas_total')
const camposConjunto = computed(() => CONJUNTOS.find((c) => c.id === conjunto.value)!.campos)
watch(conjunto, () => { indicador.value = camposConjunto.value[0].chave })
const indicadores = computed(() => [{ chave: 'familias', rotulo: 'Famílias' }, ...camposConjunto.value, { chave: 'area', rotulo: 'Área (ha)' }])
function valorIndicador(x: Grupo | Familia, chave: string): number {
  if ('totais' in x) {
    if (chave === 'familias') return x.totais.familias
    if (chave === 'area') return Math.round(x.totais.area_dm2 / 10_000) / 100
    return v(x.totais, chave)
  }
  if (chave === 'area') return Math.round((x.area_dm2 ?? 0) / 10_000) / 100
  if (chave === 'familias') return 1
  return v(x, chave)
}
const rotuloIndicador = computed(() => indicadores.value.find((i) => i.chave === indicador.value)?.rotulo ?? '')
const corIndicador = computed(() => (indicador.value === 'familias' || indicador.value === 'area' ? 'eletro' : dominioDoCampo(indicador.value)))

const grupos = computed(() => (agrupar.value === 'equipe' ? r.value?.por_equipe : r.value?.por_jornada) ?? [])
const colGrupos = computed<Coluna<Grupo>[]>(() => [
  { chave: 'nome', rotulo: agrupar.value === 'equipe' ? 'Equipe' : 'Jornada' },
  { chave: 'familias', rotulo: 'Famílias', tipo: 'numero', valor: (l) => l.totais.familias, formatar: (l) => num(l.totais.familias) },
  ...camposConjunto.value.map((c) => ({ chave: c.chave, rotulo: c.rotulo, tipo: 'numero' as const, valor: (l: Grupo) => v(l.totais, c.chave), formatar: (l: Grupo) => num(v(l.totais, c.chave)) })),
  ...(conjunto.value === 'resumo' ? [{ chave: 'area', rotulo: 'Área (ha)', tipo: 'numero' as const, valor: (l: Grupo) => l.totais.area_dm2, formatar: (l: Grupo) => l.totais.area_ha + (l.totais.sem_area ? ` (${l.totais.sem_area} sem)` : '') }] : []),
])
const totaisLinha = computed(() => {
  const x = t.value
  if (!x) return null
  const o: Record<string, string> = { nome: 'Total geral', familias: num(x.familias), area: x.area_ha + (x.sem_area ? ` (${x.sem_area} sem)` : '') }
  for (const c of camposConjunto.value) o[c.chave] = num(v(x, c.chave))
  return o
})
function detalharGrupo(g: Grupo) {
  const extra: Partial<Filtro> = agrupar.value === 'equipe' ? { equipes: [g.chave] } : { jornadas: [g.chave] }
  const qtd = indicador.value !== 'familias' && indicador.value !== 'area'
  detalhar(qtd ? indicador.value : 'familias', `${g.nome}: ${qtd ? rotuloDe(indicador.value).toLowerCase() : 'famílias'}`, extra)
}

const familias = computed(() => r.value?.familias ?? [])
const colFamilias = computed<Coluna<Familia>[]>(() => [
  { chave: 'nome', rotulo: 'Família / propriedade' },
  { chave: 'equipe', rotulo: 'Equipe', secundaria: conjunto.value !== 'resumo' },
  ...camposConjunto.value.map((c) => ({ chave: c.chave, rotulo: c.rotulo, tipo: 'numero' as const, valor: (l: Familia) => v(l, c.chave), formatar: (l: Familia) => num(v(l, c.chave)) })),
  { chave: 'area', rotulo: 'Área (ha)', tipo: 'numero', valor: (l) => haNumero(l.area_ha), formatar: (l) => l.area_ha ?? 'Não informada' },
])
const totaisFamilias = computed(() => (totaisLinha.value && t.value ? { ...totaisLinha.value, nome: `Total geral (${plural(t.value.familias, 'família', 'famílias')})`, area: t.value.area_ha } : null))
const MAX_BARRAS = 15
const topoFamilias = computed(() => [...familias.value].sort((a, b) => valorIndicador(b, indicador.value) - valorIndicador(a, indicador.value)).slice(0, MAX_BARRAS))

type Categoria = 'eletrodomestico' | 'movel' | 'ave'
const categoriaItens = ref<Categoria>('eletrodomestico')
const abasItens: { id: Categoria; rotulo: string; dominio: 'eletro' | 'moveis' | 'aves'; total: string }[] = [
  { id: 'eletrodomestico', rotulo: 'Eletrodomésticos', dominio: 'eletro', total: 'eletro_total' },
  { id: 'movel', rotulo: 'Móveis', dominio: 'moveis', total: 'moveis_total' },
  { id: 'ave', rotulo: 'Aves', dominio: 'aves', total: 'aves_total' },
]
const abaItem = computed(() => abasItens.find((c) => c.id === categoriaItens.value)!)
const itensAba = computed(() => (r.value?.itens ?? []).filter((i) => i.campo.categoria === categoriaItens.value).sort((a, b) => b.total - a.total))
const colItens: Coluna<TipoItem>[] = [
  { chave: 'nome', rotulo: 'Tipo', valor: (l) => l.campo.rotulo },
  { chave: 'origem', rotulo: 'Origem', valor: (l) => (l.campo.outros ? 'Digitado em “Outros”' : 'Catálogo'), secundaria: true },
  { chave: 'total', rotulo: 'Quantidade', tipo: 'numero', formatar: (l) => num(l.total) },
  { chave: 'familias', rotulo: 'Famílias que possuem', tipo: 'numero', formatar: (l) => num(l.familias) },
]
const totalItens = computed(() => ({ nome: 'Total', total: num(itensAba.value.reduce((s, i) => s + i.total, 0)) }))
const detalharItem = (i: TipoItem) => detalhar(i.campo.chave, `Famílias com ${rotuloCampo(i.campo).toLowerCase()}`)

const faixas = computed(() => r.value?.faixas_area ?? [])
const colFaixas: Coluna<Faixa>[] = [
  { chave: 'rotulo', rotulo: 'Tamanho', ordenavel: false },
  { chave: 'familias', rotulo: 'Famílias', tipo: 'numero', ordenavel: false, formatar: (l) => num(l.familias) },
  { chave: 'area_ha', rotulo: 'Área somada (ha)', tipo: 'numero', ordenavel: false, formatar: (l) => (l.chave === 'sem_area' ? '—' : l.area_ha) },
]
const detalharFaixa = (f: Faixa) => detalhar('familias', `Tamanho: ${f.rotulo.toLowerCase()}`, { faixas_area: [f.chave] })

const abas: { id: Aba; rotulo: string }[] = [
  { id: 'grupos', rotulo: 'Por jornada ou equipe' }, { id: 'familias', rotulo: 'Por família' }, { id: 'itens', rotulo: 'Itens por tipo' }, { id: 'tamanho', rotulo: 'Tamanho das propriedades' },
]
</script>

<template>
  <div class="pilha">
    <header class="cabeca">
      <div class="titulo">
        <h1>Painel</h1>
        <p class="suave">
          {{ estudoAtual ? `${estudoAtual.nome} · ${estudoAtual.codigo_exibicao}` : 'Nenhum estudo de situação' }}<template v-if="resumo?.lotes_importados">. {{ plural(resumo.lotes_importados, 'lote importado', 'lotes importados') }}, o último em {{ quando(resumo.ultima_importacao) }}</template>
        </p>
      </div>
      <div class="linha">
        <button :disabled="!r?.familias_base" title="PDF, Word ou Excel, com os filtros ativos" @click="exportando = true"><svg class="ic" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 4v11m0 0-4.5-4.5M12 15l4.5-4.5M5 19h14" /></svg>Exportar</button>
        <button class="primario" @click="emit('ir', estudoAtual ? 'sincronizacao' : 'estudos')">{{ estudoAtual ? 'Receber dos celulares' : 'Criar estudo' }}</button>
      </div>
    </header>

    <BarraFiltros v-model="filtro" :equipes="opcoes.equipes" :jornadas="opcoes.jornadas" :campos="campos"
                  :mostrando="t?.familias ?? null" :base="r?.familias_base ?? null" />
    <p v-if="erro" class="aviso-erro">{{ erro }}</p>

    <template v-if="r && t">
      <div v-if="r.familias_base === 0" class="bloco vazio-geral">
        <h2>Nenhuma família recebida ainda</h2>
        <p class="suave">Os números aparecem aqui assim que os celulares enviarem os cadastros.</p>
        <button class="primario" @click="emit('ir', 'sincronizacao')">Ir para Sincronização</button>
      </div>

      <template v-else>
        <section class="cartoes" aria-label="Indicadores">
          <CartaoIndicador titulo="Famílias" :principais="[{ chave: 'familias', rotulo: 'propriedades', valor: t.familias }]"
            :linhas="[{ chave: 'com_localizacao', rotulo: 'Com localização', valor: t.com_localizacao }, { chave: 'sem_localizacao', rotulo: 'Sem localização', valor: t.sem_localizacao }, { chave: 'com_foto', rotulo: 'Com foto', valor: t.com_foto }]"
            @detalhar="detalhar" />
          <CartaoIndicador titulo="Pessoas" cor="--s2" :principais="[{ chave: 'pessoas_total', rotulo: 'moradores', valor: v(t, 'pessoas_total') }]"
            :linhas="[lin('pessoas_adultos', 'Adultos (18+)'), lin('pessoas_jovens', 'Jovens (12–17)'), lin('pessoas_criancas', 'Crianças (0–11)')]" @detalhar="detalhar" />
          <CartaoIndicador titulo="Veículos" cor="--s4" :principais="[{ chave: 'veic_total', rotulo: 'no total', valor: v(t, 'veic_total') }]"
            :linhas="[lin('veic_caminhoes', 'Caminhões'), lin('veic_automoveis_utilitarios', 'Automóveis e utilitários'), lin('veic_motos_ciclomotores', 'Motos e ciclomotores')]" @detalhar="detalhar" />
          <CartaoIndicador titulo="Rebanho" cor="--s3" :principais="[{ chave: 'bov_total', rotulo: 'bovinos', valor: v(t, 'bov_total') }, { chave: 'equ_total', rotulo: 'equinos', valor: v(t, 'equ_total') }]"
            :linhas="[lin('bov_adultos', 'Bovinos adultos'), lin('bov_bezerros', 'Bezerros e bezerras'), lin('equ_adultos', 'Equinos adultos'), lin('equ_potros', 'Potros')]" @detalhar="detalhar" />
          <CartaoIndicador titulo="Aves" cor="--s6" :principais="[{ chave: 'aves_total', rotulo: 'de criação', valor: v(t, 'aves_total') }]"
            :linhas="linhasItens('ave')" :rodape="rodapeItens('ave')" vazio="Nenhuma ave registrada." @detalhar="detalhar" />
          <CartaoIndicador titulo="Eletrodomésticos" cor="--s1" :principais="[{ chave: 'eletro_total', rotulo: 'no total', valor: v(t, 'eletro_total') }]"
            :linhas="linhasItens('eletrodomestico')" :rodape="rodapeItens('eletrodomestico')" vazio="Nenhum eletrodoméstico registrado." @detalhar="detalhar" />
          <CartaoIndicador titulo="Móveis" cor="--s5" :principais="[{ chave: 'moveis_total', rotulo: 'no total', valor: v(t, 'moveis_total') }]"
            :linhas="linhasItens('movel')" :rodape="rodapeItens('movel')" vazio="Nenhum móvel registrado." @detalhar="detalhar" />
          <CartaoIndicador titulo="Área" :principais="[{ chave: 'com_area', rotulo: 'hectares informados', valor: t.area_dm2, texto: t.area_ha }]"
            :linhas="[{ chave: 'com_area', rotulo: 'Com área informada', valor: t.com_area }, { chave: 'sem_area', rotulo: 'Sem área informada', valor: t.sem_area }]"
            :rodape="t.area_media_ha ? `Média de ${t.area_media_ha} ha por família com área` : undefined" @detalhar="detalhar" />
        </section>

        <section class="analises">
          <div class="abas" role="tablist" aria-label="Análises">
            <button v-for="a in abas" :key="a.id" role="tab" :aria-selected="aba === a.id" :class="{ ativa: aba === a.id }" @click="aba = a.id">{{ a.rotulo }}</button>
          </div>

          <div v-if="aba === 'grupos' || aba === 'familias'" class="controles">
            <label v-if="aba === 'grupos'">Agrupar por
              <select v-model="agrupar"><option value="equipe">Equipe</option><option value="jornada">Jornada</option></select>
            </label>
            <label>Colunas
              <select v-model="conjunto"><option v-for="c in CONJUNTOS" :key="c.id" :value="c.id">{{ c.rotulo }}</option></select>
            </label>
            <label>Gráfico
              <select v-model="indicador"><option v-for="i in indicadores" :key="i.chave" :value="i.chave">{{ i.rotulo }}</option></select>
            </label>
          </div>

          <template v-if="aba === 'grupos'">
            <div v-if="grupos.length" class="bloco grafico-bloco">
              <h3>{{ rotuloIndicador }} por {{ agrupar }}</h3>
              <Grafico :montar="(tm) => barrasHorizontais(tm, corIndicador, grupos.map((g) => ({ nome: g.nome, valor: valorIndicador(g, indicador) })), indicador === 'area' ? 'ha' : '')"
                       :altura="alturaBarras(grupos.length)" :rotulo="`Gráfico de ${rotuloIndicador} por grupo; os valores estão na tabela`" @clicar="(i) => detalharGrupo(grupos[i])" />
            </div>
            <TabelaDados :colunas="colGrupos" :linhas="grupos" :chave="(l) => l.chave || '—'" :rotulo="`Totais por ${agrupar}`" :por-pagina="0"
                         :totais="totaisLinha" clicavel :ordem="{ chave: 'familias', dir: 'desc' }" texto-vazio="Nenhuma família com esses filtros." @abrir="detalharGrupo" />
            <p class="fraco dica">Clique numa linha ou barra para ver as famílias que compõem os números.</p>
          </template>

          <template v-else-if="aba === 'familias'">
            <div v-if="familias.length" class="bloco grafico-bloco">
              <h3>{{ rotuloIndicador }} por família{{ familias.length > MAX_BARRAS ? ` (as ${MAX_BARRAS} maiores)` : '' }}</h3>
              <Grafico :montar="(tm) => barrasHorizontais(tm, corIndicador, topoFamilias.map((p) => ({ nome: p.nome, valor: valorIndicador(p, indicador) })), indicador === 'area' ? 'ha' : '')"
                       :altura="alturaBarras(topoFamilias.length)" :rotulo="`Gráfico de ${rotuloIndicador} por família; os valores estão na tabela`" @clicar="(i) => emit('abrir', topoFamilias[i].uuid)" />
            </div>
            <TabelaDados :colunas="colFamilias" :linhas="familias" :chave="(l) => l.uuid" rotulo="Totais por família" clicavel :ordem="{ chave: 'nome', dir: 'asc' }"
                         :totais="totaisFamilias" texto-vazio="Nenhuma família com esses filtros." @abrir="(l) => emit('abrir', l.uuid)">
              <template #nome="{ linha }"><strong>{{ linha.nome }}</strong><span class="suave resp">{{ linha.responsavel || 'Responsável não informado' }}</span></template>
            </TabelaDados>
          </template>

          <template v-else-if="aba === 'itens'">
            <div class="controles">
              <div class="segmentado" role="radiogroup" aria-label="Categoria">
                <button v-for="c in abasItens" :key="c.id" type="button" role="radio" :aria-checked="categoriaItens === c.id" @click="categoriaItens = c.id">
                  {{ c.rotulo }} <span class="cont">{{ num(v(t, c.total)) }}</span>
                </button>
              </div>
            </div>
            <div v-if="itensAba.some((i) => i.total > 0)" class="bloco grafico-bloco">
              <h3>{{ abaItem.rotulo }}: quantidade por tipo</h3>
              <Grafico :key="categoriaItens" :montar="(tm) => barrasHorizontais(tm, abaItem.dominio, itensAba.slice(0, MAX_BARRAS).map((i) => ({ nome: i.campo.rotulo, valor: i.total })))"
                       :altura="alturaBarras(Math.min(itensAba.length, MAX_BARRAS))" rotulo="Gráfico da quantidade por tipo de item; os valores estão na tabela"
                       @clicar="(i) => detalharItem(itensAba[i])" />
            </div>
            <TabelaDados :colunas="colItens" :linhas="itensAba" :chave="(l) => l.campo.chave" rotulo="Itens por tipo" :por-pagina="0" clicavel :totais="totalItens"
                         :ordem="{ chave: 'total', dir: 'desc' }" texto-vazio="Nenhum item nesta categoria." @abrir="detalharItem" />
            <p class="fraco dica">Itens digitados em “Outros” são agrupados pelo nome, sem diferenciar acentos e maiúsculas. O texto original não é alterado.</p>
          </template>

          <template v-else>
            <div class="bloco grafico-bloco">
              <h3>Famílias por tamanho da propriedade</h3>
              <Grafico :montar="(tm) => barrasHorizontais(tm, 'eletro', faixas.map((f) => ({ nome: f.rotulo, valor: f.familias })), 'família(s)')"
                       :altura="alturaBarras(faixas.length)" rotulo="Gráfico de famílias por faixa de tamanho; os valores estão na tabela" @clicar="(i) => detalharFaixa(faixas[i])" />
            </div>
            <TabelaDados :colunas="colFaixas" :linhas="faixas" :chave="(l) => l.chave" rotulo="Tamanho das propriedades" :por-pagina="0" compacta clicavel
                         :totais="{ rotulo: 'Total', familias: num(t.familias), area_ha: t.area_ha }" @abrir="detalharFaixa" />
            <p class="fraco dica">Área convertida para hectares (1 alqueire paulista = 2,42 ha). Propriedades sem área informada não valem zero: aparecem à parte.</p>
          </template>
        </section>
      </template>
    </template>

    <ExportarRelatorio v-if="exportando" modo="geral" :filtro="filtro" :criterios="criterios" :familias="t?.familias ?? 0" @fechar="exportando = false" />
    <Detalhamento v-if="detalhe" :titulo="detalhe.titulo" :filtro="detalhe.filtro" :campo="detalhe.campo" :campos="campos"
                  :equipes="opcoes.equipes" :jornadas="opcoes.jornadas"
                  @fechar="detalhe = null" @abrir="(u) => emit('abrir', u)" @aplicar="aplicarDetalhe" />
  </div>
</template>

<style scoped>
.ic { width: 18px; height: 18px; fill: none; stroke: currentColor; stroke-width: 2; stroke-linecap: round; stroke-linejoin: round; vertical-align: -3px; margin-right: .35rem; }
.cabeca { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.titulo { display: flex; flex-direction: column; gap: .35rem; }
.cartoes { display: grid; gap: .9rem; grid-template-columns: repeat(auto-fill, minmax(min(100%, 235px), 1fr)); }
.vazio-geral { padding: 2.5rem 1.5rem; display: flex; flex-direction: column; align-items: center; gap: .75rem; text-align: center; }
.analises { display: flex; flex-direction: column; gap: .9rem; }
.abas { display: flex; gap: .25rem; border-bottom: 1px solid var(--regua-forte); flex-wrap: wrap; }
.abas button { border: none; border-radius: var(--raio) var(--raio) 0 0; background: transparent; color: var(--tinta-2); font-weight: 600; padding: .6rem 1rem; position: relative; }
.abas button:hover { color: var(--tinta); background: var(--folha-2); }
.abas button.ativa { color: var(--tinta); }
.abas button.ativa::after { content: ""; position: absolute; left: .6rem; right: .6rem; bottom: -1px; height: 3px; border-radius: 3px 3px 0 0; background: var(--mata); }
.controles { display: flex; gap: .75rem 1.25rem; flex-wrap: wrap; align-items: center; }
.controles label { display: inline-flex; align-items: center; gap: .5rem; color: var(--tinta-2); font-size: .92rem; }
.controles select { width: auto; min-width: 150px; }
.grafico-bloco { padding: .9rem 1.1rem; display: flex; flex-direction: column; gap: .5rem; }
.grafico-bloco h3 { font-size: .95rem; color: var(--tinta-2); }
.resp { display: block; font-size: .85rem; }
.dica { font-size: .85rem; }
.segmentado { display: inline-flex; border: 1px solid var(--regua-forte); border-radius: var(--raio); overflow: hidden; flex-wrap: wrap; }
.segmentado button { border: none; border-radius: 0; min-height: 34px; padding: .3rem .85rem; background: var(--folha); color: var(--tinta-2); font-weight: 550; }
.segmentado button + button { border-left: 1px solid var(--regua-forte); }
.segmentado button[aria-checked="true"] { background: var(--mata); color: var(--sobre-mata); }
.cont { font-stretch: 85%; font-weight: 700; margin-left: .2rem; }
</style>
