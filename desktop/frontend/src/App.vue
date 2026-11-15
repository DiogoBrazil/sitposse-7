<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref } from 'vue'
import { OperacaoService as Op, SincronizacaoService as Sinc } from '../bindings/sitposse7/desktop/app'
import type { Operacao, PedidoResumo } from '../bindings/sitposse7/desktop/internal/pareamento/models'
import Painel from './telas/Painel.vue'
import Propriedades from './telas/Propriedades.vue'
import Propriedade from './telas/Propriedade.vue'
import Equipes from './telas/Equipes.vue'
import Sincronizacao from './telas/Sincronizacao.vue'
import Cartas from './telas/Cartas.vue'
import InfoTecnica from './componentes/InfoTecnica.vue'
import { mensagem } from './formato'
import type { Tela } from './navegacao'

const menu: { id: Tela; rotulo: string; icone: string }[] = [
  { id: 'painel', rotulo: 'Painel', icone: 'M3 13h7V3H3zm0 8h7v-6H3zm11 0h7V11h-7zm0-18v6h7V3z' },
  { id: 'propriedades', rotulo: 'Propriedades', icone: 'M3 11 12 4l9 7v9a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1z' },
  { id: 'cartas', rotulo: 'Cartas-imagem', icone: 'M9 4 3 6.5v13.5l6-2.5 6 2.5 6-2.5V4l-6 2.5zM9 4v13.5M15 6.5V20' },
  { id: 'equipes', rotulo: 'Equipes', icone: 'M8 11a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7zm8 0a3 3 0 1 0 0-6M2 20c0-3.3 2.7-6 6-6s6 2.7 6 6M16 14c3 0 6 2.2 6 6' },
  { id: 'sincronizacao', rotulo: 'Sincronização', icone: 'M4 12a8 8 0 0 1 14-5.3M20 12a8 8 0 0 1-14 5.3M18 3v4h-4M6 21v-4h4' },
]
const tela = ref<Tela>('painel')
/** Propriedade aberta e a tela de onde veio (para o "Voltar"). */
const aberta = ref<{ uuid: string; de: Tela } | null>(null)
const versao = ref('')
const operacao = ref<Operacao | null>(null)
const infoAberta = ref(false)
/** Propriedade escolhida na página dela para a carta individual. */
const cartaUuid = ref<string | null>(null)
function cartaDe(uuid: string) { cartaUuid.value = uuid; ir('cartas') }

function ir(t: Tela) { tela.value = t; aberta.value = null; window.scrollTo(0, 0) }
let rolagem = 0
function abrirPropriedade(uuid: string) { rolagem = window.scrollY; aberta.value = { uuid, de: tela.value }; window.scrollTo(0, 0) }
function voltar() { aberta.value = null; nextTick(() => window.scrollTo(0, rolagem)) }

// Pedidos de pareamento: aparecem em qualquer tela (o aparelho espera até 90 s).
const pedidos = ref<PedidoResumo[]>([])
const erroPedido = ref('')
let timer: number | undefined
async function verificar() {
  try { pedidos.value = (await Sinc.Pendentes()) ?? [] } catch { /* a Central continua utilizável */ }
}
async function decidir(id: string, aceitar: boolean) {
  erroPedido.value = ''
  try { await Sinc.Decidir(id, aceitar); await verificar() } catch (e) { erroPedido.value = mensagem(e) }
}
async function carregarOperacao() { try { operacao.value = await Op.Ativa() } catch { /* */ } }
onMounted(async () => {
  verificar()
  carregarOperacao()
  timer = window.setInterval(() => { verificar(); carregarOperacao() }, 1500)
  try { versao.value = (await Sinc.Info()).versao } catch { /* */ }
})
onUnmounted(() => window.clearInterval(timer))
</script>

<template>
  <div class="app">
    <nav class="lateral" aria-label="Menu principal">
      <div class="marca">
        <img src="/appicon.png" alt="" width="38" height="38" />
        <div>
          <strong>SITPOSSE-7</strong>
          <span>Central · 7º BPM/PMRO</span>
        </div>
      </div>
      <div class="itens">
        <button v-for="m in menu" :key="m.id" :class="{ ativo: tela === m.id }" :aria-current="tela === m.id ? 'page' : undefined" @click="ir(m.id)">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="m.icone" /></svg>
          <span>{{ m.rotulo }}</span>
        </button>
      </div>
      <div class="operacao">
        <span class="op-rotulo">Operação ativa</span>
        <strong>{{ operacao?.nome ?? 'Nenhuma' }}</strong>
      </div>
      <div class="rodape">
        <button class="info" @click="infoAberta = true">
          <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9" /><path d="M12 11v6M12 7.5v.5" /></svg>
          Informações técnicas
        </button>
        <span class="versao">Versão {{ versao }}</span>
      </div>
    </nav>

    <main>
      <Propriedade v-if="aberta" :key="aberta.uuid" :uuid="aberta.uuid" :de="aberta.de" @voltar="voltar" @carta="cartaDe" />
      <!-- As telas ficam montadas (só escondidas) enquanto a propriedade está aberta: ao voltar,
           filtros, aba e detalhamento continuam como estavam. -->
      <div v-show="!aberta">
        <KeepAlive>
          <Painel v-if="tela === 'painel'" @ir="ir" @abrir="abrirPropriedade" />
          <Propriedades v-else-if="tela === 'propriedades'" @abrir="abrirPropriedade" />
          <Equipes v-else-if="tela === 'equipes'" />
          <Cartas v-else-if="tela === 'cartas'" :uuid-inicial="cartaUuid" />
          <Sincronizacao v-else @info="infoAberta = true" />
        </KeepAlive>
      </div>
    </main>

    <InfoTecnica v-if="infoAberta" @fechar="infoAberta = false" />

    <div v-if="pedidos.length" class="modal-fundo">
      <div class="modal pilha" role="alertdialog" aria-modal="true" aria-labelledby="pedido-titulo">
        <h2 id="pedido-titulo">Celular pedindo pareamento</h2>
        <div v-for="p in pedidos" :key="p.id" class="pilha pedido">
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
        <p v-if="erroPedido" class="aviso-erro">{{ erroPedido }}</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.app { display: grid; grid-template-columns: 236px minmax(0, 1fr); min-height: 100vh; }
.lateral {
  background: var(--lateral); color: var(--lateral-tinta); padding: 1.1rem .8rem .9rem;
  display: flex; flex-direction: column; gap: 1rem; position: sticky; top: 0; height: 100vh;
}
.marca { display: flex; gap: .7rem; align-items: center; padding: 0 .35rem .25rem; }
.marca img { border-radius: 9px; }
.marca div { display: flex; flex-direction: column; line-height: 1.2; }
.marca strong { font-size: 1.05rem; font-stretch: 112%; letter-spacing: .01em; }
.marca span { font-size: .8rem; color: var(--lateral-tinta-2); }
.itens { display: flex; flex-direction: column; gap: .2rem; }
.itens button {
  display: flex; align-items: center; gap: .7rem; text-align: left; border: none; background: transparent;
  color: var(--lateral-tinta-2); padding: .62rem .8rem; font-weight: 550; border-radius: var(--raio); position: relative;
}
.itens button:hover { color: var(--lateral-tinta); background: var(--lateral-ativo); }
.itens button.ativo { color: var(--lateral-tinta); background: var(--lateral-ativo); font-weight: 680; }
.itens button.ativo::before { content: ""; position: absolute; left: -.8rem; top: .45rem; bottom: .45rem; width: 4px; border-radius: 0 3px 3px 0; background: var(--ipe); }
.itens svg, .info svg { width: 19px; height: 19px; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; flex: none; }
.lateral :focus-visible { outline-color: var(--ipe); }
.operacao { margin-top: auto; padding: .75rem .8rem; border-top: 1px solid color-mix(in srgb, var(--lateral-tinta) 14%, transparent); display: flex; flex-direction: column; gap: .15rem; }
.op-rotulo { font-size: .78rem; color: var(--lateral-tinta-2); }
.operacao strong { font-size: .95rem; overflow-wrap: anywhere; }
.rodape { display: flex; flex-direction: column; gap: .35rem; }
.info { display: flex; align-items: center; gap: .55rem; border: none; background: transparent; color: var(--lateral-tinta-2); padding: .45rem .8rem; font-size: .88rem; font-weight: 500; min-height: 0; }
.info:hover { color: var(--lateral-tinta); background: var(--lateral-ativo); }
.versao { font-size: .78rem; color: var(--lateral-tinta-2); padding: 0 .8rem; opacity: .8; }
main { padding: 1.75rem 2.25rem 3rem; max-width: 1560px; width: 100%; }
.pedido + .pedido { border-top: 1px solid var(--regua); padding-top: 1rem; }

@media (max-width: 1000px) {
  .app { grid-template-columns: 1fr; }
  .lateral { position: sticky; z-index: 10; height: auto; flex-direction: row; flex-wrap: wrap; align-items: center; gap: .5rem; padding: .5rem .75rem; }
  .marca span, .operacao, .versao { display: none; }
  .itens { flex-direction: row; flex-wrap: wrap; }
  .itens button.ativo::before { left: .6rem; right: .6rem; top: auto; bottom: 0; width: auto; height: 3px; border-radius: 3px 3px 0 0; }
  .rodape { margin-left: auto; }
  .info { font-size: 0; gap: 0; padding: .5rem; }
  main { padding: 1.1rem 1rem 2rem; }
}
</style>
