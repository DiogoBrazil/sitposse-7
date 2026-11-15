<script setup lang="ts">
/**
 * Faixa de totais no estilo de caderneta de campo: grupos separados por réguas verticais,
 * números grandes condensados. É o elemento de destaque da Central (Painel e propriedade).
 */
import type { GrupoTotal } from '../graficos'
defineProps<{ grupos: GrupoTotal[] }>()
</script>

<template>
  <section class="faixa" aria-label="Totais">
    <div v-for="g in grupos" :key="g.titulo" class="grupo" :style="g.cor ? { '--marca': `var(${g.cor})` } : undefined">
      <h2 class="titulo">{{ g.titulo }}</h2>
      <div class="itens">
        <div v-for="i in g.itens" :key="i.rotulo" class="item">
          <span class="valor">{{ i.valor }}</span>
          <span class="rotulo">{{ i.rotulo }}</span>
          <span v-if="i.detalhe" class="detalhe">{{ i.detalhe }}</span>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.faixa {
  display: flex; flex-wrap: wrap;
  background: var(--folha); border: 1px solid var(--regua); border-radius: var(--raio-grande); overflow: hidden;
}
.grupo { flex: 1 0 auto; padding: .85rem clamp(.8rem, 1.1vw, 1.1rem) 1rem; border-right: 1px solid var(--regua); border-bottom: 1px solid var(--regua); margin: 0 -1px -1px 0; position: relative; }
.grupo::before { content: ""; position: absolute; left: clamp(.8rem, 1.1vw, 1.1rem); top: 0; width: 28px; height: 3px; background: var(--marca, var(--regua-forte)); border-radius: 0 0 2px 2px; }
.titulo { font-size: .88rem; font-weight: 650; font-stretch: 100%; color: var(--tinta-2); margin-bottom: .45rem; }
.itens { display: flex; gap: clamp(.9rem, 1.3vw, 1.4rem); }
.item { display: flex; flex-direction: column; min-width: 0; }
.valor { font-size: clamp(1.9rem, 2.5vw, 2.35rem); line-height: 1.05; font-weight: 720; font-stretch: 78%; letter-spacing: -0.01em; color: var(--tinta); }
.rotulo { font-size: .9rem; color: var(--tinta-2); }
.detalhe { font-size: .8rem; color: var(--tinta-3); }
</style>
