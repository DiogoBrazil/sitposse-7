import { describe, expect, it } from 'vitest'
import { chips, comExtra, dataParaMs, filtroVazio, msParaData } from './filtro'

const opcoes = {
  equipes: [{ valor: 'e1', rotulo: 'Alfa' }], areas: [{ valor: 'Linha C-25', rotulo: 'Linha C-25' }, { valor: '', rotulo: 'Sem identificação da área' }],
  operacoes: [], campos: [{ chave: 'bov_total', rotulo: 'Bovinos (total)', grupo: 'Bovinos' }, { chave: 'item:eletro.geladeira', rotulo: 'Geladeira', grupo: 'Eletrodomésticos', categoria: 'eletrodomestico' }],
}

describe('datas locais', () => {
  it('início e fim do dia local, ida e volta', () => {
    const ini = dataParaMs('2026-11-14', false)!
    const fim = dataParaMs('2026-11-14', true)!
    expect(fim - ini).toBe(86_400_000 - 1)
    expect(msParaData(ini)).toBe('2026-11-14')
    expect(msParaData(fim)).toBe('2026-11-14')
    expect(dataParaMs('14/11/2026', false)).toBeNull()
  })
})

describe('chips', () => {
  it('filtro vazio não tem chips', () => {
    expect(chips(filtroVazio(), opcoes)).toEqual([])
  })
  it('descreve e remove cada critério sem mexer nos outros', () => {
    const f = { ...filtroVazio(), equipes: ['e1'], areas_identificadas: [''], com_foto: true, area_min_ha: 10,
      condicoes: [{ campo: 'bov_total', operador: 'maior', valor: 5 }, { campo: 'item:eletro.geladeira', operador: 'nao_tem', valor: 0 }] }
    const cs = chips(f, opcoes)
    expect(cs.map((c) => c.texto)).toEqual([
      'Equipe: Alfa', 'Sem identificação da área', 'Com foto', 'Área a partir de 10 ha',
      'Bovinos (total): mais de 5', 'Não possui geladeira (eletrodoméstico)',
    ])
    const sem = cs[4].remover(f)
    expect(sem.condicoes).toEqual([{ campo: 'item:eletro.geladeira', operador: 'nao_tem', valor: 0 }])
    expect(sem.equipes).toEqual(['e1'])
    expect(f.condicoes).toHaveLength(2) // original intacto
  })
})

describe('comExtra (drill-down)', () => {
  it('soma a condição do número clicado às do filtro atual', () => {
    const f = { ...filtroVazio(), equipes: ['e1'], condicoes: [{ campo: 'bov_total', operador: 'maior', valor: 5 }] }
    const d = comExtra(f, { condicoes: [{ campo: 'veic_caminhoes', operador: 'tem', valor: 0 }] })
    expect(d.condicoes).toHaveLength(2)
    expect(d.equipes).toEqual(['e1'])
    expect(f.condicoes).toHaveLength(1)
  })
})
