<script setup lang="ts">
/** Lista suspensa com caixas de seleção (equipes, áreas, faixas). Fecha com Esc ou clique fora. */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import type { Opcao } from '../filtro'

const props = defineProps<{ rotulo: string; opcoes: Opcao[]; modelValue: string[] | null; todos?: string }>()
const emit = defineEmits<{ 'update:modelValue': [v: string[]] }>()
const aberto = ref(false)
const raiz = ref<HTMLElement | null>(null)
const sel = computed(() => props.modelValue ?? [])

const resumo = computed(() => {
  if (!sel.value.length) return props.todos ?? 'Todas'
  if (sel.value.length === 1) return props.opcoes.find((o) => o.valor === sel.value[0])?.rotulo ?? '1 selecionada'
  return `${sel.value.length} selecionadas`
})
function alternar(v: string) {
  emit('update:modelValue', sel.value.includes(v) ? sel.value.filter((x) => x !== v) : [...sel.value, v])
}
const fora = (e: MouseEvent) => { if (aberto.value && raiz.value && !raiz.value.contains(e.target as Node)) aberto.value = false }
const tecla = (e: KeyboardEvent) => { if (e.key === 'Escape') aberto.value = false }
onMounted(() => { document.addEventListener('mousedown', fora); document.addEventListener('keydown', tecla) })
onUnmounted(() => { document.removeEventListener('mousedown', fora); document.removeEventListener('keydown', tecla) })
</script>

<template>
  <div ref="raiz" class="selecao">
    <button type="button" class="gatilho" :class="{ ativo: sel.length }" :aria-expanded="aberto" aria-haspopup="listbox" @click="aberto = !aberto">
      <span class="rot">{{ rotulo }}</span>
      <span class="val">{{ resumo }}</span>
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
    </button>
    <div v-if="aberto" class="lista" role="listbox" :aria-label="rotulo" aria-multiselectable="true">
      <p v-if="!opcoes.length" class="fraco vazio">Nenhuma opção ainda.</p>
      <label v-for="o in opcoes" :key="o.valor" class="opcao">
        <input type="checkbox" :checked="sel.includes(o.valor)" @change="alternar(o.valor)" />
        <span>{{ o.rotulo }}</span>
      </label>
      <button v-if="sel.length" type="button" class="discreto limpar" @click="emit('update:modelValue', [])">Limpar seleção</button>
    </div>
  </div>
</template>

<style scoped>
.selecao { position: relative; }
.gatilho { display: inline-flex; align-items: center; gap: .45rem; width: 100%; justify-content: space-between; text-align: left; font-weight: 500; }
.gatilho.ativo { border-color: var(--mata); box-shadow: inset 0 0 0 1px var(--mata); }
.rot { color: var(--tinta-2); }
.val { font-weight: 650; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.gatilho svg { width: 16px; height: 16px; fill: none; stroke: currentColor; stroke-width: 2; flex: none; }
.lista {
  position: absolute; z-index: 20; top: calc(100% + 4px); left: 0; min-width: 100%; width: max-content; max-width: 340px; max-height: 320px; overflow: auto;
  background: var(--folha); border: 1px solid var(--regua-forte); border-radius: var(--raio); box-shadow: var(--sombra-modal); padding: .35rem;
}
.opcao { display: flex; gap: .6rem; align-items: center; padding: .45rem .55rem; border-radius: 4px; cursor: pointer; }
.opcao:hover { background: var(--folha-2); }
.opcao input { width: 16px; height: 16px; min-height: 0; accent-color: var(--mata); }
.vazio { padding: .5rem; }
.limpar { width: 100%; margin-top: .25rem; border-top: 1px solid var(--regua); border-radius: 0; }
</style>
