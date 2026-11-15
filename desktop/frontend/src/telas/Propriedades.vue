<script setup lang="ts">
import { computed, onActivated, onMounted, ref, watch } from 'vue'
import { ConsultaService as Consulta } from '../../bindings/sitposse7/desktop/app'
import type { EquipeLinha, PropriedadeLinha } from '../../bindings/sitposse7/desktop/internal/consultas/models'
import TabelaDados from '../componentes/TabelaDados.vue'
import type { Coluna } from '../tabela'
import { haNumero, mensagem, num, plural, quando } from '../formato'

const emit = defineEmits<{ abrir: [uuid: string] }>()
const lista = ref<PropriedadeLinha[]>([])
const equipes = ref<EquipeLinha[]>([])
const busca = ref('')
const equipe = ref('')
const erro = ref('')
const carregando = ref(true)

async function carregar() {
  try { lista.value = (await Consulta.Propriedades(busca.value, equipe.value)) ?? []; erro.value = '' } catch (e) { erro.value = mensagem(e) } finally { carregando.value = false }
}
let atraso: number | undefined
watch([busca, equipe], () => { window.clearTimeout(atraso); atraso = window.setTimeout(carregar, 250) })
onMounted(async () => { carregar(); try { equipes.value = (await Consulta.Equipes()) ?? [] } catch { /* */ } })
onActivated(carregar)

const colunas: Coluna<PropriedadeLinha>[] = [
  { chave: 'nome', rotulo: 'Propriedade / família', largura: '26%' },
  { chave: 'equipe', rotulo: 'Equipe' },
  { chave: 'pessoas', rotulo: 'Pessoas', tipo: 'numero', formatar: (l) => num(l.pessoas) },
  { chave: 'bovinos', rotulo: 'Bovinos', tipo: 'numero', formatar: (l) => num(l.bovinos) },
  { chave: 'area_ha', rotulo: 'Área (ha)', tipo: 'numero', valor: (l) => haNumero(l.area_ha), formatar: (l) => l.area_ha ?? 'Não informada' },
  { chave: 'tem_gps', rotulo: 'Localização' },
  { chave: 'fotos', rotulo: 'Fotos', tipo: 'numero', secundaria: true },
  { chave: 'recebido_em', rotulo: 'Recebido em', formatar: (l) => quando(l.recebido_em), secundaria: true },
]
const filtrando = computed(() => !!(busca.value.trim() || equipe.value))
const resumo = computed(() => {
  const n = lista.value.length
  return filtrando.value ? `${plural(n, 'família encontrada', 'famílias encontradas')}` : `${plural(n, 'família recebida', 'famílias recebidas')} dos celulares`
})
</script>

<template>
  <div class="pilha">
    <header class="cabeca">
      <h1>Propriedades</h1>
      <p class="suave">{{ resumo }}. Cada propriedade corresponde a uma família.</p>
    </header>
    <div class="filtros">
      <label class="busca">
        <span class="sr">Buscar</span>
        <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="6.5" /><path d="m16 16 4.5 4.5" /></svg>
        <input v-model="busca" type="search" placeholder="Buscar por propriedade ou responsável" />
      </label>
      <select v-model="equipe" aria-label="Equipe">
        <option value="">Todas as equipes</option>
        <option v-for="e in equipes" :key="e.uuid" :value="e.uuid">{{ e.nome }}</option>
      </select>
      <button v-if="filtrando" class="discreto" @click="busca = ''; equipe = ''">Limpar filtros</button>
    </div>
    <TabelaDados :colunas="colunas" :linhas="lista" :chave="(l) => l.uuid" rotulo="Propriedades recebidas" clicavel
                 :carregando="carregando" :erro="erro" :ordem="{ chave: 'nome', dir: 'asc' }" @abrir="(l) => emit('abrir', l.uuid)">
      <template #nome="{ linha }">
        <strong class="nome">{{ linha.nome }}</strong>
        <span class="suave resp">{{ linha.responsavel || 'Responsável não informado' }}</span>
      </template>
      <template #tem_gps="{ linha }">
        <span class="selo" :class="linha.tem_gps ? 'ok' : ''">{{ linha.tem_gps ? 'Com localização' : 'Sem localização' }}</span>
      </template>
      <template #vazio>
        {{ filtrando ? 'Nenhuma propriedade com esse filtro.' : 'Nenhuma propriedade recebida ainda. Receba os cadastros em Sincronização.' }}
      </template>
    </TabelaDados>
  </div>
</template>

<style scoped>
.cabeca { display: flex; flex-direction: column; gap: .35rem; }
.filtros { display: flex; gap: .6rem; flex-wrap: wrap; align-items: center; }
.busca { position: relative; flex: 1 1 340px; }
.busca svg { position: absolute; left: .7rem; top: 50%; translate: 0 -50%; width: 18px; height: 18px; fill: none; stroke: var(--tinta-3); stroke-width: 2; stroke-linecap: round; }
.busca input { padding-left: 2.2rem; }
.filtros select { flex: 0 1 240px; }
.nome { display: block; }
.resp { font-size: .88rem; }
.sr { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
</style>
