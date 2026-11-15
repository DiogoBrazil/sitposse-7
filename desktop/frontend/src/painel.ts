// Montagem dos indicadores do Painel a partir do Resultado do FiltroEstudo (§22-A).
import type { Familia, Grupo, Resultado, Totais } from '../bindings/sitposse7/desktop/internal/estatisticas/models'
import type { Condicao, Filtro } from './filtro'

export type { Familia, Grupo, Resultado, Totais }

/** Linha de detalhamento de um cartão; `chave` identifica o drill-down. */
export interface LinhaIndicador { rotulo: string; valor: number; texto?: string; chave: string }
export interface Principal { rotulo: string; valor: number; texto?: string; chave: string }

export const v = (t: Totais | Familia | undefined | null, chave: string): number => (t?.valores?.[chave] ?? 0)

/**
 * Drill-down: o critério extra que restringe o filtro às famílias que compõem um número.
 * Quantidade → "possui"; famílias/área/localização/foto → o critério correspondente.
 */
export function extraDoIndicador(chave: string): Partial<Filtro> {
  switch (chave) {
    case 'familias': return {}
    case 'com_localizacao': return { com_localizacao: true }
    case 'sem_localizacao': return { com_localizacao: false }
    case 'com_foto': return { com_foto: true }
    case 'sem_foto': return { com_foto: false }
    case 'com_area': return { com_area: true }
    case 'sem_area': return { com_area: false }
    default: return { condicoes: [{ campo: chave, operador: 'tem', valor: 0 } as Condicao] }
  }
}

/** Conjuntos de colunas das tabelas analíticas ("colunas escolhíveis"). */
export const CONJUNTOS: { id: string; rotulo: string; campos: { chave: string; rotulo: string }[] }[] = [
  { id: 'resumo', rotulo: 'Resumo', campos: [
    { chave: 'pessoas_total', rotulo: 'Pessoas' }, { chave: 'veic_total', rotulo: 'Veículos' }, { chave: 'bov_total', rotulo: 'Bovinos' },
    { chave: 'equ_total', rotulo: 'Equinos' }, { chave: 'aves_total', rotulo: 'Aves' }, { chave: 'eletro_total', rotulo: 'Eletrodomésticos' }, { chave: 'moveis_total', rotulo: 'Móveis' },
  ] },
  { id: 'pessoas', rotulo: 'Pessoas', campos: [
    { chave: 'pessoas_adultos', rotulo: 'Adultos' }, { chave: 'pessoas_jovens', rotulo: 'Jovens' }, { chave: 'pessoas_criancas', rotulo: 'Crianças' }, { chave: 'pessoas_total', rotulo: 'Total' },
  ] },
  { id: 'veiculos', rotulo: 'Veículos', campos: [
    { chave: 'veic_caminhoes', rotulo: 'Caminhões' }, { chave: 'veic_automoveis_utilitarios', rotulo: 'Automóveis e utilitários' }, { chave: 'veic_motos_ciclomotores', rotulo: 'Motos e ciclomotores' }, { chave: 'veic_total', rotulo: 'Total' },
  ] },
  { id: 'rebanho', rotulo: 'Rebanho', campos: [
    { chave: 'bov_adultos', rotulo: 'Bovinos adultos' }, { chave: 'bov_bezerros', rotulo: 'Bezerros' }, { chave: 'bov_total', rotulo: 'Bovinos' },
    { chave: 'equ_adultos', rotulo: 'Equinos adultos' }, { chave: 'equ_potros', rotulo: 'Potros' }, { chave: 'equ_total', rotulo: 'Equinos' },
  ] },
  { id: 'bens', rotulo: 'Bens e criações', campos: [
    { chave: 'aves_total', rotulo: 'Aves' }, { chave: 'eletro_total', rotulo: 'Eletrodomésticos' }, { chave: 'moveis_total', rotulo: 'Móveis' }, { chave: 'fotos', rotulo: 'Fotos' },
  ] },
]

/** Domínio (cor) de um campo, para os gráficos. */
export function dominioDoCampo(chave: string): 'pessoas' | 'veiculos' | 'rebanho' | 'aves' | 'eletro' | 'moveis' {
  if (chave.startsWith('pessoas')) return 'pessoas'
  if (chave.startsWith('veic')) return 'veiculos'
  if (chave.startsWith('bov') || chave.startsWith('equ')) return 'rebanho'
  if (chave.startsWith('aves') || chave.startsWith('item:ave')) return 'aves'
  if (chave.startsWith('moveis') || chave.startsWith('item:movel')) return 'moveis'
  return 'eletro'
}
