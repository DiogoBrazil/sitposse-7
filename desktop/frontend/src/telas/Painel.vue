<script setup lang="ts">
import { computed, onActivated, onDeactivated, onMounted, onUnmounted, ref } from 'vue'
import { ConsultaService as Consulta, OperacaoService as Op } from '../../bindings/sitposse7/desktop/app'
import type { PropriedadeLinha, Resumo } from '../../bindings/sitposse7/desktop/internal/consultas/models'
import type { Operacao } from '../../bindings/sitposse7/desktop/internal/pareamento/models'
import FaixaTotais from '../componentes/FaixaTotais.vue'
import Grafico from '../componentes/Grafico.vue'
import TabelaDados from '../componentes/TabelaDados.vue'
import { alturaBarras, barrasHorizontais, type GrupoTotal } from '../graficos'
import { ha, haNumero, mensagem, num, plural, quando } from '../formato'
import type { Tela } from '../navegacao'
import type { Coluna } from '../tabela'

const emit = defineEmits<{ ir: [tela: Tela]; abrir: [uuid: string] }>()
const r = ref<Resumo | null>(null)
const familias = ref<PropriedadeLinha[]>([])
const op = ref<Operacao | null>(null)
const erro = ref('')

async function carregar() {
  try {
    const [a, b, c] = await Promise.all([Consulta.Resumo(), Consulta.Propriedades('', ''), Op.Ativa()])
    r.value = a; familias.value = b ?? []; op.value = c; erro.value = ''
  } catch (e) { erro.value = mensagem(e) }
}
// Atualiza a cada 5 s só enquanto o Painel está na tela.
let timer: number | undefined
const iniciar = () => { carregar(); window.clearInterval(timer); timer = window.setInterval(carregar, 5000) }
const parar = () => window.clearInterval(timer)
onMounted(iniciar); onActivated(iniciar); onDeactivated(parar); onUnmounted(parar)

const totais = computed<GrupoTotal[]>(() => {
  const x = r.value
  if (!x) return []
  return [
    { titulo: 'Famílias', itens: [{ valor: num(x.propriedades), rotulo: 'propriedades' }, { valor: num(x.equipes), rotulo: 'equipes', detalhe: plural(x.aparelhos, 'aparelho', 'aparelhos') }] },
    { titulo: 'Pessoas', cor: '--s2', itens: [{ valor: num(x.pessoas), rotulo: 'moradores' }] },
    { titulo: 'Rebanho', cor: '--s3', itens: [{ valor: num(x.bovinos), rotulo: 'bovinos' }, { valor: num(x.equinos), rotulo: 'equinos' }] },
    { titulo: 'Aves', cor: '--s6', itens: [{ valor: num(x.aves), rotulo: 'de criação' }] },
    { titulo: 'Veículos', cor: '--s4', itens: [{ valor: num(x.veiculos), rotulo: 'no total' }] },
    { titulo: 'Bens', cor: '--s1', itens: [{ valor: num(x.eletrodomesticos), rotulo: 'eletrodomésticos' }, { valor: num(x.moveis), rotulo: 'móveis' }] },
    { titulo: 'Território', itens: [{ valor: x.area_ha, rotulo: 'hectares informados', detalhe: x.sem_area ? `${plural(x.sem_area, 'propriedade', 'propriedades')} sem área` : 'todas com área' }] },
  ]
})

const cobertura = computed(() => {
  const x = r.value
  const n = x?.propriedades ?? 0
  const comFoto = familias.value.filter((p) => p.fotos > 0).length
  return [
    { rotulo: 'Com localização (GPS ou digitada)', valor: x?.com_gps ?? 0, n },
    { rotulo: 'Com área informada', valor: x?.com_area ?? 0, n },
    { rotulo: 'Com pelo menos uma foto', valor: comFoto, n },
  ]
})
const pct = (v: number, n: number) => (n ? Math.round((v / n) * 100) : 0)

interface LinhaEquipe { uuid: string; equipe: string; familias: number; pessoas: number; bovinos: number; area: number; semArea: number; comGps: number }
const porEquipe = computed<LinhaEquipe[]>(() => {
  const m = new Map<string, LinhaEquipe>()
  for (const p of familias.value) {
    const e = m.get(p.equipe_uuid) ?? { uuid: p.equipe_uuid, equipe: p.equipe, familias: 0, pessoas: 0, bovinos: 0, area: 0, semArea: 0, comGps: 0 }
    const a = haNumero(p.area_ha)
    e.familias++; e.pessoas += p.pessoas; e.bovinos += p.bovinos; e.comGps += p.tem_gps ? 1 : 0
    if (a == null) e.semArea++; else e.area += a
    m.set(p.equipe_uuid, e)
  }
  return [...m.values()]
})
const colEquipe: Coluna<LinhaEquipe>[] = [
  { chave: 'equipe', rotulo: 'Equipe' },
  { chave: 'familias', rotulo: 'Famílias', tipo: 'numero', formatar: (l) => num(l.familias) },
  { chave: 'pessoas', rotulo: 'Pessoas', tipo: 'numero', formatar: (l) => num(l.pessoas) },
  { chave: 'bovinos', rotulo: 'Bovinos', tipo: 'numero', formatar: (l) => num(l.bovinos) },
  { chave: 'area', rotulo: 'Área (ha)', tipo: 'numero', formatar: (l) => ha(l.area) + (l.semArea ? ` (${l.semArea} sem área)` : '') },
  { chave: 'comGps', rotulo: 'Com localização', tipo: 'numero', formatar: (l) => `${l.comGps} de ${l.familias}` },
]
const totalEquipes = computed(() => {
  const s = (f: (l: LinhaEquipe) => number) => porEquipe.value.reduce((t, l) => t + f(l), 0)
  const semArea = s((l) => l.semArea)
  return {
    equipe: 'Total geral', familias: num(s((l) => l.familias)), pessoas: num(s((l) => l.pessoas)), bovinos: num(s((l) => l.bovinos)),
    area: ha(s((l) => l.area)) + (semArea ? ` (${semArea} sem área)` : ''), comGps: `${s((l) => l.comGps)} de ${s((l) => l.familias)}`,
  }
})

const colFamilia: Coluna<PropriedadeLinha>[] = [
  { chave: 'nome', rotulo: 'Família / propriedade' },
  { chave: 'equipe', rotulo: 'Equipe' },
  { chave: 'pessoas', rotulo: 'Pessoas', tipo: 'numero', formatar: (l) => num(l.pessoas) },
  { chave: 'bovinos', rotulo: 'Bovinos', tipo: 'numero', formatar: (l) => num(l.bovinos) },
  { chave: 'area_ha', rotulo: 'Área (ha)', tipo: 'numero', valor: (l) => haNumero(l.area_ha), formatar: (l) => l.area_ha ?? 'Não informada' },
  { chave: 'tem_gps', rotulo: 'Localização', valor: (l) => (l.tem_gps ? 'Com localização' : 'Sem localização'), secundaria: true },
]
const totalFamilias = computed(() => ({
  nome: `Total geral (${plural(familias.value.length, 'família', 'famílias')})`,
  pessoas: num(r.value?.pessoas ?? 0), bovinos: num(r.value?.bovinos ?? 0), area_ha: r.value?.area_ha ?? '',
}))

const MAX_BARRAS = 12
const topo = (f: (p: PropriedadeLinha) => number) =>
  [...familias.value].sort((a, b) => f(b) - f(a)).slice(0, MAX_BARRAS).map((p) => ({ nome: p.nome, valor: f(p) }))
const pessoasFamilia = computed(() => topo((p) => p.pessoas))
const bovinosFamilia = computed(() => topo((p) => p.bovinos))
const recorte = computed(() => familias.value.length > MAX_BARRAS ? ` (as ${MAX_BARRAS} maiores)` : '')
</script>

<template>
  <div class="pilha">
    <header class="cabeca">
      <div class="titulo">
        <h1>Painel</h1>
        <p class="suave">{{ op ? `Operação ${op.nome}` : 'Nenhuma operação ativa' }}<template v-if="r?.ultima_importacao"> · última importação em {{ quando(r.ultima_importacao) }}</template></p>
      </div>
      <button class="primario" @click="emit('ir', 'sincronizacao')">{{ op ? 'Receber dos celulares' : 'Criar operação' }}</button>
    </header>
    <p v-if="erro" class="aviso-erro">{{ erro }}</p>

    <template v-if="r">
      <FaixaTotais :grupos="totais" />

      <div class="grade-2">
        <section class="bloco">
          <div class="bloco-cabeca"><h2>Cobertura do levantamento</h2><span class="suave">{{ plural(r.propriedades, 'família', 'famílias') }}</span></div>
          <div class="bloco-corpo medidores">
            <div v-for="c in cobertura" :key="c.rotulo" class="medidor">
              <div class="medidor-txt"><span>{{ c.rotulo }}</span><strong>{{ c.valor }} de {{ c.n }}</strong></div>
              <div class="trilho" role="meter" :aria-label="c.rotulo" :aria-valuenow="c.valor" aria-valuemin="0" :aria-valuemax="c.n">
                <div class="preenchido" :style="{ width: `${pct(c.valor, c.n)}%` }"></div>
              </div>
              <span class="fraco">{{ pct(c.valor, c.n) }}%</span>
            </div>
          </div>
        </section>
        <section class="bloco">
          <div class="bloco-cabeca"><h2>Recepção</h2><button class="discreto" @click="emit('ir', 'sincronizacao')">Ver lotes</button></div>
          <div class="bloco-corpo">
            <dl class="dados">
              <dt>Lotes importados</dt><dd><strong>{{ num(r.lotes_importados) }}</strong></dd>
              <dt>Última importação</dt><dd>{{ quando(r.ultima_importacao) }}</dd>
              <dt>Fotos recebidas</dt><dd>{{ num(r.fotos) }}</dd>
              <dt>Equipes e aparelhos</dt><dd>{{ plural(r.equipes, 'equipe', 'equipes') }}, {{ plural(r.aparelhos, 'aparelho', 'aparelhos') }}</dd>
            </dl>
          </div>
        </section>
      </div>

      <section class="pilha secao">
        <h2>Por equipe</h2>
        <TabelaDados :colunas="colEquipe" :linhas="porEquipe" :chave="(l) => l.uuid" rotulo="Totais por equipe" :por-pagina="0" :totais="totalEquipes"
                     :ordem="{ chave: 'familias', dir: 'desc' }" texto-vazio="Nenhuma equipe com cadastros recebidos." />
      </section>

      <section class="pilha secao">
        <h2>Por família</h2>
        <div v-if="familias.length" class="grade-2">
          <div class="bloco">
            <div class="bloco-cabeca"><h3>Pessoas por família{{ recorte }}</h3></div>
            <div class="bloco-corpo"><Grafico :montar="(t) => barrasHorizontais(t, 'pessoas', pessoasFamilia, 'pessoa(s)')" :altura="alturaBarras(pessoasFamilia.length)" rotulo="Gráfico de pessoas por família; os valores estão na tabela Por família" /></div>
          </div>
          <div class="bloco">
            <div class="bloco-cabeca"><h3>Bovinos por família{{ recorte }}</h3></div>
            <div class="bloco-corpo"><Grafico :montar="(t) => barrasHorizontais(t, 'rebanho', bovinosFamilia, 'bovino(s)')" :altura="alturaBarras(bovinosFamilia.length)" rotulo="Gráfico de bovinos por família; os valores estão na tabela Por família" /></div>
          </div>
        </div>
        <TabelaDados :colunas="colFamilia" :linhas="familias" :chave="(l) => l.uuid" rotulo="Totais por família" clicavel :totais="totalFamilias"
                     :ordem="{ chave: 'nome', dir: 'asc' }" texto-vazio="Nenhuma família recebida ainda." @abrir="(l) => emit('abrir', l.uuid)" />
      </section>

      <p class="fraco">Os números consideram só os cadastros vigentes (sem os excluídos) e são calculados neste computador, sem internet.</p>
    </template>
  </div>
</template>

<style scoped>
.cabeca { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.titulo { display: flex; flex-direction: column; gap: .35rem; }
.grade-2 { display: grid; gap: 1.25rem; grid-template-columns: repeat(auto-fit, minmax(min(100%, 440px), 1fr)); align-items: start; }
.secao { gap: .75rem; }
.medidores { display: flex; flex-direction: column; gap: .9rem; }
.medidor { display: grid; grid-template-columns: 1fr 3.2rem; gap: .3rem .75rem; align-items: center; }
.medidor-txt { grid-column: 1 / -1; display: flex; justify-content: space-between; gap: 1rem; }
.trilho { height: 8px; border-radius: 4px; background: var(--folha-2); border: 1px solid var(--regua); overflow: hidden; }
.preenchido { height: 100%; background: var(--mata); border-radius: 0 4px 4px 0; }
.medidor .fraco { text-align: right; font-size: .85rem; }
</style>
