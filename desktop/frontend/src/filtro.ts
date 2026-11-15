// FiltroEstudo no frontend (docs/requirements/estatisticas.md §22-A): estado da barra de
// filtros, conversão de datas e os "chips" de filtros ativos. Lógica pura, testada com Vitest.
import type { Campo, Condicao, Filtro, Opcao } from '../bindings/sitposse7/desktop/internal/estatisticas/models'

export type { Campo, Condicao, Filtro, Opcao }

/** Filtro vazio dentro do estudo em foco (A28: Painel, Propriedades e cartas mostram um estudo por vez). */
export function filtroVazio(estudo = ''): Filtro {
  return {
    estudo, jornadas: [], equipes: [], busca: '', periodo_campo: 'cadastro', de: null, ate: null,
    com_localizacao: null, com_foto: null, com_area: null, faixas_area: [], area_min_ha: null, area_max_ha: null, condicoes: [],
  }
}

export const OPERADORES: { valor: string; rotulo: string; usaValor: boolean }[] = [
  { valor: 'tem', rotulo: 'possui', usaValor: false },
  { valor: 'nao_tem', rotulo: 'não possui', usaValor: false },
  { valor: 'maior', rotulo: 'mais de', usaValor: true },
  { valor: 'pelo_menos', rotulo: 'pelo menos', usaValor: true },
  { valor: 'menor', rotulo: 'menos de', usaValor: true },
  { valor: 'igual', rotulo: 'exatamente', usaValor: true },
]

export const FAIXAS: Opcao[] = [
  { valor: 'ate10', rotulo: 'Até 10 ha' },
  { valor: '10a25', rotulo: 'De 10 a 25 ha' },
  { valor: '25a50', rotulo: 'De 25 a 50 ha' },
  { valor: '50a100', rotulo: 'De 50 a 100 ha' },
  { valor: 'acima100', rotulo: '100 ha ou mais' },
  { valor: 'sem_area', rotulo: 'Sem área informada' },
]

/** Data local "AAAA-MM-DD" → ms UTC do início (00:00) ou do fim (23:59:59.999) do dia local. */
export function dataParaMs(data: string, fim: boolean): number | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(data)
  if (!m) return null
  const d = fim ? new Date(+m[1], +m[2] - 1, +m[3], 23, 59, 59, 999) : new Date(+m[1], +m[2] - 1, +m[3])
  return d.getTime()
}

/** ms → "AAAA-MM-DD" no fuso local (para o campo de data). */
export function msParaData(ms: number | null | undefined): string {
  if (ms == null) return ''
  const d = new Date(ms)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

const dataCurta = (ms: number) => new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short' }).format(new Date(ms))
const numero = (n: number) => new Intl.NumberFormat('pt-BR', { maximumFractionDigits: 2 }).format(n)

export function descreverCondicao(c: Condicao, campos: Campo[]): string {
  const campo = campos.find((x) => x.chave === c.campo)
  const nome = campo ? rotuloCampo(campo) : c.campo
  const op = OPERADORES.find((o) => o.valor === c.operador)
  if (!op) return nome
  return op.usaValor ? `${nome}: ${op.rotulo} ${numero(c.valor)}` : `${op.rotulo === 'possui' ? 'Possui' : 'Não possui'} ${nome.toLowerCase()}`
}

/** Rótulo de um campo, com a categoria nos itens ("Geladeira" → "Geladeira (eletrodoméstico)"). */
export function rotuloCampo(c: Campo): string {
  if (!c.categoria) return c.rotulo
  const cat = { eletrodomestico: 'eletrodoméstico', movel: 'móvel', ave: 'ave' }[c.categoria] ?? c.categoria
  return `${c.rotulo} (${cat}${c.outros ? ', personalizado' : ''})`
}

export interface Chip { chave: string; texto: string; remover: (f: Filtro) => Filtro }

const sem = <T>(lista: T[] | null | undefined, x: T) => (lista ?? []).filter((y) => y !== x)

/** Filtros ativos, um chip por critério (o estudo em foco aparece no cabeçalho, não como chip). */
export function chips(f: Filtro, o: { equipes: Opcao[]; jornadas: Opcao[]; campos: Campo[] }): Chip[] {
  const out: Chip[] = []
  const rot = (lista: Opcao[], v: string) => lista.find((x) => x.valor === v)?.rotulo ?? v
  for (const v of f.jornadas ?? []) out.push({ chave: `jo:${v}`, texto: rot(o.jornadas, v), remover: (x) => ({ ...x, jornadas: sem(x.jornadas, v) }) })
  for (const v of f.equipes ?? []) out.push({ chave: `eq:${v}`, texto: `Equipe: ${rot(o.equipes, v)}`, remover: (x) => ({ ...x, equipes: sem(x.equipes, v) }) })
  if (f.busca.trim()) out.push({ chave: 'busca', texto: `Busca: “${f.busca.trim()}”`, remover: (x) => ({ ...x, busca: '' }) })
  if (f.de != null || f.ate != null) {
    const qual = f.periodo_campo === 'recebimento' ? 'Recebidas' : 'Cadastradas'
    const txt = f.de != null && f.ate != null ? `${qual} de ${dataCurta(f.de)} a ${dataCurta(f.ate)}` : f.de != null ? `${qual} desde ${dataCurta(f.de)}` : `${qual} até ${dataCurta(f.ate!)}`
    out.push({ chave: 'periodo', texto: txt, remover: (x) => ({ ...x, de: null, ate: null }) })
  }
  if (f.com_localizacao != null) out.push({ chave: 'loc', texto: f.com_localizacao ? 'Com localização' : 'Sem localização', remover: (x) => ({ ...x, com_localizacao: null }) })
  if (f.com_foto != null) out.push({ chave: 'foto', texto: f.com_foto ? 'Com foto' : 'Sem foto', remover: (x) => ({ ...x, com_foto: null }) })
  if (f.com_area != null) out.push({ chave: 'area', texto: f.com_area ? 'Com área informada' : 'Sem área informada', remover: (x) => ({ ...x, com_area: null }) })
  for (const v of f.faixas_area ?? []) out.push({ chave: `fx:${v}`, texto: `Tamanho: ${rot(FAIXAS, v)}`, remover: (x) => ({ ...x, faixas_area: sem(x.faixas_area, v) }) })
  if (f.area_min_ha != null || f.area_max_ha != null) {
    const txt = f.area_min_ha != null && f.area_max_ha != null ? `Área de ${numero(f.area_min_ha)} a ${numero(f.area_max_ha)} ha`
      : f.area_min_ha != null ? `Área a partir de ${numero(f.area_min_ha)} ha` : `Área até ${numero(f.area_max_ha!)} ha`
    out.push({ chave: 'areaminmax', texto: txt, remover: (x) => ({ ...x, area_min_ha: null, area_max_ha: null }) })
  }
  ;(f.condicoes ?? []).forEach((c, i) => out.push({
    chave: `c:${i}:${c.campo}`, texto: descreverCondicao(c, o.campos),
    remover: (x) => ({ ...x, condicoes: (x.condicoes ?? []).filter((_, j) => j !== i) }),
  }))
  return out
}

/** Junta um critério extra (drill-down) ao filtro atual, sem alterar o original. */
export function comExtra(f: Filtro, extra: Partial<Filtro>): Filtro {
  return {
    ...f, ...extra,
    equipes: extra.equipes ?? f.equipes, jornadas: extra.jornadas ?? f.jornadas,
    faixas_area: extra.faixas_area ?? f.faixas_area,
    condicoes: [...(f.condicoes ?? []), ...(extra.condicoes ?? [])],
  }
}
