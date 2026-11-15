<script setup lang="ts">
/**
 * Exportação de relatórios (A26, RF-C07): relatório geral com os filtros ativos
 * (panorama + famílias + fichas) ou a ficha de uma propriedade, em PDF, Word ou Excel.
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RelatorioService as Relatorio } from '../../bindings/sitposse7/desktop/app'
import type { Filtro } from '../filtro'
import { mensagem, plural } from '../formato'

const props = defineProps<{
  modo: 'geral' | 'ficha'
  filtro?: Filtro
  criterios?: string[]
  familias?: number
  uuid?: string
  nome?: string
}>()
const emit = defineEmits<{ fechar: [] }>()

type Formato = 'pdf' | 'docx' | 'xlsx'
const opcoes = computed(() => [
  { id: 'pdf' as Formato, rotulo: 'PDF', descr: 'Para imprimir ou enviar. Não pode ser editado.' },
  { id: 'docx' as Formato, rotulo: 'Word (.docx)', descr: 'Documento editável, com o mesmo conteúdo do PDF.' },
  ...(props.modo === 'geral' ? [{ id: 'xlsx' as Formato, rotulo: 'Planilha Excel (.xlsx)', descr: 'Uma aba por visão, com filtros, totais e gráficos.' }] : []),
])
const formato = ref<Formato>('pdf')
const fichas = ref(true)
const estado = ref<'escolhendo' | 'gerando' | 'pronto'>('escolhendo')
const caminho = ref('')
const erro = ref('')

async function salvar() {
  erro.value = ''
  estado.value = 'gerando'
  try {
    const c = props.modo === 'ficha'
      ? await Relatorio.ExportarFicha(formato.value, props.uuid!)
      : await Relatorio.Exportar(formato.value, props.filtro!, props.criterios ?? [], fichas.value && formato.value !== 'xlsx')
    if (!c) { estado.value = 'escolhendo'; return } // cancelado no "Salvar como"
    caminho.value = c
    estado.value = 'pronto'
  } catch (e) { erro.value = mensagem(e); estado.value = 'escolhendo' }
}
async function acao(f: (c: string) => Promise<void>) {
  try { await f(caminho.value) } catch (e) { erro.value = mensagem(e) }
}
const tecla = (e: KeyboardEvent) => { if (e.key === 'Escape' && estado.value !== 'gerando') emit('fechar') }
onMounted(() => window.addEventListener('keydown', tecla))
onUnmounted(() => window.removeEventListener('keydown', tecla))
</script>

<template>
  <div class="modal-fundo" @click.self="estado !== 'gerando' && emit('fechar')">
    <div class="modal pilha" role="dialog" aria-modal="true" aria-labelledby="exp-titulo">
      <div>
        <h2 id="exp-titulo">{{ modo === 'ficha' ? 'Exportar ficha da propriedade' : 'Exportar relatório' }}</h2>
        <p class="suave sub">
          <template v-if="modo === 'ficha'">{{ nome }}</template>
          <template v-else>Panorama geral, totais por equipe e por área, itens, tamanhos e {{ plural(familias ?? 0, 'família', 'famílias') }}.</template>
        </p>
      </div>

      <template v-if="estado !== 'pronto'">
        <fieldset class="formatos" :disabled="estado === 'gerando'">
          <legend class="sr">Formato</legend>
          <label v-for="o in opcoes" :key="o.id" class="formato" :class="{ sel: formato === o.id }">
            <input v-model="formato" type="radio" name="formato" :value="o.id" />
            <span><strong>{{ o.rotulo }}</strong><span class="suave">{{ o.descr }}</span></span>
          </label>
        </fieldset>

        <label v-if="modo === 'geral' && formato !== 'xlsx'" class="opcao">
          <input v-model="fichas" type="checkbox" :disabled="estado === 'gerando'" />
          <span>Incluir a ficha de cada família <span class="suave">(uma por página)</span></span>
        </label>

        <div v-if="modo === 'geral'" class="criterios">
          <span class="suave">Filtros aplicados:</span>
          <ul v-if="criterios?.length"><li v-for="c in criterios" :key="c">{{ c }}</li></ul>
          <span v-else>nenhum (todas as famílias recebidas)</span>
        </div>

        <p v-if="erro" class="aviso-erro">{{ erro }}</p>
        <div class="acoes">
          <button :disabled="estado === 'gerando'" @click="emit('fechar')">Cancelar</button>
          <button class="primario" :disabled="estado === 'gerando'" @click="salvar">{{ estado === 'gerando' ? 'Gerando…' : 'Salvar…' }}</button>
        </div>
      </template>

      <template v-else>
        <p class="ok-msg"><strong>Arquivo salvo.</strong><br /><span class="mono">{{ caminho }}</span></p>
        <p v-if="erro" class="aviso-erro">{{ erro }}</p>
        <div class="acoes">
          <button @click="acao(Relatorio.MostrarNaPasta)">Mostrar na pasta</button>
          <button @click="acao(Relatorio.Abrir)">Abrir arquivo</button>
          <button class="primario" autofocus @click="emit('fechar')">Concluir</button>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.sub { margin-top: .3rem; }
.formatos { border: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: .5rem; }
.formato { display: flex; gap: .75rem; align-items: flex-start; border: 1px solid var(--regua-forte); border-radius: var(--raio); padding: .7rem .85rem; cursor: pointer; }
.formato:hover { background: var(--folha-2); }
.formato.sel { border-color: var(--mata); box-shadow: inset 0 0 0 1px var(--mata); background: color-mix(in srgb, var(--mata-clara) 45%, transparent); }
.formato input { width: 18px; height: 18px; min-height: 0; margin-top: .1rem; accent-color: var(--mata); }
.formato > span { display: flex; flex-direction: column; gap: .1rem; }
.formato .suave { font-size: .88rem; }
.opcao { display: flex; gap: .6rem; align-items: center; cursor: pointer; }
.opcao input { width: 18px; height: 18px; min-height: 0; accent-color: var(--mata); }
.criterios { display: flex; flex-wrap: wrap; gap: .35rem .5rem; align-items: center; font-size: .92rem; }
.criterios ul { display: contents; list-style: none; }
.criterios li { background: var(--mata-clara); border-radius: 999px; padding: .1rem .65rem; font-size: .85rem; font-weight: 550; }
.acoes { display: flex; justify-content: flex-end; gap: .6rem; flex-wrap: wrap; }
.ok-msg { background: var(--ok-fundo); color: var(--ok); border-radius: var(--raio); padding: .8rem 1rem; }
.ok-msg .mono { color: var(--tinta); }
.sr { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
</style>
