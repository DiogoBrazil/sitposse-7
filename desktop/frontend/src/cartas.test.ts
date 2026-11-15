import { describe, expect, it } from 'vitest'
import { agruparPinos, primeiroPNG } from './cartas'

describe('agruparPinos', () => {
  it('mantém separados os pinos distantes e junta os sobrepostos, sem perder nenhum', () => {
    const g = agruparPinos([
      { numero: 3, x: 100, y: 100 }, { numero: 1, x: 10, y: 10 }, { numero: 2, x: 14, y: 12 }, { numero: 4, x: 100, y: 100 },
    ], 18)
    expect(g).toEqual([{ x: 10, y: 10, numeros: [1, 2] }, { x: 100, y: 100, numeros: [3, 4] }])
    expect(g.flatMap((x) => x.numeros).sort()).toEqual([1, 2, 3, 4])
  })
  it('é determinístico (a ordem de entrada não muda os grupos)', () => {
    const pinos = Array.from({ length: 300 }, (_, i) => ({ numero: i + 1, x: (i * 37) % 500, y: (i * 53) % 400 }))
    const a = agruparPinos(pinos, 18)
    const b = agruparPinos([...pinos].reverse(), 18)
    expect(b).toEqual(a)
    expect(a.flatMap((x) => x.numeros).length).toBe(300)
  })
  it('coordenadas iguais formam um grupo com todos os números', () => {
    expect(agruparPinos([{ numero: 7, x: 5, y: 5 }, { numero: 9, x: 5, y: 5 }, { numero: 8, x: 5, y: 5 }], 18)).toEqual([{ x: 5, y: 5, numeros: [7, 8, 9] }])
  })
})

describe('primeiroPNG', () => {
  it('escolhe a folha geral (A30) e aceita as cartas antigas (A27)', () => {
    expect(primeiroPNG(['carta.pdf', 'carta-02-setor-A.png', 'carta-01-geral.png'])).toBe('carta-01-geral.png')
    expect(primeiroPNG(['carta.pdf', 'carta.png'])).toBe('carta.png')
  })
})
