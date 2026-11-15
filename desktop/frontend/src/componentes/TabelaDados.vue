<script setup lang="ts" generic="T">
/**
 * Tabela da Central (ADR-0019): divisórias visíveis, cabeçalho fixo, ordenação acessível,
 * paginação, estados vazio/carregando/erro e layout empilhado em largura estreita.
 * Células personalizadas: slot com o nome da coluna (`#nome="{ linha }"`).
 */
import { computed, ref, watch } from 'vue'
import { type Coluna, type Direcao, ordenar, paginar } from '../tabela'


const props = withDefaults(defineProps<{
  colunas: Coluna<T>[]
  linhas: T[]
  chave: (l: T) => string
  rotulo: string
  porPagina?: number
  ordem?: { chave: string; dir: Direcao }
  carregando?: boolean
  erro?: string
  textoVazio?: string
  clicavel?: boolean
  selecionada?: string | null
  /** Linha de totais (já formatados) por chave de coluna. */
  totais?: Record<string, string> | null
  /** Tabela pequena (duas ou três colunas): não empilha as linhas em largura estreita. */
  compacta?: boolean
}>(), { porPagina: 25, carregando: false, erro: '', textoVazio: 'Nada para mostrar.', clicavel: false, selecionada: null, totais: null })

const emit = defineEmits<{ abrir: [linha: T] }>()

const chaveOrdem = ref(props.ordem?.chave ?? '')
const dir = ref<Direcao>(props.ordem?.dir ?? 'asc')
const pagina = ref(1)
const tamanho = ref(props.porPagina)

const valorDe = (c: Coluna<T>, l: T) => (c.valor ? c.valor(l) : (l as Record<string, unknown>)[c.chave])
function texto(c: Coluna<T>, l: T): string {
  if (c.formatar) return c.formatar(l)
  const v = valorDe(c, l)
  return v === null || v === undefined || v === '' ? '—' : String(v)
}

const ordenadas = computed(() => {
  const c = props.colunas.find((x) => x.chave === chaveOrdem.value)
  return c ? ordenar(props.linhas, (l) => valorDe(c, l), dir.value) : [...props.linhas]
})
const pag = computed(() => paginar(ordenadas.value, pagina.value, tamanho.value))
watch(() => props.linhas, () => { pagina.value = pag.value.pagina })

function ordenarPor(c: Coluna<T>) {
  if (c.ordenavel === false) return
  if (chaveOrdem.value === c.chave) dir.value = dir.value === 'asc' ? 'desc' : 'asc'
  else { chaveOrdem.value = c.chave; dir.value = c.tipo === 'numero' ? 'desc' : 'asc' }
  pagina.value = 1
}
function ariaSort(c: Coluna<T>) {
  return chaveOrdem.value === c.chave ? (dir.value === 'asc' ? 'ascending' : 'descending') : 'none'
}
function teclaLinha(e: KeyboardEvent, l: T) {
  if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); emit('abrir', l) }
}
const tamanhos = [10, 25, 50, 100]
</script>

<template>
  <div class="tabela-dados" :class="{ empilha: !compacta }">
    <div class="rolagem">
      <table :aria-label="rotulo" :aria-busy="carregando">
        <thead>
          <tr>
            <th v-for="c in colunas" :key="c.chave" scope="col" :aria-sort="c.ordenavel === false ? undefined : ariaSort(c)"
                :class="{ num: c.tipo === 'numero', secundaria: c.secundaria }" :style="c.largura ? { width: c.largura } : undefined">
              <button v-if="c.ordenavel !== false" type="button" class="ordenar" @click="ordenarPor(c)">
                <span>{{ c.rotulo }}</span>
                <svg class="seta" :class="{ ativa: chaveOrdem === c.chave, desc: chaveOrdem === c.chave && dir === 'desc' }" viewBox="0 0 10 10" aria-hidden="true"><path d="M5 2 8.5 7h-7z" /></svg>
              </button>
              <span v-else>{{ c.rotulo }}</span>
            </th>
          </tr>
        </thead>
        <tbody v-if="!erro && pag.itens.length">
          <tr v-for="l in pag.itens" :key="chave(l)" :class="{ clicavel, sel: selecionada === chave(l) }"
              :tabindex="clicavel ? 0 : undefined" @click="clicavel && emit('abrir', l)" @keydown="clicavel && teclaLinha($event, l)">
            <td v-for="c in colunas" :key="c.chave" :data-rotulo="c.rotulo" :class="{ num: c.tipo === 'numero', secundaria: c.secundaria }">
              <div class="cel"><slot :name="c.chave" :linha="l">{{ texto(c, l) }}</slot></div>
            </td>
          </tr>
        </tbody>
        <tfoot v-if="totais && !erro && pag.itens.length">
          <tr>
            <td v-for="(c, i) in colunas" :key="c.chave" :data-rotulo="c.rotulo" :class="{ num: c.tipo === 'numero', secundaria: c.secundaria, vazia: i > 0 && !totais[c.chave] }">
              {{ i === 0 ? (totais[c.chave] ?? 'Total') : (totais[c.chave] ?? '') }}
            </td>
          </tr>
        </tfoot>
      </table>
      <div v-if="carregando && !linhas.length" class="estado" role="status">Carregando…</div>
      <div v-else-if="erro" class="estado erro" role="alert">{{ erro }}</div>
      <div v-else-if="!linhas.length" class="estado"><slot name="vazio">{{ textoVazio }}</slot></div>
    </div>
    <div v-if="!erro && tamanho > 0 && pag.total > Math.min(tamanho, tamanhos[0]) && (pag.paginas > 1 || tamanho !== porPagina)" class="rodape">
      <span class="suave">{{ pag.inicio }}–{{ pag.fim }} de {{ pag.total }}</span>
      <label class="por-pagina suave">Por página
        <select v-model.number="tamanho" @change="pagina = 1"><option v-for="t in tamanhos" :key="t" :value="t">{{ t }}</option></select>
      </label>
      <div class="paginas">
        <button type="button" class="discreto" :disabled="pag.pagina <= 1" aria-label="Página anterior" @click="pagina = pag.pagina - 1">‹ Anterior</button>
        <span>{{ pag.pagina }} / {{ pag.paginas }}</span>
        <button type="button" class="discreto" :disabled="pag.pagina >= pag.paginas" aria-label="Próxima página" @click="pagina = pag.pagina + 1">Próxima ›</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tabela-dados { container-type: inline-size; background: var(--folha); border: 1px solid var(--regua); border-radius: var(--raio-grande); overflow: hidden; }
.rolagem { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; }
th, td { padding: .7rem .9rem; border-bottom: 1px solid var(--regua); text-align: left; vertical-align: middle; }
thead th { position: sticky; top: 0; z-index: 1; background: var(--folha-2); border-bottom: 1px solid var(--regua-forte); font-size: .85rem; font-weight: 650; color: var(--tinta-2); white-space: nowrap; padding-top: .55rem; padding-bottom: .55rem; }
th.num, td.num { text-align: right; white-space: nowrap; }
.ordenar { all: unset; cursor: pointer; display: inline-flex; align-items: center; gap: .35rem; border-radius: 4px; }
th.num .ordenar { flex-direction: row-reverse; }
.ordenar:focus-visible { outline: 2px solid var(--foco); outline-offset: 2px; }
.ordenar:hover span { color: var(--tinta); }
.seta { width: 9px; height: 9px; fill: var(--regua-forte); transform: rotate(180deg); transition: transform .15s, opacity .15s; opacity: 0; }
.ordenar:hover .seta, .ordenar:focus-visible .seta { opacity: 1; }
.seta.ativa { fill: var(--mata); transform: rotate(0deg); opacity: 1; }
.seta.ativa.desc { transform: rotate(180deg); }
tbody tr:last-child td { border-bottom: none; }
tbody tr.clicavel { cursor: pointer; }
tbody tr.clicavel:hover { background: var(--folha-2); }
tbody tr.clicavel:focus-visible { outline: 2px solid var(--foco); outline-offset: -2px; }
tbody tr.sel { background: var(--mata-clara); box-shadow: inset 3px 0 0 var(--mata); }
tfoot td { border-top: 1px solid var(--regua-forte); border-bottom: none; font-weight: 700; background: var(--folha-2); }
.estado { padding: 2.25rem 1rem; text-align: center; color: var(--tinta-2); }
.estado.erro { color: var(--erro); }
.rodape { display: flex; align-items: center; gap: 1rem; flex-wrap: wrap; padding: .5rem .9rem; border-top: 1px solid var(--regua-forte); background: var(--folha-2); font-size: .9rem; }
.por-pagina { display: inline-flex; align-items: center; gap: .5rem; }
.por-pagina select { width: auto; min-height: 32px; padding: .2rem .5rem; }
.paginas { margin-left: auto; display: inline-flex; align-items: center; gap: .5rem; }

/* Largura estreita: cada linha vira um bloco com rótulo: valor, mantendo as divisórias. */
/* Largura média: as colunas secundárias saem primeiro. */
@container (max-width: 900px) {
  th.secundaria, td.secundaria { display: none; }
}
@container (max-width: 640px) {
  .empilha thead { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
  .empilha table, .empilha tbody, .empilha tfoot, .empilha tr, .empilha td { display: block; width: 100%; }
  .empilha tbody tr, .empilha tfoot tr { padding: .6rem .9rem; border-bottom: 1px solid var(--regua); }
  .empilha td { border: none; padding: .2rem 0; display: grid; grid-template-columns: minmax(7rem, 40%) 1fr; gap: .75rem; text-align: left !important; }
  .empilha td::before { content: attr(data-rotulo); color: var(--tinta-2); font-size: .85rem; }
  .empilha td .cel { min-width: 0; }
  .empilha td.secundaria, .empilha tfoot td.vazia { display: none; }
}
</style>
