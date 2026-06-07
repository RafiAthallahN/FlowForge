<script setup lang="ts">
import type { WorkflowRun } from '../types'

const props = defineProps<{
  runs: WorkflowRun[]
  loading: boolean
}>()

const emit = defineEmits<{
  (e: 'select', run: WorkflowRun): void
}>()

function statusBadgeClass(status: string): string {
  const map: Record<string, string> = {
    Pending: 'badge-pending',
    Running: 'badge-running',
    Success: 'badge-success',
    Failed: 'badge-failed',
  }
  return `app-badge ${map[status] || 'badge-pending'}`
}

function formatDuration(run: WorkflowRun): string {
  if (!run.started_at || !run.completed_at) return '—'
  const start = new Date(run.started_at).getTime()
  const end = new Date(run.completed_at).getTime()
  const ms = end - start
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function formatTime(iso: string | null): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <div v-if="loading" class="text-center py-8 text-brand-slate text-sm font-medium animate-pulse select-none">
      Loading runs...
    </div>

    <div v-else-if="runs.length === 0" class="text-center py-10 border border-dashed border-border-subtle/50 rounded-lg flex flex-col items-center gap-2">
      <span class="text-2xl select-none">📋</span>
      <p class="text-brand-slate text-sm font-medium">No runs found yet</p>
    </div>

    <div v-else class="overflow-x-auto w-full">
      <table class="w-full text-left border-collapse">
        <thead>
          <tr class="border-b border-border-subtle select-none">
            <th class="px-4 py-3 text-[10px] font-bold text-brand-slate uppercase tracking-wider">Run ID</th>
            <th class="px-4 py-3 text-[10px] font-bold text-brand-slate uppercase tracking-wider">Version</th>
            <th class="px-4 py-3 text-[10px] font-bold text-brand-slate uppercase tracking-wider">Status</th>
            <th class="px-4 py-3 text-[10px] font-bold text-brand-slate uppercase tracking-wider">Duration</th>
            <th class="px-4 py-3 text-[10px] font-bold text-brand-slate uppercase tracking-wider">Started</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border-subtle/20">
          <tr
            v-for="run in runs"
            :key="run.id"
            class="hover:bg-brand-primary/5 transition-colors cursor-pointer"
            @click="emit('select', run)"
          >
            <td class="px-4 py-3.5 font-mono text-xs max-w-[140px] truncate text-white/95 select-all">{{ run.id }}</td>
            <td class="px-4 py-3.5 text-xs text-white/80 select-none">v{{ run.workflow_version }}</td>
            <td class="px-4 py-3.5 text-xs"><span :class="statusBadgeClass(run.status)">{{ run.status }}</span></td>
            <td class="px-4 py-3.5 text-xs text-white/80 select-none">{{ formatDuration(run) }}</td>
            <td class="px-4 py-3.5 text-xs text-brand-slate/80 select-none">{{ formatTime(run.started_at) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
