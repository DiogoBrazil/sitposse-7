import { calcular, montarBase } from './estatisticas'
// Dados fictícios para revisar o visual no navegador (`npm run demo`). Só entram com
// `vite --mode demo` (alias em vite.config.ts); o build de produção usa os bindings reais.
/* eslint-disable */
const espera = <T>(v: T) => new Promise<T>((r) => setTimeout(() => r(v), 60))
const H = 3600_000
const t0 = Date.UTC(2026, 10, 14, 13, 0)
const equipes = [
  { uuid: 'e1', nome: 'Bravo', responsavel: 'Sgt. Ribeiro', identificacao_area: 'Linha C-25, Rio Branco' },
  { uuid: 'e2', nome: 'Charlie', responsavel: 'Cb. Almeida', identificacao_area: 'Linha C-28' },
  { uuid: 'e3', nome: 'Delta', responsavel: 'Sd. Nogueira', identificacao_area: null },
]
const nomes = ['Sítio Boa Esperança', 'Chácara Lua de Prata', 'Fazenda Jaraguá', 'Sítio Ipê Amarelo', 'Chácara São José', 'Sítio Três Irmãos', 'Fazendinha Recanto Verde', 'Sítio Água Boa', 'Chácara Bela Vista', 'Sítio Santa Luzia', 'Sítio do Igarapé', 'Chácara Nova Aliança', 'Sítio Castanheira', 'Fazenda Primavera']
const resp = ['Pedro Pereira', 'João Souza', 'Maria Aparecida Lima', null, 'Antônio Carlos', 'Francisca Gomes', 'José Ribamar', 'Luzia Santos', 'Raimundo Nonato', 'Ana Paula Ferreira', 'Sebastião Rocha', null, 'Edinaldo Matos', 'Rosa Maria Dias']
function rnd(i: number, m: number) { return Math.abs(Math.sin(i * 9301 + m * 49297) * 233280) % 1 }
const ints = (i: number, m: number, max: number) => Math.floor(rnd(i, m) * (max + 1))
const foto = (cor: string) => 'data:image/svg+xml;utf8,' + encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" width="800" height="600"><defs><linearGradient id="g" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#9cc6e8"/><stop offset=".55" stop-color="#d8e8c8"/><stop offset=".56" stop-color="${cor}"/><stop offset="1" stop-color="#5b7a3a"/></linearGradient></defs><rect width="800" height="600" fill="url(#g)"/><rect x="420" y="250" width="200" height="110" fill="#c98b5a"/><polygon points="400,255 520,185 640,255" fill="#8a4b2a"/></svg>`)
const propriedades = nomes.map((nome, i) => {
  const e = equipes[i % 3]
  const semArea = i % 5 === 3
  const semGps = i % 6 === 4
  const ad = 1 + ints(i, 1, 4), jo = ints(i, 2, 3), cr = ints(i, 3, 4)
  const itens = [
    { uuid: `i${i}a`, categoria: 'eletrodomestico', codigo: 'eletro.geladeira', nome: 'Geladeira', outros: false, quantidade: 1 },
    { uuid: `i${i}b`, categoria: 'eletrodomestico', codigo: 'eletro.fogao', nome: 'Fogão', outros: false, quantidade: 1 + ints(i, 4, 1) },
    { uuid: `i${i}c`, categoria: 'eletrodomestico', codigo: 'eletro.tv', nome: 'Televisor', outros: false, quantidade: ints(i, 5, 2) },
    { uuid: `i${i}d`, categoria: 'movel', codigo: 'movel.sofa', nome: 'Sofá', outros: false, quantidade: ints(i, 6, 2) },
    { uuid: `i${i}e`, categoria: 'movel', codigo: 'movel.mesa', nome: 'Mesa de jantar', outros: false, quantidade: 1 },
    { uuid: `i${i}f`, categoria: 'ave', codigo: 'ave.galinha', nome: 'Galinhas', outros: false, quantidade: ints(i, 7, 40) },
    ...(i % 4 === 0 ? [{ uuid: `i${i}g`, categoria: 'ave', codigo: 'ave.outros', nome: 'Codornas', outros: true, quantidade: ints(i, 8, 30) }] : []),
  ].filter((x) => x.quantidade > 0)
  return {
    uuid: `p${i}`, nome, responsavel: resp[i], equipe_uuid: e.uuid, equipe: e.nome, equipe_area: e.identificacao_area,
    pessoas_adultos: ad, pessoas_jovens: jo, pessoas_criancas: cr,
    veic_caminhoes: ints(i, 9, 1), veic_automoveis_utilitarios: ints(i, 10, 2), veic_motos_ciclomotores: ints(i, 11, 3),
    bov_adultos: ints(i, 12, 60), bov_bezerros: ints(i, 13, 20), equ_adultos: ints(i, 14, 4), equ_potros: ints(i, 15, 2),
    area_milesimos: semArea ? null : 1000 + ints(i, 16, 90000), area_unidade: semArea ? null : i % 2 ? 'alq_paulista' : 'ha',
    area_ha: semArea ? null : (5 + rnd(i, 17) * 120).toFixed(2).replace('.', ','),
    lat: semGps ? null : -10.43 - rnd(i, 18) / 10, lon: semGps ? null : -62.47 - rnd(i, 19) / 10,
    gps_origem: semGps ? null : i % 7 === 2 ? 'manual' : 'gnss', gps_precisao_m: semGps ? null : 4 + rnd(i, 20) * 12, gps_capturado_em: t0 + i * H, gps_satelites: semGps ? null : 8 + ints(i, 21, 10),
    criado_em: t0 + i * H, atualizado_em: t0 + i * H + 600_000, recebido_em: t0 + 6 * H + (i % 3) * H, rev: 1 + (i % 3),
    itens, fotos: Array.from({ length: i % 4 }, (_, k) => ({ uuid: `f${i}${k}`, url: foto(['#7aa04b', '#8fb05a', '#a3b46a'][k % 3]), largura: 2560, altura: 1920, tamanho_bytes: 800_000, capturada_em: t0 + i * H + k * 60_000 })),
  }
})
const linha = (p: (typeof propriedades)[number]) => ({
  uuid: p.uuid, nome: p.nome, responsavel: p.responsavel, equipe_uuid: p.equipe_uuid, equipe: p.equipe,
  pessoas: p.pessoas_adultos + p.pessoas_jovens + p.pessoas_criancas, bovinos: p.bov_adultos + p.bov_bezerros,
  area_ha: p.area_ha, tem_gps: p.lat != null, fotos: p.fotos.length, atualizado_em: p.atualizado_em, recebido_em: p.recebido_em,
})
const sem = (s: string) => s.normalize('NFD').replace(/\p{M}/gu, '').toLowerCase()
const somaCat = (c: string) => propriedades.reduce((s, p) => s + p.itens.filter((i) => i.categoria === c).reduce((t, i) => t + i.quantidade, 0), 0)
const somaHa = propriedades.reduce((s, p) => s + (p.area_ha ? Number(p.area_ha.replace(',', '.')) : 0), 0)
let pedidos: any[] = []
const baseEst = montarBase(propriedades)

export const ConsultaService = {
  Estatisticas: (f: any) => espera(calcular(baseEst, f)),
  Resumo: () => espera({
    equipes: 3, aparelhos: 4, propriedades: propriedades.length,
    pessoas: propriedades.reduce((s, p) => s + p.pessoas_adultos + p.pessoas_jovens + p.pessoas_criancas, 0),
    veiculos: propriedades.reduce((s, p) => s + p.veic_caminhoes + p.veic_automoveis_utilitarios + p.veic_motos_ciclomotores, 0),
    bovinos: propriedades.reduce((s, p) => s + p.bov_adultos + p.bov_bezerros, 0),
    equinos: propriedades.reduce((s, p) => s + p.equ_adultos + p.equ_potros, 0),
    aves: somaCat('ave'), eletrodomesticos: somaCat('eletrodomestico'), moveis: somaCat('movel'),
    fotos: propriedades.reduce((s, p) => s + p.fotos.length, 0),
    area_ha: somaHa.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 }),
    com_area: propriedades.filter((p) => p.area_ha).length, sem_area: propriedades.filter((p) => !p.area_ha).length,
    com_gps: propriedades.filter((p) => p.lat != null).length, lotes_importados: 6, ultima_importacao: t0 + 8 * H,
  }),
  Propriedades: (busca: string, equipe: string) => espera(propriedades
    .filter((p) => (!equipe || p.equipe_uuid === equipe) && (!busca || sem(`${p.nome} ${p.responsavel ?? ''}`).includes(sem(busca))))
    .map(linha)),
  Propriedade: (uuid: string) => espera(propriedades.find((p) => p.uuid === uuid) ?? null),
  Equipes: () => espera(equipes.map((e, i) => ({ ...e, aparelhos: i === 0 ? 2 : 1, propriedades: propriedades.filter((p) => p.equipe_uuid === e.uuid).length, ultima_importacao: t0 + (6 + i) * H }))),
}
export const OperacaoService = {
  Ativa: () => espera({ uuid: 'o1', nome: 'Levantamento Linha C-25', descricao: '', status: 'ativa', criada_em: t0 }),
  Criar: (nome: string) => espera({ uuid: 'o1', nome, descricao: '', status: 'ativa', criada_em: t0 }),
}
export const SincronizacaoService = {
  Info: () => espera({ versao: '0.2.0-demo', central_id: 'fa79dbdb-4663-43cb-93c7-d90d7e549b20', pasta_dados: 'C:\\Users\\operador\\AppData\\Local\\SITPOSSE-7', ips: ['192.168.137.1', '192.168.0.24'], porta: 38807, pin: 'msApM8wO8uNTqXyvekNN1gf0iPjSEBVuS7ZlNu1EeFY', servidor_ativo: true, erro_servidor: '', versao_esquema: 2, fotos_sem_arquivo: [] }),
  Pendentes: () => espera(location.hash === '#pedido' ? [{ id: 'x', equipe_uuid: 'e1', equipe_nome: 'Bravo', responsavel: 'Sgt. Ribeiro', identificacao_area: 'Linha C-25', rotulo_aparelho: 'motorola moto g56 5G', instalacao_uuid: 'a', equipe_conhecida: true, recebido_em: Date.now() }] : pedidos),
  Decidir: () => espera(undefined),
  GerarQRPareamento: () => espera({ payload: { v: 1, central_id: 'fa79dbdb', ips: ['192.168.137.1'], porta: 38807, spki_sha256: 'msApM8wO8uNTqXyvekNN1gf0iPjSEBVuS7ZlNu1EeFY', token: 'demo-token' }, expira_em: Date.now() + 600_000 }),
  Lotes: () => espera(Array.from({ length: 6 }, (_, i) => ({ uuid: `l${i}`, equipe: equipes[i % 3].nome, aparelho: i % 2 ? 'motorola moto g56 5G' : 'Samsung SM-A155M', status: i === 0 ? 'recebendo' : 'importado', recebido_em: t0 + (8 - i) * H, importado_em: i === 0 ? null : t0 + (8 - i) * H + 4000, aplicados: 4 + i * 3, duplicados: i % 3, rejeitados: i === 2 ? 1 : 0, conflitos: 0 }))),
  Rejeicoes: () => espera([{ entidade: 'item', uuid: '3f2a9c1e-0000-4000-8000-000000000001', motivo: 'quantidade negativa' }]),
}
