// Exibição sempre em horário local (ADR-0010) e números em pt-BR.
const dataHora = new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short', timeStyle: 'short' })
const numero = new Intl.NumberFormat('pt-BR')

export function quando(ms?: number | null): string {
  return ms ? dataHora.format(new Date(ms)) : '—'
}

export function num(n?: number | null): string {
  return n == null ? '—' : numero.format(n)
}

export function mensagem(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === 'string') return e
  return String(e)
}
