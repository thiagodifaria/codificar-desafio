<script setup lang="ts">
import { CheckCircle2, X, XCircle } from '@lucide/vue'
import { toastState } from '../services/toast'
</script>

<template>
  <Transition
    enter-active-class="transition duration-200"
    enter-from-class="translate-y-3 opacity-0"
    leave-active-class="transition duration-150"
    leave-to-class="translate-y-3 opacity-0"
  >
    <div
      v-if="toastState.visible"
      class="fixed bottom-5 right-5 z-50 flex max-w-sm items-center gap-3 rounded-2xl border bg-white p-4 shadow-xl"
      :class="toastState.tone === 'success' ? 'border-emerald-200' : 'border-rose-200'"
      role="status"
    >
      <CheckCircle2 v-if="toastState.tone === 'success'" :size="21" class="shrink-0 text-emerald-600" />
      <XCircle v-else :size="21" class="shrink-0 text-rose-600" />
      <p class="text-sm font-semibold text-slate-700">{{ toastState.message }}</p>
      <button class="ml-2 rounded-lg p-1 text-slate-400 hover:bg-slate-100" aria-label="Fechar aviso" @click="toastState.visible = false">
        <X :size="16" />
      </button>
    </div>
  </Transition>
</template>

