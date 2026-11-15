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

/** Número de uma área já formatada pela Central em pt-BR ("1.234,50" → 1234.5); null se não informada. */
export function haNumero(s?: string | null): number | null {
  if (!s) return null
  const n = Number(s.replace(/\./g, '').replace(',', '.'))
  return Number.isFinite(n) ? n : null
}

/** Formata hectares com duas casas (para somas feitas na tela). */
export function ha(n: number): string {
  return new Intl.NumberFormat('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(n)
}

/** "1 família" / "2 famílias". */
export function plural(n: number, um: string, varios: string): string {
  return `${num(n)} ${n === 1 ? um : varios}`
}
