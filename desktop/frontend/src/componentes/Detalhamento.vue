<script setup lang="ts">
/**
 * Drill-down (§22-A.4, RN-82): a relação das famílias que compõem um número, calculada com
 * o mesmo FiltroEstudo do Painel mais o critério do número clicado. Cada linha abre o cadastro.
 */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { ConsultaService as Consulta } from '../../bindings/sitposse7/desktop/app'
import TabelaDados from './TabelaDados.vue'
import { type Campo, chips, type Filtro, type Opcao } from '../filtro'
import { type Familia, type Resultado, v } from '../painel'
import { haNumero, mensagem, num, plural } from '../formato'
import type { Coluna } from '../tabela'

const props = defineProps<{ titulo: string; filtro: Filtro; campo?: string | null; campos: Campo[]; equipes: Opcao[]; areas: Opcao[]; operacoes: Opcao[] }>()
const emit = defineEmits<{ fechar: []; abrir: [uuid: string]; aplicar: [f: Filtro] }>()
const r = ref<Resultado | null>(null)
const erro = ref('')
const carregando = ref(true)

async function carregar() {
  carregando.value = true
  try { r.value = await Consulta.Estatisticas(props.filtro); erro.value = '' } catch (e) { erro.value = mensagem(e) } finally { carregando.value = false }
}
watch(() => props.filtro, carregar, { deep: true })
const tecla = (e: KeyboardEvent) => { if (e.key === 'Escape') emit('fechar') }
onMounted(() => { carregar(); window.addEventListener('keydown', tecla) })
onUnmounted(() => window.removeEventListener('keydown', tecla))

const rotuloCampo = computed(() => props.campos.find((c) => c.chave === props.campo)?.rotulo ?? '')
const criterios = computed(() => chips(props.filtro, { equipes: props.equipes, areas: props.areas, operacoes: props.operacoes, campos: props.campos }).map((c) => c.texto))

const colunas = computed<Coluna<Familia>[]>(() => [
  { chave: 'nome', rotulo: 'Família / propriedade' },
  { chave: 'equipe', rotulo: 'Equipe' },
  ...(props.campo && !['pessoas_total', 'bov_total'].includes(props.campo)
    ? [{ chave: 'campo', rotulo: rotuloCampo.value, tipo: 'numero' as const, valor: (l: Familia) => v(l, props.campo!), formatar: (l: Familia) => num(v(l, props.campo!)) }]
    : []),
  { chave: 'pessoas', rotulo: 'Pessoas', tipo: 'numero', valor: (l) => v(l, 'pessoas_total'), formatar: (l) => num(v(l, 'pessoas_total')) },
  { chave: 'bovinos', rotulo: 'Bovinos', tipo: 'numero', valor: (l) => v(l, 'bov_total'), formatar: (l) => num(v(l, 'bov_total')) },
  { chave: 'area', rotulo: 'Área (ha)', tipo: 'numero', valor: (l) => haNumero(l.area_ha), formatar: (l) => l.area_ha ?? 'Não informada' },
  { chave: 'loc', rotulo: 'Localização', valor: (l) => (l.com_localizacao ? 'Com localização' : 'Sem localização'), secundaria: true },
])
const totais = computed(() => {
  const t = r.value?.totais
  if (!t) return null
  const out: Record<string, string> = { nome: `Total (${plural(t.familias, 'família', 'famílias')})`, pessoas: num(v(t, 'pessoas_total')), bovinos: num(v(t, 'bov_total')), area: t.area_ha }
  if (props.campo) out.campo = num(v(t, props.campo))
  return out
})
</script>

<template>
  <div class="modal-fundo" @click.self="emit('fechar')">
    <div class="modal largo pilha" role="dialog" aria-modal="true" aria-labelledby="det-titulo">
      <div class="cabeca">
        <div>
          <h2 id="det-titulo">{{ titulo }}</h2>
          <p class="suave">{{ r ? plural(r.totais.familias, 'família compõe', 'famílias compõem') + ' este número' : 'Calculando…' }}</p>
        </div>
        <button class="discreto" autofocus @click="emit('fechar')">Fechar</button>
      </div>
      <ul v-if="criterios.length" class="criterios" aria-label="Critérios">
        <li v-for="c in criterios" :key="c">{{ c }}</li>
      </ul>
      <TabelaDados :colunas="colunas" :linhas="r?.familias ?? []" :chave="(l) => l.uuid" :rotulo="titulo" clicavel :carregando="carregando" :erro="erro"
                   :totais="totais" :ordem="campo ? { chave: 'campo', dir: 'desc' } : { chave: 'nome', dir: 'asc' }" :por-pagina="10"
                   texto-vazio="Nenhuma família com esses critérios." @abrir="(l) => emit('abrir', l.uuid)">
        <template #nome="{ linha }">
          <strong>{{ linha.nome }}</strong>
          <span class="suave resp">{{ linha.responsavel || 'Responsável não informado' }}</span>
        </template>
      </TabelaDados>
      <div class="acoes">
        <span class="fraco">Clique numa família para abrir o cadastro.</span>
        <button class="primario" @click="emit('aplicar', filtro)">Filtrar o Painel por estes critérios</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.largo { max-width: 1040px; }
.cabeca { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; }
.cabeca p { margin-top: .25rem; }
.criterios { list-style: none; margin: 0; padding: 0; display: flex; flex-wrap: wrap; gap: .4rem; }
.criterios li { background: var(--mata-clara); border-radius: 999px; padding: .15rem .7rem; font-size: .85rem; font-weight: 550; }
.resp { display: block; font-size: .85rem; }
.acoes { display: flex; justify-content: space-between; align-items: center; gap: 1rem; flex-wrap: wrap; }
</style>
