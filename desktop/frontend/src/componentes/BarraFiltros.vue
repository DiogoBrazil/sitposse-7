<script setup lang="ts">
/**
 * Barra do FiltroEstudo (§22-A.2–22-A.4): busca, equipe e área sempre à vista; os demais
 * critérios em "Mais filtros". Os filtros ativos aparecem como chips removíveis, e
 * "Limpar filtros" volta à visão geral.
 */
import { computed, ref, watch } from 'vue'
import SelecaoMultipla from './SelecaoMultipla.vue'
import { type Campo, chips, dataParaMs, FAIXAS, type Filtro, filtroVazio, msParaData, type Opcao, OPERADORES, rotuloCampo } from '../filtro'
import { num, plural } from '../formato'

const props = defineProps<{
  modelValue: Filtro
  equipes: Opcao[]
  areas: Opcao[]
  operacoes: Opcao[]
  campos: Campo[]
  mostrando?: number | null
  base?: number | null
}>()
const emit = defineEmits<{ 'update:modelValue': [f: Filtro] }>()
const f = computed(() => props.modelValue)
const mudar = (parcial: Partial<Filtro>) => emit('update:modelValue', { ...f.value, ...parcial })

const avancado = ref(false)
const ativos = computed(() => chips(f.value, { equipes: props.equipes, areas: props.areas, operacoes: props.operacoes, campos: props.campos }))
const nAvancados = computed(() => ativos.value.filter((c) => !/^(eq|ar|busca)/.test(c.chave)).length)

// Busca com pequeno atraso para não recalcular a cada tecla.
const busca = ref(f.value.busca)
watch(() => f.value.busca, (v) => { if (v !== busca.value) busca.value = v })
let atraso: number | undefined
watch(busca, (v) => { window.clearTimeout(atraso); atraso = window.setTimeout(() => mudar({ busca: v }), 250) })

type Tri = 'todas' | 'com' | 'sem'
const tri = (v: boolean | null | undefined): Tri => (v == null ? 'todas' : v ? 'com' : 'sem')
const deTri = (t: Tri) => (t === 'todas' ? null : t === 'com')
const opcoesTri: { v: Tri; r: string }[] = [{ v: 'todas', r: 'Todas' }, { v: 'com', r: 'Com' }, { v: 'sem', r: 'Sem' }]

function numeroOuNulo(e: Event): number | null {
  const s = (e.target as HTMLInputElement).value.replace(',', '.').trim()
  if (s === '') return null
  const n = Number(s)
  return Number.isFinite(n) && n >= 0 ? n : null
}

// Condições sobre quantidades, agrupadas como no cadastro.
const grupos = computed(() => {
  const m = new Map<string, Campo[]>()
  for (const c of props.campos) {
    if (!m.has(c.grupo)) m.set(c.grupo, [])
    m.get(c.grupo)!.push(c)
  }
  return [...m.entries()]
})
function novaCondicao() {
  mudar({ condicoes: [...(f.value.condicoes ?? []), { campo: 'bov_total', operador: 'maior', valor: 5 }] })
  avancado.value = true
}
function mudarCondicao(i: number, parcial: Partial<{ campo: string; operador: string; valor: number }>) {
  mudar({ condicoes: (f.value.condicoes ?? []).map((c, j) => (j === i ? { ...c, ...parcial } : c)) })
}
const usaValor = (op: string) => OPERADORES.find((o) => o.valor === op)?.usaValor ?? false
</script>

<template>
  <section class="barra" aria-label="Filtros">
    <div class="linha-principal">
      <label class="busca">
        <span class="sr">Buscar</span>
        <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="6.5" /><path d="m16 16 4.5 4.5" /></svg>
        <input v-model="busca" type="search" placeholder="Buscar família, propriedade ou responsável" />
      </label>
      <SelecaoMultipla rotulo="Equipe" todos="Todas" :opcoes="equipes" :model-value="f.equipes" @update:model-value="(v) => mudar({ equipes: v })" />
      <SelecaoMultipla rotulo="Área" todos="Todas" :opcoes="areas" :model-value="f.areas_identificadas" @update:model-value="(v) => mudar({ areas_identificadas: v })" />
      <button type="button" class="mais" :class="{ aberto: avancado }" :aria-expanded="avancado" @click="avancado = !avancado">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 6h16M7 12h10M10 18h4" /></svg>
        Mais filtros<span v-if="nAvancados" class="cont">{{ nAvancados }}</span>
      </button>
    </div>

    <div v-if="avancado" class="avancado">
      <fieldset>
        <legend>Período</legend>
        <div class="segmentado" role="radiogroup" aria-label="Data considerada">
          <button type="button" role="radio" :aria-checked="f.periodo_campo !== 'recebimento'" @click="mudar({ periodo_campo: 'cadastro' })">Cadastro</button>
          <button type="button" role="radio" :aria-checked="f.periodo_campo === 'recebimento'" @click="mudar({ periodo_campo: 'recebimento' })">Recebimento</button>
        </div>
        <div class="dupla">
          <label>De <input type="date" :value="msParaData(f.de)" @change="mudar({ de: dataParaMs(($event.target as HTMLInputElement).value, false) })" /></label>
          <label>Até <input type="date" :value="msParaData(f.ate)" @change="mudar({ ate: dataParaMs(($event.target as HTMLInputElement).value, true) })" /></label>
        </div>
      </fieldset>

      <fieldset>
        <legend>Registro</legend>
        <div v-for="[campo, rotulo] in ([['com_localizacao', 'Localização'], ['com_foto', 'Fotos'], ['com_area', 'Área informada']] as const)" :key="campo" class="tri">
          <span>{{ rotulo }}</span>
          <div class="segmentado" role="radiogroup" :aria-label="rotulo">
            <button v-for="o in opcoesTri" :key="o.v" type="button" role="radio" :aria-checked="tri(f[campo]) === o.v" @click="mudar({ [campo]: deTri(o.v) })">{{ o.r }}</button>
          </div>
        </div>
      </fieldset>

      <fieldset>
        <legend>Tamanho da propriedade</legend>
        <SelecaoMultipla rotulo="Faixas" todos="Todas" :opcoes="FAIXAS" :model-value="f.faixas_area" @update:model-value="(v) => mudar({ faixas_area: v })" />
        <div class="dupla">
          <label>De (ha) <input inputmode="decimal" :value="f.area_min_ha ?? ''" placeholder="0" @change="mudar({ area_min_ha: numeroOuNulo($event) })" /></label>
          <label>Até (ha) <input inputmode="decimal" :value="f.area_max_ha ?? ''" placeholder="sem limite" @change="mudar({ area_max_ha: numeroOuNulo($event) })" /></label>
        </div>
        <p class="fraco dica">Propriedades sem área informada não entram numa faixa em hectares.</p>
      </fieldset>

      <fieldset v-if="operacoes.length > 1">
        <legend>Operação</legend>
        <SelecaoMultipla rotulo="Operação" todos="Todas" :opcoes="operacoes" :model-value="f.operacoes" @update:model-value="(v) => mudar({ operacoes: v })" />
      </fieldset>

      <fieldset class="condicoes">
        <legend>Pessoas, veículos, animais e bens</legend>
        <p v-if="!(f.condicoes ?? []).length" class="fraco">Por exemplo: bovinos mais de 5, possui caminhão, não possui geladeira.</p>
        <div v-for="(c, i) in f.condicoes ?? []" :key="i" class="condicao">
          <select :value="c.campo" aria-label="Quantidade" @change="mudarCondicao(i, { campo: ($event.target as HTMLSelectElement).value })">
            <optgroup v-for="[g, cs] in grupos" :key="g" :label="g">
              <option v-for="x in cs" :key="x.chave" :value="x.chave">{{ rotuloCampo(x) }}</option>
            </optgroup>
          </select>
          <select :value="c.operador" aria-label="Condição" @change="mudarCondicao(i, { operador: ($event.target as HTMLSelectElement).value })">
            <option v-for="o in OPERADORES" :key="o.valor" :value="o.valor">{{ o.rotulo }}</option>
          </select>
          <input v-if="usaValor(c.operador)" type="number" min="0" step="1" :value="c.valor" aria-label="Valor"
                 @change="mudarCondicao(i, { valor: Math.max(0, Math.floor(Number(($event.target as HTMLInputElement).value) || 0)) })" />
          <span v-else></span>
          <button type="button" class="discreto" :aria-label="`Remover condição ${i + 1}`" @click="mudar({ condicoes: (f.condicoes ?? []).filter((_, j) => j !== i) })">Remover</button>
        </div>
        <button type="button" class="adicionar" @click="novaCondicao">+ Adicionar condição</button>
      </fieldset>
    </div>

    <div v-if="ativos.length || mostrando != null" class="ativos">
      <span v-if="mostrando != null && base != null" class="contagem">
        <strong>{{ num(mostrando) }}</strong> de {{ plural(base, 'família', 'famílias') }}
      </span>
      <ul v-if="ativos.length" class="chips" aria-label="Filtros ativos">
        <li v-for="c in ativos" :key="c.chave">
          {{ c.texto }}
          <button type="button" :aria-label="`Remover filtro: ${c.texto}`" @click="emit('update:modelValue', c.remover(f))">×</button>
        </li>
      </ul>
      <button v-if="ativos.length" type="button" class="discreto limpar" @click="busca = ''; emit('update:modelValue', filtroVazio())">Limpar filtros</button>
    </div>
  </section>
</template>

<style scoped>
.barra { display: flex; flex-direction: column; gap: .7rem; }
.linha-principal { display: grid; grid-template-columns: minmax(220px, 1fr) minmax(170px, 230px) minmax(170px, 230px) auto; gap: .6rem; }
.busca { position: relative; }
.busca svg { position: absolute; left: .7rem; top: 50%; translate: 0 -50%; width: 18px; height: 18px; fill: none; stroke: var(--tinta-3); stroke-width: 2; stroke-linecap: round; }
.busca input { padding-left: 2.2rem; }
.mais { display: inline-flex; align-items: center; gap: .45rem; white-space: nowrap; }
.mais svg { width: 18px; height: 18px; fill: none; stroke: currentColor; stroke-width: 2; stroke-linecap: round; }
.mais.aberto { background: var(--folha-2); }
.cont { background: var(--mata); color: var(--sobre-mata); border-radius: 999px; font-size: .78rem; padding: 0 .45rem; min-width: 1.3rem; text-align: center; }
.avancado {
  display: grid; gap: 1rem 1.5rem; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  background: var(--folha); border: 1px solid var(--regua); border-radius: var(--raio-grande); padding: 1rem 1.1rem;
}
fieldset { border: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: .55rem; min-width: 0; }
legend { font-weight: 650; padding: 0; margin-bottom: .35rem; }
.dupla { display: grid; grid-template-columns: 1fr 1fr; gap: .5rem; }
.dupla label { display: flex; flex-direction: column; gap: .2rem; font-size: .88rem; color: var(--tinta-2); }
.dica { font-size: .82rem; }
.tri { display: flex; align-items: center; justify-content: space-between; gap: .75rem; }
.segmentado { display: inline-flex; align-self: flex-start; border: 1px solid var(--regua-forte); border-radius: var(--raio); overflow: hidden; }
.segmentado button { border: none; border-radius: 0; min-height: 32px; padding: .25rem .7rem; background: var(--folha); font-weight: 500; color: var(--tinta-2); }
.segmentado button + button { border-left: 1px solid var(--regua-forte); }
.segmentado button[aria-checked="true"] { background: var(--mata); color: var(--sobre-mata); font-weight: 650; }
.condicoes { grid-column: 1 / -1; }
.condicao { display: grid; grid-template-columns: minmax(200px, 2fr) minmax(130px, 1fr) minmax(80px, 110px) auto; gap: .5rem; align-items: center; }
.adicionar { align-self: flex-start; border-style: dashed; }
.ativos { display: flex; align-items: center; gap: .6rem; flex-wrap: wrap; }
.contagem { color: var(--tinta-2); margin-right: .25rem; }
.contagem strong { color: var(--tinta); font-stretch: 85%; font-size: 1.1rem; }
.chips { display: contents; list-style: none; }
.chips li { display: inline-flex; align-items: center; gap: .2rem; background: var(--mata-clara); color: var(--tinta); border-radius: 999px; padding: .15rem .2rem .15rem .75rem; font-size: .88rem; font-weight: 550; }
.chips button { border: none; background: transparent; min-height: 0; width: 1.6rem; height: 1.6rem; padding: 0; border-radius: 50%; font-size: 1.1rem; line-height: 1; color: var(--tinta-2); }
.chips button:hover { background: color-mix(in srgb, var(--mata) 18%, transparent); color: var(--tinta); }
.limpar { font-weight: 650; color: var(--mata); }
.sr { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
@media (max-width: 1100px) { .linha-principal { grid-template-columns: 1fr 1fr auto; } .busca { grid-column: 1 / -1; } }
@media (max-width: 640px) { .linha-principal { grid-template-columns: 1fr; } .mais { justify-self: start; } .condicao { grid-template-columns: 1fr 1fr; } }
</style>
