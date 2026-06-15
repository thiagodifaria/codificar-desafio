<script setup lang="ts">
import { AlertTriangle, Calendar, Plus, RotateCcw, Search, SlidersHorizontal, TicketX, UserRound } from '@lucide/vue'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import LoadingState from '../components/LoadingState.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { api } from '../services/api'
import { needsAttention, relativeDate } from '../services/format'
import type { Assignee, Ticket, TicketSort } from '../types/ticket'

const route = useRoute()
const router = useRouter()
const tickets = ref<Ticket[]>([])
const assignees = ref<Assignee[]>([])
const loading = ref(true)
const error = ref('')
const filters = reactive({
  search: '',
  status: '',
  priority: '',
  assigneeId: 0,
  sort: 'priority_desc' as TicketSort,
})
let searchTimer: ReturnType<typeof setTimeout> | undefined

const hasFilters = computed(() =>
  Boolean(filters.search || filters.status || filters.priority || filters.assigneeId || filters.sort !== 'priority_desc'),
)

async function loadTickets() {
  loading.value = true
  error.value = ''
  try {
    tickets.value = await api.tickets(filters)
  } catch {
    error.value = 'Não foi possível carregar os chamados.'
  } finally {
    loading.value = false
  }
}

function filtersFromRoute() {
  return {
    search: String(route.query.search ?? ''),
    status: String(route.query.status ?? ''),
    priority: String(route.query.priority ?? ''),
    assigneeId: Number(route.query.assigneeId ?? 0),
    sort: (route.query.sort as TicketSort | undefined) ?? 'priority_desc',
  }
}

function queryFromFilters() {
  const query: Record<string, string> = {}
  if (filters.search) query.search = filters.search
  if (filters.status) query.status = filters.status
  if (filters.priority) query.priority = filters.priority
  if (filters.assigneeId) query.assigneeId = String(filters.assigneeId)
  if (filters.sort !== 'priority_desc') query.sort = filters.sort
  return query
}

function queriesMatch(left: Record<string, unknown>, right: Record<string, string>) {
  const normalizedLeft = Object.fromEntries(
    Object.entries(left).map(([key, value]) => [key, String(value)]),
  )
  return JSON.stringify(normalizedLeft) === JSON.stringify(right)
}

watch(
  () => [filters.search, filters.status, filters.priority, filters.assigneeId, filters.sort],
  () => {
    clearTimeout(searchTimer)
    searchTimer = setTimeout(() => {
      const query = queryFromFilters()
      if (!queriesMatch(route.query, query)) {
        void router.push({ query })
      }
    }, 250)
  },
)

watch(
  () => route.query,
  async () => {
    clearTimeout(searchTimer)
    Object.assign(filters, filtersFromRoute())
    await loadTickets()
  },
  { immediate: true },
)

function clearFilters() {
  Object.assign(filters, {
    search: '',
    status: '',
    priority: '',
    assigneeId: 0,
    sort: 'priority_desc',
  })
}

onMounted(async () => {
  assignees.value = await api.assignees().catch(() => [])
})
</script>

<template>
  <section>
    <div>
      <p class="text-sm font-semibold text-emerald-700">Acompanhamento</p>
      <h1 class="mt-1 text-3xl font-bold tracking-tight text-slate-950">Chamados</h1>
      <p class="mt-2 text-sm text-slate-500">Encontre rapidamente o que precisa de atenção.</p>
    </div>

    <div class="mt-8">
      <div class="flex flex-col items-start justify-between gap-4 xl:flex-row xl:items-center">
        <div class="grid w-full flex-1 gap-3 sm:grid-cols-2 xl:grid-cols-[minmax(240px,1fr)_160px_150px_190px_190px]">
          <div class="relative sm:col-span-2 xl:col-span-1">
            <Search class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" :size="18" />
            <input
              v-model="filters.search"
              type="text"
              placeholder="Buscar por título, descrição ou solicitante..."
              class="field py-2 pl-10"
            />
          </div>

          <div class="relative">
            <select v-model="filters.status" class="field appearance-none py-2 pr-10" aria-label="Filtrar por status">
              <option value="">Todos os status</option>
              <option value="open">Aberto</option>
              <option value="in_progress">Em andamento</option>
              <option value="resolved">Resolvido</option>
              <option value="closed">Fechado</option>
            </select>
            <SlidersHorizontal class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-slate-400" :size="16" />
          </div>

          <select v-model="filters.priority" class="field py-2" aria-label="Filtrar por prioridade">
            <option value="">Prioridades</option>
            <option value="high">Alta</option>
            <option value="medium">Média</option>
            <option value="low">Baixa</option>
          </select>

          <select v-model="filters.assigneeId" class="field py-2" aria-label="Filtrar por responsável">
            <option :value="0">Todos os responsáveis</option>
            <option v-for="item in assignees" :key="item.id" :value="item.id">{{ item.name }}</option>
          </select>

          <select v-model="filters.sort" class="field py-2" aria-label="Ordenar chamados">
            <option value="priority_desc">Maior prioridade</option>
            <option value="opened_desc">Mais recentes</option>
            <option value="opened_asc">Mais antigos</option>
            <option value="updated_desc">Atualizados recentemente</option>
          </select>
        </div>

        <RouterLink to="/chamados/novo" class="button-primary w-full shrink-0 xl:w-auto">
          <Plus :size="18" />
          Novo chamado
        </RouterLink>
      </div>

      <button v-if="hasFilters" class="mt-3 inline-flex items-center gap-2 text-xs font-semibold text-slate-500 hover:text-emerald-700" @click="clearFilters">
        <RotateCcw :size="14" />
        Limpar filtros
      </button>
    </div>

    <LoadingState v-if="loading" variant="list" />

    <div v-else-if="error" class="panel mt-6 p-8 text-center">
      <p class="font-semibold text-slate-800">{{ error }}</p>
      <button class="button-secondary mt-4" @click="loadTickets">Tentar novamente</button>
    </div>

    <div v-else-if="tickets.length" class="mt-6 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
      <div class="overflow-x-auto">
        <table class="w-full border-collapse text-left">
          <thead>
            <tr class="border-b border-slate-200 bg-slate-50 text-[11px] uppercase tracking-wider text-slate-500">
              <th class="px-6 py-4 font-medium">Chamado</th>
              <th class="px-6 py-4 font-medium">Status</th>
              <th class="px-6 py-4 font-medium">Prioridade</th>
              <th class="px-6 py-4 font-medium">Responsável</th>
              <th class="px-6 py-4 font-medium">Abertura</th>
              <th class="px-6 py-4 text-right font-medium">Ações</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr
              v-for="item in tickets"
              :key="item.id"
              class="group border-l-4 transition-colors hover:bg-slate-50/50"
              :class="needsAttention(item.openedAt, item.priority, item.status) ? 'border-l-amber-400 bg-amber-50/30' : 'border-l-transparent'"
            >
              <td class="px-6 py-4">
                <RouterLink :to="`/chamados/${item.id}`" class="flex flex-col">
                  <span class="mb-1 flex items-center gap-1.5 font-mono text-[10px] text-slate-400">
                    <AlertTriangle v-if="needsAttention(item.openedAt, item.priority, item.status)" :size="12" class="text-amber-500" />
                    #CH-{{ item.id.toString().padStart(4, '0') }}
                  </span>
                  <span class="line-clamp-1 text-sm font-medium text-slate-800 transition-colors group-hover:text-emerald-700">{{ item.title }}</span>
                  <span class="mt-1 text-[11px] text-slate-500">{{ item.requesterName }} · {{ relativeDate(item.openedAt) }}</span>
                </RouterLink>
              </td>
              <td class="px-6 py-4"><StatusBadge :status="item.status" /></td>
              <td class="px-6 py-4"><StatusBadge :priority="item.priority" /></td>
              <td class="px-6 py-4">
                <div class="flex items-center gap-2">
                  <span class="grid size-7 place-items-center rounded-lg bg-slate-100 text-slate-500">
                    <UserRound :size="14" />
                  </span>
                  <span class="text-xs font-medium text-slate-700">{{ item.assigneeName || 'Não atribuído' }}</span>
                </div>
              </td>
              <td class="px-6 py-4">
                <div class="flex items-center gap-2 text-xs text-slate-500">
                  <Calendar :size="14" />
                  {{ new Date(item.openedAt).toLocaleDateString('pt-BR', { day: '2-digit', month: 'short' }) }}
                </div>
              </td>
              <td class="px-6 py-4 text-right">
                <RouterLink :to="`/chamados/${item.id}/editar`" class="rounded px-3 py-1.5 text-xs font-medium text-emerald-700 transition-colors hover:bg-emerald-50 hover:text-emerald-900">
                  Editar
                </RouterLink>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-else class="panel mt-6 flex flex-col items-center px-6 py-16 text-center">
      <span class="grid size-14 place-items-center rounded-2xl bg-slate-100 text-slate-400">
        <TicketX :size="26" />
      </span>
      <h2 class="mt-4 font-bold text-slate-800">Nenhum chamado encontrado</h2>
      <p class="mt-1 max-w-sm text-sm text-slate-500">Ajuste os filtros ou registre uma nova solicitação para começar.</p>
      <RouterLink to="/chamados/novo" class="button-primary mt-5">Criar chamado</RouterLink>
    </div>

    <p v-if="!loading && tickets.length" class="mt-4 flex items-center gap-2 text-xs text-slate-500">
      <SlidersHorizontal :size="14" />
      {{ tickets.length }} {{ tickets.length === 1 ? 'resultado encontrado' : 'resultados encontrados' }}
    </p>
  </section>
</template>
