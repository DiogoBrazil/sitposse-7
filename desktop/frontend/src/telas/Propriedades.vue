<script setup lang="ts">
/**
 * Lista de propriedades/famílias com os mesmos filtros do Painel (§22-A.2 "Propriedades":
 * nome, responsável, equipe, área identificada, com/sem coordenadas, faixas de tamanho,
 * presença ou ausência de bens ou animais). Clique abre a página da propriedade.
 */
import { computed, onActivated, onMounted, ref, watch } from 'vue'
import { ConsultaService as Consulta } from '../../bindings/sitposse7/desktop/app'
import BarraFiltros from '../componentes/BarraFiltros.vue'
import TabelaDados from '../componentes/TabelaDados.vue'
import { type Filtro, filtroVazio } from '../filtro'
import { type Familia, type Resultado, v } from '../painel'
import { haNumero, mensagem, num, plural, quando } from '../formato'
import type { Coluna } from '../tabela'

const emit = defineEmits<{ abrir: [uuid: string] }>()
const filtro = ref<Filtro>(filtroVazio())
const r = ref<Resultado | null>(null)
const erro = ref('')
const carregando = ref(true)

async function carregar() {
  try { r.value = await Consulta.Estatisticas(filtro.value); erro.value = '' } catch (e) { erro.value = mensagem(e) } finally { carregando.value = false }
}
let atraso: number | undefined
watch(filtro, () => { window.clearTimeout(atraso); atraso = window.setTimeout(carregar, 120) }, { deep: true })
onMounted(carregar)
onActivated(carregar)

const familias = computed(() => r.value?.familias ?? [])
const colunas: Coluna<Familia>[] = [
  { chave: 'nome', rotulo: 'Propriedade / família', largura: '24%' },
  { chave: 'equipe', rotulo: 'Equipe' },
  { chave: 'area_identificada', rotulo: 'Área identificada', secundaria: true },
  { chave: 'pessoas', rotulo: 'Pessoas', tipo: 'numero', valor: (l) => v(l, 'pessoas_total'), formatar: (l) => num(v(l, 'pessoas_total')) },
  { chave: 'bovinos', rotulo: 'Bovinos', tipo: 'numero', valor: (l) => v(l, 'bov_total'), formatar: (l) => num(v(l, 'bov_total')) },
  { chave: 'veiculos', rotulo: 'Veículos', tipo: 'numero', valor: (l) => v(l, 'veic_total'), formatar: (l) => num(v(l, 'veic_total')), secundaria: true },
  { chave: 'area', rotulo: 'Área (ha)', tipo: 'numero', valor: (l) => haNumero(l.area_ha), formatar: (l) => l.area_ha ?? 'Não informada' },
  { chave: 'loc', rotulo: 'Localização', valor: (l) => (l.com_localizacao ? 1 : 0) },
  { chave: 'fotos', rotulo: 'Fotos', tipo: 'numero', valor: (l) => v(l, 'fotos'), secundaria: true },
  { chave: 'recebido_em', rotulo: 'Recebido em', formatar: (l) => quando(l.recebido_em), secundaria: true },
]
const totais = computed(() => {
  const t = r.value?.totais
  return t ? { nome: `Total (${plural(t.familias, 'família', 'famílias')})`, pessoas: num(v(t, 'pessoas_total')), bovinos: num(v(t, 'bov_total')), veiculos: num(v(t, 'veic_total')), area: t.area_ha, fotos: num(v(t, 'fotos')) } : null
})
</script>

<template>
  <div class="pilha">
    <header class="cabeca">
      <h1>Propriedades</h1>
      <p class="suave">Cada propriedade corresponde a uma família. Clique numa linha para ver o cadastro completo.</p>
    </header>
    <BarraFiltros v-model="filtro" :equipes="r?.equipes ?? []" :areas="r?.areas ?? []" :operacoes="r?.operacoes ?? []" :campos="r?.campos ?? []"
                  :mostrando="r?.totais.familias ?? null" :base="r?.familias_base ?? null" />
    <TabelaDados :colunas="colunas" :linhas="familias" :chave="(l) => l.uuid" rotulo="Propriedades recebidas" clicavel :totais="totais"
                 :carregando="carregando" :erro="erro" :ordem="{ chave: 'nome', dir: 'asc' }" @abrir="(l) => emit('abrir', l.uuid)">
      <template #nome="{ linha }">
        <strong class="nome">{{ linha.nome }}</strong>
        <span class="suave resp">{{ linha.responsavel || 'Responsável não informado' }}</span>
      </template>
      <template #area_identificada="{ linha }">{{ linha.area_identificada || '—' }}</template>
      <template #loc="{ linha }">
        <span class="selo" :class="linha.com_localizacao ? 'ok' : ''">{{ linha.com_localizacao ? 'Com localização' : 'Sem localização' }}</span>
      </template>
      <template #vazio>
        {{ r?.familias_base ? 'Nenhuma propriedade com esses filtros.' : 'Nenhuma propriedade recebida ainda. Receba os cadastros em Sincronização.' }}
      </template>
    </TabelaDados>
  </div>
</template>

<style scoped>
.cabeca { display: flex; flex-direction: column; gap: .35rem; }
.nome { display: block; }
.resp { font-size: .88rem; }
</style>
