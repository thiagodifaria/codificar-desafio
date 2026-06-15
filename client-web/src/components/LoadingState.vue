<script setup lang="ts">
withDefaults(defineProps<{ variant?: 'default' | 'dashboard' | 'list' }>(), {
  variant: 'default',
})
</script>

<template>
  <div v-if="variant === 'dashboard'" class="mt-8 animate-pulse space-y-6" aria-label="Carregando visão geral">
    <div class="grid gap-4 md:grid-cols-3">
      <div v-for="item in 3" :key="item" class="panel h-36 bg-slate-50" />
    </div>
    <div class="grid gap-6 xl:grid-cols-[1.45fr_1fr]">
      <div class="panel h-96 bg-slate-50" />
      <div class="panel h-96 bg-slate-50" />
    </div>
  </div>
  <div v-else-if="variant === 'list'" class="panel mt-6 animate-pulse divide-y divide-slate-100 overflow-hidden" aria-label="Carregando chamados">
    <div v-for="item in 6" :key="item" class="flex items-center gap-4 px-6 py-5">
      <span class="size-10 rounded-xl bg-slate-100" />
      <span class="h-4 flex-1 rounded bg-slate-100" />
      <span class="h-7 w-24 rounded-full bg-slate-100" />
    </div>
  </div>
  <div v-else class="flex min-h-64 items-center justify-center" aria-live="polite">
    <div class="flex items-center gap-3 text-sm font-medium text-slate-500">
      <span class="size-5 animate-spin rounded-full border-2 border-slate-200 border-t-emerald-600" />
      Carregando informações
    </div>
  </div>
</template>
