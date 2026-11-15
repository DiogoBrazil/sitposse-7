<script setup lang="ts">
/**
 * Cartas-imagem (A27, ADR-0022): imagem Sentinel-2 (Copernicus) com os pinos das propriedades.
 * Fluxo: escolher as propriedades (individual ou pelos filtros do Painel) → buscar as datas
 * de imagem disponíveis → prévia (arrastar e aproximar ajusta o enquadramento) → gerar a carta
 * (PDF + PNG, guardada na pasta de dados e reaberta sem internet).
 */
import { computed, onActivated, onMounted, ref, watch } from 'vue'
import { CartaService as Cartas, ConsultaService as Consulta } from '../../bindings/sitposse7/desktop/app'
import type { EstadoConexao, ImagemPrevia, PassagensResposta, Plano } from '../../bindings/sitposse7/desktop/app/models'
import type { CartaGuardada } from '../../bindings/sitposse7/desktop/internal/cartografia/models'
import type { PropriedadeLinha } from '../../bindings/sitposse7/desktop/internal/consultas/models'
import BarraFiltros from '../componentes/BarraFiltros.vue'
import ConfigCopernicus from '../componentes/ConfigCopernicus.vue'
import MapaCarta from '../componentes/MapaCarta.vue'
import TabelaDados from '../componentes/TabelaDados.vue'
import { diaUTC, ehSemConexao, type Retangulo3857 } from '../cartas'
import { chips, dataParaMs, type Filtro, filtroVazio, msParaData } from '../filtro'
import { mensagem, num, plural, quando } from '../formato'
import type { Resultado } from '../painel'
import type { Coluna } from '../tabela'

const props = defineProps<{ uuidInicial?: string | null }>()

const aba = ref<'nova' | 'salvas'>('nova')
const estado = ref<EstadoConexao | null>(null)
const configAberta = ref(false)
const erro = ref('')
const ocupado = ref('')

async function carregarEstado() { try { estado.value = await Cartas.Estado() } catch { /* */ } }

// ---------- seleção ----------
const tipo = ref<'individual' | 'geral'>('geral')
const propriedades = ref<PropriedadeLinha[]>([])
const uuid = ref('')
const filtro = ref<Filtro>(filtroVazio())
const resultado = ref<Resultado | null>(null)
const plano = ref<Plano | null>(null)
const grupo = ref(-1) // -1 = todas numa carta
const criterios = computed(() => tipo.value === 'geral' && resultado.value
  ? chips(filtro.value, { equipes: resultado.value.equipes ?? [], areas: resultado.value.areas ?? [], operacoes: resultado.value.operacoes ?? [], campos: resultado.value.campos ?? [] }).map((c) => c.texto)
  : [])

async function planejar() {
  erro.value = ''; plano.value = null; previa.value = null; passagens.value = null; gerada.value = null; grupo.value = -1
  try {
    if (tipo.value === 'individual') {
      if (!uuid.value) return
      plano.value = await Cartas.PlanejarIndividual(uuid.value)
    } else {
      resultado.value = await Consulta.Estatisticas(filtro.value)
      plano.value = await Cartas.PlanejarGeral(filtro.value)
    }
    enquadrar.value = plano.value ? { so: plano.value.so, ne: plano.value.ne } : null
  } catch (e) { erro.value = mensagem(e) }
}
let atraso: number | undefined
watch([tipo, uuid], planejar)
watch(filtro, () => { window.clearTimeout(atraso); atraso = window.setTimeout(planejar, 250) }, { deep: true })

const marcadores = computed(() => {
  const ms = plano.value?.marcadores ?? []
  const g = plano.value?.grupos?.[grupo.value]
  return g ? g.map((i) => ms[i]) : ms
})
const uuidsSelecionados = computed(() => {
  const us = plano.value?.uuids ?? []
  const g = plano.value?.grupos?.[grupo.value]
  return g ? g.map((i) => us[i]) : us
})
const pinos = computed(() => marcadores.value.map((m) => ({ numero: m.numero, nome: m.nome, lat: m.ponto.lat, lon: m.ponto.lon })))
watch(grupo, async () => {
  previa.value = null; passagens.value = null; gerada.value = null
  if (grupo.value < 0 || !plano.value) { if (plano.value) enquadrar.value = { so: plano.value.so, ne: plano.value.ne }; return }
  try {
    const r = await Cartas.EnquadrarGrupo(marcadores.value.map((m) => m.ponto))
    retangulo.value = r
    const sw = mercatorParaGraus(r.min_x, r.min_y), ne = mercatorParaGraus(r.max_x, r.max_y)
    enquadrar.value = { so: sw, ne }
  } catch (e) { erro.value = mensagem(e) }
})

// ---------- mapa ----------
const enquadrar = ref<{ so: { lat: number; lon: number }; ne: { lat: number; lon: number } } | null>(null)
const retangulo = ref<Retangulo3857 | null>(null)
const foraDaImagem = ref(true)
function aoMudarVista(r: Retangulo3857, fora: boolean) { retangulo.value = r; foraDaImagem.value = fora; gerada.value = null }
const R = 6378137
function mercatorParaGraus(x: number, y: number) {
  return { lat: (2 * Math.atan(Math.exp(y / R)) - Math.PI / 2) * 180 / Math.PI, lon: x / R * 180 / Math.PI }
}

// ---------- imagens ----------
const hoje = new Date()
const de = ref(msParaData(hoje.getTime() - 90 * 86_400_000))
const ate = ref(msParaData(hoje.getTime()))
const passagens = ref<PassagensResposta | null>(null)
const escolha = ref<string>('') // ISO da passagem ou "mosaico"
const previa = ref<ImagemPrevia | null>(null)

function pedidoAtual() {
  const r = retangulo.value ?? plano.value?.retangulo
  if (!r) throw new Error('Escolha as propriedades primeiro.')
  if (escolha.value === 'mosaico') {
    return { retangulo: r, data: null, de: new Date(dataParaMs(de.value, false) ?? 0).toISOString(), ate: new Date(dataParaMs(ate.value, true) ?? Date.now()).toISOString() }
  }
  if (!escolha.value) throw new Error('Escolha uma data de imagem.')
  return { retangulo: r, data: escolha.value, de: null, ate: null }
}

async function buscarPassagens() {
  erro.value = ''; ocupado.value = 'Procurando imagens disponíveis…'
  try {
    const r = retangulo.value ?? plano.value?.retangulo
    if (!r) return
    const ini = dataParaMs(de.value, false), fim = dataParaMs(ate.value, true)
    passagens.value = await Cartas.Passagens(r, ini ? new Date(ini).toISOString() : null, fim ? new Date(fim).toISOString() : null)
    const nova = passagens.value?.sugerida ?? (passagens.value?.passagens?.length ? '' : 'mosaico')
    if (nova === escolha.value) { if (nova) await carregarPrevia() } else escolha.value = nova // o watch carrega a prévia
  } catch (e) { erro.value = mensagem(e) } finally { ocupado.value = '' }
}

async function carregarPrevia() {
  erro.value = ''; ocupado.value = 'Obtendo a imagem de satélite…'; gerada.value = null
  try { previa.value = await Cartas.Imagem(pedidoAtual()); await carregarEstado() } catch (e) { erro.value = mensagem(e) } finally { ocupado.value = '' }
}
watch(escolha, (v) => { if (v && passagens.value) carregarPrevia() })

// ---------- gerar ----------
const gerada = ref<CartaGuardada | null>(null)
async function gerar() {
  erro.value = ''; ocupado.value = 'Gerando a carta (PDF e PNG)…'
  try {
    gerada.value = await Cartas.GerarCarta({ ...pedidoAtual(), tipo: tipo.value, uuids: uuidsSelecionados.value, criterios: criterios.value })
    await Promise.all([carregarSalvas(), carregarEstado()])
  } catch (e) { erro.value = mensagem(e) } finally { ocupado.value = '' }
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

onMounted(async () => {
  await carregarEstado()
  try { propriedades.value = (await Consulta.Propriedades('', '')) ?? [] } catch { /* */ }
  if (props.uuidInicial) { tipo.value = 'individual'; uuid.value = props.uuidInicial } else planejar()
  carregarSalvas()
})
onActivated(() => { carregarEstado(); carregarSalvas() })
watch(() => props.uuidInicial, (u) => { if (u) { aba.value = 'nova'; tipo.value = 'individual'; uuid.value = u } })

const comLocalizacao = computed(() => propriedades.value.filter((p) => p.tem_gps))
const semLocalizacaoLista = computed(() => propriedades.value.filter((p) => !p.tem_gps))
const offline = computed(() => ehSemConexao(erro.value))
</script>

<template>
  <div class="pilha">
    <header class="cabeca">
      <div class="titulo">
        <h1>Cartas-imagem</h1>
        <p class="suave">Imagens de satélite Sentinel-2 com a localização das propriedades. Precisa de internet só para buscar imagens novas.</p>
      </div>
      <button class="estado" :class="estado?.configurado ? 'ativo' : 'parado'" @click="configAberta = true">
        <span class="ponto" aria-hidden="true"></span>
        {{ estado?.configurado ? 'Copernicus configurado' : 'Configurar acesso ao Copernicus' }}
      </button>
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
            <option v-for="p in comLocalizacao" :key="p.uuid" :value="p.uuid">{{ p.nome }} · {{ p.equipe }}</option>
            <optgroup v-if="semLocalizacaoLista.length" label="Sem localização (não podem ser representadas)">
              <option v-for="p in semLocalizacaoLista" :key="p.uuid" :value="p.uuid" disabled>{{ p.nome }}</option>
            </optgroup>
          </select>
        </div>
        <BarraFiltros v-if="tipo === 'geral'" v-model="filtro" :equipes="resultado?.equipes ?? []" :areas="resultado?.areas ?? []" :operacoes="resultado?.operacoes ?? []"
                      :campos="resultado?.campos ?? []" :mostrando="resultado?.totais.familias ?? null" :base="resultado?.familias_base ?? null" />
      </section>

      <div v-if="erro" class="aviso-erro erro-bloco" role="alert">
        <p>{{ erro }}</p>
        <div v-if="offline" class="linha"><button @click="buscarPassagens">Tentar novamente</button><button class="discreto" @click="erro = ''">Voltar</button></div>
        <div v-else-if="erro.includes('credenciais') || erro.includes('Configurar')" class="linha"><button @click="configAberta = true">Configurar acesso</button></div>
      </div>

      <div v-if="plano" class="area">
        <aside class="painel pilha">
          <section v-if="plano.grupos?.length" class="aviso">
            As propriedades se espalham por cerca de {{ num(Math.round(plano.extensao_km)) }} km: numa carta só, cada pixel cobriria ≈ {{ num(Math.round(plano.resolucao_m)) }} m.
            Para manter a legibilidade, gere uma carta por grupo.
            <select v-model.number="grupo" aria-label="Grupo" class="sel-grupo">
              <option :value="-1">Todas numa carta só</option>
              <option v-for="(g, i) in plano.grupos" :key="i" :value="i">Grupo {{ i + 1 }} ({{ plural(g?.length ?? 0, 'propriedade', 'propriedades') }})</option>
            </select>
          </section>

          <section class="pilha bloco-p">
            <h2>1. Data da imagem</h2>
            <div class="dupla">
              <label>De <input v-model="de" type="date" /></label>
              <label>Até <input v-model="ate" type="date" /></label>
            </div>
            <button :disabled="!!ocupado || !estado?.configurado" @click="buscarPassagens">Buscar imagens disponíveis</button>
            <p v-if="!estado?.configurado" class="fraco">Configure o acesso ao Copernicus para buscar imagens.</p>
            <div v-if="passagens" class="passagens" role="radiogroup" aria-label="Imagens disponíveis">
              <p v-if="!passagens.passagens?.length" class="fraco">Nenhuma passagem do satélite no período. Amplie o período.</p>
              <label v-for="p in passagens.passagens ?? []" :key="p.data" class="passagem" :class="{ sel: escolha === p.data }">
                <input v-model="escolha" type="radio" :value="p.data" />
                <span class="pd">{{ diaUTC(p.data) }}</span>
                <span class="nuvens" :class="{ muitas: p.nuvens > 30 }" :title="'Nuvens estimadas pelo catálogo (por tile, não exatamente na área)'">
                  <span class="barra"><span :style="{ width: `${Math.min(100, p.nuvens)}%` }"></span></span>{{ Math.round(p.nuvens) }}% nuvens
                </span>
                <span v-if="p.data === passagens.sugerida" class="selo ok">Sugerida</span>
              </label>
              <label class="passagem" :class="{ sel: escolha === 'mosaico' }">
                <input v-model="escolha" type="radio" value="mosaico" />
                <span>Mosaico do período (menos nuvens; a carta citará o período)</span>
              </label>
            </div>
          </section>

          <section class="pilha bloco-p">
            <h2>2. Enquadramento</h2>
            <p class="fraco">Arraste e aproxime o mapa: a área visível é a área da carta (proporção 4:3).</p>
            <button v-if="previa && foraDaImagem" :disabled="!!ocupado" @click="carregarPrevia">Atualizar a imagem para esta área</button>
          </section>

          <section class="pilha bloco-p">
            <h2>3. Gerar</h2>
            <button class="primario" :disabled="!!ocupado || !previa || !marcadores.length" @click="gerar">Gerar carta (PDF e PNG)</button>
            <div v-if="gerada" class="msg-ok pilha">
              <strong>Carta gerada e guardada.</strong>
              <div class="linha">
                <button @click="abrir(gerada!, 'carta.pdf')">Abrir PDF</button>
                <button @click="abrir(gerada!, 'carta.png')">Abrir PNG</button>
                <button class="discreto" @click="mostrar(gerada!)">Mostrar na pasta</button>
              </div>
            </div>
          </section>

          <section class="pilha bloco-p legenda">
            <h2>Legenda <span class="fraco">({{ marcadores.length }})</span></h2>
            <ol>
              <li v-for="m in marcadores" :key="m.numero"><span class="num">{{ m.numero }}</span>{{ m.nome }}</li>
            </ol>
            <details v-if="plano.sem_localizacao?.length">
              <summary>{{ plural(plano.sem_localizacao.length, 'propriedade sem localização', 'propriedades sem localização') }}</summary>
              <p class="fraco">Não podem ser representadas até que tenham coordenada coletada no celular.</p>
              <ul><li v-for="n in plano.sem_localizacao" :key="n">{{ n }}</li></ul>
            </details>
          </section>
        </aside>

        <section class="pilha mapa-col">
          <div class="info-mapa">
            <span v-if="previa"><strong>{{ previa.periodo }}</strong><template v-if="previa.guardada"> · guardada neste computador</template></span>
            <span v-else class="fraco">Sem imagem ainda: busque as imagens disponíveis.</span>
            <span v-if="previa && previa.sem_dados > 0.0005" class="selo alerta">{{ Math.round(previa.sem_dados * 100) }}% sem dados: escolha outra data</span>
          </div>
          <div class="mapa-envolta">
            <MapaCarta :pinos="pinos" :imagem="previa ? { url: previa.url, so: previa.so, ne: previa.ne } : null" :enquadrar="enquadrar"
                       :nomes="pinos.length <= 12" atribuicao="Contains modified Copernicus Sentinel data" @vista="aoMudarVista" />
            <div v-if="ocupado" class="carregando" role="status"><span class="giro" aria-hidden="true"></span>{{ ocupado }}</div>
          </div>
          <p class="fraco nota">
            Imagem Sentinel-2 com resolução de 10 m: aproximar não aumenta a resolução. O pino é a coordenada coletada em campo, com a precisão do GPS do celular.
          </p>
        </section>
      </div>
    </template>

    <template v-else>
      <TabelaDados :colunas="colSalvas" :linhas="salvas" :chave="(l) => l.id" rotulo="Cartas salvas" :ordem="{ chave: 'gerada_em', dir: 'desc' }"
                   texto-vazio="Nenhuma carta gerada ainda.">
        <template #miniatura="{ linha }">
          <img class="mini" :src="`/cartografia/cartas/${linha.id}/carta.png`" alt="" loading="lazy" @click="abrir(linha, 'carta.png')" />
        </template>
        <template #acoes="{ linha }">
          <div class="acoes">
            <button @click="abrir(linha, 'carta.pdf')">PDF</button>
            <button @click="abrir(linha, 'carta.png')">PNG</button>
            <button class="discreto" title="Salvar uma cópia em outra pasta" @click="copia(linha, 'carta.pdf')">Salvar cópia</button>
            <button class="discreto" @click="mostrar(linha)">Pasta</button>
          </div>
        </template>
      </TabelaDados>
      <p class="fraco">As cartas ficam guardadas na pasta de dados da Central e abrem sem internet.</p>
    </template>

    <ConfigCopernicus v-if="configAberta" @fechar="configAberta = false" @mudou="carregarEstado" />
  </div>
</template>

<style scoped>
.cabeca { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.titulo { display: flex; flex-direction: column; gap: .35rem; }
.estado { display: inline-flex; align-items: center; gap: .5rem; border-radius: 999px; font-weight: 600; }
.estado .ponto { width: 9px; height: 9px; border-radius: 50%; background: var(--alerta); }
.estado.ativo .ponto { background: var(--ok); box-shadow: 0 0 0 3px var(--ok-fundo); }
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
.area { display: grid; grid-template-columns: minmax(280px, 340px) minmax(0, 1fr); gap: 1.25rem; align-items: start; }
.painel { position: sticky; top: 1rem; }
.bloco-p { background: var(--folha); border: 1px solid var(--regua); border-radius: var(--raio-grande); padding: .9rem 1rem; gap: .6rem; }
.bloco-p h2 { font-size: 1rem; }
.dupla { display: grid; grid-template-columns: 1fr 1fr; gap: .5rem; }
.dupla label { display: flex; flex-direction: column; gap: .2rem; font-size: .88rem; color: var(--tinta-2); }
.passagens { display: flex; flex-direction: column; gap: .3rem; max-height: 260px; overflow: auto; }
.passagem { display: flex; flex-wrap: wrap; align-items: center; gap: .45rem; padding: .35rem .5rem; border: 1px solid var(--regua); border-radius: var(--raio); cursor: pointer; font-size: .9rem; }
.passagem.sel { border-color: var(--mata); box-shadow: inset 0 0 0 1px var(--mata); }
.passagem input { width: 16px; height: 16px; min-height: 0; accent-color: var(--mata); }
.pd { font-weight: 650; min-width: 5.6rem; }
.nuvens { display: inline-flex; align-items: center; gap: .35rem; color: var(--tinta-2); font-size: .85rem; flex: 1; white-space: nowrap; }
.nuvens .barra { width: 34px; height: 6px; border-radius: 3px; background: var(--folha-2); border: 1px solid var(--regua); overflow: hidden; }
.nuvens .barra span { display: block; height: 100%; background: var(--s2); }
.nuvens.muitas .barra span { background: var(--alerta); }
.sel-grupo { margin-top: .5rem; }
.legenda ol { margin: 0; padding: 0; list-style: none; display: flex; flex-direction: column; gap: .3rem; max-height: 280px; overflow: auto; }
.legenda li { display: flex; align-items: center; gap: .5rem; font-size: .9rem; }
.legenda .num { width: 22px; height: 22px; flex: none; border-radius: 50%; background: #f2b705; border: 2px solid #101412; color: #101412; display: grid; place-items: center; font-size: .75rem; font-weight: 700; }
.legenda details { font-size: .9rem; }
.legenda summary { cursor: pointer; color: var(--alerta); font-weight: 600; }
.mapa-col { gap: .6rem; }
.info-mapa { display: flex; justify-content: space-between; gap: .75rem; flex-wrap: wrap; align-items: center; }
.mapa-envolta { position: relative; }
.carregando { position: absolute; inset: 0; display: grid; place-content: center; justify-items: center; gap: .6rem; background: rgb(10 20 14 / 55%); color: #fff; font-weight: 600; border-radius: var(--raio-grande); z-index: 1000; }
.giro { width: 34px; height: 34px; border-radius: 50%; border: 3px solid rgb(255 255 255 / 30%); border-top-color: #fff; animation: giro 0.9s linear infinite; }
@keyframes giro { to { transform: rotate(360deg); } }
.nota { font-size: .85rem; }
.msg-ok { background: var(--ok-fundo); color: var(--ok); border-radius: var(--raio); padding: .7rem .85rem; gap: .5rem; }
.mini { width: 100px; aspect-ratio: 4 / 3.4; object-fit: cover; border-radius: 4px; border: 1px solid var(--regua); cursor: zoom-in; background: var(--folha-2); }
.acoes { display: flex; gap: .35rem; justify-content: center; flex-wrap: wrap; }
.acoes button { min-height: 32px; padding: .25rem .6rem; }
@media (max-width: 1100px) { .area { grid-template-columns: 1fr; } .painel { position: static; } }
</style>
