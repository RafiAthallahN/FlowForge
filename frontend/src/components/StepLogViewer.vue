<script setup lang="ts">
import { ref } from 'vue'
import type { ExecutionLog } from '../types'

defineProps<{
  logs: ExecutionLog[]
}>()

const expandedSteps = ref<Set<string>>(new Set())

function toggle(stepId: string) {
  if (expandedSteps.value.has(stepId)) {
    expandedSteps.value.delete(stepId)
  } else {
    expandedSteps.value.add(stepId)
  }
  expandedSteps.value = new Set(expandedSteps.value)
}

function statusBadgeClass(status: string): string {
  const map: Record<string, string> = {
    Pending: 'badge-pending',
    Running: 'badge-running',
    Success: 'badge-success',
    Failed: 'badge-failed',
  }
  return `app-badge ${map[status] || 'badge-pending'}`
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <div v-if="logs.length === 0" class="text-center py-12 text-brand-slate text-sm font-medium select-none">
      No execution logs available
    </div>

    <div v-else class="flex flex-col gap-3">
      <div
        v-for="log in logs"
        :key="log.id"
        class="glass-card bg-[#111123]/25 overflow-hidden transition-all duration-300"
        :class="{ '!bg-[#191932]/45': expandedSteps.has(log.step_id) }"
      >
        <!-- Log Header (Clickable Accordion) -->
        <div class="flex justify-between items-center px-4 py-3.5 cursor-pointer hover:bg-brand-primary/5 transition-colors select-none" @click="toggle(log.step_id)">
          <div class="flex items-center gap-3">
            <span class="text-sm font-bold font-mono text-white/90">{{ log.step_id }}</span>
            <span :class="statusBadgeClass(log.status)">{{ log.status }}</span>
          </div>
          <div class="flex items-center gap-4 text-xs text-brand-slate">
            <span class="flex items-center gap-1">⏱️ {{ log.duration_ms }}ms</span>
            <span class="flex items-center gap-1" v-if="log.retry_count > 0">🔄 {{ log.retry_count }} retries</span>
            <span class="text-[10px] shrink-0 ml-1.5 transition-transform duration-200" :class="{ 'rotate-180': expandedSteps.has(log.step_id) }">▼</span>
          </div>
        </div>

        <!-- Log Body -->
        <div v-if="expandedSteps.has(log.step_id)" class="px-4 pb-4 animate-fade-in flex flex-col gap-4">
          <!-- AI Diagnostics Insight Box -->
          <div v-if="log.failure_reason || log.suggested_fix" class="bg-brand-purple/5 border border-brand-purple/20 p-4 rounded-md flex flex-col gap-3.5 animate-fade-in shadow-inner">
            <div class="inline-flex items-center gap-1.5 bg-brand-purple/15 border border-brand-purple/30 px-3 py-1 rounded-full w-fit select-none">
              <span class="text-xs">✨</span>
              <span class="text-[9px] font-bold text-brand-purple uppercase tracking-wider">AI DIAGNOSTIC INSIGHT</span>
            </div>
            
            <div class="flex flex-col" v-if="log.failure_reason">
              <span class="text-[10px] font-bold text-brand-purple/80 uppercase tracking-wider mb-1 select-none">Root Cause</span>
              <p class="text-xs text-white/95 bg-slate-950/45 p-3 rounded border-l-2 border-brand-purple leading-relaxed">{{ log.failure_reason }}</p>
            </div>
            
            <div class="flex flex-col" v-if="log.suggested_fix">
              <span class="text-[10px] font-bold text-brand-purple/80 uppercase tracking-wider mb-1 select-none">Suggested Fix</span>
              <p class="text-xs text-white/95 bg-slate-950/45 p-3 rounded border-l-2 border-brand-purple leading-relaxed">{{ log.suggested_fix }}</p>
            </div>
          </div>

          <!-- Log Output Console -->
          <pre class="bg-slate-950/50 border border-border-subtle/50 rounded p-4 text-[11px] font-mono text-brand-green/90 overflow-x-auto whitespace-pre-wrap word-break-all max-h-[300px] overflow-y-auto leading-relaxed shadow-inner">{{ log.log_output || 'No output' }}</pre>
        </div>
      </div>
    </div>
  </div>
</template>
