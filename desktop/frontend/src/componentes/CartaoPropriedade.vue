<script setup lang="ts">
/**
 * Cartão da propriedade no mapa da carta-imagem (A30): número fixo no estudo, nome e
 * responsável. Fica aberto até ser fechado (×, Esc ou clique no mapa) ou até outra
 * propriedade ser escolhida. Para pinos sobrepostos, lista todos e permite escolher um.
 */
import { computed } from 'vue'
import type { PinoMapa } from '../cartas'

const props = defineProps<{ pinos: PinoMapa[] }>()
const emit = defineEmits<{ fechar: []; escolher: [numero: number] }>()
const unico = computed(() => (props.pinos.length === 1 ? props.pinos[0] : null))
</script>

<template>
  <aside v-if="pinos.length" class="cartao-prop" role="dialog" aria-live="polite" :aria-label="unico ? `Propriedade nº ${unico.numero}` : `${pinos.length} propriedades`">
    <button class="fechar discreto" title="Fechar (Esc)" aria-label="Fechar" @click="emit('fechar')">×</button>
    <template v-if="unico">
      <span class="num">{{ unico.numero }}</span>
      <div class="pilha-mini">
        <span class="rot">Propriedade nº {{ unico.numero }}</span>
        <strong class="nome">{{ unico.nome }}</strong>
        <span>Responsável: {{ unico.responsavel || '—' }}</span>
      </div>
    </template>
    <template v-else>
      <div class="pilha-mini largo">
        <strong>{{ pinos.length }} propriedades neste ponto</strong>
        <ul>
          <li v-for="p in pinos" :key="p.numero">
            <button class="item" @click="emit('escolher', p.numero)">
              <span class="num pequeno">{{ p.numero }}</span>
              <span><strong>{{ p.nome }}</strong><br><span class="suave">Responsável: {{ p.responsavel || '—' }}</span></span>
            </button>
          </li>
        </ul>
      </div>
    </template>
  </aside>
</template>

<style scoped>
.cartao-prop {
  position: absolute; top: 12px; right: 12px; z-index: 900; max-width: min(340px, calc(100% - 24px));
  display: flex; gap: .75rem; align-items: flex-start; padding: .8rem 2.2rem .8rem .85rem;
  background: var(--folha); color: var(--tinta); border: 1px solid var(--regua); border-radius: var(--raio-grande); box-shadow: 0 6px 22px rgb(0 0 0 / 28%);
}
.fechar { position: absolute; top: .3rem; right: .35rem; min-height: 28px; width: 28px; padding: 0; font-size: 1.25rem; line-height: 1; }
.num { width: 34px; height: 34px; flex: none; border-radius: 50%; background: #f2b705; border: 2px solid #101412; color: #101412; display: grid; place-items: center; font-weight: 700; }
.num.pequeno { width: 24px; height: 24px; font-size: .75rem; }
.pilha-mini { display: flex; flex-direction: column; gap: .15rem; min-width: 0; }
.pilha-mini.largo { width: 100%; }
.rot { font-size: .8rem; color: var(--tinta-2); text-transform: uppercase; letter-spacing: .04em; }
.nome { font-size: 1.05rem; overflow-wrap: anywhere; }
ul { list-style: none; margin: .4rem 0 0; padding: 0; display: flex; flex-direction: column; gap: .25rem; max-height: 240px; overflow: auto; }
.item { display: flex; gap: .55rem; align-items: center; width: 100%; text-align: left; border: 1px solid var(--regua); background: var(--folha-2); min-height: 0; padding: .4rem .5rem; }
.item:hover { border-color: var(--mata); }
</style>
