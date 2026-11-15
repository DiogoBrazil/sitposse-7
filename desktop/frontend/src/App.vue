<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref } from 'vue'
import { SincronizacaoService as Sinc } from '../bindings/sitposse7/desktop/app'
import type { PedidoResumo } from '../bindings/sitposse7/desktop/internal/pareamento/models'
import type { LoteAguardando } from '../bindings/sitposse7/desktop/internal/importacao/models'
import Painel from './telas/Painel.vue'
import Propriedades from './telas/Propriedades.vue'
import Propriedade from './telas/Propriedade.vue'
import Estudos from './telas/Estudos.vue'
import Sincronizacao from './telas/Sincronizacao.vue'
import Cartas from './telas/Cartas.vue'
import InfoTecnica from './componentes/InfoTecnica.vue'
import ModalDestino from './componentes/ModalDestino.vue'
import { mensagem } from './formato'
import { carregarEstudos, definirFoco, estudoFoco, estudos } from './estudo'
import type { Tela } from './navegacao'

const menu: { id: Tela; rotulo: string; icone: string }[] = [
  { id: 'painel', rotulo: 'Painel', icone: 'M3 13h7V3H3zm0 8h7v-6H3zm11 0h7V11h-7zm0-18v6h7V3z' },
  { id: 'propriedades', rotulo: 'Propriedades', icone: 'M3 11 12 4l9 7v9a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1z' },
  { id: 'cartas', rotulo: 'Cartas-imagem', icone: 'M9 4 3 6.5v13.5l6-2.5 6 2.5 6-2.5V4l-6 2.5zM9 4v13.5M15 6.5V20' },
  { id: 'estudos', rotulo: 'Estudos', icone: 'M8 11a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7zm8 0a3 3 0 1 0 0-6M2 20c0-3.3 2.7-6 6-6s6 2.7 6 6M16 14c3 0 6 2.2 6 6' },
  { id: 'sincronizacao', rotulo: 'Sincronização', icone: 'M4 12a8 8 0 0 1 14-5.3M20 12a8 8 0 0 1-14 5.3M18 3v4h-4M6 21v-4h4' },
]
const tela = ref<Tela>('painel')
/** Propriedade aberta e a tela de onde veio (para o "Voltar"). */
const aberta = ref<{ uuid: string; de: Tela } | null>(null)
const versao = ref('')
const bloqueio = ref('')
const infoAberta = ref(false)
/** Propriedade escolhida na página dela para a carta individual. */
const cartaUuid = ref<string | null>(null)
function cartaDe(uuid: string) { cartaUuid.value = uuid; ir('cartas') }

function ir(t: Tela) { tela.value = t; aberta.value = null; window.scrollTo(0, 0) }
let rolagem = 0
function abrirPropriedade(uuid: string) { rolagem = window.scrollY; aberta.value = { uuid, de: tela.value }; window.scrollTo(0, 0) }
function voltar() { aberta.value = null; nextTick(() => window.scrollTo(0, rolagem)) }

// Pedidos de pareamento e lotes esperando o destino: aparecem em qualquer tela.
const pedidos = ref<PedidoResumo[]>([])
const aguardando = ref<LoteAguardando[]>([])
const erroPedido = ref('')
let timer: number | undefined
async function verificar() {
  try { pedidos.value = (await Sinc.Pendentes()) ?? [] } catch { /* a Central continua utilizável */ }
  try { aguardando.value = (await Sinc.Aguardando()) ?? [] } catch { /* */ }
}
async function aposDecidir() { await verificar(); await carregarEstudos() }
async function decidir(id: string, aceitar: boolean) {
  erroPedido.value = ''
  try { await Sinc.Decidir(id, aceitar); await verificar() } catch (e) { erroPedido.value = mensagem(e) }
}
onMounted(async () => {
  try { const i = await Sinc.Info(); versao.value = i.versao; bloqueio.value = i.bloqueio } catch { /* */ }
  if (bloqueio.value) return
  try { await carregarEstudos() } catch { /* */ }
  verificar()
  timer = window.setInterval(verificar, 1500)
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
        <label class="op-rotulo" for="estudo-foco">Estudo em foco</label>
        <select v-if="estudos.length" id="estudo-foco" :value="estudoFoco" @change="definirFoco(($event.target as HTMLSelectElement).value)">
          <option v-for="e in estudos" :key="e.uuid" :value="e.uuid">{{ e.nome }}</option>
        </select>
        <button v-else class="discreto novo-estudo" @click="ir('estudos')">Criar estudo</button>
      </div>
      <div class="rodape">
        <button class="info" @click="infoAberta = true">
          <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9" /><path d="M12 11v6M12 7.5v.5" /></svg>
          Informações técnicas
        </button>
        <span class="versao">Versão {{ versao }}</span>
      </div>
    </nav>

    <main v-if="bloqueio">
      <section class="bloco">
        <div class="bloco-cabeca"><h2>Dados de teste antigos</h2></div>
        <div class="bloco-corpo pilha">
          <p class="aviso-erro">{{ bloqueio }}</p>
          <p>Feche a Central e siga o procedimento de reinício dos dados de teste (docs/operacao/teste-de-campo-mvp.md): os arquivos antigos são movidos para uma pasta de cópia, sem apagar nada.</p>
        </div>
      </section>
    </main>
    <main v-else>
      <Propriedade v-if="aberta" :key="aberta.uuid" :uuid="aberta.uuid" :de="aberta.de" @voltar="voltar" @carta="cartaDe" />
      <!-- As telas ficam montadas (só escondidas) enquanto a propriedade está aberta: ao voltar,
           filtros, aba e detalhamento continuam como estavam. -->
      <div v-show="!aberta">
        <KeepAlive>
          <Painel v-if="tela === 'painel'" @ir="ir" @abrir="abrirPropriedade" />
          <Propriedades v-else-if="tela === 'propriedades'" @abrir="abrirPropriedade" />
          <Estudos v-else-if="tela === 'estudos'" />
          <Cartas v-else-if="tela === 'cartas'" :uuid-inicial="cartaUuid" />
          <Sincronizacao v-else @info="infoAberta = true" />
        </KeepAlive>
      </div>
    </main>

    <InfoTecnica v-if="infoAberta" @fechar="infoAberta = false" />

    <ModalDestino v-if="aguardando.length && !pedidos.length" :lotes="aguardando" @decidido="aposDecidir" />

    <div v-if="pedidos.length" class="modal-fundo">
      <div class="modal pilha" role="alertdialog" aria-modal="true" aria-labelledby="pedido-titulo">
        <h2 id="pedido-titulo">Celular pedindo pareamento</h2>
        <div v-for="p in pedidos" :key="p.id" class="pilha pedido">
          <dl class="dados">
            <dt>Aparelho</dt><dd><strong>{{ p.rotulo_aparelho || '—' }}</strong></dd>
            <dt>Instalação</dt><dd class="mono">{{ p.instalacao_uuid.slice(0, 8) }}</dd>
          </dl>
          <p class="suave">{{ p.aparelho_conhecido ? 'Aparelho já pareado antes: a credencial dele será renovada.' : 'Aparelho novo.' }} O estudo e a equipe de cada envio são confirmados depois, pelo código.</p>
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
.operacao select { width: 100%; background: var(--lateral-ativo); color: var(--lateral-tinta); border-color: color-mix(in srgb, var(--lateral-tinta) 25%, transparent); font-weight: 600; }
.novo-estudo { color: var(--lateral-tinta); justify-content: flex-start; padding-left: 0; }
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
