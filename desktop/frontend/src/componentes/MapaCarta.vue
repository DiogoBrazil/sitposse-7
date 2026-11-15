<script setup lang="ts">
/**
 * Mapa da carta-imagem (A30, ADR-0027; Leaflet 1.9.4, ADR-0013).
 * - Imagem: tiles da composição servidos pela própria Central (/cartografia/composicoes/…),
 *   em duas camadas: a geral (até o zoom da carta geral) e a de detalhe (setores, até a
 *   resolução nativa). Nenhum mapa-base de terceiros.
 * - Pinos: posição = coordenada coletada (WGS 84), projetada pelo Leaflet com a mesma
 *   fórmula do Go. Passar o mouse mostra nome e responsável; clicar abre o cartão. Pinos que
 *   se sobrepõem na tela viram um selo com a contagem, e o cartão lista todos (nenhum some).
 * - Os contornos tracejados são as folhas da carta (geral e setores): a área exportada não
 *   depende de onde o operador deixou a vista.
 */
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'

import { agruparPinos, type PinoMapa } from '../cartas'

export interface CamadaImagem { tiles: string; zMin: number; zGeral: number; zNativo: number; so: { lat: number; lon: number }; ne: { lat: number; lon: number } }
export interface FolhaMapa { rotulo: string; so: { lat: number; lon: number }; ne: { lat: number; lon: number } }

const props = defineProps<{
  pinos: PinoMapa[]
  camada: CamadaImagem | null
  folhas: FolhaMapa[]
  enquadrar: { so: { lat: number; lon: number }; ne: { lat: number; lon: number } } | null
  selecionados: number[]
  atribuicao?: string
}>()
const emit = defineEmits<{ selecionar: [numeros: number[]] }>()

const el = ref<HTMLDivElement | null>(null)
let mapa: L.Map | null = null
let camadaGeral: L.TileLayer | null = null
let camadaDetalhe: L.TileLayer | null = null
const camadaFolhas = L.layerGroup()
const camadaPinos = L.layerGroup()
const DISTANCIA_GRUPO = 18 // px na tela

function limites(r: { so: { lat: number; lon: number }; ne: { lat: number; lon: number } }) {
  return L.latLngBounds([r.so.lat, r.so.lon], [r.ne.lat, r.ne.lon])
}

/** Conteúdo do tooltip como DOM (nome e responsável vêm do cadastro: nunca como HTML). */
function dica(ps: PinoMapa[]): HTMLElement {
  const d = document.createElement('div')
  if (ps.length === 1) {
    const b = document.createElement('strong')
    b.textContent = ps[0].nome
    const r = document.createElement('div')
    r.textContent = `Responsável: ${ps[0].responsavel || '—'}`
    d.append(b, r)
  } else {
    const b = document.createElement('strong')
    b.textContent = `${ps.length} propriedades neste ponto`
    const r = document.createElement('div')
    r.textContent = `Nº ${ps.map((p) => p.numero).join(', ')} — clique para ver`
    d.append(b, r)
  }
  return d
}

function desenharPinos() {
  camadaPinos.clearLayers()
  if (!mapa) return
  const porNumero = new Map(props.pinos.map((p) => [p.numero, p]))
  const tela = props.pinos.map((p) => {
    const pt = mapa!.latLngToLayerPoint([p.lat, p.lon])
    return { numero: p.numero, x: pt.x, y: pt.y }
  })
  const sel = new Set(props.selecionados)
  for (const g of agruparPinos(tela, DISTANCIA_GRUPO)) {
    const ps = g.numeros.map((n) => porNumero.get(n)!)
    const ativo = g.numeros.some((n) => sel.has(n))
    const unico = ps.length === 1
    const icone = L.divIcon({
      className: `pino-carta${unico ? '' : ' grupo'}${ativo ? ' ativo' : ''}`,
      html: unico ? `<span>${ps[0].numero}</span>` : `<span>${ps.length}</span>`,
      iconSize: [28, 28], iconAnchor: [14, 14],
    })
    const m = L.marker([ps[0].lat, ps[0].lon], { icon: icone, keyboard: true, riseOnHover: true,
      title: unico ? `Propriedade nº ${ps[0].numero}` : `${ps.length} propriedades` })
    m.bindTooltip(dica(ps), { direction: 'top', offset: [0, -14], className: 'dica-pino', opacity: 1 })
    m.on('click', () => emit('selecionar', g.numeros))
    camadaPinos.addLayer(m)
  }
}

function desenharFolhas() {
  camadaFolhas.clearLayers()
  for (const f of props.folhas) {
    const r = L.rectangle(limites(f), { color: '#ffffff', weight: 2, dashArray: '6 6', fill: false, interactive: false })
    camadaFolhas.addLayer(r)
    if (f.rotulo) {
      // Rótulo no canto noroeste (no centro ele cobriria pinos).
      const rot = L.divIcon({ className: 'rotulo-setor', html: `Setor ${f.rotulo.replace(/[^A-Z]/g, '')}`, iconSize: null as unknown as L.PointExpression, iconAnchor: [-4, -4] })
      camadaFolhas.addLayer(L.marker([f.ne.lat, f.so.lon], { icon: rot, interactive: false, keyboard: false }))
    }
  }
}

function aplicarCamada() {
  if (!mapa) return
  camadaGeral?.remove(); camadaDetalhe?.remove()
  camadaGeral = camadaDetalhe = null
  const c = props.camada
  if (!c) return
  const b = limites(c)
  camadaGeral = L.tileLayer(c.tiles, { minNativeZoom: c.zMin, maxNativeZoom: c.zGeral, maxZoom: 19, bounds: b, keepBuffer: 2 }).addTo(mapa)
  if (c.zNativo > c.zGeral) {
    camadaDetalhe = L.tileLayer(c.tiles, { minZoom: c.zGeral + 1, minNativeZoom: c.zGeral + 1, maxNativeZoom: c.zNativo, maxZoom: 19, bounds: b, keepBuffer: 2 }).addTo(mapa)
  }
  mapa.setMaxZoom(c.zNativo + 2)
  camadaGeral.bringToBack()
}

// Enquadramento pendente: o mapa pode nascer com tamanho zero (tela ainda em layout).
let enquadrarPendente = true
function ajustar() {
  enquadrarPendente = true
  if (!mapa || !props.enquadrar || !el.value || el.value.clientWidth < 50 || el.value.clientHeight < 50) return
  mapa.invalidateSize({ animate: false })
  mapa.fitBounds(limites(props.enquadrar), { animate: false, padding: [12, 12] })
  enquadrarPendente = false
}

function centralizar(lat: number, lon: number) {
  if (!mapa) return
  mapa.setView([lat, lon], Math.max(mapa.getZoom(), (props.camada?.zGeral ?? 14) + 1))
}

let observador: ResizeObserver | null = null

function teclado(e: KeyboardEvent) { if (e.key === 'Escape') emit('selecionar', []) }

onMounted(() => {
  if (!el.value) return
  mapa = L.map(el.value, { zoomSnap: 0.25, zoomDelta: 0.5, attributionControl: true, minZoom: 3, maxZoom: 18 })
  mapa.attributionControl.setPrefix(false)
  if (props.atribuicao) mapa.attributionControl.addAttribution(props.atribuicao)
  L.control.scale({ imperial: false, position: 'bottomleft' }).addTo(mapa)
  camadaFolhas.addTo(mapa)
  camadaPinos.addTo(mapa)
  aplicarCamada()
  desenharFolhas()
  ajustar()
  desenharPinos()
  mapa.on('zoomend', desenharPinos)
  mapa.on('click', () => emit('selecionar', []))
  observador = new ResizeObserver(() => {
    if (!mapa || !el.value || el.value.clientWidth < 50) return
    mapa.invalidateSize({ animate: false })
    if (enquadrarPendente) ajustar()
  })
  observador.observe(el.value)
  window.addEventListener('keydown', teclado)
})
onBeforeUnmount(() => { observador?.disconnect(); window.removeEventListener('keydown', teclado); mapa?.remove(); mapa = null })

watch(() => [props.pinos, props.selecionados], desenharPinos, { deep: true })
watch(() => props.camada?.tiles, aplicarCamada)
watch(() => props.folhas, desenharFolhas, { deep: true })
watch(() => props.enquadrar, ajustar, { deep: true })
watch(() => props.atribuicao, (novo, antigo) => {
  if (!mapa) return
  if (antigo) mapa.attributionControl.removeAttribution(antigo)
  if (novo) mapa.attributionControl.addAttribution(novo)
})

defineExpose({ ajustar, centralizar })
</script>

<template>
  <div class="moldura-mapa">
    <div ref="el" class="mapa" role="application"
         aria-label="Mapa da carta-imagem: arraste para mover, use a roda do mouse ou os botões + e − para aproximar; passe o mouse sobre um pino para ver a propriedade e clique para abrir o cartão" />
    <slot />
  </div>
</template>

<style scoped>
.moldura-mapa { position: relative; width: 100%; aspect-ratio: 4 / 3; border: 1px solid var(--regua-forte); border-radius: var(--raio-grande); overflow: hidden; background: #2b332d; }
.mapa { width: 100%; height: 100%; background: #2b332d; }
:deep(.pino-carta) { display: grid; place-items: center; cursor: pointer; }
:deep(.pino-carta span) {
  width: 24px; height: 24px; border-radius: 50%; background: #f2b705; border: 2px solid #101412;
  display: grid; place-items: center; font: 700 12px/1 "Archivo Variable", sans-serif; color: #101412; box-shadow: 0 1px 3px rgb(0 0 0 / 40%);
  transition: transform .12s ease;
}
:deep(.pino-carta:hover span) { transform: scale(1.15); }
:deep(.pino-carta.grupo span) { background: #ffffff; border-radius: 8px; width: 26px; box-shadow: 3px 3px 0 #f2b705, 0 1px 4px rgb(0 0 0 / 45%); }
:deep(.pino-carta.ativo span) { outline: 3px solid #ffffff; outline-offset: 1px; }
:deep(.dica-pino) { font: 13px/1.35 "Archivo Variable", sans-serif; padding: 6px 9px; border-radius: 8px; border: none; box-shadow: 0 2px 8px rgb(0 0 0 / 30%); }
:deep(.rotulo-setor) { background: rgb(16 20 18 / 78%); color: #fff; border-radius: 4px; font: 700 13px/1.4 "Archivo Variable", sans-serif; padding: 1px 8px; white-space: nowrap; width: auto !important; height: auto !important; }
:deep(.leaflet-control-attribution) { font: 11px "Archivo Variable", sans-serif; background: rgb(255 255 255 / 85%); }
</style>
