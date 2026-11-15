// Gráficos da Central (ADR-0019): ECharts sob demanda, cores lidas do tema (estilo.css).
// Regras (dataviz): cor segue a entidade; marcas finas com ponta arredondada; grade discreta;
// rótulos na cor do texto; uma série → sem legenda (o título nomeia); sempre há tabela ao lado.
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent, AriaComponent } from 'echarts/components'
import type { EChartsOption } from 'echarts'

use([CanvasRenderer, BarChart, GridComponent, TooltipComponent, LegendComponent, AriaComponent])

/** Domínios com cor fixa em toda a Central (a cor acompanha o assunto, nunca a posição). */
export type Dominio = 'eletro' | 'pessoas' | 'rebanho' | 'veiculos' | 'moveis' | 'aves'
const SERIE: Record<Dominio, string> = { eletro: '--s1', pessoas: '--s2', rebanho: '--s3', veiculos: '--s4', moveis: '--s5', aves: '--s6' }

export interface Tema { tinta: string; tinta2: string; tinta3: string; regua: string; folha: string; cor: (d: Dominio) => string }

export function lerTema(): Tema {
  const s = getComputedStyle(document.documentElement)
  const v = (n: string) => s.getPropertyValue(n).trim()
  return { tinta: v('--tinta'), tinta2: v('--tinta-2'), tinta3: v('--tinta-3'), regua: v('--regua'), folha: v('--folha'), cor: (d) => v(SERIE[d]) }
}

export interface Ponto { nome: string; valor: number }

/** Rótulos de categoria mais longos que isto são cortados com reticências (o nome completo fica no tooltip e na tabela). */
const ROTULO_MAX = 190

const fmt = new Intl.NumberFormat('pt-BR', { maximumFractionDigits: 2 })
const fonte = '"Archivo Variable", "Segoe UI", sans-serif'

function tooltipBase(t: Tema) {
  return {
    backgroundColor: t.folha, borderColor: t.regua, borderWidth: 1, padding: [6, 10],
    textStyle: { color: t.tinta, fontFamily: fonte, fontSize: 13 },
    extraCssText: 'box-shadow: 0 6px 18px rgb(0 0 0 / 14%); border-radius: 6px;',
  }
}

/**
 * Barras horizontais de uma série (magnitude por categoria): rótulo do valor na ponta,
 * na cor do texto. A ordem dos pontos é mantida (categorias com ordem natural, como faixas etárias).
 */
export function barrasHorizontais(t: Tema, dominio: Dominio, pontos: Ponto[], unidade = ''): EChartsOption {
  const cor = t.cor(dominio)
  const nomes = pontos.map((p) => p.nome)
  return {
    aria: { enabled: true },
    animationDuration: 400,
    textStyle: { fontFamily: fonte },
    grid: { left: Math.min(larguraTexto(nomes, 13), ROTULO_MAX) + 18, right: 44, top: 4, bottom: 4, outerBoundsMode: 'none' },
    tooltip: { ...tooltipBase(t), trigger: 'item', formatter: (p: any) => `${p.name}<br/><b>${fmt.format(p.value)}</b>${unidade ? ' ' + unidade : ''}` },
    xAxis: { type: 'value', minInterval: 1, splitNumber: 3, axisLabel: { show: false }, splitLine: { lineStyle: { color: t.regua, type: [3, 4] } }, axisLine: { show: false } },
    yAxis: {
      type: 'category', data: nomes, inverse: true, axisTick: { show: false },
      axisLine: { lineStyle: { color: t.regua } }, axisLabel: { color: t.tinta2, fontSize: 13, margin: 10, fontFamily: fonte, width: ROTULO_MAX, overflow: 'truncate' },
    },
    series: [{
      type: 'bar', data: pontos.map((p) => p.valor), barMaxWidth: 16, barMinHeight: 0,
      itemStyle: { color: cor, borderRadius: [0, 4, 4, 0] },
      emphasis: { itemStyle: { color: cor, opacity: 0.85 } },
      cursor: 'pointer',
      label: { show: true, position: 'right', color: t.tinta, fontSize: 13, fontWeight: 600, formatter: (p: any) => fmt.format(p.value) },
    }],
  }
}

let ctx: CanvasRenderingContext2D | null = null
/** Largura (px) do maior texto, para reservar a margem dos rótulos do eixo (ECharts não encolhe a área de plotagem). */
export function larguraTexto(textos: string[], tamanho: number): number {
  ctx ??= document.createElement('canvas').getContext('2d')
  if (!ctx) return Math.max(0, ...textos.map((t) => t.length)) * tamanho * 0.55
  ctx.font = `${tamanho}px ${fonte}`
  return Math.ceil(Math.max(0, ...textos.map((t) => ctx!.measureText(t).width)))
}

/** Altura sugerida para barras horizontais (a área do gráfico cresce com as categorias). */
export function alturaBarras(n: number): number {
  return Math.max(96, n * 34 + 16)
}

/** Faixa de totais (componente FaixaTotais). `cor` é uma variável de série, por exemplo "--s2". */
export interface ItemTotal { valor: string; rotulo: string; detalhe?: string }
export interface GrupoTotal { titulo: string; itens: ItemTotal[]; cor?: string }
