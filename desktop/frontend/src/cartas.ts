// Tipos e funções da tela Cartas-imagem (A27, ADR-0022; A30, ADR-0027).

/** Pino de uma propriedade: o número é o fixo do estudo (RN-96). */
export interface PinoMapa { numero: number; uuid: string; nome: string; responsavel: string; lat: number; lon: number }
export interface Retangulo3857 { min_x: number; min_y: number; max_x: number; max_y: number }

/** Mensagem da RN-91 (sem conexão): igual à do Go (cartografia.ErrSemConexao). */
export const MSG_SEM_CONEXAO = 'O mapa e a carta-imagem precisam de conexão com a internet. Seus cadastros, coordenadas, estatísticas e filtros continuam disponíveis offline. Conecte-se e tente novamente.'

export const ehSemConexao = (msg: string) => msg.startsWith('O mapa e a carta-imagem precisam de conexão')

/** "28/10/2026" a partir da data UTC do catálogo (o dia da passagem, sem fuso). */
export function diaUTC(iso: string): string {
  const d = new Date(iso)
  return `${String(d.getUTCDate()).padStart(2, '0')}/${String(d.getUTCMonth() + 1).padStart(2, '0')}/${d.getUTCFullYear()}`
}

export function megabytes(b: number): string {
  return new Intl.NumberFormat('pt-BR', { maximumFractionDigits: 1 }).format(b / 1024 / 1024) + ' MB'
}

/** Pino com a posição na tela (pixels do contêiner do mapa). */
export interface PinoTela { numero: number; x: number; y: number }

/** Grupo de pinos que se sobrepõem na tela: mostrado como um selo com a contagem. */
export interface GrupoPinos { x: number; y: number; numeros: number[] }

/**
 * Agrupa os pinos a menos de `distancia` px uns dos outros (na tela). Determinístico: percorre
 * os pinos pelo número; o primeiro livre ancora o grupo e reúne os livres próximos a ele.
 * Nenhum pino some: todo número aparece em exatamente um grupo, e o grupo lista todos.
 */
export function agruparPinos(pinos: PinoTela[], distancia: number): GrupoPinos[] {
  const ordem = [...pinos].sort((a, b) => a.numero - b.numero)
  const usado = new Set<number>()
  const grupos: GrupoPinos[] = []
  const d2 = distancia * distancia
  for (const p of ordem) {
    if (usado.has(p.numero)) continue
    const g: GrupoPinos = { x: p.x, y: p.y, numeros: [p.numero] }
    usado.add(p.numero)
    for (const q of ordem) {
      if (usado.has(q.numero)) continue
      const dx = q.x - p.x, dy = q.y - p.y
      if (dx * dx + dy * dy < d2) { g.numeros.push(q.numero); usado.add(q.numero) }
    }
    grupos.push(g)
  }
  return grupos
}

/** "16/11/2026 às 14:05" (horário local) para mostrar quando a imagem foi preparada. */
export function quandoPreparada(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return `${d.toLocaleDateString('pt-BR')} às ${d.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })}`
}

/** Primeiro PNG de uma carta salva (A30: carta-01-geral.png; A27: carta.png). */
export function primeiroPNG(arquivos: string[] | null | undefined): string {
  const pngs = (arquivos ?? []).filter((a) => a.endsWith('.png')).sort()
  return pngs[0] ?? 'carta.png'
}
