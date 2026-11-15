<script setup lang="ts">
/**
 * Cartão de indicador do Painel: número principal e detalhamento por tipo, com barra de
 * proporção na cor do assunto. Cada número abre a relação das famílias que o compõem (§22-A.4).
 */
import { computed } from 'vue'
import { num } from '../formato'

import type { LinhaIndicador, Principal } from '../painel'

const props = defineProps<{ titulo: string; principais: Principal[]; linhas: LinhaIndicador[]; cor?: string; rodape?: string; vazio?: string }>()
const emit = defineEmits<{ detalhar: [chave: string, rotulo: string] }>()
const maximo = computed(() => Math.max(1, ...props.linhas.map((l) => l.valor)))
</script>

<template>
  <article class="cartao" :style="cor ? { '--cor': `var(${cor})` } : undefined">
    <h3>{{ titulo }}</h3>
    <div class="principais">
      <button v-for="p in principais" :key="p.chave" type="button" class="principal" :title="`Ver as famílias: ${p.rotulo}`" @click="emit('detalhar', p.chave, p.rotulo)">
        <span class="valor">{{ p.texto ?? num(p.valor) }}</span>
        <span class="rotulo">{{ p.rotulo }}</span>
      </button>
    </div>
    <ul v-if="linhas.length" class="linhas">
      <li v-for="l in linhas" :key="l.chave">
        <button type="button" :title="`Ver as famílias: ${l.rotulo}`" @click="emit('detalhar', l.chave, l.rotulo)">
          <span class="l-rot">{{ l.rotulo }}</span>
          <span class="l-val">{{ l.texto ?? num(l.valor) }}</span>
          <span class="trilho" aria-hidden="true"><span :style="{ width: `${(l.valor / maximo) * 100}%` }"></span></span>
        </button>
      </li>
    </ul>
    <p v-else-if="vazio" class="fraco vazio">{{ vazio }}</p>
    <p v-if="rodape" class="rodape fraco">{{ rodape }}</p>
  </article>
</template>

<style scoped>
.cartao { background: var(--folha); border: 1px solid var(--regua); border-radius: var(--raio-grande); padding: .95rem 1rem 1rem; display: flex; flex-direction: column; gap: .55rem; min-width: 0; }
h3 { font-size: .95rem; font-weight: 650; color: var(--tinta-2); }
.principais { display: flex; gap: 1.1rem; flex-wrap: wrap; }
.principal { all: unset; cursor: pointer; display: flex; flex-direction: column; border-radius: 4px; }
.principal:hover .valor { color: var(--mata); }
.principal:focus-visible { outline: 2px solid var(--foco); outline-offset: 3px; }
.valor { font-size: clamp(1.9rem, 2.4vw, 2.3rem); line-height: 1.05; font-weight: 720; font-stretch: 78%; letter-spacing: -0.01em; }
.rotulo { font-size: .85rem; color: var(--tinta-2); }
.linhas { list-style: none; margin: .1rem 0 0; padding: .55rem 0 0; border-top: 1px solid var(--regua); display: flex; flex-direction: column; gap: .15rem; }
.linhas button { all: unset; cursor: pointer; display: grid; grid-template-columns: 1fr auto; gap: .1rem .75rem; width: 100%; padding: .2rem .3rem; margin: 0 -.3rem; border-radius: 4px; }
.linhas button:hover { background: var(--folha-2); }
.linhas button:focus-visible { outline: 2px solid var(--foco); }
.l-rot { font-size: .88rem; color: var(--tinta-2); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.l-val { font-size: .92rem; font-weight: 650; text-align: right; }
.trilho { grid-column: 1 / -1; height: 4px; border-radius: 2px; background: var(--folha-2); overflow: hidden; }
.trilho span { display: block; height: 100%; border-radius: 2px; background: var(--cor, var(--mata)); }
.rodape { font-size: .82rem; margin-top: auto; }
.vazio { font-size: .88rem; }
</style>
