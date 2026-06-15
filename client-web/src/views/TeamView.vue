<script setup lang="ts">
import { UserRound } from '@lucide/vue'
import { onMounted, ref } from 'vue'
import LoadingState from '../components/LoadingState.vue'
import { api } from '../services/api'
import type { Assignee } from '../types/ticket'

const team = ref<Assignee[]>([])
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const dashboard = await api.dashboard()
    team.value = dashboard.assignees
  } catch {
    error.value = 'Não foi possível carregar as informações da equipe.'
  } finally {
    loading.value = false
  }
}

onMounted(load)

</script>

<template>
  <section>
    <div>
      <p class="text-sm font-semibold text-emerald-700">Recursos humanos</p>
      <h1 class="mt-1 text-3xl font-bold tracking-tight text-slate-950">Equipe</h1>
      <p class="mt-2 text-sm text-slate-500">Acompanhe a carga de trabalho de cada membro.</p>
    </div>

    <LoadingState v-if="loading" variant="list" />

    <div v-else-if="error" class="panel mt-8 p-8 text-center">
      <p class="font-semibold text-slate-800">{{ error }}</p>
      <button class="button-secondary mt-4" @click="load">Tentar novamente</button>
    </div>

    <div v-else class="mt-8 grid grid-cols-1 gap-6 md:grid-cols-3">
      <div v-for="member in team" :key="member.id" class="bg-white rounded-xl border border-slate-200 p-6 shadow-sm flex flex-col items-center text-center hover:border-emerald-200 transition-colors">
        <div class="mb-4 grid size-16 place-items-center rounded-xl bg-emerald-50 text-emerald-700">
          <UserRound :size="30" />
        </div>
        <h3 class="font-bold text-lg text-slate-800">{{ member.name }}</h3>
        <p class="text-sm text-slate-500 mb-6">Membro da Equipe</p>

        <div class="w-full grid grid-cols-2 gap-4 border-t border-slate-100 pt-5 mt-auto">
          <div class="flex flex-col items-center">
            <div class="text-2xl font-bold text-emerald-600">{{ member.openTickets }}</div>
            <div class="text-[10px] text-slate-500 uppercase tracking-wider font-semibold">Em Fila</div>
          </div>
          <div class="flex flex-col items-center">
            <div class="text-2xl font-bold text-slate-600">{{ member.completedTickets }}</div>
            <div class="text-[10px] text-slate-500 uppercase tracking-wider font-semibold">Concluídos</div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
