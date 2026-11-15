// Tipos e textos da tela Cartas-imagem (A27, ADR-0022).
export interface PinoMapa { numero: number; nome: string; lat: number; lon: number }
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
