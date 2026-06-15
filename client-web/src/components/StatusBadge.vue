<script setup lang="ts">
import { AlertCircle, CheckCircle, Clock, X } from '@lucide/vue'
import { computed } from 'vue'
import { priorityLabel, statusLabel } from '../services/format'
import type { Priority, TicketStatus } from '../types/ticket'

const props = defineProps<{
  status?: TicketStatus
  priority?: Priority
}>()

const label = computed(() =>
  props.status ? statusLabel[props.status] : priorityLabel[props.priority!],
)

const icon = computed(() => {
  if (props.priority) return null
  return {
    open: AlertCircle,
    in_progress: Clock,
    resolved: CheckCircle,
    closed: X,
  }[props.status!]
})

const classes = computed(() => {
  const value = props.status ?? props.priority
  return {
    open: 'bg-yellow-100 text-yellow-800 border-yellow-200',
    in_progress: 'bg-blue-100 text-blue-800 border-blue-200',
    resolved: 'bg-emerald-100 text-emerald-800 border-emerald-200',
    closed: 'bg-gray-100 text-gray-800 border-gray-200',
    low: 'text-emerald-600 bg-emerald-50 border-emerald-100',
    medium: 'text-orange-600 bg-orange-50 border-orange-100',
    high: 'text-rose-600 bg-rose-50 border-rose-100',
  }[value!]
})
</script>

<template>
  <span
    class="inline-flex items-center rounded-full px-2.5 py-1 text-[10px] font-medium border"
    :class="classes"
  >
    <component v-if="icon" :is="icon" :size="12" class="mr-1.5" />
    {{ label }}
  </span>
</template>

