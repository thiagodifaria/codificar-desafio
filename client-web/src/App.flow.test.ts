// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App.vue'
import TicketDetailView from './views/TicketDetailView.vue'
import TicketFormView from './views/TicketFormView.vue'
import TicketListView from './views/TicketListView.vue'
import type { Assignee, Ticket, TicketInput, TicketStatus } from './types/ticket'

const assignees: Assignee[] = [{
  id: 1,
  name: 'Ana Souza',
  active: true,
  openTickets: 0,
  completedTickets: 0,
  lastAssignedAt: null,
}]

let currentTicket: Ticket

const { apiMock } = vi.hoisted(() => ({
  apiMock: {
    assignees: vi.fn(),
    tickets: vi.fn(),
    ticket: vi.fn(),
    createTicket: vi.fn(),
    updateTicket: vi.fn(),
    changeTicketStatus: vi.fn(),
  },
}))

vi.mock('./services/api', () => ({
  ApiError: class ApiError extends Error {
    fields: Record<string, string>

    constructor(message: string, _status: number, fields: Record<string, string> = {}) {
      super(message)
      this.fields = fields
    }
  },
  api: apiMock,
}))

function makeTicket(input: TicketInput): Ticket {
  return {
    id: 99,
    ...input,
    assigneeId: 1,
    assigneeName: 'Ana Souza',
    openedAt: '2026-06-15T12:00:00Z',
    resolvedAt: null,
    createdAt: '2026-06-15T12:00:00Z',
    updatedAt: '2026-06-15T12:00:00Z',
  }
}

async function settle() {
  await flushPromises()
  await flushPromises()
}

describe('interface do fluxo principal', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
    currentTicket = makeTicket({
      title: 'Chamado inicial',
      description: 'Descrição inicial',
      requesterName: 'Financeiro',
      priority: 'medium',
      status: 'open',
      assignmentMode: 'automatic',
      assigneeId: 0,
      redistribute: false,
    })
    apiMock.assignees.mockImplementation(async () => assignees)
    apiMock.tickets.mockImplementation(async () => [currentTicket])
    apiMock.ticket.mockImplementation(async () => currentTicket)
    apiMock.createTicket.mockImplementation(async (input: TicketInput) => {
      currentTicket = makeTicket(input)
      return currentTicket
    })
    apiMock.updateTicket.mockImplementation(async (_id: number, input: TicketInput) => {
      currentTicket = { ...currentTicket, ...input, updatedAt: '2026-06-15T13:00:00Z' }
      return currentTicket
    })
    apiMock.changeTicketStatus.mockImplementation(async (_id: number, status: TicketStatus) => {
      currentTicket = {
        ...currentTicket,
        status,
        resolvedAt: status === 'resolved' ? '2026-06-15T14:00:00Z' : null,
      }
      return currentTicket
    })
  })

  it('cria, edita e conclui um chamado pela interface', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/chamados', component: TicketListView },
        { path: '/equipe', component: { template: '<div />' } },
        { path: '/configuracoes', component: { template: '<div />' } },
        { path: '/chamados/novo', component: TicketFormView },
        { path: '/chamados/:id', component: TicketDetailView },
        { path: '/chamados/:id/editar', component: TicketFormView },
      ],
    })
    await router.push('/chamados/novo')
    await router.isReady()

    const wrapper = mount(App, { global: { plugins: [router] } })
    await settle()

    await wrapper.get('input[name="title"]').setValue('Impressora sem conexão')
    await wrapper.get('textarea[name="description"]').setValue('A impressora do financeiro não responde.')
    await wrapper.get('input[placeholder="Nome do solicitante"]').setValue('Mariana Alves')
    await wrapper.get('#ticket-form').trigger('submit')
    await settle()

    expect(apiMock.createTicket).toHaveBeenCalledOnce()
    expect(router.currentRoute.value.path).toBe('/chamados/99')
    expect(wrapper.text()).toContain('Impressora sem conexão')

    await wrapper.get('a[href="/chamados/99/editar"]').trigger('click')
    await settle()
    await wrapper.get('input[name="title"]').setValue('Impressora reconectada')
    await wrapper.get('#ticket-form').trigger('submit')
    await settle()

    expect(apiMock.updateTicket).toHaveBeenCalledOnce()
    expect(router.currentRoute.value.path).toBe('/chamados/99')
    expect(wrapper.text()).toContain('Impressora reconectada')

    const resolveButton = wrapper.findAll('button').find((button) => button.text().includes('Marcar como resolvido'))
    expect(resolveButton).toBeTruthy()
    await resolveButton!.trigger('click')
    await settle()

    expect(apiMock.changeTicketStatus).toHaveBeenCalledWith(99, 'resolved')
    expect(wrapper.text()).toContain('Resolvido')
  })

  it('mantém a criação aberta e mostra o erro do responsável junto ao campo manual', async () => {
    const { ApiError } = await import('./services/api')
    apiMock.createTicket.mockRejectedValueOnce(
      new ApiError('Revise os campos informados.', 422, {
        assigneeId: 'Selecione um responsável.',
      }),
    )
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/chamados', component: { template: '<div />' } },
        { path: '/equipe', component: { template: '<div />' } },
        { path: '/configuracoes', component: { template: '<div />' } },
        { path: '/chamados/novo', component: TicketFormView },
      ],
    })
    await router.push('/chamados/novo')
    await router.isReady()

    const wrapper = mount(App, { global: { plugins: [router] } })
    await settle()

    expect(wrapper.find('select[name="status"]').exists()).toBe(false)
    await wrapper.get('input[value="manual"]').setValue()
    await wrapper.get('input[name="title"]').setValue('Acesso bloqueado')
    await wrapper.get('textarea[name="description"]').setValue('Usuário sem acesso ao sistema.')
    await wrapper.get('input[name="requesterName"]').setValue('Financeiro')
    await wrapper.get('#ticket-form').trigger('submit')
    await settle()

    const assignee = wrapper.get('select[name="assigneeId"]')
    expect(assignee.attributes('aria-invalid')).toBe('true')
    expect(wrapper.get('#ticket-assignee-error').text()).toBe('Selecione um responsável.')
    expect(apiMock.createTicket).toHaveBeenCalledWith(
      expect.objectContaining({ status: 'open', assignmentMode: 'manual' }),
    )
  })
})
