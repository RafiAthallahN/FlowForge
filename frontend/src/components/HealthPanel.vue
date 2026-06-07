<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { useRunStore } from '../stores/runs'

const runStore = useRunStore()
const refreshInterval = ref<any>(null)

// Refresh health metrics every 30s
onMounted(async () => {
  await runStore.fetchHealthMetrics()
  refreshInterval.value = setInterval(() => {
    runStore.fetchHealthMetrics()
  }, 30000)
})

onUnmounted(() => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
  }
})

const successPercent = computed(() => {
  if (!runStore.healthMetrics) return 0
  return Math.round(runStore.healthMetrics.success_rate)
})

const avgDurationSec = computed(() => {
  if (!runStore.healthMetrics) return '0.00'
  return (runStore.healthMetrics.avg_duration_ms / 1000).toFixed(2)
})
</script>

<template>
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5" v-if="runStore.healthMetrics">
    <!-- Active Runs Card -->
    <div class="glass-card p-6 flex justify-between items-center select-none">
      <div class="flex flex-col">
        <span class="text-[10px] font-bold uppercase tracking-wider text-brand-slate">Active Executions</span>
        <span class="text-3xl font-extrabold text-brand-primary mt-1.5 leading-none">{{ runStore.healthMetrics.active_runs }}</span>
        <span class="text-[11px] text-brand-slate/60 mt-2">Currently running steps</span>
      </div>
      <div class="w-12 h-12 rounded-md bg-brand-primary/10 flex items-center justify-center shrink-0">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#3b82f6" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="animate-pulse"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline></svg>
      </div>
    </div>

    <!-- Avg Duration Card -->
    <div class="glass-card p-6 flex justify-between items-center select-none">
      <div class="flex flex-col">
        <span class="text-[10px] font-bold uppercase tracking-wider text-brand-slate">Avg Execution Time</span>
        <span class="text-3xl font-extrabold text-brand-purple mt-1.5 leading-none">{{ avgDurationSec }}s</span>
        <span class="text-[11px] text-brand-slate/60 mt-2">Rolling 24h average</span>
      </div>
      <div class="w-12 h-12 rounded-md bg-brand-purple/10 flex items-center justify-center shrink-0">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#8b5cf6" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg>
      </div>
    </div>

    <!-- Success Rate Card -->
    <div class="glass-card p-6 flex justify-between items-center col-span-1 sm:col-span-2">
      <div class="flex flex-col w-full">
        <div class="flex justify-between items-center w-full select-none">
          <span class="text-[10px] font-bold uppercase tracking-wider text-brand-slate">Success Rate</span>
          <span class="text-lg font-extrabold text-brand-green">{{ successPercent }}%</span>
        </div>
        
        <!-- Progress Bar -->
        <div class="w-full h-2 bg-white/5 rounded-full overflow-hidden mt-3 select-none">
          <div class="h-full rounded-full bg-brand-green transition-all duration-500" :style="{ width: `${successPercent}%` }"></div>
        </div>

        <div class="flex justify-between text-brand-slate/70 text-xs mt-3.5 w-full select-none">
          <span>Successes: <strong class="text-white">{{ runStore.healthMetrics.success_count }}</strong></span>
          <span>Failures: <strong class="text-white">{{ runStore.healthMetrics.failure_count }}</strong></span>
        </div>
      </div>
    </div>
  </div>
  <div class="glass-card p-6 text-center text-brand-slate text-sm font-medium animate-pulse" v-else>
    Loading health metrics...
  </div>
</template>
