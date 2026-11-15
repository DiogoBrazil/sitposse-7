// Só para `npm run demo`: versão simplificada do cálculo do Go (internal/estatisticas) sobre os
// dados fictícios, para revisar a tela no navegador. A regra oficial e testada é a do Go.
/* eslint-disable @typescript-eslint/no-explicit-any */
const sem = (s: string) => s.normalize('NFD').replace(/\p{M}/gu, '').toLowerCase().trim()
const fmtHa = (dm2: number) => { const c = Math.floor((dm2 + 5000) / 10000); return `${Math.floor(c / 100)},${String(c % 100).padStart(2, '0')}` }
const FIXOS: [string, string, string][] = [
  ['pessoas_total', 'Pessoas (total)', 'Pessoas'], ['pessoas_adultos', 'Adultos (18+)', 'Pessoas'], ['pessoas_jovens', 'Jovens (12–17)', 'Pessoas'], ['pessoas_criancas', 'Crianças (0–11)', 'Pessoas'],
  ['veic_total', 'Veículos (total)', 'Veículos'], ['veic_caminhoes', 'Caminhões', 'Veículos'], ['veic_automoveis_utilitarios', 'Automóveis, caminhonetes e utilitários', 'Veículos'], ['veic_motos_ciclomotores', 'Motocicletas e ciclomotores', 'Veículos'],
  ['bov_total', 'Bovinos (total)', 'Bovinos'], ['bov_adultos', 'Bovinos adultos', 'Bovinos'], ['bov_bezerros', 'Bezerros e bezerras', 'Bovinos'],
  ['equ_total', 'Equinos (total)', 'Equinos'], ['equ_adultos', 'Equinos adultos', 'Equinos'], ['equ_potros', 'Potros', 'Equinos'],
  ['aves_total', 'Aves (total)', 'Aves'], ['eletro_total', 'Eletrodomésticos (total)', 'Eletrodomésticos'], ['moveis_total', 'Móveis (total)', 'Móveis'], ['fotos', 'Fotos', 'Fotos'],
]
const FAIXAS: [string, string, number, number][] = [['ate10', 'Até 10 ha', 0, 10e6], ['10a25', 'De 10 a 25 ha', 10e6, 25e6], ['25a50', 'De 25 a 50 ha', 25e6, 50e6], ['50a100', 'De 50 a 100 ha', 50e6, 100e6], ['acima100', '100 ha ou mais', 100e6, Infinity]]
const faixaDe = (d: number | null) => (d == null ? 'sem_area' : FAIXAS.find((f) => d >= f[2] && d < f[3])![0])
const grupoCat: Record<string, string> = { ave: 'Aves', eletrodomestico: 'Eletrodomésticos', movel: 'Móveis' }
const totalCat: Record<string, string> = { ave: 'aves_total', eletrodomestico: 'eletro_total', movel: 'moveis_total' }

export const ESTUDOS = [{ uuid: 's1', nome: 'Fazenda Santa Maria' }, { uuid: 's2', nome: 'Gleba Rio Preto' }]

export function montarBase(props: any[]) {
  const itens = new Map<string, any>()
  const familias = props.map((p) => {
    const val: Record<string, number> = {
      pessoas_adultos: p.pessoas_adultos, pessoas_jovens: p.pessoas_jovens, pessoas_criancas: p.pessoas_criancas,
      veic_caminhoes: p.veic_caminhoes, veic_automoveis_utilitarios: p.veic_automoveis_utilitarios, veic_motos_ciclomotores: p.veic_motos_ciclomotores,
      bov_adultos: p.bov_adultos, bov_bezerros: p.bov_bezerros, equ_adultos: p.equ_adultos, equ_potros: p.equ_potros, fotos: p.fotos.length,
      aves_total: 0, eletro_total: 0, moveis_total: 0,
    }
    val.pessoas_total = val.pessoas_adultos + val.pessoas_jovens + val.pessoas_criancas
    val.veic_total = val.veic_caminhoes + val.veic_automoveis_utilitarios + val.veic_motos_ciclomotores
    val.bov_total = val.bov_adultos + val.bov_bezerros
    val.equ_total = val.equ_adultos + val.equ_potros
    for (const i of p.itens) {
      const chave = i.outros ? `item:${i.codigo}:${sem(i.nome)}` : `item:${i.codigo}`
      if (!itens.has(chave)) itens.set(chave, { chave, rotulo: i.nome, grupo: grupoCat[i.categoria], categoria: i.categoria, outros: i.outros })
      val[chave] = (val[chave] ?? 0) + i.quantidade
      val[totalCat[i.categoria]] += i.quantidade
    }
    const dm2 = p.area_ha ? Math.round(Number(p.area_ha.replace(',', '.')) * 1e6) : null
    return { uuid: p.uuid, nome: p.nome, responsavel: p.responsavel, observacoes: p.observacoes, equipe_uuid: p.equipe_uuid, equipe: p.equipe, equipe_codigo: p.equipe_codigo,
      jornada_uuid: p.jornada_uuid, jornada: p.jornada, jornada_numero: p.jornada_numero, jornada_data: p.jornada_data, estudo_uuid: p.estudo_uuid, estudo: p.estudo,
      valores: val, area_dm2: dm2, area_ha: p.area_ha, area_unidade: p.area_unidade, com_localizacao: p.lat != null, criado_em: p.criado_em, recebido_em: p.recebido_em }
  })
  return { familias, itens: [...itens.values()] }
}

function passa(p: any, f: any): boolean {
  const termo = sem(f.busca ?? '')
  if (f.equipes?.length && !f.equipes.includes(p.equipe_uuid)) return false
  if (f.estudo && p.estudo_uuid !== f.estudo) return false
  if (f.jornadas?.length && !f.jornadas.includes(p.jornada_uuid)) return false
  if (termo && !sem(p.nome).includes(termo) && !sem(p.responsavel ?? '').includes(termo)) return false
  const t = f.periodo_campo === 'recebimento' ? p.recebido_em : p.criado_em
  if ((f.de != null && t < f.de) || (f.ate != null && t > f.ate)) return false
  if (f.com_localizacao != null && p.com_localizacao !== f.com_localizacao) return false
  if (f.com_foto != null && (p.valores.fotos > 0) !== f.com_foto) return false
  if (f.com_area != null && (p.area_dm2 != null) !== f.com_area) return false
  if (f.faixas_area?.length && !f.faixas_area.includes(faixaDe(p.area_dm2))) return false
  if (f.area_min_ha != null || f.area_max_ha != null) {
    if (p.area_dm2 == null) return false
    if ((f.area_min_ha != null && p.area_dm2 < f.area_min_ha * 1e6) || (f.area_max_ha != null && p.area_dm2 > f.area_max_ha * 1e6)) return false
  }
  for (const c of f.condicoes ?? []) {
    const x = p.valores[c.campo] ?? 0
    const ok = { tem: x > 0, nao_tem: x === 0, maior: x > c.valor, pelo_menos: x >= c.valor, menor: x < c.valor, igual: x === c.valor }[c.operador as string]
    if (!ok) return false
  }
  return true
}

function totais(fs: any[]) {
  const t: any = { familias: 0, valores: {}, area_dm2: 0, com_area: 0, sem_area: 0, com_localizacao: 0, sem_localizacao: 0, com_foto: 0, sem_foto: 0 }
  for (const p of fs) {
    t.familias++
    for (const [k, x] of Object.entries(p.valores)) t.valores[k] = (t.valores[k] ?? 0) + (x as number)
    if (p.area_dm2 != null) { t.com_area++; t.area_dm2 += p.area_dm2 } else t.sem_area++
    if (p.com_localizacao) t.com_localizacao++; else t.sem_localizacao++
    if (p.valores.fotos > 0) t.com_foto++; else t.sem_foto++
  }
  t.area_ha = fmtHa(t.area_dm2)
  t.area_media_ha = t.com_area ? fmtHa(Math.round(t.area_dm2 / t.com_area)) : null
  return t
}

export function calcular(base: ReturnType<typeof montarBase>, f: any) {
  const doEstudo = base.familias.filter((p) => !f.estudo || p.estudo_uuid === f.estudo)
  const fs = doEstudo.filter((p) => passa(p, f))
  const agrupar = (chave: (p: any) => string, nome: (p: any) => string) => {
    const m = new Map<string, any[]>()
    for (const p of fs) { const k = chave(p); if (!m.has(k)) m.set(k, []); m.get(k)!.push(p) }
    return [...m.entries()].map(([k, ps]) => ({ chave: k, nome: nome(ps[0]), totais: totais(ps) })).sort((a, b) => a.nome.localeCompare(b.nome, 'pt-BR'))
  }
  const rotEq = (p: any) => `${p.equipe} · J${p.jornada_numero} (${p.jornada_data.split('-').reverse().join('/')})`
  const eqs = new Map(doEstudo.map((p) => [p.equipe_uuid, rotEq(p)]))
  const jors = new Map(doEstudo.map((p) => [p.jornada_uuid, p.jornada]))
  return {
    familias_base: doEstudo.length, totais: totais(fs), familias: fs,
    por_equipe: agrupar((p) => p.equipe_uuid, rotEq),
    por_jornada: agrupar((p) => p.jornada_uuid, (p) => p.jornada),
    itens: base.itens.map((c) => ({ campo: c, total: fs.reduce((s, p) => s + (p.valores[c.chave] ?? 0), 0), familias: fs.filter((p) => (p.valores[c.chave] ?? 0) > 0).length })),
    faixas_area: [...FAIXAS.map(([k, r]) => { const ps = fs.filter((p) => faixaDe(p.area_dm2) === k); const d = ps.reduce((s, p) => s + (p.area_dm2 ?? 0), 0); return { chave: k, rotulo: r, familias: ps.length, area_dm2: d, area_ha: fmtHa(d) } }),
      { chave: 'sem_area', rotulo: 'Sem área informada', familias: fs.filter((p) => p.area_dm2 == null).length, area_dm2: 0, area_ha: '0,00' }],
    campos: [...FIXOS.map(([chave, rotulo, grupo]) => ({ chave, rotulo, grupo })), ...base.itens],
    equipes: [...eqs.entries()].map(([valor, rotulo]) => ({ valor, rotulo })),
    jornadas: [...jors.entries()].map(([valor, rotulo]) => ({ valor, rotulo })),
    estudos: ESTUDOS.map((e) => ({ valor: e.uuid, rotulo: e.nome })),
  }
}
