<script setup lang="ts">
/**
 * Gráfico ECharts que acompanha o tema claro/escuro do Windows: as cores vêm de estilo.css
 * e o gráfico é refeito quando o tema muda. Sempre acompanhado de tabela (acessibilidade).
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import VChart from 'vue-echarts'
import type { EChartsOption } from 'echarts'
import { lerTema, type Tema } from '../graficos'

const props = defineProps<{ montar: (t: Tema) => EChartsOption; altura: number; rotulo: string }>()

const tema = ref<Tema | null>(null)
const opcao = computed(() => (tema.value ? props.montar(tema.value) : undefined))
const midia = window.matchMedia('(prefers-color-scheme: dark)')
const atualizar = () => { tema.value = lerTema() }
onMounted(() => { atualizar(); midia.addEventListener('change', atualizar) })
onUnmounted(() => midia.removeEventListener('change', atualizar))
</script>

<template>
  <div class="grafico" role="img" :aria-label="rotulo" :style="{ height: `${altura}px` }">
    <VChart v-if="opcao" :option="opcao" autoresize />
  </div>
</template>

<style scoped>
.grafico { width: 100%; min-width: 0; }
</style>
