import { calcular, ESTUDOS, montarBase } from './estatisticas'
// Dados fictícios para revisar o visual no navegador (`npm run demo`). Só entram com
// `vite --mode demo` (alias em vite.config.ts); o build de produção usa os bindings reais.
/* eslint-disable */
const espera = <T>(v: T) => new Promise<T>((r) => setTimeout(() => r(v), 60))
const H = 3600_000
const t0 = Date.UTC(2026, 10, 14, 13, 0)
// Estudo "Fazenda Santa Maria": jornada 1 (Alfa, Bravo) e jornada 2 (Alfa, Charlie). Outro estudo vazio.
const equipes = [
  { uuid: 'e1', nome: 'Equipe Alfa', codigo: 'EQ-7K3M-9QX1', jornada: 'j1' },
  { uuid: 'e2', nome: 'Equipe Bravo', codigo: 'EQ-4HZ2-RTB5', jornada: 'j1' },
  { uuid: 'e3', nome: 'Equipe Alfa', codigo: 'EQ-0A1B-0C1X', jornada: 'j2' },
  { uuid: 'e4', nome: 'Equipe Charlie', codigo: 'EQ-10Q0-K1MX', jornada: 'j2' },
]
const jornadas: Record<string, { numero: number; data: string }> = { j1: { numero: 1, data: '2026-10-10' }, j2: { numero: 2, data: '2026-10-11' } }
const nomes = ['Sítio Boa Esperança', 'Chácara Lua de Prata', 'Fazenda Jaraguá', 'Sítio Ipê Amarelo', 'Chácara São José', 'Sítio Três Irmãos', 'Fazendinha Recanto Verde', 'Sítio Água Boa', 'Chácara Bela Vista', 'Sítio Santa Luzia', 'Sítio do Igarapé', 'Chácara Nova Aliança', 'Sítio Castanheira', 'Fazenda Primavera']
const resp = ['Pedro Pereira', 'João Souza', 'Maria Aparecida Lima', 'Joaquim Barros', 'Antônio Carlos', 'Francisca Gomes', 'José Ribamar', 'Luzia Santos', 'Raimundo Nonato', 'Ana Paula Ferreira', 'Sebastião Rocha', 'Benedita Alves', 'Edinaldo Matos', 'Rosa Maria Dias']
function rnd(i: number, m: number) { return Math.abs(Math.sin(i * 9301 + m * 49297) * 233280) % 1 }
const ints = (i: number, m: number, max: number) => Math.floor(rnd(i, m) * (max + 1))
const foto = (cor: string) => 'data:image/svg+xml;utf8,' + encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" width="800" height="600"><defs><linearGradient id="g" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#9cc6e8"/><stop offset=".55" stop-color="#d8e8c8"/><stop offset=".56" stop-color="${cor}"/><stop offset="1" stop-color="#5b7a3a"/></linearGradient></defs><rect width="800" height="600" fill="url(#g)"/><rect x="420" y="250" width="200" height="110" fill="#c98b5a"/><polygon points="400,255 520,185 640,255" fill="#8a4b2a"/></svg>`)
const propriedades = nomes.map((nome, i) => {
  const e = equipes[i % 4]
  const j = jornadas[e.jornada]
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
    uuid: `p${i}`, nome, responsavel: resp[i], equipe_uuid: e.uuid, equipe: e.nome, equipe_codigo: e.codigo,
    observacoes: i % 5 === 0 ? 'Família pede prazo para retirar o gado.\nHá um idoso acamado na casa principal.' : null,
    jornada_uuid: e.jornada, jornada_numero: j.numero, jornada_data: j.data, jornada: `Jornada ${j.numero} · ${j.data.split('-').reverse().join('/')}`,
    estudo_uuid: 's1', estudo: 'Fazenda Santa Maria', estudo_codigo: 'SITPOSSE7-A82F-91C6',
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
  DocumentoEdicao:(id:string)=>{const p:any=propriedades.find(p=>p.uuid===id);return espera(JSON.stringify({token:'demo',propriedade:{...p,nome_propriedade:p.nome,responsavel_familia:p.responsavel,area_valor_milesimos:p.area_milesimos,observacoes_relevantes:p.observacoes},itens:p.itens.map((i:any)=>({...i,codigo_catalogo:i.codigo,nome_personalizado:i.outros?i.nome:null})),fotos:p.fotos,catalogo:p.itens.map((i:any)=>({codigo:i.codigo,categoria:i.categoria,nome_exibicao:i.nome}))}))},
  SalvarEdicao:()=>espera(undefined),ExcluirPropriedade:()=>espera(undefined),BaixarFoto:()=>espera('foto.jpg'),
  Estatisticas: (f: any) => espera(calcular(baseEst, f)),
  Resumo: () => espera({
    equipes: 4, aparelhos: 4, propriedades: propriedades.length,
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
  Propriedades: (busca: string, estudo: string, equipe: string) => espera(propriedades
    .filter((p) => (!estudo || p.estudo_uuid === estudo) && (!equipe || p.equipe_uuid === equipe) && (!busca || sem(`${p.nome} ${p.responsavel ?? ''}`).includes(sem(busca))))
    .map(linha)),
  Propriedade: (uuid: string) => espera(propriedades.find((p) => p.uuid === uuid) ?? null),
}
const detalheEstudo = (uuid: string) => {
  const s = ESTUDOS.find((e) => e.uuid === uuid)!
  const doEstudo = uuid === 's1' ? propriedades : []
  const lista_jornadas = uuid === 's1' ? Object.entries(jornadas).map(([ju, j]) => ({
    uuid: ju, estudo_uuid: uuid, numero: j.numero, data: j.data, familias: doEstudo.filter((p) => p.jornada_uuid === ju).length,
    equipes: equipes.filter((e) => e.jornada === ju).map((e) => ({ uuid: e.uuid, jornada_uuid: ju, nome: e.nome, codigo: e.codigo.replace(/-/g, ''), codigo_exibicao: e.codigo,
      coletas: 1, familias: doEstudo.filter((p) => p.equipe_uuid === e.uuid).length, ultima_importacao: t0 + 6 * H })),
  })) : [{ uuid: 'j9', estudo_uuid: uuid, numero: 1, data: '2026-11-20', familias: 0, equipes: [] }]
  return { uuid, codigo: 'SITPOSSE7A82F91C6', codigo_exibicao: uuid === 's1' ? 'SITPOSSE7-A82F-91C6' : 'SITPOSSE7-3QJT-W8M2', nome: s.nome, criado_em: t0,
    jornadas: lista_jornadas.length, equipes: lista_jornadas.reduce((n, j) => n + j.equipes.length, 0), familias: doEstudo.length,
    primeira_data: lista_jornadas[0].data, ultima_data: lista_jornadas.at(-1)!.data, ultima_importacao: uuid === 's1' ? t0 + 8 * H : null, lista_jornadas }
}
let foco = 's1'
export const EstudoService = {
 Excluir:()=>espera(undefined),
  Listar: () => espera(ESTUDOS.map((e) => { const { lista_jornadas, ...r } = detalheEstudo(e.uuid); return r })),
  Detalhe: (u: string) => espera(detalheEstudo(u)),
  EmFoco: () => espera(foco),
  DefinirEmFoco: (u: string) => { foco = u; return espera(undefined) },
  Criar: (nome: string) => espera({ ...detalheEstudo('s2'), nome }),
  CriarJornada: () => espera(null), AdicionarEquipes: () => espera([]), Renomear: () => espera(undefined),
  RenomearEquipe: () => espera(undefined), ExcluirEquipe: () => espera(undefined), ExcluirJornada: () => espera(undefined), AlterarDataJornada: () => espera(undefined),
}
const destinoDemo = { estudo_uuid: 's1', estudo_codigo: 'SITPOSSE7A82F91C6', estudo_nome: 'Fazenda Santa Maria', jornada_uuid: 'j2', jornada_numero: 2, jornada_data: '2026-10-11', equipe_uuid: 'e3', equipe_nome: 'Equipe Alfa', equipe_codigo: 'EQ0A1B0C1X' }
const aguardandoDemo = () => location.hash === '#destino'
  ? [{ lote_uuid: 'l-demo', instalacao_uuid: 'a', rotulo_aparelho: 'motorola moto g56 5G', coleta_uuid: 'c1', codigo_informado: 'EQ0A1B0C1X', codigo_exibicao: 'EQ-0A1B-0C1X', aguardando_desde: Date.now(), novos: 7, atualizados: 2, excluidos: 0, fotos: 15, destino: destinoDemo, travado: false, destino_travado: null }]
  : location.hash === '#codigo-errado'
    ? [{ lote_uuid: 'l-demo2', instalacao_uuid: 'a', rotulo_aparelho: 'Samsung SM-A155M', coleta_uuid: 'c2', codigo_informado: 'EQ0A1B0C2X', codigo_exibicao: 'EQ-0A1B-0C2X', aguardando_desde: Date.now(), novos: 3, atualizados: 0, excluidos: 0, fotos: 4, destino: null, travado: false, destino_travado: null }]
    : []
export const SincronizacaoService = {
  Aguardando: () => espera(aguardandoDemo()),
  DecidirDestino: () => { location.hash = ''; return espera('importado') },
  Info: () => espera({ bloqueio: location.hash === '#bloqueio' ? 'Este banco de dados é de uma versão de teste anterior da Central (antes dos estudos de situação) e não pode ser aberto por esta versão; nada foi alterado.' : '', versao: '0.6.0-demo', central_id: 'fa79dbdb-4663-43cb-93c7-d90d7e549b20', pasta_dados: 'C:\\Users\\operador\\AppData\\Local\\SITPOSSE-7', ips: ['192.168.137.1', '192.168.0.24'], porta: 38807, pin: 'msApM8wO8uNTqXyvekNN1gf0iPjSEBVuS7ZlNu1EeFY', servidor_ativo: true, erro_servidor: '', versao_esquema: 2, fotos_sem_arquivo: [] }),
  Pendentes: () => espera(location.hash === '#pedido' ? [{ id: 'x', rotulo_aparelho: 'motorola moto g56 5G', instalacao_uuid: 'a7c41e2b-0000-4000-8000-000000000001', aparelho_conhecido: true, recebido_em: Date.now() }] : pedidos),
  Decidir: () => espera(undefined),
  GerarQRPareamento: () => espera({ payload: { v: 2, central_id: 'fa79dbdb', ips: ['192.168.137.1'], porta: 38807, spki_sha256: 'msApM8wO8uNTqXyvekNN1gf0iPjSEBVuS7ZlNu1EeFY', token: 'demo-token' }, expira_em: Date.now() + 600_000 }),
  Lotes: () => espera(Array.from({ length: 6 }, (_, i) => {
    const st = ['aguardando_confirmacao', 'importado', 'importado', 'cancelado', 'importado', 'importado'][i]
    const e = equipes[i % 4]
    return { uuid: `l${i}`, codigo_informado: i === 3 ? 'EQ-0A1B-0C2X' : e.codigo, estudo: st === 'importado' ? 'Fazenda Santa Maria' : null,
      jornada: st === 'importado' ? `Jornada ${jornadas[e.jornada].numero} · ${jornadas[e.jornada].data.split('-').reverse().join('/')}` : null,
      equipe: st === 'importado' ? e.nome : null, equipe_codigo: st === 'importado' ? e.codigo : null, corrigido: i === 2, motivo: st === 'cancelado' ? 'codigo_inexistente' : null,
      aparelho: i % 2 ? 'motorola moto g56 5G' : 'Samsung SM-A155M', status: st, recebido_em: t0 + (8 - i) * H, importado_em: st === 'importado' ? t0 + (8 - i) * H + 4000 : null,
      aplicados: st === 'importado' ? 4 + i * 3 : 0, duplicados: st === 'importado' ? i % 3 : 0, rejeitados: i === 2 ? 1 : 0, conflitos: 0 }
  })),
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
    marcadores: com.map((p, i) => ({ numero: i + 1, uuid: p.uuid, nome: p.nome, responsavel: p.responsavel, ponto: { lat: p.lat, lon: p.lon }, precisao_m: p.gps_precisao_m, origem: p.gps_origem, equipe: p.equipe, jornada: p.jornada })),
    uuids: com.map((p) => p.uuid), sem_localizacao: ps.filter((p) => p.lat == null).map((p) => p.nome),
    suspeitas: [], individual: com.length === 1, retangulo: r, so: graus(r.min_x, r.min_y), ne: graus(r.max_x, r.max_y), extensao_km: (r.max_x - r.min_x) / 1000 * Math.cos(10.4 * Math.PI / 180),
  }
}
let imagemDemo = ''
const tarefasDemo: Record<string, number> = {}
function resultadoDemo(pl: any) {
  imagemDemo ||= satelite()
  const r = pl.retangulo
  const meio = (r.min_x + r.max_x) / 2
  const setores = pl.marcadores.length > 6 ? [
    { rotulo: 'A', so: graus(r.min_x + 300, (r.min_y + r.max_y) / 2 - 200), ne: graus(meio, r.max_y - 300), propriedades: Math.ceil(pl.marcadores.length / 2) },
    { rotulo: 'B', so: graus(meio, r.min_y + 300), ne: graus(r.max_x - 300, (r.min_y + r.max_y) / 2 + 200), propriedades: Math.floor(pl.marcadores.length / 2) },
  ] : []
  return {
    composicao_id: 'demo', tiles: imagemDemo, z_min: 11, z_geral: 14, z_nativo: 16, geral: { rotulo: '', so: pl.so, ne: pl.ne, propriedades: pl.marcadores.length }, setores,
    partes: [
      { rotulo: 'CBERS-4A/WPM (INPE)', periodo: '13/10/2026', resolucao_m: 2, fracao: 0.94, nuvem_estimada: true, atribuicao: 'Imagem CBERS-4A/WPM © INPE (CC BY 4.0)', fonte: 'cbers4a-wpm' },
      { rotulo: 'Sentinel-2 L2A (Copernicus/INPE)', periodo: '16/09/2026', resolucao_m: 10, fracao: 0.06, nuvem_estimada: false, atribuicao: 'Contains modified Copernicus Sentinel data 2026', fonte: 'sentinel2-l2a' },
    ],
    mosaico: true, resolucao_m: 2, guardada: false, reserva: false, criada_em: new Date().toISOString(), avisos: [],
  }
}
let ultimoPlano: any = null
export const CartaService = {
  Estado: () => espera({ configurado: false, client_id: '', provedor: 'INPE', uso: { imagens: 3, bytes_imagens: 34_200_000, cartas: 1, bytes_cartas: 2_100_000 } }),
  PlanejarGeral: (f: any) => espera(ultimoPlano = planoDe(propriedades.filter((p) => (!f.equipes?.length || f.equipes.includes(p.equipe_uuid))))),
  PlanejarIndividual: (u: string) => espera(ultimoPlano = planoDe(propriedades.filter((p) => p.uuid === u))),
  Preparar: () => { const id = String(Date.now()); tarefasDemo[id] = Date.now(); return espera(id) },
  Andamento: (id: string) => {
    const t = (Date.now() - tarefasDemo[id]) / 1000
    if (t < 0.8) return espera({ fase: 'Buscando imagens disponíveis', feito: 0, total: 0, bytes_rede: 0, concluida: false, erro: '', sem_conexao: false, resultado: null })
    if (t < 1.8) return espera({ fase: 'Avaliando a qualidade das imagens', feito: Math.floor((t - 0.8) * 12), total: 15, bytes_rede: 12_400_000 * t, concluida: false, erro: '', sem_conexao: false, resultado: null })
    if (t < 2.6) return espera({ fase: 'Preparando a imagem da região', feito: Math.floor((t - 1.8) * 200), total: 186, bytes_rede: 31_000_000, concluida: false, erro: '', sem_conexao: false, resultado: null })
    return espera({ fase: '', feito: 0, total: 0, bytes_rede: 0, concluida: true, erro: '', sem_conexao: false, resultado: resultadoDemo(ultimoPlano) })
  },
  Cancelar: () => espera(undefined),
  GerarCarta: () => new Promise((ok) => setTimeout(() => ok({ id: '20261115-221500-abcd1234', titulo: 'Carta-imagem de localização das propriedades — 12 propriedade(s)', tipo: 'geral', propriedades: 12, periodo: '13/10/2026; 16/09/2026', criterios: [], gerada_em: new Date().toISOString(), arquivos: ['carta-01-geral.png', 'carta-02-setor-A.png', 'carta-03-setor-B.png', 'carta.pdf'] }), 900)),
  Cartas: () => espera([{ id: '20261114-201000-abcd1234', titulo: 'Carta-imagem de localização das propriedades — Fazenda Jaraguá', tipo: 'individual', propriedades: 1, periodo: 'Aquisição de 28/10/2026', criterios: [], gerada_em: '2026-11-14T20:10:00Z', arquivos: ['carta.pdf', 'carta.png'] }]),
  AbrirCarta: () => espera(undefined), MostrarCarta: () => espera(undefined), SalvarCopia: () => espera(''),
  SalvarCredenciais: () => espera(undefined), TestarConexao: () => espera(undefined), RemoverCredenciais: () => espera(undefined), LimparImagens: () => espera(3),
}
