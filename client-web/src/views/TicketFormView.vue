<script setup lang="ts">
import { ArrowLeft, Check, Wand2, UserRound, Plus, Settings } from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { onBeforeRouteLeave, RouterLink, useRoute, useRouter } from 'vue-router'
import LoadingState from '../components/LoadingState.vue'
import { ApiError, api } from '../services/api'
import { showToast } from '../services/toast'
import type { Assignee, TicketInput } from '../types/ticket'

const route = useRoute()
const router = useRouter()
const ticketID = computed(() => Number(route.params.id || 0))
const editing = computed(() => ticketID.value > 0)
const loading = ref(editing.value)
const saving = ref(false)
const assignees = ref<Assignee[]>([])
const errors = ref<Record<string, string>>({})
const generalError = ref('')
const initialSnapshot = ref('')
const originalAssignmentMode = ref<'manual' | 'automatic'>('automatic')

const form = reactive<TicketInput>({
  title: '',
  description: '',
  requesterName: '',
  priority: 'medium',
  status: 'open',
  assignmentMode: localStorage.getItem('autoAssignment') === 'false' ? 'manual' : 'automatic',
  assigneeId: 0,
  redistribute: false,
})

const dirty = computed(() =>
  Boolean(initialSnapshot.value) && JSON.stringify(form) !== initialSnapshot.value,
)

async function load() {
  loading.value = true
  generalError.value = ''
  try {
    const requests: [Promise<Assignee[]>, ReturnType<typeof api.ticket> | undefined] = [
      api.assignees(),
      editing.value ? api.ticket(ticketID.value) : undefined,
    ]
    const [availableAssignees, ticket] = await Promise.all(requests)
    assignees.value = availableAssignees
    if (ticket) {
      Object.assign(form, {
        title: ticket.title,
        description: ticket.description,
        requesterName: ticket.requesterName,
        priority: ticket.priority,
        status: ticket.status,
        assignmentMode: ticket.assignmentMode,
        assigneeId: ticket.assigneeId,
        redistribute: false,
      })
      originalAssignmentMode.value = ticket.assignmentMode
    }
    initialSnapshot.value = JSON.stringify(form)
  } catch {
    generalError.value = 'Não foi possível carregar os dados do chamado.'
  } finally {
    loading.value = false
  }
}

async function submit() {
  saving.value = true
  errors.value = {}
  generalError.value = ''
  try {
    const saved = editing.value
      ? await api.updateTicket(ticketID.value, form)
      : await api.createTicket(form)
    initialSnapshot.value = JSON.stringify(form)
    showToast(editing.value ? 'Chamado atualizado com sucesso.' : 'Chamado criado e atribuído com sucesso.')
    await router.push(`/chamados/${saved.id}`)
  } catch (error) {
    if (error instanceof ApiError) {
      errors.value = error.fields
      generalError.value = error.message
    } else {
      generalError.value = 'Não foi possível salvar o chamado.'
    }
  } finally {
    saving.value = false
  }
}

function beforeUnload(event: BeforeUnloadEvent) {
  if (!dirty.value) return
  event.preventDefault()
  event.returnValue = ''
}

onBeforeRouteLeave(() => {
  if (!dirty.value) return true
  return window.confirm('Existem alterações não salvas. Deseja sair mesmo assim?')
})

onMounted(load)
onMounted(() => window.addEventListener('beforeunload', beforeUnload))
onBeforeUnmount(() => window.removeEventListener('beforeunload', beforeUnload))
</script>

<template>
  <div class="max-w-4xl mx-auto">
    <!-- MODAL-LIKE HEADER -->
    <div class="px-6 py-4 border-b border-slate-200 flex items-center justify-between bg-slate-50 rounded-t-2xl">
      <h2 class="text-lg font-bold text-slate-800 flex items-center gap-2">
        <component :is="editing ? Settings : Plus" :size="20" class="text-emerald-600"/>
        {{ editing ? `Editar Chamado #CH-${ticketID.toString().padStart(4,'0')}` : 'Abertura de Novo Chamado' }}
      </h2>
      <RouterLink :to="editing ? `/chamados/${ticketID}` : '/chamados'" class="p-2 text-slate-400 hover:text-slate-600 hover:bg-slate-200 rounded-full transition-colors">
        <ArrowLeft :size="20" />
      </RouterLink>
    </div>

    <LoadingState v-if="loading" />

    <div v-else class="bg-white rounded-b-2xl shadow-xl overflow-hidden flex flex-col border border-t-0 border-slate-200">
      <div class="p-6 overflow-y-auto">
        <form id="ticket-form" @submit.prevent="submit" class="space-y-5">

          <div>
            <label for="ticket-title" class="block text-sm font-medium text-slate-700 mb-1">Título do problema</label>
            <input
              id="ticket-title"
              required
              name="title"
              v-model="form.title"
              maxlength="160"
              :aria-invalid="Boolean(errors.title)"
              :aria-describedby="errors.title ? 'ticket-title-error' : undefined"
              placeholder="Ex: Impressora sem tinta"
              class="w-full px-4 py-2.5 rounded-lg border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 transition-shadow text-sm"
            />
            <p v-if="errors.title" id="ticket-title-error" class="field-error">{{ errors.title }}</p>
          </div>

          <div>
            <label for="ticket-description" class="block text-sm font-medium text-slate-700 mb-1">Descrição detalhada</label>
            <textarea
              id="ticket-description"
              required
              name="description"
              v-model="form.description"
              rows="4"
              :aria-invalid="Boolean(errors.description)"
              :aria-describedby="errors.description ? 'ticket-description-error' : undefined"
              placeholder="Descreva o que está acontecendo..."
              class="w-full px-4 py-2.5 rounded-lg border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 transition-shadow resize-none text-sm"
            />
            <p v-if="errors.description" id="ticket-description-error" class="field-error">{{ errors.description }}</p>
          </div>

          <div class="grid grid-cols-1 gap-5" :class="editing ? 'md:grid-cols-3' : 'md:grid-cols-2'">
            <div>
              <label for="ticket-requester" class="block text-sm font-medium text-slate-700 mb-1">Solicitante</label>
              <input
                id="ticket-requester"
                required
                name="requesterName"
                v-model="form.requesterName"
                maxlength="120"
                :aria-invalid="Boolean(errors.requesterName)"
                :aria-describedby="errors.requesterName ? 'ticket-requester-error' : undefined"
                placeholder="Nome do solicitante"
                class="w-full px-4 py-2.5 bg-white rounded-lg border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 transition-shadow text-sm"
              />
              <p v-if="errors.requesterName" id="ticket-requester-error" class="field-error">{{ errors.requesterName }}</p>
            </div>

            <div>
              <label for="ticket-priority" class="block text-sm font-medium text-slate-700 mb-1">Prioridade</label>
              <select
                id="ticket-priority"
                name="priority"
                v-model="form.priority"
                :aria-invalid="Boolean(errors.priority)"
                :aria-describedby="errors.priority ? 'ticket-priority-error' : undefined"
                class="w-full px-4 py-2.5 bg-white rounded-lg border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 transition-shadow text-sm"
              >
                <option value="low">Baixa</option>
                <option value="medium">Média</option>
                <option value="high">Alta</option>
              </select>
              <p v-if="errors.priority" id="ticket-priority-error" class="field-error">{{ errors.priority }}</p>
            </div>

            <div v-if="editing">
              <label for="ticket-status" class="block text-sm font-medium text-slate-700 mb-1">Status</label>
              <select
                id="ticket-status"
                name="status"
                v-model="form.status"
                :aria-invalid="Boolean(errors.status)"
                :aria-describedby="errors.status ? 'ticket-status-error' : undefined"
                class="w-full px-4 py-2.5 bg-white rounded-lg border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 transition-shadow text-sm"
              >
                <option value="open">Aberto</option>
                <option value="in_progress">Em andamento</option>
                <option value="resolved">Resolvido</option>
                <option value="closed">Fechado</option>
              </select>
              <p v-if="errors.status" id="ticket-status-error" class="field-error">{{ errors.status }}</p>
            </div>
          </div>

          <!-- ATRIBUIÇÃO -->
          <div class="bg-slate-50 p-4 rounded-xl border border-slate-200">
            <label class="block text-sm font-medium text-slate-800 mb-3 flex items-center gap-2">
              <UserRound :size="16" class="text-slate-500"/>
              Responsável pelo atendimento
            </label>

            <div class="flex flex-col md:flex-row gap-3">
              <div class="flex-1 space-y-3">
                <label class="flex items-center gap-2 text-sm text-slate-700 cursor-pointer">
                  <input type="radio" v-model="form.assignmentMode" value="automatic" class="w-4 h-4 text-emerald-600 focus:ring-emerald-500 border-gray-300" />
                  Atribuição Inteligente (Balanceada)
                </label>
                <label class="flex items-center gap-2 text-sm text-slate-700 cursor-pointer">
                  <input type="radio" v-model="form.assignmentMode" value="manual" class="w-4 h-4 text-emerald-600 focus:ring-emerald-500 border-gray-300" />
                  Selecionar manualmente
                </label>
              </div>

              <div class="flex-1">
                <select
                  v-if="form.assignmentMode === 'manual'"
                  id="ticket-assignee"
                  name="assigneeId"
                  v-model="form.assigneeId"
                  :aria-invalid="Boolean(errors.assigneeId)"
                  :aria-describedby="errors.assigneeId ? 'ticket-assignee-error' : undefined"
                  class="w-full px-4 py-2.5 bg-white rounded-lg border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 transition-shadow text-sm"
                >
                  <option :value="0" disabled>-- Selecionar responsável --</option>
                  <option v-for="user in assignees" :key="user.id" :value="user.id">{{ user.name }} ({{ user.openTickets }} ativos)</option>
                </select>
                <div v-else class="flex items-center justify-center gap-2 px-4 py-2.5 bg-emerald-100 text-emerald-700 rounded-lg font-medium text-sm">
                  <Wand2 :size="16" />
                  Atribuição Inteligente Ativa
                </div>
              </div>
            </div>
            <p v-if="errors.assigneeId" id="ticket-assignee-error" class="field-error mt-2">
              {{ errors.assigneeId }}
            </p>
            <p v-if="errors.assignmentMode" class="field-error mt-2">{{ errors.assignmentMode }}</p>

            <div v-if="editing && originalAssignmentMode === 'automatic' && form.assignmentMode === 'automatic'" class="mt-4">
              <label class="flex items-center gap-2 text-xs text-slate-600 cursor-pointer">
                <input type="checkbox" v-model="form.redistribute" class="w-3.5 h-3.5 text-emerald-600 focus:ring-emerald-500 border-gray-300 rounded" />
                Redistribuir agora (Recalcular carga do time)
              </label>
            </div>

            <p class="text-[10px] text-slate-500 mt-3">
              A atribuição inteligente avalia a carga de trabalho atual da equipe e escolhe o membro mais disponível.
            </p>
          </div>

          <div v-if="generalError" class="rounded-xl border border-rose-200 bg-rose-50 p-4 text-sm font-medium text-rose-700">
            {{ generalError }}
          </div>
        </form>
      </div>

      <!-- MODAL FOOTER -->
      <div class="px-6 py-4 border-t border-slate-200 bg-slate-50 flex justify-end gap-3">
        <RouterLink
          :to="editing ? `/chamados/${ticketID}` : '/chamados'"
          class="px-5 py-2.5 text-slate-600 font-medium hover:bg-slate-200 rounded-lg transition-colors text-sm"
        >
          Cancelar
        </RouterLink>
        <button
          form="ticket-form"
          type="submit"
          :disabled="saving"
          class="px-5 py-2.5 bg-emerald-600 hover:bg-emerald-700 text-white font-medium rounded-lg shadow-sm transition-colors text-sm flex items-center gap-2"
        >
          <span v-if="saving" class="size-4 animate-spin rounded-full border-2 border-white/40 border-t-white" />
          <Check v-else :size="18" />
          {{ saving ? 'Salvando...' : editing ? 'Salvar Alterações' : 'Criar Chamado' }}
        </button>
      </div>
    </div>
  </div>
</template>
