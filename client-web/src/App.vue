<script setup lang="ts">
import { LayoutDashboard, Menu, TicketCheck, Users, Settings, X } from '@lucide/vue'
import { ref, watch } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import ToastHost from './components/ToastHost.vue'

const route = useRoute()
const menuOpen = ref(false)
watch(() => route.fullPath, () => {
  menuOpen.value = false
})

const navigation = [
  { name: 'Dashboard', to: '/', icon: LayoutDashboard },
  { name: 'Chamados', to: '/chamados', icon: TicketCheck },
  { name: 'Equipe', to: '/equipe', icon: Users },
]
</script>

<template>
  <div class="min-h-screen bg-[#f6f7f5] flex font-sans text-slate-800">
    <!-- SIDEBAR -->
    <aside
      class="fixed inset-y-0 left-0 z-40 w-64 transform bg-[#101c16] border-r border-white/5 text-white transition-transform duration-200 lg:translate-x-0 flex flex-col"
      :class="menuOpen ? 'translate-x-0' : '-translate-x-full'"
    >
      <div class="p-6 flex items-center justify-between">
        <RouterLink to="/" class="flex items-center gap-3">
          <div class="w-9 h-9 bg-emerald-400 rounded-xl flex items-center justify-center text-[#101c16]">
            <TicketCheck :size="21" />
          </div>
          <span class="text-xl font-bold text-white">Codificar Desk</span>
        </RouterLink>
        <button class="rounded-lg p-2 text-white/60 lg:hidden" aria-label="Fechar menu" @click="menuOpen = false">
          <X :size="20" />
        </button>
      </div>

      <nav class="flex-1 px-4 space-y-1 mt-4" aria-label="Navegação principal">
        <RouterLink
          v-for="item in navigation"
          :key="item.name"
          :to="item.to"
          class="flex items-center gap-3 px-3 py-2.5 rounded-lg font-medium transition-colors text-white/60 hover:bg-white/5 hover:text-white"
          active-class="!bg-white/10 !text-white"
        >
          <component :is="item.icon" :size="18" />
          {{ item.name }}
        </RouterLink>
      </nav>

      <div class="p-4 border-t border-white/10">
        <RouterLink
          to="/configuracoes"
          class="flex items-center gap-3 px-3 py-2 rounded-lg font-medium transition-colors text-white/60 hover:bg-white/5 hover:text-white"
          active-class="!bg-white/10 !text-white"
        >
          <Settings :size="18" />
          Configurações
        </RouterLink>
      </div>
    </aside>

    <div v-if="menuOpen" class="fixed inset-0 z-30 bg-slate-950/40 backdrop-blur-sm lg:hidden" @click="menuOpen = false" />

    <div class="flex-1 flex flex-col min-w-0 lg:pl-64">
      <header class="h-16 bg-[#f6f7f5]/90 backdrop-blur border-b border-slate-200/80 flex items-center px-6 flex-shrink-0 sticky top-0 z-20">
        <div class="flex items-center gap-4">
          <button class="rounded-xl border border-slate-200 bg-white p-2 text-slate-700 lg:hidden" aria-label="Abrir menu" @click="menuOpen = true">
            <Menu :size="20" />
          </button>
          <h1 class="text-xl font-semibold capitalize text-slate-800">
            {{ route.name === 'dashboard' ? 'Dashboard' : route.name === 'tickets' ? 'Chamados' : route.name === 'team' ? 'Equipe' : route.name === 'settings' ? 'Configurações' : 'Codificar Desk' }}
          </h1>
        </div>
      </header>

      <main class="flex-1 overflow-y-auto p-6 md:p-8">
        <RouterView />
      </main>
    </div>
    <ToastHost />
  </div>
</template>
