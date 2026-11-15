<script setup lang="ts">
/**
 * Prévia da carta-imagem (A27, ADR-0022) com Leaflet 1.9.4 (ADR-0013).
 * - Sem mapa-base de terceiros: a única camada de imagem é a Sentinel-2 obtida pela Central.
 * - A imagem vem em EPSG:3857, a mesma projeção do Leaflet: o ImageOverlay fica exato.
 * - Pinos numa camada separada; a posição é a coordenada coletada (WGS 84), projetada pelo
 *   Leaflet com a mesma fórmula do Go (internal/cartografia/geo).
 * - A área visível tem a proporção 4:3 da carta: o que se vê é o que vai para a carta.
 */
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'

import type { PinoMapa, Retangulo3857 } from '../cartas'

const props = defineProps<{
  pinos: PinoMapa[]
  imagem: { url: string; so: { lat: number; lon: number }; ne: { lat: number; lon: number } } | null
  enquadrar: { so: { lat: number; lon: number }; ne: { lat: number; lon: number } } | null
  nomes: boolean
  atribuicao?: string
}>()
const emit = defineEmits<{ vista: [r: Retangulo3857, foraDaImagem: boolean] }>()

const el = ref<HTMLDivElement | null>(null)
let mapa: L.Map | null = null
let camadaImagem: L.ImageOverlay | null = null
const camadaPinos = L.layerGroup()

function retanguloVisivel(): Retangulo3857 | null {
  if (!mapa) return null
  const b = mapa.getBounds()
  const so = L.Projection.SphericalMercator.project(b.getSouthWest())
  const ne = L.Projection.SphericalMercator.project(b.getNorthEast())
  return { min_x: so.x, min_y: so.y, max_x: ne.x, max_y: ne.y }
}

function avisarVista() {
  const r = retanguloVisivel()
  if (!r || !mapa) return
  const fora = !camadaImagem || !camadaImagem.getBounds().contains(mapa.getBounds())
  emit('vista', r, fora)
}

function desenharPinos() {
  camadaPinos.clearLayers()
  for (const p of props.pinos) {
    const icone = L.divIcon({ className: 'pino-carta', html: `<span>${p.numero}</span>`, iconSize: [26, 26], iconAnchor: [13, 13] })
    const m = L.marker([p.lat, p.lon], { icon: icone, keyboard: false, title: `${p.numero}. ${p.nome}` })
    if (props.nomes) m.bindTooltip(p.nome, { permanent: true, direction: 'right', offset: [14, 0], className: 'rotulo-pino' })
    camadaPinos.addLayer(m)
  }
}

function aplicarImagem() {
  if (!mapa) return
  if (camadaImagem) { camadaImagem.remove(); camadaImagem = null }
  if (props.imagem) {
    const b = L.latLngBounds([props.imagem.so.lat, props.imagem.so.lon], [props.imagem.ne.lat, props.imagem.ne.lon])
    camadaImagem = L.imageOverlay(props.imagem.url, b, { interactive: false }).addTo(mapa)
    camadaImagem.bringToBack()
  }
  avisarVista()
}

function aplicarEnquadramento() {
  if (!mapa || !props.enquadrar) return
  const b = L.latLngBounds([props.enquadrar.so.lat, props.enquadrar.so.lon], [props.enquadrar.ne.lat, props.enquadrar.ne.lon])
  mapa.fitBounds(b, { animate: false, padding: [0, 0] })
}

onMounted(() => {
  if (!el.value) return
  mapa = L.map(el.value, { zoomSnap: 0.25, zoomDelta: 0.5, attributionControl: true, minZoom: 3, maxZoom: 19 })
  mapa.attributionControl.setPrefix(false)
  if (props.atribuicao) mapa.attributionControl.addAttribution(props.atribuicao)
  L.control.scale({ imperial: false, position: 'bottomleft' }).addTo(mapa)
  camadaPinos.addTo(mapa)
  desenharPinos()
  aplicarEnquadramento()
  aplicarImagem()
  mapa.on('moveend', avisarVista)
})
onBeforeUnmount(() => { mapa?.remove(); mapa = null })

watch(() => [props.pinos, props.nomes], desenharPinos, { deep: true })
watch(() => props.imagem?.url, aplicarImagem)
watch(() => props.enquadrar, aplicarEnquadramento, { deep: true })

defineExpose({ retanguloVisivel })
</script>

<template>
  <div class="moldura-mapa">
    <div ref="el" class="mapa" role="application" aria-label="Prévia da carta-imagem: arraste para mover, use a roda do mouse ou os botões + e − para aproximar" />
  </div>
</template>

<style scoped>
.moldura-mapa { width: 100%; aspect-ratio: 4 / 3; border: 1px solid var(--regua-forte); border-radius: var(--raio-grande); overflow: hidden; background: #2b332d; }
.mapa { width: 100%; height: 100%; background: #2b332d; }
:deep(.pino-carta) { display: grid; place-items: center; }
:deep(.pino-carta span) {
  width: 24px; height: 24px; border-radius: 50%; background: #f2b705; border: 2px solid #101412;
  display: grid; place-items: center; font: 700 12px/1 "Archivo Variable", sans-serif; color: #101412; box-shadow: 0 1px 3px rgb(0 0 0 / 40%);
}
:deep(.rotulo-pino) { background: rgb(16 20 18 / 78%); color: #fff; border: none; box-shadow: none; font: 600 12px "Archivo Variable", sans-serif; padding: 2px 6px; }
:deep(.rotulo-pino::before) { display: none; }
:deep(.leaflet-control-attribution) { font: 11px "Archivo Variable", sans-serif; background: rgb(255 255 255 / 85%); }
</style>
