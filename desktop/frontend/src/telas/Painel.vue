<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { ConsultaService as Consulta, OperacaoService as Op } from '../../bindings/sitposse7/desktop/app'
import type { Resumo } from '../../bindings/sitposse7/desktop/internal/consultas/models'
import type { Operacao } from '../../bindings/sitposse7/desktop/internal/pareamento/models'
import { mensagem, num, quando } from '../formato'

const emit = defineEmits<{ ir: [tela: 'propriedades' | 'equipes' | 'sincronizacao'] }>()
const r = ref<Resumo | null>(null)
const op = ref<Operacao | null>(null)
const erro = ref('')
let timer: number | undefined

async function carregar() {
  try {
    ;[r.value, op.value] = await Promise.all([Consulta.Resumo(), Op.Ativa()])
    erro.value = ''
  } catch (e) { erro.value = mensagem(e) }
}
onMounted(() => { carregar(); timer = window.setInterval(carregar, 5000) })
onUnmounted(() => window.clearInterval(timer))
</script>

<template>
  <div class="pilha">
    <div class="linha" style="justify-content: space-between">
      <div>
        <h1>Painel</h1>
        <p class="suave" style="margin: .25rem 0 0">{{ op ? `Operação ativa: ${op.nome}` : 'Nenhuma operação ativa' }}</p>
      </div>
      <button class="primario" @click="emit('ir', 'sincronizacao')">{{ op ? 'Receber dos celulares' : 'Criar operação' }}</button>
    </div>
    <p v-if="erro" class="aviso-erro">{{ erro }}</p>
    <template v-if="r">
      <div class="grade-cartoes">
        <button class="cartao numero" @click="emit('ir', 'propriedades')"><span>{{ num(r.propriedades) }}</span>Propriedades</button>
        <div class="cartao numero"><span>{{ num(r.pessoas) }}</span>Pessoas</div>
        <div class="cartao numero"><span>{{ num(r.veiculos) }}</span>Veículos</div>
        <div class="cartao numero"><span>{{ num(r.bovinos) }}</span>Bovinos</div>
        <div class="cartao numero"><span>{{ num(r.equinos) }}</span>Equinos</div>
        <div class="cartao numero"><span>{{ num(r.aves) }}</span>Aves</div>
        <div class="cartao numero"><span>{{ num(r.eletrodomesticos) }}</span>Eletrodomésticos</div>
        <div class="cartao numero"><span>{{ num(r.moveis) }}</span>Móveis</div>
        <div class="cartao numero"><span>{{ r.area_ha }} ha</span>Área informada<small class="suave">{{ r.sem_area }} sem área informada</small></div>
        <div class="cartao numero"><span>{{ num(r.com_gps) }}</span>Com localização<small class="suave">{{ r.propriedades - r.com_gps }} sem</small></div>
        <div class="cartao numero"><span>{{ num(r.fotos) }}</span>Fotos</div>
        <button class="cartao numero" @click="emit('ir', 'equipes')"><span>{{ num(r.equipes) }}</span>Equipes<small class="suave">{{ r.aparelhos }} aparelho(s)</small></button>
      </div>
      <div class="cartao">
        <h2>Recepção</h2>
        <p style="margin: .5rem 0 0">{{ num(r.lotes_importados) }} lote(s) importado(s) · última importação: {{ quando(r.ultima_importacao) }}</p>
      </div>
      <p class="suave">Os números consideram só os cadastros vigentes (sem os excluídos) e são calculados neste computador, sem internet.</p>
    </template>
  </div>
</template>

<style scoped>
.numero { display: flex; flex-direction: column; gap: .2rem; text-align: left; font-weight: 500; color: var(--texto-suave); min-height: 96px; }
.numero span { font-size: 1.8rem; font-weight: 700; color: var(--texto); font-variant-numeric: tabular-nums; }
.numero small { font-size: .78rem; font-weight: 400; }
button.numero:hover { border-color: var(--primaria); }
</style>
