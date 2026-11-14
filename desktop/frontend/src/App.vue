<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { SincronizacaoService as Sinc } from '../bindings/sitposse7/desktop/app'
import type { PedidoResumo } from '../bindings/sitposse7/desktop/internal/pareamento/models'
import Painel from './telas/Painel.vue'
import Propriedades from './telas/Propriedades.vue'
import Equipes from './telas/Equipes.vue'
import Sincronizacao from './telas/Sincronizacao.vue'
import { mensagem } from './formato'

type Tela = 'painel' | 'propriedades' | 'equipes' | 'sincronizacao'
const menu: { id: Tela; rotulo: string }[] = [
  { id: 'painel', rotulo: 'Painel' },
  { id: 'propriedades', rotulo: 'Propriedades' },
  { id: 'equipes', rotulo: 'Equipes' },
  { id: 'sincronizacao', rotulo: 'Sincronização' },
]
const tela = ref<Tela>('painel')
const versao = ref('')

// Pedidos de pareamento: aparecem em qualquer tela (o aparelho espera até 90 s).
const pedidos = ref<PedidoResumo[]>([])
const erroPedido = ref('')
let timer: number | undefined
async function verificarPedidos() {
  try { pedidos.value = (await Sinc.Pendentes()) ?? [] } catch { /* a Central continua utilizável */ }
}
async function decidir(id: string, aceitar: boolean) {
  erroPedido.value = ''
  try { await Sinc.Decidir(id, aceitar); await verificarPedidos() } catch (e) { erroPedido.value = mensagem(e) }
}
onMounted(async () => {
  verificarPedidos()
  timer = window.setInterval(verificarPedidos, 1500)
  try { versao.value = (await Sinc.Info()).versao } catch { /* */ }
})
onUnmounted(() => window.clearInterval(timer))
</script>

<template>
  <div class="app">
    <nav class="lateral">
      <div class="marca">
        <img src="/appicon.png" alt="" width="36" height="36" />
        <div>
          <strong>SITPOSSE-7</strong>
          <span class="suave">Central · 7º BPM/PMRO</span>
        </div>
      </div>
      <button v-for="m in menu" :key="m.id" :class="{ ativo: tela === m.id }" @click="tela = m.id">{{ m.rotulo }}</button>
      <span class="versao suave">versão {{ versao }}</span>
    </nav>
    <main>
      <Painel v-if="tela === 'painel'" @ir="(t: Tela) => (tela = t)" />
      <Propriedades v-else-if="tela === 'propriedades'" />
      <Equipes v-else-if="tela === 'equipes'" />
      <Sincronizacao v-else />
    </main>

    <div v-if="pedidos.length" class="modal-fundo">
      <div class="modal pilha">
        <h2>Aparelho pedindo pareamento</h2>
        <div v-for="p in pedidos" :key="p.id" class="pilha">
          <dl class="dados">
            <dt>Equipe</dt><dd><strong>{{ p.equipe_nome }}</strong></dd>
            <dt>Responsável</dt><dd>{{ p.responsavel }}</dd>
            <dt v-if="p.identificacao_area">Área</dt><dd v-if="p.identificacao_area">{{ p.identificacao_area }}</dd>
            <dt>Aparelho</dt><dd>{{ p.rotulo_aparelho || '—' }}</dd>
          </dl>
          <p class="suave">{{ p.equipe_conhecida ? 'Equipe já conhecida: um novo aparelho dela será liberado.' : 'Equipe nova: será incluída na operação ativa.' }}</p>
          <div class="linha" style="justify-content: flex-end">
            <button class="perigo" @click="decidir(p.id, false)">Recusar</button>
            <button class="primario" @click="decidir(p.id, true)">Confirmar pareamento</button>
          </div>
        </div>
        <p v-if="erroPedido" class="erro">{{ erroPedido }}</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.app { display: grid; grid-template-columns: 230px 1fr; min-height: 100vh; }
.lateral { background: var(--superficie); border-right: 1px solid var(--borda); padding: 1rem .75rem; display: flex; flex-direction: column; gap: .35rem; position: sticky; top: 0; height: 100vh; }
.marca { display: flex; gap: .65rem; align-items: center; padding: .25rem .4rem 1rem; }
.marca div { display: flex; flex-direction: column; line-height: 1.2; }
.marca span { font-size: .8rem; }
.lateral button { text-align: left; border: none; background: transparent; padding: .7rem .9rem; font-weight: 500; }
.lateral button.ativo { background: var(--primaria-clara); color: var(--primaria); font-weight: 700; }
.versao { margin-top: auto; font-size: .78rem; padding: 0 .9rem; }
main { padding: 1.5rem 2rem; max-width: 1500px; width: 100%; }
@media (max-width: 1000px) {
  .app { grid-template-columns: 1fr; }
  .lateral { position: static; height: auto; flex-direction: row; flex-wrap: wrap; align-items: center; border-right: none; border-bottom: 1px solid var(--borda); padding: .5rem; }
  .marca { padding: 0 .5rem; }
  .marca span, .versao { display: none; }
  main { padding: 1rem; }
}
</style>
