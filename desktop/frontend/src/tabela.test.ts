import { describe, expect, it } from 'vitest'
import { comparar, ordenar, paginar } from './tabela'

describe('comparar', () => {
  it('ignora acentos e maiúsculas e ordena números dentro do texto', () => {
    expect(comparar('Árvore', 'arvore')).toBe(0)
    expect(comparar('Linha 9', 'Linha 25')).toBeLessThan(0)
    expect(comparar(2, 10)).toBeLessThan(0)
  })
})

describe('ordenar', () => {
  const linhas = [
    { nome: 'Jaraguá', area: 24.2 },
    { nome: 'Lua de Prata', area: null },
    { nome: 'água boa', area: 60.5 },
    { nome: 'Boa Vista', area: 0 },
  ]
  it('ordena textos em pt-BR sem diferenciar acentos', () => {
    expect(ordenar(linhas, (l) => l.nome, 'asc').map((l) => l.nome)).toEqual(['água boa', 'Boa Vista', 'Jaraguá', 'Lua de Prata'])
  })
  it('mantém ausentes no fim nas duas direções e trata zero como valor', () => {
    expect(ordenar(linhas, (l) => l.area, 'asc').map((l) => l.area)).toEqual([0, 24.2, 60.5, null])
    expect(ordenar(linhas, (l) => l.area, 'desc').map((l) => l.area)).toEqual([60.5, 24.2, 0, null])
  })
  it('é estável e não altera a lista original', () => {
    const iguais = [{ k: 1, id: 'a' }, { k: 1, id: 'b' }, { k: 0, id: 'c' }]
    expect(ordenar(iguais, (l) => l.k, 'desc').map((l) => l.id)).toEqual(['a', 'b', 'c'])
    expect(iguais.map((l) => l.id)).toEqual(['a', 'b', 'c'])
  })
})

describe('paginar', () => {
  const n = Array.from({ length: 53 }, (_, i) => i + 1)
  it('fatia e informa a faixa mostrada', () => {
    const p = paginar(n, 2, 25)
    expect(p.itens[0]).toBe(26)
    expect([p.inicio, p.fim, p.total, p.paginas]).toEqual([26, 50, 53, 3])
  })
  it('corrige página inexistente (por exemplo, depois de filtrar)', () => {
    expect(paginar(n, 9, 25).pagina).toBe(3)
    expect(paginar(n, 0, 25).pagina).toBe(1)
  })
  it('lista vazia e sem paginação', () => {
    expect(paginar([], 1, 25)).toMatchObject({ itens: [], pagina: 1, paginas: 1, inicio: 0, fim: 0, total: 0 })
    expect(paginar(n, 1, 0).itens.length).toBe(53)
  })
})

it('paginação de cinco mantém o conjunto completo para totais e exportação',()=>{
 const base=Array.from({length:11},(_,i)=>({nome:`Família ${i}`,quantidade:i}));
 const ordenadas=ordenar(base,l=>l.quantidade,'desc');
 expect(paginar(ordenadas,1,5).itens.map(l=>l.quantidade)).toEqual([10,9,8,7,6]);
 expect(paginar(ordenadas,3,5).itens.map(l=>l.quantidade)).toEqual([0]);
 expect(base.length).toBe(11);expect(base.reduce((n,l)=>n+l.quantidade,0)).toBe(55);
 expect(paginar(ordenadas.filter(l=>l.quantidade>8),3,5)).toMatchObject({pagina:1,total:2});
});
