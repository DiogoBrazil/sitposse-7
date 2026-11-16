import { describe, expect, it } from 'vitest'
import { agruparPinos, avisoMudanca, compararPlanos, exigePreparo, primeiroPNG, type PinoMapa } from './cartas'

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

describe('compararPlanos (RN-98)', () => {
  const p = (uuid: string, numero: number, lat: number, lon: number, nome = `Sítio ${numero}`): PinoMapa =>
    ({ uuid, numero, nome, responsavel: 'Fulano', lat, lon })
  const antes = [p('a', 1, -10.4, -62.5), p('b', 2, -10.42, -62.47), p('c', 3, -10.43, -62.46)]

  it('nada mudou: nada a fazer', () => {
    const m = compararPlanos(antes, [...antes].reverse())
    expect(m).toEqual({ entraram: 0, sairam: 0, movidas: [], textos: false })
    expect(exigePreparo(m)).toBe(false)
    expect(avisoMudanca(m)).toBe('')
  })
  it('só o nome ou o responsável mudou: redesenha os pinos sem preparar a imagem', () => {
    const m = compararPlanos(antes, [p('a', 1, -10.4, -62.5, 'Novo nome'), antes[1], antes[2]])
    expect(m.textos).toBe(true)
    expect(exigePreparo(m)).toBe(false)
  })
  it('coordenada alterada: exige preparo e cita o número', () => {
    const m = compararPlanos(antes, [antes[0], p('b', 2, -10.425, -62.47), antes[2]])
    expect(m.movidas).toEqual([2])
    expect(exigePreparo(m)).toBe(true)
    expect(avisoMudanca(m)).toBe('Desde a preparação da imagem, a localização da propriedade nº 2 mudou.')
  })
  it('propriedades que entram ou saem (ganhou ou perdeu coordenada) também exigem preparo', () => {
    const m = compararPlanos(antes, [antes[0], p('b', 2, -10.41, -62.47), p('d', 4, -10.44, -62.45)])
    expect(m).toEqual({ entraram: 1, sairam: 1, movidas: [2], textos: false })
    expect(avisoMudanca(m)).toBe('Desde a preparação da imagem, a localização da propriedade nº 2 mudou; 1 propriedade entrou na carta; 1 propriedade saiu da carta.')
  })
  it('lista no máximo 8 números', () => {
    const muitos = Array.from({ length: 10 }, (_, i) => p(`u${i}`, i + 1, -10, -62))
    const m = compararPlanos(muitos, muitos.map((x) => ({ ...x, lat: -10.01 })))
    expect(avisoMudanca(m)).toBe('Desde a preparação da imagem, a localização de 10 propriedades mudou (nº 1, nº 2, nº 3, nº 4, nº 5, nº 6, nº 7, nº 8…).')
  })
})
