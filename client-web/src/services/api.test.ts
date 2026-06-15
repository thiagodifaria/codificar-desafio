import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from './api'
import type { Ticket, TicketInput } from '../types/ticket'

const input: TicketInput = {
  title: 'Impressora sem conexão',
  description: 'A impressora do financeiro não responde.',
  requesterName: 'Financeiro',
  priority: 'high',
  status: 'open',
  assignmentMode: 'automatic',
  assigneeId: 0,
  redistribute: false,
}

const ticket: Ticket = {
  id: 12,
  ...input,
  assigneeId: 2,
  assigneeName: 'Bruno Lima',
  openedAt: '2026-06-15T12:00:00Z',
  resolvedAt: null,
  createdAt: '2026-06-15T12:00:00Z',
  updatedAt: '2026-06-15T12:00:00Z',
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('fluxo principal de chamados', () => {
  it('cria, edita e conclui um chamado pelo contrato HTTP', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(ticket), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...ticket, title: 'Impressora reconectada' }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...ticket, status: 'resolved' }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await api.createTicket(input)
    await api.updateTicket(ticket.id, { ...input, title: 'Impressora reconectada' })
    const resolved = await api.changeTicketStatus(ticket.id, 'resolved')

    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/tickets', expect.objectContaining({ method: 'POST' }))
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/tickets/12', expect.objectContaining({ method: 'PUT' }))
    expect(fetchMock).toHaveBeenNthCalledWith(3, '/api/tickets/12/status', expect.objectContaining({ method: 'PATCH' }))
    expect(resolved.status).toBe('resolved')
  })
})
