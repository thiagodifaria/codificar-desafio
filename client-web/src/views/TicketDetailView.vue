<script setup lang="ts">
import {
  ArrowLeft,
  CalendarClock,
  CheckCircle2,
  CirclePlay,
  LockKeyhole,
  Pencil,
  RotateCcw,
  Scale,
  UserRound,
} from '@lucide/vue'
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import LoadingState from '../components/LoadingState.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { api } from '../services/api'
import { formatDate } from '../services/format'
import { showToast } from '../services/toast'
import type { Ticket, TicketStatus } from '../types/ticket'

const route = useRoute()
const ticket = ref<Ticket | null>(null)
const loading = ref(true)
const error = ref('')
const updatingStatus = ref(false)
const ticketID = Number(route.params.id)

const statusActions = computed(() => {
  if (!ticket.value) return []
  const actions: Array<{ label: string; status: TicketStatus; icon: typeof CirclePlay; primary?: boolean }> = []
  if (ticket.value.status === 'open') {
    actions.push({ label: 'Iniciar atendimento', status: 'in_progress', icon: CirclePlay, primary: true })
  }
  if (ticket.value.status === 'open' || ticket.value.status === 'in_progress') {
    actions.push({ label: 'Marcar como resolvido', status: 'resolved', icon: CheckCircle2 })
  }
  if (ticket.value.status === 'resolved') {
    actions.push({ label: 'Fechar chamado', status: 'closed', icon: LockKeyhole, primary: true })
    actions.push({ label: 'Reabrir', status: 'open', icon: RotateCcw })
  }
  if (ticket.value.status === 'closed') {
    actions.push({ label: 'Reabrir', status: 'open', icon: RotateCcw })
  }
  return actions
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    ticket.value = await api.ticket(ticketID)
  } catch {
    error.value = 'Chamado não encontrado ou indisponível.'
  } finally {
    loading.value = false
  }
}

async function changeStatus(status: TicketStatus) {
  updatingStatus.value = true
  try {
    ticket.value = await api.changeTicketStatus(ticketID, status)
    showToast('Status do chamado atualizado.')
  } catch {
    showToast('Não foi possível atualizar o status.', 'error')
  } finally {
    updatingStatus.value = false
  }
}

onMounted(load)
</script>

<template>
  <section>
    <RouterLink to="/chamados" class="inline-flex items-center gap-2 text-sm font-semibold text-slate-500 hover:text-emerald-700">
      <ArrowLeft :size="17" />
      Todos os chamados
    </RouterLink>

    <LoadingState v-if="loading" />

    <div v-else-if="error" class="panel mt-8 p-8 text-center">
      <p class="font-semibold text-slate-800">{{ error }}</p>
      <RouterLink to="/chamados" class="button-secondary mt-4">Voltar à lista</RouterLink>
    </div>

    <template v-else-if="ticket">
      <div class="mt-6 flex flex-col justify-between gap-5 sm:flex-row sm:items-start">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-sm font-bold text-slate-500">#{{ ticket.id }}</span>
            <StatusBadge :status="ticket.status" />
            <StatusBadge :priority="ticket.priority" />
          </div>
          <h1 class="mt-4 max-w-4xl text-3xl font-bold tracking-tight text-slate-950">{{ ticket.title }}</h1>
          <p class="mt-2 text-sm text-slate-500">Solicitado por <strong class="font-semibold text-slate-700">{{ ticket.requesterName }}</strong></p>
        </div>
        <RouterLink :to="`/chamados/${ticket.id}/editar`" class="button-secondary shrink-0">
          <Pencil :size="17" />
          Editar chamado
        </RouterLink>
      </div>

      <div class="mt-8 grid gap-6 xl:grid-cols-[1.4fr_0.8fr]">
        <div class="space-y-6">
          <section v-if="statusActions.length" class="panel flex flex-wrap items-center gap-3 p-4">
            <p class="mr-auto text-sm font-semibold text-slate-600">Próxima ação</p>
            <button
              v-for="action in statusActions"
              :key="action.status"
              type="button"
              :class="action.primary ? 'button-primary' : 'button-secondary'"
              :disabled="updatingStatus"
              @click="changeStatus(action.status)"
            >
              <component :is="action.icon" :size="17" />
              {{ action.label }}
            </button>
          </section>

          <article class="panel p-5 sm:p-7">
            <h2 class="text-sm font-bold uppercase tracking-[0.12em] text-slate-500">Descrição</h2>
            <p class="mt-5 whitespace-pre-wrap text-[15px] leading-7 text-slate-700">{{ ticket.description }}</p>
          </article>
        </div>

        <aside class="panel divide-y divide-slate-100 px-5 sm:px-6">
          <div class="flex gap-3 py-5">
            <span class="grid size-10 shrink-0 place-items-center rounded-xl bg-emerald-50 text-emerald-700">
              <UserRound :size="19" />
            </span>
            <div>
              <p class="text-xs font-semibold uppercase tracking-wide text-slate-500">Responsável</p>
              <p class="mt-1 text-sm font-bold text-slate-800">{{ ticket.assigneeName }}</p>
            </div>
          </div>
          <div class="flex gap-3 py-5">
            <span class="grid size-10 shrink-0 place-items-center rounded-xl bg-violet-50 text-violet-700">
              <Scale :size="19" />
            </span>
            <div>
              <p class="text-xs font-semibold uppercase tracking-wide text-slate-500">Atribuição</p>
              <p class="mt-1 text-sm font-bold text-slate-800">
                {{ ticket.assignmentMode === 'automatic' ? 'Automática' : 'Manual' }}
              </p>
            </div>
          </div>
          <div class="flex gap-3 py-5">
            <span class="grid size-10 shrink-0 place-items-center rounded-xl bg-sky-50 text-sky-700">
              <CalendarClock :size="19" />
            </span>
            <div>
              <p class="text-xs font-semibold uppercase tracking-wide text-slate-500">Abertura</p>
              <p class="mt-1 text-sm font-bold text-slate-800">{{ formatDate(ticket.openedAt) }}</p>
              <p v-if="ticket.resolvedAt" class="mt-1 text-xs text-slate-500">Concluído em {{ formatDate(ticket.resolvedAt) }}</p>
              <p class="mt-1 text-xs text-slate-500">Última atualização em {{ formatDate(ticket.updatedAt) }}</p>
            </div>
          </div>
        </aside>
      </div>
    </template>
  </section>
</template>
