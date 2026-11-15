<script setup lang="ts">
/**
 * Lista de propriedades/famílias com os mesmos filtros do Painel (§22-A.2 "Propriedades":
 * nome, responsável, jornada, equipe, com/sem coordenadas, faixas de tamanho,
 * presença ou ausência de bens ou animais). Clique abre a página da propriedade.
 */
import { computed, onActivated, onMounted, ref, watch } from 'vue'
import { ConsultaService as Consulta } from '../../bindings/sitposse7/desktop/app'
import AcaoIcone from '../componentes/AcaoIcone.vue'
import EditarPropriedade from '../componentes/EditarPropriedade.vue'
import ConfirmarExclusao from '../componentes/ConfirmarExclusao.vue'
import BarraFiltros from '../componentes/BarraFiltros.vue'
import TabelaDados from '../componentes/TabelaDados.vue'
import ExportarRelatorio from '../componentes/ExportarRelatorio.vue'
import { chips, type Filtro, filtroVazio } from '../filtro'
import { type Familia, type Resultado, v } from '../painel'
import { haNumero, mensagem, num, plural, quando } from '../formato'
import { estudoAtual, estudoFoco } from '../estudo'
import type { Coluna } from '../tabela'

const emit = defineEmits<{ abrir: [uuid: string] }>()
const filtro = ref<Filtro>(filtroVazio(estudoFoco.value))
watch(estudoFoco, (e) => { if (filtro.value.estudo !== e) filtro.value = filtroVazio(e) })
const r = ref<Resultado | null>(null)
const erro = ref('')
const carregando = ref(true)

async function carregar() {
  try { r.value = await Consulta.Estatisticas(filtro.value); erro.value = '' } catch (e) { erro.value = mensagem(e) } finally { carregando.value = false }
}
let atraso: number | undefined
watch(filtro, () => { window.clearTimeout(atraso); atraso = window.setTimeout(carregar, 120) }, { deep: true })
onMounted(carregar)
onActivated(carregar)

const familias = computed(() => r.value?.familias ?? [])
const exportando = ref(false)
const editando=ref(''), excluindo=ref<any>(null)
async function excluir(id:string){try{excluindo.value=JSON.parse(await Consulta.DocumentoEdicao(id))}catch(e){erro.value=mensagem(e)}}
async function confirmarExclusao(){await Consulta.ExcluirPropriedade(excluindo.value.propriedade.uuid,excluindo.value.token)}
async function atualizou(){editando.value='';excluindo.value=null;await carregar();window.dispatchEvent(new Event('cadastros-atualizados'))}

const criterios = computed(() => chips(filtro.value, { equipes: r.value?.equipes ?? [], jornadas: r.value?.jornadas ?? [], campos: r.value?.campos ?? [] }).map((c) => c.texto))
const colunas: Coluna<Familia>[] = [
  { chave: 'nome', rotulo: 'Propriedade / família', largura: '18%' },
  { chave: 'equipe', rotulo: 'Equipe' },
  { chave: 'jornada', rotulo: 'Jornada', secundaria: true },
  { chave: 'pessoas', rotulo: 'Pessoas', tipo: 'numero', valor: (l) => v(l, 'pessoas_total'), formatar: (l) => num(v(l, 'pessoas_total')) },
  { chave: 'bovinos', rotulo: 'Bovinos', tipo: 'numero', valor: (l) => v(l, 'bov_total'), formatar: (l) => num(v(l, 'bov_total')) },
  { chave: 'veiculos', rotulo: 'Veículos', tipo: 'numero', valor: (l) => v(l, 'veic_total'), formatar: (l) => num(v(l, 'veic_total')), secundaria: true },
  { chave: 'area', rotulo: 'Área (ha)', tipo: 'numero', valor: (l) => haNumero(l.area_ha), formatar: (l) => l.area_ha ?? 'Não informada' },
  { chave: 'loc', rotulo: 'Localização', valor: (l) => (l.com_localizacao ? 1 : 0) },
  { chave: 'fotos', rotulo: 'Fotos', tipo: 'numero', valor: (l) => v(l, 'fotos'), secundaria: true },
  { chave: 'recebido_em', rotulo: 'Recebido em', formatar: (l) => quando(l.recebido_em), secundaria: true },
  { chave: 'acao', rotulo: 'Ação', ordenavel: false, largura: '120px' },
]
const totais = computed(() => {
  const t = r.value?.totais
  return t ? { nome: `Total (${plural(t.familias, 'família', 'famílias')})`, pessoas: num(v(t, 'pessoas_total')), bovinos: num(v(t, 'bov_total')), veiculos: num(v(t, 'veic_total')), area: t.area_ha, fotos: num(v(t, 'fotos')) } : null
})
</script>

<template>
  <div class="pilha">
    <EditarPropriedade v-if="editando" :uuid="editando" @fechar="editando=''" @salvo="atualizou" />
    <ConfirmarExclusao v-if="excluindo" :nome="excluindo.propriedade.nome_propriedade" :resumo="`Serão excluídos o cadastro, ${excluindo.itens.length} itens e ${excluindo.fotos.length} fotos.`" :executar="confirmarExclusao" @fechar="excluindo=null" @concluido="atualizou" />
    <header class="cabeca">
      <div class="titulo">
        <h1>Propriedades</h1>
        <p class="suave"><template v-if="estudoAtual">{{ estudoAtual.nome }} · {{ estudoAtual.codigo_exibicao }}. </template>Cada propriedade corresponde a uma família. Clique numa linha para ver o cadastro completo.</p>
      </div>
      <button :disabled="!r?.familias_base" title="PDF, Word ou Excel, com os filtros ativos" @click="exportando = true"><svg class="ic" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 4v11m0 0-4.5-4.5M12 15l4.5-4.5M5 19h14" /></svg>Exportar</button>
    </header>
    <ExportarRelatorio v-if="exportando" modo="geral" :filtro="filtro" :criterios="criterios" :familias="r?.totais.familias ?? 0" @fechar="exportando = false" />
    <BarraFiltros v-model="filtro" :equipes="r?.equipes ?? []" :jornadas="r?.jornadas ?? []" :campos="r?.campos ?? []"
                  :mostrando="r?.totais.familias ?? null" :base="r?.familias_base ?? null" />
    <TabelaDados class="tabela-propriedades" :colunas="colunas" :linhas="familias" :chave="(l) => l.uuid" rotulo="Propriedades recebidas" clicavel :totais="totais"
                 :carregando="carregando" :erro="erro" :ordem="{ chave: 'nome', dir: 'asc' }" @abrir="(l) => emit('abrir', l.uuid)">
      <template #acao="{linha}"><div class="acoes-tabela"><AcaoIcone acao="Editar propriedade" icone="editar" @click="editando=linha.uuid"/><AcaoIcone acao="Excluir propriedade" icone="excluir" @click="excluir(linha.uuid)"/></div></template>
      <template #nome="{ linha }">
        <strong class="nome">{{ linha.nome }}</strong>
        <span class="suave resp">{{ linha.responsavel || 'Responsável não informado' }}</span>
      </template>
      <template #equipe="{ linha }">{{ linha.equipe }}<span class="suave resp">{{ linha.jornada }}</span></template>
      <template #loc="{ linha }">
        <span class="selo" :class="linha.com_localizacao ? 'ok' : ''">{{ linha.com_localizacao ? 'Com localização' : 'Sem localização' }}</span>
      </template>
      <template #vazio>
        {{ r?.familias_base ? 'Nenhuma propriedade com esses filtros.' : 'Nenhuma propriedade recebida ainda. Receba os cadastros em Sincronização.' }}
      </template>
    </TabelaDados>
  </div>
</template>

<style scoped>
.cabeca { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.titulo { display: flex; flex-direction: column; gap: .35rem; }
.ic { width: 18px; height: 18px; fill: none; stroke: currentColor; stroke-width: 2; stroke-linecap: round; stroke-linejoin: round; vertical-align: -3px; margin-right: .35rem; }

.tabela-propriedades :deep(th), .tabela-propriedades :deep(td) { padding-left: .45rem; padding-right: .45rem; }
.nome { display: block; }
.resp { font-size: .88rem; }
</style>
