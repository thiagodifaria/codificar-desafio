import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: () => import('../views/DashboardView.vue'),
    },
    {
      path: '/chamados',
      name: 'tickets',
      component: () => import('../views/TicketListView.vue'),
    },
    {
      path: '/equipe',
      name: 'team',
      component: () => import('../views/TeamView.vue'),
    },
    {
      path: '/configuracoes',
      name: 'settings',
      component: () => import('../views/SettingsView.vue'),
    },
    {
      path: '/chamados/novo',
      name: 'ticket-create',
      component: () => import('../views/TicketFormView.vue'),
    },
    {
      path: '/chamados/:id',
      name: 'ticket-detail',
      component: () => import('../views/TicketDetailView.vue'),
    },
    {
      path: '/chamados/:id/editar',
      name: 'ticket-edit',
      component: () => import('../views/TicketFormView.vue'),
    },
  ],
})

export default router

