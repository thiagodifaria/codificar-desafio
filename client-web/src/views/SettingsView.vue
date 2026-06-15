<script setup lang="ts">
import { Check, Pencil, Plus, Settings, Trash2, UserRound, Users } from '@lucide/vue'
import { onMounted, ref } from 'vue'
import { ApiError, api } from '../services/api'
import { showToast } from '../services/toast'
import type { Assignee } from '../types/ticket'

const autoAssignment = ref(localStorage.getItem('autoAssignment') !== 'false')
const team = ref<Assignee[]>([])
const loadingTeam = ref(true)
const newMemberName = ref('')
const editingID = ref(0)
const editingName = ref('')
const savingID = ref(0)
const adding = ref(false)

function toggleAutoAssignment() {
  autoAssignment.value = !autoAssignment.value
  localStorage.setItem('autoAssignment', String(autoAssignment.value))
  showToast('Preferência atualizada.')
}

async function loadTeam() {
  loadingTeam.value = true
  try {
    team.value = await api.assignees()
  } catch {
    showToast('Não foi possível carregar a equipe.', 'error')
  } finally {
    loadingTeam.value = false
  }
}

async function addMember() {
  const name = newMemberName.value.trim()
  if (!name) {
    showToast('Informe o nome do novo membro.', 'error')
    return
  }
  adding.value = true
  try {
    await api.createAssignee({ name })
    newMemberName.value = ''
    await loadTeam()
    showToast('Membro adicionado à equipe.')
  } catch (error) {
    showToast(error instanceof ApiError ? error.message : 'Não foi possível adicionar o membro.', 'error')
  } finally {
    adding.value = false
  }
}

function startEditing(member: Assignee) {
  editingID.value = member.id
  editingName.value = member.name
}

async function saveMember(member: Assignee) {
  const name = editingName.value.trim()
  if (!name) {
    showToast('Informe o nome do membro.', 'error')
    return
  }
  savingID.value = member.id
  try {
    await api.updateAssignee(member.id, { name })
    editingID.value = 0
    await loadTeam()
    showToast('Membro atualizado.')
  } catch (error) {
    showToast(error instanceof ApiError ? error.message : 'Não foi possível atualizar o membro.', 'error')
  } finally {
    savingID.value = 0
  }
}

async function removeMember(member: Assignee) {
  if (!window.confirm(`Remover ${member.name} da equipe?`)) return

  savingID.value = member.id
  try {
    await api.deleteAssignee(member.id)
    await loadTeam()
    showToast('Membro removido da equipe.')
  } catch (error) {
    showToast(error instanceof ApiError ? error.message : 'Não foi possível remover o membro.', 'error')
  } finally {
    savingID.value = 0
  }
}

onMounted(loadTeam)
</script>

<template>
  <section class="max-w-5xl">
    <div>
      <p class="text-sm font-semibold text-emerald-700">Administração</p>
      <h1 class="mt-1 text-3xl font-bold tracking-tight text-slate-950">Configurações</h1>
      <p class="mt-2 text-sm text-slate-500">Gerencie o comportamento da operação e os membros da equipe.</p>
    </div>

    <div class="mt-8 grid gap-6">
      <section class="panel overflow-hidden">
        <header class="flex items-center gap-3 border-b border-slate-100 px-6 py-5">
          <span class="grid size-10 place-items-center rounded-xl bg-emerald-50 text-emerald-700">
            <Settings :size="19" />
          </span>
          <div>
            <h2 class="font-bold text-slate-900">Preferências deste navegador</h2>
            <p class="text-sm text-slate-500">Ajustes salvos somente neste navegador.</p>
          </div>
        </header>

        <div class="flex items-center justify-between gap-6 px-6 py-5">
          <div>
            <h3 class="font-medium text-slate-800">Usar atribuição automática ao criar chamados</h3>
            <p class="mt-1 text-sm text-slate-500">Preferência local para priorizar o balanceamento de carga no formulário.</p>
          </div>
          <button
            type="button"
            role="switch"
            :aria-checked="autoAssignment"
            class="flex h-6 w-12 shrink-0 items-center rounded-full px-1 transition-colors"
            :class="autoAssignment ? 'bg-emerald-700' : 'bg-slate-200'"
            @click="toggleAutoAssignment"
          >
            <span class="size-4 rounded-full bg-white shadow-sm transition-transform" :class="autoAssignment ? 'translate-x-6' : ''" />
          </button>
        </div>
      </section>

      <section class="panel overflow-hidden">
        <header class="flex flex-col gap-4 border-b border-slate-100 px-6 py-5 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex items-center gap-3">
            <span class="grid size-10 place-items-center rounded-xl bg-emerald-50 text-emerald-700">
              <Users :size="19" />
            </span>
            <div>
              <h2 class="font-bold text-slate-900">Gestão da equipe</h2>
              <p class="text-sm text-slate-500">Cadastre, renomeie ou remova membros sem histórico de chamados.</p>
            </div>
          </div>

          <form class="flex w-full gap-2 sm:w-auto" @submit.prevent="addMember">
            <input v-model="newMemberName" class="field min-w-0 sm:w-64" maxlength="120" placeholder="Nome do novo membro" />
            <button class="button-primary shrink-0" :disabled="adding">
              <Plus :size="17" />
              Adicionar
            </button>
          </form>
        </header>

        <div v-if="loadingTeam" class="p-6 text-sm text-slate-500">Carregando equipe...</div>
        <div v-else class="divide-y divide-slate-100">
          <article v-for="member in team" :key="member.id" class="flex flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center">
            <span class="grid size-10 shrink-0 place-items-center rounded-xl bg-emerald-50 text-emerald-700">
              <UserRound :size="19" />
            </span>

            <div class="min-w-0 flex-1">
              <div v-if="editingID === member.id" class="flex max-w-md gap-2">
                <input v-model="editingName" class="field py-2" maxlength="120" @keyup.enter="saveMember(member)" />
                <button class="button-primary px-3" :disabled="savingID === member.id" @click="saveMember(member)">
                  <Check :size="17" />
                </button>
              </div>
              <template v-else>
                <h3 class="font-semibold text-slate-800">{{ member.name }}</h3>
                <p class="mt-1 text-xs text-slate-500">{{ member.openTickets }} chamado(s) ativo(s)</p>
              </template>
            </div>

            <div class="flex gap-2">
              <button class="button-secondary px-3" :disabled="savingID === member.id" @click="startEditing(member)">
                <Pencil :size="16" />
                Renomear
              </button>
              <button class="button-secondary px-3 !text-rose-700" :disabled="savingID === member.id" @click="removeMember(member)">
                <Trash2 :size="16" />
                Remover
              </button>
            </div>
          </article>

          <div v-if="!team.length" class="px-6 py-10 text-center text-sm text-slate-500">
            Nenhum membro cadastrado.
          </div>
        </div>
      </section>
    </div>
  </section>
</template>
