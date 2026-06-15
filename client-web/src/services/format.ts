import type { Priority, TicketStatus } from '../types/ticket'

export const priorityLabel: Record<Priority, string> = {
  low: 'Baixa',
  medium: 'Média',
  high: 'Alta',
}

export const statusLabel: Record<TicketStatus, string> = {
  open: 'Aberto',
  in_progress: 'Em andamento',
  resolved: 'Resolvido',
  closed: 'Fechado',
}

export function formatDate(value: string): string {
  return new Intl.DateTimeFormat('pt-BR', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value))
}

export function relativeDate(value: string): string {
  const date = new Date(value)
  const seconds = Math.round((date.getTime() - Date.now()) / 1000)
  const formatter = new Intl.RelativeTimeFormat('pt-BR', { numeric: 'auto' })
  const ranges: Array<[Intl.RelativeTimeFormatUnit, number]> = [
    ['year', 31_536_000],
    ['month', 2_592_000],
    ['week', 604_800],
    ['day', 86_400],
    ['hour', 3_600],
    ['minute', 60],
  ]
  for (const [unit, divisor] of ranges) {
    if (Math.abs(seconds) >= divisor) {
      return formatter.format(Math.round(seconds / divisor), unit)
    }
  }
  return 'agora'
}

export function hoursSince(value: string): number {
  return Math.max(0, (Date.now() - new Date(value).getTime()) / 3_600_000)
}

export function needsAttention(openedAt: string, priority: Priority, status: TicketStatus): boolean {
  if (status === 'resolved' || status === 'closed') {
    return false
  }
  return priority === 'high' || hoursSince(openedAt) >= 24
}
