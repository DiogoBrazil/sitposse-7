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
export const RelatorioService = {
  Exportar: (formato: string) => new Promise<string>((r) => setTimeout(() => r(`C:\\Users\\operador\\Documents\\SITPOSSE-7 - Panorama - Levantamento Linha C-25.${formato}`), 700)),
  ExportarFicha: (formato: string) => new Promise<string>((r) => setTimeout(() => r(`C:\\Users\\operador\\Documents\\SITPOSSE-7 - Ficha.${formato}`), 500)),
  Abrir: () => espera(undefined),
  MostrarNaPasta: () => espera(undefined),
}

// ---------- cartas-imagem (demonstração: sem Copernicus) ----------
const RM = 6378137
const merc = (lat: number, lon: number) => ({ x: RM * lon * Math.PI / 180, y: RM * Math.log(Math.tan(Math.PI / 4 + lat * Math.PI / 360)) })
const graus = (x: number, y: number) => ({ lat: (2 * Math.atan(Math.exp(y / RM)) - Math.PI / 2) * 180 / Math.PI, lon: x / RM * 180 / Math.PI })
function quadro(r: { min_x: number; min_y: number; max_x: number; max_y: number }) {
  const cx = (r.min_x + r.max_x) / 2, cy = (r.min_y + r.max_y) / 2
  let w = r.max_x - r.min_x, h = r.max_y - r.min_y
  if (w / h > 4 / 3) h = w * 3 / 4; else w = h * 4 / 3
  return { min_x: cx - w / 2, max_x: cx + w / 2, min_y: cy - h / 2, max_y: cy + h / 2 }
}
function satelite(): string {
  const c = document.createElement('canvas'); c.width = 800; c.height = 600
  const g = c.getContext('2d')!
  for (let y = 0; y < 600; y += 6) for (let x = 0; x < 800; x += 6) {
    const v = 70 + 40 * Math.sin(x / 41) * Math.cos(y / 29) + 25 * Math.sin((x + y) / 13)
    g.fillStyle = `rgb(${v * 0.6 | 0},${v + 25 | 0},${v * 0.4 | 0})`; g.fillRect(x, y, 6, 6)
  }
  g.strokeStyle = '#b98d5c'; g.lineWidth = 7; g.beginPath(); g.moveTo(0, 320); g.bezierCurveTo(250, 260, 500, 380, 800, 300); g.stroke()
  return c.toDataURL('image/png')
}
function planoDe(ps: any[]) {
  const com = ps.filter((p) => p.lat != null)
  const pts = com.map((p) => merc(p.lat, p.lon))
  const m = 1500
  const r = quadro({ min_x: Math.min(...pts.map((p) => p.x)) - m, max_x: Math.max(...pts.map((p) => p.x)) + m, min_y: Math.min(...pts.map((p) => p.y)) - m, max_y: Math.max(...pts.map((p) => p.y)) + m })
  return {
    marcadores: com.map((p, i) => ({ numero: i + 1, nome: p.nome, ponto: { lat: p.lat, lon: p.lon }, precisao_m: p.gps_precisao_m, origem: p.gps_origem, equipe: p.equipe, area_identificada: p.equipe_area ?? '' })),
    uuids: com.map((p) => p.uuid), sem_localizacao: ps.filter((p) => p.lat == null).map((p) => p.nome),
    retangulo: r, so: graus(r.min_x, r.min_y), ne: graus(r.max_x, r.max_y), extensao_km: (r.max_x - r.min_x) / 1000 * Math.cos(10.4 * Math.PI / 180), resolucao_m: 6, grupos: [], largura: 2000, altura: 1500,
  }
}
let imagemDemo = ''
export const CartaService = {
  Estado: () => espera({ configurado: true, client_id: 'sh-1•••••••a9f2', provedor: 'Copernicus', uso: { imagens: 3, bytes_imagens: 14_200_000, cartas: 1, bytes_cartas: 2_100_000 } }),
  PlanejarGeral: (f: any) => espera(planoDe(propriedades.filter((p) => (!f.equipes?.length || f.equipes.includes(p.equipe_uuid))))),
  PlanejarIndividual: (u: string) => espera(planoDe(propriedades.filter((p) => p.uuid === u))),
  EnquadrarGrupo: (pts: any[]) => espera(quadro({ min_x: 0, min_y: 0, max_x: 1, max_y: 1, ...pts })),
  Passagens: () => espera({ passagens: [
    { data: '2026-11-07T00:00:00Z', nuvens: 64.2, tiles: 2 }, { data: '2026-11-02T00:00:00Z', nuvens: 38.5, tiles: 2 },
    { data: '2026-10-28T00:00:00Z', nuvens: 4.1, tiles: 2 }, { data: '2026-10-13T00:00:00Z', nuvens: 12.7, tiles: 1 }, { data: '2026-09-18T00:00:00Z', nuvens: 0.8, tiles: 2 },
  ], sugerida: '2026-10-28T00:00:00Z' }),
  Imagem: (t: any) => new Promise((ok) => setTimeout(() => {
    imagemDemo ||= satelite()
    const r = quadro(t.retangulo)
    ok({ url: imagemDemo, retangulo: r, so: graus(r.min_x, r.min_y), ne: graus(r.max_x, r.max_y), periodo: t.data ? 'Aquisição de 28/10/2026' : 'Mosaico de aquisições', sem_dados: 0, guardada: false })
  }, 500)),
  GerarCarta: () => new Promise((ok) => setTimeout(() => ok({ id: '20261114-221500-abcd1234', titulo: 'Carta-imagem de localização das propriedades — 12 propriedade(s)', tipo: 'geral', propriedades: 12, periodo: 'Aquisição de 28/10/2026', criterios: [], gerada_em: new Date().toISOString(), arquivos: ['carta.pdf', 'carta.png'] }), 900)),
  Cartas: () => espera([{ id: '20261114-201000-abcd1234', titulo: 'Carta-imagem de localização das propriedades — Fazenda Jaraguá', tipo: 'individual', propriedades: 1, periodo: 'Aquisição de 28/10/2026', criterios: [], gerada_em: '2026-11-14T20:10:00Z', arquivos: ['carta.pdf', 'carta.png'] }]),
  AbrirCarta: () => espera(undefined), MostrarCarta: () => espera(undefined), SalvarCopia: () => espera(''),
  SalvarCredenciais: () => espera(undefined), TestarConexao: () => espera(undefined), RemoverCredenciais: () => espera(undefined), LimparImagens: () => espera(3),
}
