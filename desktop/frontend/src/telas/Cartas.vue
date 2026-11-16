<script setup lang="ts">
/**
 * Cartas-imagem (A30, ADR-0027): a melhor imagem gratuita é escolhida automaticamente (INPE:
 * CBERS-4A de 2 m, Sentinel-2 de 10 m; Copernicus como reserva), com enquadramento pelas
 * propriedades e, quando a área é extensa, folhas de setor. Fluxo: escolher as propriedades
 * (individual ou pelos filtros do Painel) → a imagem é preparada sozinha (ou reaberta, se já
 * guardada) → explorar o mapa (pinos com nome e responsável) → gerar a carta (PDF + PNG).
 * Referências (RN-98): ao voltar à tela e antes de gerar, as propriedades são relidas; se
 * coordenadas mudaram (edição ou sincronização), a carta só sai depois de "Atualizar referências".
 */
import { computed, onActivated, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { CartaService as Cartas, ConsultaService as Consulta } from '../../bindings/sitposse7/desktop/app'
import type { Andamento, EstadoConexao, Plano, ResultadoPreparo } from '../../bindings/sitposse7/desktop/app/models'
import type { CartaGuardada } from '../../bindings/sitposse7/desktop/internal/cartografia/models'
import type { PropriedadeLinha } from '../../bindings/sitposse7/desktop/internal/consultas/models'
import BarraFiltros from '../componentes/BarraFiltros.vue'
import CartaoPropriedade from '../componentes/CartaoPropriedade.vue'
import ConfigCopernicus from '../componentes/ConfigCopernicus.vue'
import MapaCarta from '../componentes/MapaCarta.vue'
import TabelaDados from '../componentes/TabelaDados.vue'
import { avisoMudanca, compararPlanos, ehSemConexao, exigePreparo, megabytes, primeiroPNG, quandoPreparada, type PinoMapa } from '../cartas'
import { chips, type Filtro, filtroVazio } from '../filtro'
import { mensagem, num, plural, quando } from '../formato'
import type { Resultado } from '../painel'
import type { Coluna } from '../tabela'
import { estudoFoco } from '../estudo'

const props = defineProps<{ uuidInicial?: string | null }>()

const aba = ref<'nova' | 'salvas'>('nova')
const estado = ref<EstadoConexao | null>(null)
const configAberta = ref(false)
const erro = ref('')

async function carregarEstado() { try { estado.value = await Cartas.Estado() } catch { /* */ } }

// ---------- seleção ----------
const tipo = ref<'individual' | 'geral'>('geral')
const propriedades = ref<PropriedadeLinha[]>([])
const uuid = ref('')
const filtro = ref<Filtro>(filtroVazio(estudoFoco.value))
async function carregarPropriedades() { try { propriedades.value = (await Consulta.Propriedades('', estudoFoco.value, '')) ?? [] } catch { /* */ } }
watch(estudoFoco, (e) => { if (filtro.value.estudo !== e) filtro.value = filtroVazio(e); carregarPropriedades() })
const resultado = ref<Resultado | null>(null)
const plano = ref<Plano | null>(null)
const criterios = computed(() => tipo.value === 'geral' && resultado.value
  ? chips(filtro.value, { equipes: resultado.value.equipes ?? [], jornadas: resultado.value.jornadas ?? [], campos: resultado.value.campos ?? [] }).map((c) => c.texto)
  : [])

// Só replaneja (e prepara a imagem de novo) quando a seleção muda de verdade: o filtro é
// observado em profundidade e pode ser reemitido igual pela barra de filtros.
let ultimaChave = ''
const chaveSelecao = () => JSON.stringify([tipo.value, uuid.value, tipo.value === 'geral' ? filtro.value : null])
async function planejar() {
  const chave = chaveSelecao()
  if (chave === ultimaChave && plano.value) return
  ultimaChave = chave
  erro.value = ''; gerada.value = null; selecionados.value = []
  try {
    let novo: Plano | null
    if (tipo.value === 'individual') {
      if (!uuid.value) { plano.value = null; return }
      novo = await Cartas.PlanejarIndividual(uuid.value)
    } else {
      resultado.value = await Consulta.Estatisticas(filtro.value)
      novo = await Cartas.PlanejarGeral(filtro.value)
    }
    if (chave !== ultimaChave) return // outra seleção chegou enquanto esta era calculada
    const mesmas = !!novo && !!plano.value && (novo.uuids ?? []).join() === (plano.value.uuids ?? []).join()
    plano.value = novo
    if (!novo) { cancelar(); preparo.value = null; return }
    if (!mesmas) { preparo.value = null; preparar(false) }
  } catch (e) { erro.value = mensagem(e); ultimaChave = '' }
}
let atraso: number | undefined
watch([tipo, uuid], planejar)

// Relê as propriedades da mesma seleção (RN-98): coordenadas, nomes ou cadastros podem ter
// mudado (edição na Central ou sincronização). Os pinos passam a mostrar os dados atuais; se a
// imagem foi preparada para outras coordenadas, o aviso pede "Atualizar referências".
let revisando: Promise<void> | null = null
function revisar(): Promise<void> {
  if (!plano.value || chaveSelecao() !== ultimaChave) return Promise.resolve()
  if (revisando) return revisando
  const chave = ultimaChave
  revisando = (async () => {
    try {
      let novo: Plano | null
      if (tipo.value === 'individual') novo = await Cartas.PlanejarIndividual(uuid.value)
      else { resultado.value = await Consulta.Estatisticas(filtro.value); novo = await Cartas.PlanejarGeral(filtro.value) }
      if (chave === ultimaChave && novo) plano.value = novo
    } catch (e) { if (chave === ultimaChave) erro.value = mensagem(e) } finally { revisando = null }
  })()
  return revisando
}
watch(filtro, () => { window.clearTimeout(atraso); atraso = window.setTimeout(planejar, 400) }, { deep: true })

const pinos = computed<PinoMapa[]>(() => (plano.value?.marcadores ?? []).map((m) => ({
  numero: m.numero, uuid: m.uuid, nome: m.nome, responsavel: m.responsavel, lat: m.ponto.lat, lon: m.ponto.lon,
})))
const enquadramento = computed(() => {
  const g = preparo.value?.geral
  if (g) return { so: g.so, ne: g.ne }
  return plano.value ? { so: plano.value.so, ne: plano.value.ne } : null
})

// ---------- preparo da imagem (tarefa em segundo plano) ----------
const preparo = ref<ResultadoPreparo | null>(null)
// Pinos (com coordenadas) para os quais a imagem atual foi pedida: base da comparação (RN-98).
const pinosPreparo = ref<PinoMapa[]>([])
const mudanca = computed(() => preparo.value ? compararPlanos(pinosPreparo.value, pinos.value) : null)
const desatualizada = computed(() => !!mudanca.value && exigePreparo(mudanca.value))
async function atualizarReferencias() {
  erro.value = ''
  await revisar()
  if (!erro.value) preparar(false)
}
const andamento = ref<Andamento | null>(null)
let tarefa = ''
let relogio: number | undefined

function cancelar() {
  window.clearInterval(relogio)
  if (tarefa) { Cartas.Cancelar(tarefa).catch(() => {}); tarefa = '' }
  andamento.value = null
}

async function preparar(atualizar: boolean) {
  if (!plano.value) return
  cancelar()
  erro.value = ''; gerada.value = null
  const base = pinos.value.map((p) => ({ ...p }))
  try {
    const id = await Cartas.Preparar({ tipo: tipo.value, uuids: plano.value.uuids, atualizar })
    tarefa = id
    andamento.value = { fase: 'Iniciando', feito: 0, total: 0, bytes_rede: 0, concluida: false, erro: '', sem_conexao: false, resultado: null }
    relogio = window.setInterval(async () => {
      if (tarefa !== id) return
      try {
        const a = await Cartas.Andamento(id)
        if (tarefa !== id || !a) return
        andamento.value = a
        if (!a.concluida) return
        window.clearInterval(relogio); tarefa = ''; andamento.value = null
        if (a.erro) { if (a.erro !== 'Operação cancelada.') erro.value = a.erro; return }
        preparo.value = a.resultado
        pinosPreparo.value = base
        carregarEstado()
      } catch (e) { window.clearInterval(relogio); tarefa = ''; andamento.value = null; erro.value = mensagem(e) }
    }, 400)
  } catch (e) { erro.value = mensagem(e) }
}
onBeforeUnmount(cancelar)

const camada = computed(() => preparo.value ? {
  tiles: preparo.value.tiles, zMin: preparo.value.z_min, zGeral: preparo.value.z_geral, zNativo: preparo.value.z_nativo,
  so: preparo.value.geral.so, ne: preparo.value.geral.ne,
} : null)
const folhas = computed(() => preparo.value ? [{ rotulo: '', so: preparo.value.geral.so, ne: preparo.value.geral.ne }, ...(preparo.value.setores ?? [])] : [])
const partesUsadas = computed(() => (preparo.value?.partes ?? []).filter((p) => p.fracao > 0))
const estimada = computed(() => partesUsadas.value.some((p) => p.nuvem_estimada))
const atribuicao = computed(() => [...new Set(partesUsadas.value.map((p) => p.atribuicao))].join(' · '))
const pct = (f: number) => `${Math.max(1, Math.round(100 * f))}%`
const mt = (v: number) => `${String(v).replace('.', ',')} m`

// ---------- pinos e cartão ----------
const selecionados = ref<number[]>([])
const pinosSelecionados = computed(() => selecionados.value.map((n) => pinos.value.find((p) => p.numero === n)).filter((p): p is PinoMapa => !!p))
const mapaRef = ref<InstanceType<typeof MapaCarta> | null>(null)
function escolher(n: number) {
  selecionados.value = [n]
  const p = pinos.value.find((x) => x.numero === n)
  if (p) mapaRef.value?.centralizar(p.lat, p.lon)
}

// ---------- gerar ----------
const gerada = ref<CartaGuardada | null>(null)
const gerando = ref(false)
async function gerar() {
  if (!plano.value || !preparo.value) return
  erro.value = ''; gerando.value = true
  try {
    // Uma sincronização pode ter chegado com a tela aberta: confere antes de gerar (RN-98).
    await revisar()
    if (desatualizada.value || erro.value || !plano.value || !preparo.value) return
    gerada.value = await Cartas.GerarCarta({ tipo: tipo.value, uuids: plano.value.uuids, composicao_id: preparo.value.composicao_id,
      criterios: criterios.value, enquadramento_anterior: !!preparo.value.anterior })
    await Promise.all([carregarSalvas(), carregarEstado()])
  } catch (e) { erro.value = mensagem(e); revisar() } finally { gerando.value = false }
}
async function abrir(c: CartaGuardada, arq: string) { try { await Cartas.AbrirCarta(c.id, arq) } catch (e) { erro.value = mensagem(e) } }
async function mostrar(c: CartaGuardada) { try { await Cartas.MostrarCarta(c.id, 'carta.pdf') } catch (e) { erro.value = mensagem(e) } }
async function copia(c: CartaGuardada, arq: string) { try { await Cartas.SalvarCopia(c.id, arq) } catch (e) { erro.value = mensagem(e) } }

// ---------- cartas salvas ----------
const salvas = ref<CartaGuardada[]>([])
async function carregarSalvas() { try { salvas.value = (await Cartas.Cartas()) ?? [] } catch (e) { erro.value = mensagem(e) } }
const colSalvas: Coluna<CartaGuardada>[] = [
  { chave: 'miniatura', rotulo: 'Carta', ordenavel: false, largura: '120px' },
  { chave: 'gerada_em', rotulo: 'Gerada em', valor: (l) => l.gerada_em, formatar: (l) => quando(new Date(l.gerada_em).getTime()) },
  { chave: 'titulo', rotulo: 'Título', valor: (l) => l.titulo.replace(/^Carta-imagem de localização das propriedades — /, '') },
  { chave: 'propriedades', rotulo: 'Propriedades', tipo: 'numero' },
  { chave: 'periodo', rotulo: 'Imagem', secundaria: true },
  { chave: 'acoes', rotulo: 'Arquivos', ordenavel: false },
]
const folhasDe = (c: CartaGuardada) => (c.arquivos ?? []).filter((a) => a.endsWith('.png')).length

onMounted(async () => {
  await carregarEstado()
  await carregarPropriedades()
  if (props.uuidInicial) { tipo.value = 'individual'; uuid.value = props.uuidInicial } else planejar()
  carregarSalvas()
})
onActivated(() => { carregarEstado(); carregarSalvas(); revisar() })
watch(() => props.uuidInicial, (u) => { if (u) { aba.value = 'nova'; tipo.value = 'individual'; uuid.value = u } })

const comLocalizacao = computed(() => propriedades.value.filter((p) => p.tem_gps))
const semLocalizacaoLista = computed(() => propriedades.value.filter((p) => !p.tem_gps))
const offline = computed(() => ehSemConexao(erro.value))
const nFolhas = computed(() => 1 + (preparo.value?.setores?.length ?? 0))
</script>

<template>
  <div class="pilha">
    <header class="cabeca">
      <div class="titulo">
        <h1>Cartas-imagem</h1>
        <p class="suave">A melhor imagem de satélite gratuita é escolhida automaticamente para a área das propriedades. Precisa de internet só para buscar imagens novas.</p>
      </div>
      <button class="discreto" @click="configAberta = true">Fonte de reserva (opcional)</button>
    </header>

    <div class="abas" role="tablist">
      <button role="tab" :aria-selected="aba === 'nova'" :class="{ ativa: aba === 'nova' }" @click="aba = 'nova'">Nova carta</button>
      <button role="tab" :aria-selected="aba === 'salvas'" :class="{ ativa: aba === 'salvas' }" @click="aba = 'salvas'; carregarSalvas()">Cartas salvas <span class="cont">{{ salvas.length }}</span></button>
    </div>

    <template v-if="aba === 'nova'">
      <section class="selecao pilha">
        <div class="linha">
          <div class="segmentado" role="radiogroup" aria-label="Tipo de carta">
            <button type="button" role="radio" :aria-checked="tipo === 'geral'" @click="tipo = 'geral'">Geral (filtros)</button>
            <button type="button" role="radio" :aria-checked="tipo === 'individual'" @click="tipo = 'individual'">Individual</button>
          </div>
          <select v-if="tipo === 'individual'" v-model="uuid" aria-label="Propriedade" class="sel-prop">
            <option value="" disabled>Escolha a propriedade…</option>
            <option v-for="p in comLocalizacao" :key="p.uuid" :value="p.uuid">{{ p.numero ? `${p.numero} · ` : '' }}{{ p.nome }} · {{ p.equipe }}</option>
            <optgroup v-if="semLocalizacaoLista.length" label="Sem localização (não podem ser representadas)">
              <option v-for="p in semLocalizacaoLista" :key="p.uuid" :value="p.uuid" disabled>{{ p.nome }}</option>
            </optgroup>
          </select>
        </div>
        <BarraFiltros v-if="tipo === 'geral'" v-model="filtro" :equipes="resultado?.equipes ?? []" :jornadas="resultado?.jornadas ?? []"
                      :campos="resultado?.campos ?? []" :mostrando="resultado?.totais.familias ?? null" :base="resultado?.familias_base ?? null" />
      </section>

      <div v-if="erro" class="aviso-erro erro-bloco" role="alert">
        <p>{{ erro }}</p>
        <div v-if="offline" class="linha"><button @click="preparar(false)">Tentar novamente</button><button class="discreto" @click="erro = ''">Voltar</button></div>
      </div>

      <div v-if="plano && preparo && !andamento && (desatualizada || preparo.anterior)" class="aviso referencias" role="status">
        <p v-if="desatualizada">
          <strong>{{ avisoMudanca(mudanca!) }}</strong>
          A imagem e as folhas foram preparadas para as localizações anteriores; os pinos já mostram as atuais. Atualize as referências para gerar a carta.
        </p>
        <p v-else>
          <strong>Sem internet: imagem preparada antes, com o enquadramento anterior.</strong>
          Todos os pinos continuam dentro das folhas e a carta pode ser gerada. Atualize as referências quando houver internet.
        </p>
        <button class="primario" :disabled="gerando" @click="atualizarReferencias">Atualizar referências</button>
      </div>

      <div v-if="plano" class="area">
        <aside class="painel pilha">
          <section class="pilha bloco-p">
            <h2>Imagem</h2>
            <template v-if="preparo">
              <p v-if="preparo.mosaico" class="selo alerta largo-selo">Mosaico de imagens de datas diferentes</p>
              <ul class="partes">
                <li v-for="p in partesUsadas" :key="p.rotulo + p.periodo">
                  <strong>{{ p.rotulo }}</strong>
                  <span>{{ p.periodo }} · {{ mt(p.resolucao_m) }}<template v-if="partesUsadas.length > 1"> · {{ pct(p.fracao) }} da área</template></span>
                </li>
              </ul>
              <p v-if="estimada" class="fraco">Nuvens do CBERS-4A estimadas pelo sistema (a fonte não traz máscara): áreas com nuvem estimada usam outra imagem.</p>
              <p v-if="preparo.guardada" class="fraco">Imagem guardada neste computador, preparada em {{ quandoPreparada(preparo.criada_em) }}.</p>
              <p v-if="preparo.reserva" class="aviso">O INPE não respondeu: foi usada a fonte de reserva (Copernicus, 10 m).</p>
              <ul v-if="preparo.avisos?.length" class="avisos"><li v-for="a in preparo.avisos" :key="a">{{ a }}</li></ul>
            </template>
            <p v-else-if="!andamento" class="fraco">Sem imagem ainda.</p>
            <button :disabled="!!andamento || gerando" @click="preparar(true)">Atualizar melhor imagem</button>
          </section>

          <section class="pilha bloco-p">
            <h2>Carta</h2>
            <p v-if="preparo" class="fraco">
              {{ nFolhas === 1 ? 'Uma folha.' : `${nFolhas} folhas: a geral e ${plural(nFolhas - 1, 'setor', 'setores')} (${(preparo.setores ?? []).map((s) => s.rotulo).join(', ')}), com a mesma numeração.` }}
            </p>
            <button class="primario" :disabled="!preparo || !!andamento || gerando || !pinos.length || desatualizada" @click="gerar">{{ gerando ? 'Gerando a carta…' : 'Gerar carta (PDF e PNG)' }}</button>
            <div v-if="gerada" class="msg-ok pilha">
              <strong>Carta gerada e guardada ({{ plural(folhasDe(gerada), 'folha', 'folhas') }}).</strong>
              <div class="linha">
                <button @click="abrir(gerada!, 'carta.pdf')">Abrir PDF</button>
                <button @click="abrir(gerada!, primeiroPNG(gerada!.arquivos))">Abrir PNG</button>
                <button class="discreto" @click="mostrar(gerada!)">Mostrar na pasta</button>
              </div>
            </div>
          </section>

          <section class="pilha bloco-p legenda">
            <h2>Legenda <span class="fraco">({{ pinos.length }})</span></h2>
            <ol>
              <li v-for="m in pinos" :key="m.numero">
                <button class="item-legenda" :class="{ sel: selecionados.includes(m.numero) }" @click="escolher(m.numero)">
                  <span class="num">{{ m.numero }}</span><span class="nome-leg">{{ m.nome }}</span>
                </button>
              </li>
            </ol>
            <details v-if="plano.suspeitas?.length" open>
              <summary class="alerta-txt">{{ plural(plano.suspeitas.length, 'coordenada suspeita', 'coordenadas suspeitas') }}</summary>
              <p class="fraco">Fora do Brasil ou a mais de 50 km das demais. Confira no cadastro; elas continuam na carta.</p>
              <ul><li v-for="n in plano.suspeitas" :key="n">{{ n }}</li></ul>
            </details>
            <details v-if="plano.sem_localizacao?.length">
              <summary>{{ plural(plano.sem_localizacao.length, 'propriedade sem localização', 'propriedades sem localização') }}</summary>
              <p class="fraco">Não podem ser representadas até que tenham coordenada coletada no celular.</p>
              <ul><li v-for="n in plano.sem_localizacao" :key="n">{{ n }}</li></ul>
            </details>
          </section>
        </aside>

        <section class="pilha mapa-col">
          <div class="mapa-envolta">
            <MapaCarta ref="mapaRef" :pinos="pinos" :camada="camada" :folhas="folhas" :enquadrar="enquadramento" :selecionados="selecionados"
                       :atribuicao="atribuicao" @selecionar="(ns) => (selecionados = ns)">
              <CartaoPropriedade :pinos="pinosSelecionados" @fechar="selecionados = []" @escolher="escolher" />
              <button class="ajustar" title="Voltar ao enquadramento das propriedades" @click="mapaRef?.ajustar()">Ajustar às propriedades</button>
            </MapaCarta>
            <div v-if="andamento" class="carregando" role="status">
              <span class="giro" aria-hidden="true"></span>
              <strong>{{ andamento.fase }}…</strong>
              <span v-if="andamento.total > 0">{{ num(andamento.feito) }} de {{ num(andamento.total) }}</span>
              <span v-if="andamento.bytes_rede > 0" class="mb">{{ megabytes(andamento.bytes_rede) }} baixados</span>
              <button class="discreto claro" @click="cancelar">Cancelar</button>
            </div>
          </div>
          <p class="fraco nota">
            O pino é a coordenada coletada em campo, com a precisão do GPS do celular. Os contornos tracejados são as folhas da carta
            (geral e setores). Aproximar a imagem não aumenta a resolução{{ preparo ? ` (${mt(preparo.resolucao_m)} na melhor parte)` : '' }}.
          </p>
        </section>
      </div>
    </template>

    <template v-else>
      <TabelaDados :colunas="colSalvas" :linhas="salvas" :chave="(l) => l.id" rotulo="Cartas salvas" :ordem="{ chave: 'gerada_em', dir: 'desc' }"
                   texto-vazio="Nenhuma carta gerada ainda.">
        <template #miniatura="{ linha }">
          <img class="mini" :src="`/cartografia/cartas/${linha.id}/${primeiroPNG(linha.arquivos)}`" alt="" loading="lazy" @click="abrir(linha, primeiroPNG(linha.arquivos))" />
        </template>
        <template #acoes="{ linha }">
          <div class="acoes">
            <button @click="abrir(linha, 'carta.pdf')">PDF</button>
            <button @click="abrir(linha, primeiroPNG(linha.arquivos))">PNG</button>
            <button class="discreto" title="Salvar uma cópia em outra pasta" @click="copia(linha, 'carta.pdf')">Salvar cópia</button>
            <button class="discreto" @click="mostrar(linha)">Pasta</button>
          </div>
        </template>
      </TabelaDados>
      <p class="fraco">As cartas ficam guardadas na pasta de dados da Central e abrem sem internet. Cartas com setores têm um PNG por folha na pasta.</p>
    </template>

    <ConfigCopernicus v-if="configAberta" @fechar="configAberta = false" @mudou="carregarEstado" />
  </div>
</template>

<style scoped>
.cabeca { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.titulo { display: flex; flex-direction: column; gap: .35rem; }
.abas { display: flex; gap: .25rem; border-bottom: 1px solid var(--regua-forte); }
.abas button { border: none; border-radius: var(--raio) var(--raio) 0 0; background: transparent; color: var(--tinta-2); font-weight: 600; padding: .6rem 1rem; position: relative; }
.abas button.ativa { color: var(--tinta); }
.abas button.ativa::after { content: ""; position: absolute; left: .6rem; right: .6rem; bottom: -1px; height: 3px; border-radius: 3px 3px 0 0; background: var(--mata); }
.cont { font-stretch: 85%; font-weight: 700; margin-left: .2rem; }
.segmentado { display: inline-flex; border: 1px solid var(--regua-forte); border-radius: var(--raio); overflow: hidden; }
.segmentado button { border: none; border-radius: 0; min-height: 38px; padding: .3rem .9rem; background: var(--folha); color: var(--tinta-2); font-weight: 550; }
.segmentado button + button { border-left: 1px solid var(--regua-forte); }
.segmentado button[aria-checked="true"] { background: var(--mata); color: var(--sobre-mata); }
.sel-prop { flex: 1 1 320px; width: auto; }
.erro-bloco { display: flex; flex-direction: column; gap: .6rem; }
.referencias { display: flex; align-items: center; justify-content: space-between; gap: 1rem; flex-wrap: wrap; }
.referencias p { margin: 0; flex: 1 1 420px; }
.referencias strong { display: block; margin-bottom: .15rem; }
.area { display: grid; grid-template-columns: minmax(280px, 340px) minmax(0, 1fr); gap: 1.25rem; align-items: start; }
.painel { position: sticky; top: 1rem; }
.bloco-p { background: var(--folha); border: 1px solid var(--regua); border-radius: var(--raio-grande); padding: .9rem 1rem; gap: .6rem; }
.bloco-p h2 { font-size: 1rem; }
.partes { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: .45rem; }
.partes li { display: flex; flex-direction: column; gap: .1rem; font-size: .92rem; }
.partes li span { color: var(--tinta-2); }
.largo-selo { align-self: flex-start; }
.avisos { margin: 0; padding-left: 1.1rem; font-size: .85rem; color: var(--tinta-2); }
.legenda ol { margin: 0; padding: 0; list-style: none; display: flex; flex-direction: column; gap: .15rem; max-height: 300px; overflow: auto; }
.item-legenda { display: flex; align-items: center; gap: .5rem; width: 100%; text-align: left; border: 1px solid transparent; background: transparent; min-height: 0; padding: .25rem .35rem; font-size: .9rem; font-weight: 400; }
.item-legenda:hover, .item-legenda.sel { border-color: var(--regua-forte); background: var(--folha-2); }
.nome-leg { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.legenda .num { width: 22px; height: 22px; flex: none; border-radius: 50%; background: #f2b705; border: 2px solid #101412; color: #101412; display: grid; place-items: center; font-size: .75rem; font-weight: 700; }
.legenda details { font-size: .9rem; }
.legenda summary { cursor: pointer; color: var(--alerta); font-weight: 600; }
.mapa-col { gap: .6rem; }
.mapa-envolta { position: relative; }
.ajustar { position: absolute; left: 12px; bottom: 36px; z-index: 800; min-height: 32px; padding: .2rem .7rem; font-size: .85rem; box-shadow: 0 1px 4px rgb(0 0 0 / 30%); }
.carregando { position: absolute; inset: 0; display: grid; place-content: center; justify-items: center; gap: .45rem; background: rgb(10 20 14 / 62%); color: #fff; font-weight: 600; border-radius: var(--raio-grande); z-index: 1000; text-align: center; padding: 1rem; }
.carregando .mb { font-weight: 400; opacity: .85; font-size: .9rem; }
.carregando .claro { color: #fff; border-color: rgb(255 255 255 / 60%); margin-top: .3rem; }
.giro { width: 34px; height: 34px; border-radius: 50%; border: 3px solid rgb(255 255 255 / 30%); border-top-color: #fff; animation: giro 0.9s linear infinite; }
@keyframes giro { to { transform: rotate(360deg); } }
.nota { font-size: .85rem; }
.msg-ok { background: var(--ok-fundo); color: var(--ok); border-radius: var(--raio); padding: .7rem .85rem; gap: .5rem; }
.mini { width: 100px; aspect-ratio: 4 / 3.4; object-fit: cover; border-radius: 4px; border: 1px solid var(--regua); cursor: zoom-in; background: var(--folha-2); }
.acoes { display: flex; gap: .35rem; justify-content: center; flex-wrap: wrap; }
.acoes button { min-height: 32px; padding: .25rem .6rem; }
.alerta-txt { color: var(--alerta); }
@media (max-width: 1100px) { .area { grid-template-columns: 1fr; } .painel { position: static; } }
</style>
