<script setup lang="ts">
import { ArrowRight, CircleDot, Sparkles, Users } from '@lucide/vue'
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import LoadingState from '../components/LoadingState.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { api } from '../services/api'
import { relativeDate } from '../services/format'
import type { Dashboard, Ticket } from '../types/ticket'

const dashboard = ref<Dashboard | null>(null)
const recentTickets = ref<Ticket[]>([])
const loading = ref(true)
const error = ref('')
const loadedAt = ref(new Date())

const maximumLoad = computed(() =>
  Math.max(1, ...(dashboard.value?.assignees.map((item) => item.openTickets) ?? [])),
)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [summary, tickets] = await Promise.all([
      api.dashboard(),
      api.tickets({ sort: 'opened_desc', limit: 5 }),
    ])
    dashboard.value = summary
    recentTickets.value = tickets
    loadedAt.value = new Date()
  } catch {
    error.value = 'Não foi possível carregar a visão geral.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section>
    <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div>
        <p class="text-sm font-semibold text-emerald-700">Visão geral</p>
        <h1 class="mt-1 text-3xl font-bold tracking-tight text-slate-950">Bom trabalho por aqui.</h1>
        <p class="mt-2 text-sm text-slate-600">Acompanhe a operação e mantenha a carga do time equilibrada.</p>
      </div>
      <p class="inline-flex items-center gap-2 self-start rounded-full bg-white px-3 py-1.5 text-xs font-medium text-slate-500 ring-1 ring-slate-200">
        <CircleDot :size="14" class="text-emerald-500" />
        Atualizado {{ relativeDate(loadedAt.toISOString()) }}
      </p>
    </div>

    <LoadingState v-if="loading" variant="dashboard" />

    <div v-else-if="error" class="panel mt-8 p-8 text-center">
      <p class="font-semibold text-slate-800">{{ error }}</p>
      <button class="button-secondary mt-4" @click="load">Tentar novamente</button>
    </div>

    <template v-else-if="dashboard">
      <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mt-8">
        <div class="bg-white rounded-xl border border-slate-200 p-5 shadow-sm">
          <div class="text-slate-500 text-sm font-medium mb-1">Total de Chamados</div>
          <div class="text-3xl font-bold text-slate-800">{{ dashboard.total || (dashboard.open + dashboard.inProgress + dashboard.resolved + (dashboard.closed || 0)) }}</div>
        </div>
        <div class="bg-white rounded-xl border border-slate-200 p-5 shadow-sm">
          <div class="text-slate-500 text-sm font-medium mb-1">Abertos</div>
          <div class="text-3xl font-bold text-amber-600">{{ dashboard.open }}</div>
        </div>
        <div class="bg-white rounded-xl border border-slate-200 p-5 shadow-sm">
          <div class="text-slate-500 text-sm font-medium mb-1">Em Andamento</div>
          <div class="text-3xl font-bold text-emerald-600">{{ dashboard.inProgress }}</div>
        </div>
        <div class="bg-white rounded-xl border border-slate-200 p-5 shadow-sm">
          <div class="text-slate-500 text-sm font-medium mb-1">Resolvidos/Fechados</div>
          <div class="text-3xl font-bold text-blue-600">{{ dashboard.resolved + (dashboard.closed || 0) }}</div>
        </div>
      </div>

      <div class="bg-gradient-to-r from-emerald-600 to-teal-700 p-8 rounded-2xl shadow-sm text-white mt-6">
        <h2 class="text-2xl font-bold mb-2">Bem-vindo(a) ao seu Painel</h2>
        <p class="text-emerald-50 max-w-2xl mb-6">
          Você tem {{ dashboard.open }} chamados aguardando a sua atenção primária hoje. Utilize a aba de "Equipe" para entender como as demandas estão distribuídas.
        </p>
        <RouterLink
          to="/chamados"
          class="bg-white text-emerald-700 px-5 py-2 rounded-lg font-bold text-sm shadow-sm hover:bg-emerald-50 transition-colors inline-block"
        >
          Ir para Chamados
        </RouterLink>
      </div>

      <div class="mt-8 grid gap-6 xl:grid-cols-[1.45fr_1fr]">
        <section class="panel overflow-hidden">
          <div class="flex items-center justify-between border-b border-slate-100 px-5 py-5 sm:px-6">
            <div>
              <h2 class="font-bold text-slate-900">Chamados recentes</h2>
              <p class="mt-1 text-xs text-slate-500">Itens que merecem atenção primeiro.</p>
            </div>
            <RouterLink to="/chamados" class="flex items-center gap-1 text-sm font-semibold text-emerald-700 hover:text-emerald-800">
              Ver todos
              <ArrowRight :size="16" />
            </RouterLink>
          </div>

          <div v-if="recentTickets.length" class="divide-y divide-slate-100">
            <RouterLink
              v-for="item in recentTickets"
              :key="item.id"
              :to="`/chamados/${item.id}`"
              class="group flex items-center gap-4 px-5 py-4 transition hover:bg-slate-50/50 sm:px-6"
            >
              <span class="min-w-0 flex-1">
                <span class="flex min-w-0 items-baseline gap-2">
                  <span class="shrink-0 font-mono text-[11px] font-semibold text-slate-400">#CH-{{ item.id.toString().padStart(4, '0') }}</span>
                  <span class="truncate text-sm font-semibold text-slate-800 transition-colors group-hover:text-emerald-700">{{ item.title }}</span>
                </span>
                <span class="mt-1 block text-[11px] text-slate-500">{{ item.assigneeName || 'Não atribuído' }} · {{ relativeDate(item.openedAt) }}</span>
              </span>
              <StatusBadge :status="item.status" />
            </RouterLink>
          </div>
          <div v-else class="px-6 py-14 text-center text-sm text-slate-500">
            Nenhum chamado registrado ainda.
          </div>
        </section>

        <section class="panel p-5 sm:p-6">
          <div class="flex items-center gap-3">
            <span class="grid size-10 place-items-center rounded-xl bg-violet-50 text-violet-700">
              <Users :size="20" />
            </span>
            <div>
              <h2 class="font-bold text-slate-900">Carga do time</h2>
              <p class="text-xs text-slate-500">Chamados ainda não concluídos.</p>
            </div>
          </div>

          <div class="mt-7 space-y-6">
            <div v-for="assignee in dashboard.assignees" :key="assignee.id">
              <div class="mb-2.5 flex items-center justify-between text-sm">
                <span class="font-semibold text-slate-700">{{ assignee.name }}</span>
                <span class="text-xs font-medium text-slate-500">
                  {{ assignee.openTickets }} {{ assignee.openTickets === 1 ? 'chamado' : 'chamados' }}
                </span>
              </div>
              <div class="h-2 overflow-hidden rounded-full bg-slate-100">
                <div
                  class="h-full rounded-full bg-emerald-500 transition-all duration-500"
                  :style="{ width: `${Math.max(assignee.openTickets ? 12 : 0, (assignee.openTickets / maximumLoad) * 100)}%` }"
                />
              </div>
            </div>
          </div>

          <div class="mt-7 rounded-xl bg-emerald-50 p-4 text-xs leading-5 text-emerald-800">
            Novos chamados automáticos são enviados para quem possui menos itens ativos. Empates respeitam a ordem de atendimento.
          </div>

          <div v-if="dashboard.nextAssignee" class="mt-4 flex items-center gap-3 rounded-xl border border-violet-100 bg-violet-50/70 p-4">
            <span class="grid size-9 shrink-0 place-items-center rounded-lg bg-white text-violet-700 shadow-sm">
              <Sparkles :size="18" />
            </span>
            <div>
              <p class="text-xs font-semibold uppercase tracking-wide text-violet-700">Próxima atribuição</p>
              <p class="mt-0.5 text-sm font-bold text-slate-800">{{ dashboard.nextAssignee.name }}</p>
            </div>
          </div>
        </section>
      </div>
    </template>
  </section>
</template>
