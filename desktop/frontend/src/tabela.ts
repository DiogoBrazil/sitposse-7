// Lógica pura das tabelas (ADR-0019): ordenação e paginação, testadas com Vitest.

export type Direcao = 'asc' | 'desc'

const colacao = new Intl.Collator('pt-BR', { sensitivity: 'base', numeric: true })

const vazio = (v: unknown) => v === null || v === undefined || v === ''

/** Compara dois valores: números numericamente, textos em pt-BR sem diferenciar acentos e maiúsculas. */
export function comparar(a: unknown, b: unknown): number {
  if (typeof a === 'number' && typeof b === 'number') return a - b
  if (typeof a === 'boolean' && typeof b === 'boolean') return Number(a) - Number(b)
  return colacao.compare(String(a), String(b))
}

/**
 * Ordena sem alterar a lista original. A ordenação é estável, e valores ausentes
 * (null, undefined ou texto vazio) ficam sempre no fim, em qualquer direção:
 * "área não informada" não é zero (AGENTS.md §3).
 */
export function ordenar<T>(linhas: readonly T[], valor: (l: T) => unknown, dir: Direcao): T[] {
  const sinal = dir === 'asc' ? 1 : -1
  return linhas
    .map((l, i) => ({ l, i, v: valor(l) }))
    .sort((x, y) => {
      const vx = vazio(x.v), vy = vazio(y.v)
      if (vx || vy) return vx === vy ? x.i - y.i : vx ? 1 : -1
      return comparar(x.v, y.v) * sinal || x.i - y.i
    })
    .map((x) => x.l)
}

export interface Pagina<T> {
  itens: T[]
  /** Página efetiva (1..paginas), corrigida se a pedida não existe mais. */
  pagina: number
  paginas: number
  /** Posição do primeiro e do último item mostrados (1-based; 0 se vazio). */
  inicio: number
  fim: number
  total: number
}

/** Fatia a lista; porPagina <= 0 mostra tudo numa página. */
export function paginar<T>(linhas: readonly T[], pagina: number, porPagina: number): Pagina<T> {
  const total = linhas.length
  if (porPagina <= 0) return { itens: [...linhas], pagina: 1, paginas: 1, inicio: total ? 1 : 0, fim: total, total }
  const paginas = Math.max(1, Math.ceil(total / porPagina))
  const p = Math.min(Math.max(1, Math.floor(pagina) || 1), paginas)
  const ini = (p - 1) * porPagina
  const itens = linhas.slice(ini, ini + porPagina)
  return { itens, pagina: p, paginas, inicio: total ? ini + 1 : 0, fim: ini + itens.length, total }
}

/** Coluna do componente TabelaDados. */
export interface Coluna<L> {
  chave: string
  rotulo: string
  tipo?: 'texto' | 'numero'
  /** Valor usado para ordenar e para a célula padrão (padrão: linha[chave]). */
  valor?: (l: L) => unknown
  /** Texto da célula padrão (padrão: o valor). */
  formatar?: (l: L) => string
  ordenavel?: boolean
  /** Coluna secundária: some no layout empilhado. */
  secundaria?: boolean
  largura?: string
}
