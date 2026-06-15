// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DashboardView from './views/DashboardView.vue'
import TeamView from './views/TeamView.vue'
import TicketListView from './views/TicketListView.vue'
import type { Dashboard, Ticket } from './types/ticket'

const ticket: Ticket = {
  id: 1,
  title: 'Impressora sem conexão',
  description: 'Equipamento indisponível.',
  requesterName: 'Financeiro',
  priority: 'high',
  status: 'open',
  assigneeId: 1,
  assigneeName: 'Ana Souza',
  assignmentMode: 'automatic',
  openedAt: '2026-06-15T12:00:00Z',
  resolvedAt: null,
  createdAt: '2026-06-15T12:00:00Z',
  updatedAt: '2026-06-15T12:00:00Z',
}

const dashboard: Dashboard = {
  total: 4,
  open: 1,
  inProgress: 1,
  resolved: 1,
  closed: 1,
  assignees: [{
    id: 1,
    name: 'Ana Souza',
    active: true,
    openTickets: 2,
    completedTickets: 7,
    lastAssignedAt: null,
  }],
  nextAssignee: null,
}

const { apiMock } = vi.hoisted(() => ({
  apiMock: {
    dashboard: vi.fn(),
    assignees: vi.fn(),
    tickets: vi.fn(),
  },
}))

vi.mock('./services/api', () => ({
  api: apiMock,
}))

async function settle() {
  await flushPromises()
  await flushPromises()
}

function listRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/chamados', component: TicketListView },
      { path: '/chamados/novo', component: { template: '<div />' } },
      { path: '/chamados/:id', component: { template: '<div />' } },
      { path: '/chamados/:id/editar', component: { template: '<div />' } },
    ],
  })
}

describe('visões operacionais', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMock.dashboard.mockResolvedValue(dashboard)
    apiMock.assignees.mockResolvedValue(dashboard.assignees)
    apiMock.tickets.mockResolvedValue([ticket])
  })

  it('sincroniza os filtros quando o histórico do navegador altera a URL', async () => {
    const router = listRouter()
    await router.push('/chamados?status=open')
    await router.isReady()
    const wrapper = mount(TicketListView, { global: { plugins: [router] } })
    await settle()

    expect(wrapper.get('select[aria-label="Filtrar por status"]').element).toHaveProperty('value', 'open')

    await router.push('/chamados?status=closed')
    await settle()
    expect(wrapper.get('select[aria-label="Filtrar por status"]').element).toHaveProperty('value', 'closed')

    router.back()
    await vi.waitFor(() => {
      expect(wrapper.get('select[aria-label="Filtrar por status"]').element).toHaveProperty('value', 'open')
    })
    expect(apiMock.tickets).toHaveBeenLastCalledWith(expect.objectContaining({ status: 'open' }))
  })

  it('exibe erro da listagem e permite tentar novamente', async () => {
    apiMock.tickets
      .mockRejectedValueOnce(new Error('offline'))
      .mockResolvedValueOnce([ticket])
    const router = listRouter()
    await router.push('/chamados')
    await router.isReady()
    const wrapper = mount(TicketListView, { global: { plugins: [router] } })
    await settle()

    expect(wrapper.text()).toContain('Não foi possível carregar os chamados.')
    await wrapper.get('button').trigger('click')
    await settle()

    expect(wrapper.text()).toContain('Impressora sem conexão')
  })

  it('apresenta os indicadores e chamados recentes no dashboard', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: DashboardView },
        { path: '/chamados', component: { template: '<div />' } },
        { path: '/chamados/:id', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    const wrapper = mount(DashboardView, { global: { plugins: [router] } })
    await settle()

    expect(wrapper.text()).toContain('Total de Chamados')
    expect(wrapper.text()).toContain('Impressora sem conexão')
    expect(wrapper.text()).toContain('Resolvidos/Fechados')
  })

  it('apresenta cargas ativas e concluídas da equipe', async () => {
    const wrapper = mount(TeamView)
    await settle()

    expect(wrapper.text()).toContain('Ana Souza')
    expect(wrapper.text()).toContain('Em Fila')
    expect(wrapper.text()).toContain('Concluídos')
    expect(wrapper.text()).toContain('7')
  })
})
