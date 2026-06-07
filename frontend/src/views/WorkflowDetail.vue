<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useWorkflowStore } from '../stores/workflows'
import { useRunStore } from '../stores/runs'
import { useSSE } from '../composables/useSSE'
import DagViewer from '../components/DagViewer.vue'
import RunHistoryPanel from '../components/RunHistoryPanel.vue'
import StepLogViewer from '../components/StepLogViewer.vue'

const route = useRoute()
const workflowStore = useWorkflowStore()
const runStore = useRunStore()
const { connected, connect, disconnect } = useSSE()

const workflowId = route.params.id as string
const selectedVersion = ref<number | null>(null)
const page = ref(1)

// Fetch data on mount
onMounted(async () => {
  await workflowStore.fetchWorkflow(workflowId)
  if (workflowStore.selectedWorkflow) {
    selectedVersion.value = workflowStore.selectedWorkflow.version
  }
  await runStore.fetchRuns(workflowId, page.value, 10)
  
  // Connect to SSE for real-time updates
  connect()
})

onUnmounted(() => {
  disconnect()
  runStore.clearLiveEvents()
})

// Watch selected run status
watch(() => runStore.selectedRun?.status, (newStatus) => {
  if (newStatus === 'Running') {
    // Keep checking runs list for updates
    runStore.fetchRuns(workflowId, page.value, 10)
  }
})

// Watch page change
watch(page, (newPage) => {
  runStore.fetchRuns(workflowId, newPage, 10)
})

// Compute merged step statuses for DAG viewer
const stepStatuses = computed(() => {
  const statusMap = new Map<string, string>()
  
  // Set all steps to Pending initially
  if (workflowStore.selectedWorkflow) {
    try {
      const def = JSON.parse(workflowStore.selectedWorkflow.definition)
      if (def && def.steps) {
        for (const step of def.steps) {
          statusMap.set(step.id, 'Pending')
        }
      }
    } catch (e) {
      // Ignored
    }
  }

  // Overlay historical logs if a run is selected
  if (runStore.selectedRun && runStore.runLogs) {
    for (const log of runStore.runLogs) {
      statusMap.set(log.step_id, log.status)
    }
  }

  // Overlay live SSE events if currently running
  if (runStore.selectedRun && runStore.selectedRun.status === 'Running') {
    for (const [stepId, event] of runStore.liveStepEvents.entries()) {
      if (event.run_id === runStore.selectedRun.id) {
        statusMap.set(stepId, event.status)
      }
    }
  }

  return statusMap
})

// Handle triggering a run
async function triggerRun() {
  if (!workflowStore.selectedWorkflow) return
  try {
    const run = await workflowStore.runWorkflow(workflowStore.selectedWorkflow.id)
    runStore.clearLiveEvents()
    // Poll/Refetch runs
    await runStore.fetchRuns(workflowId, page.value, 10)
    // Select the newly created run
    await runStore.fetchRun(run.id)
  } catch (err) {
    console.error('Failed to trigger run:', err)
  }
}

// Handle rollback
async function rollbackToVersion() {
  if (!workflowStore.selectedWorkflow || !selectedVersion.value) return
  if (selectedVersion.value === workflowStore.selectedWorkflow.version) return
  
  if (confirm(`Are you sure you want to rollback to version ${selectedVersion.value}?`)) {
    try {
      await workflowStore.rollbackWorkflow(workflowStore.selectedWorkflow.id, selectedVersion.value)
      selectedVersion.value = workflowStore.selectedWorkflow.version
      alert('Rollback successful!')
    } catch (err) {
      alert('Rollback failed')
    }
  }
}

// Handle run row selection
async function handleSelectRun(run: any) {
  await runStore.fetchRun(run.id)
}
</script>

<template>
  <div class="max-w-[1450px] mx-auto px-6 py-8 animate-fade-in flex flex-col gap-6" v-if="workflowStore.selectedWorkflow">
    <!-- Header Card -->
    <div class="glass-card p-6 flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
      <div>
        <div class="flex items-center gap-3">
          <router-link to="/" class="text-brand-slate hover:text-white transition-colors cursor-pointer inline-flex items-center">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="19" y1="12" x2="5" y2="12"></line><polyline points="12 19 5 12 12 5"></polyline></svg>
          </router-link>
          <h1 class="text-2xl font-extrabold tracking-tight">{{ workflowStore.selectedWorkflow.name }}</h1>
          <span class="app-badge badge-pending">v{{ workflowStore.selectedWorkflow.version }}</span>
        </div>
        <p class="text-brand-slate text-sm mt-1 select-none">
          ID: <span class="font-mono text-xs select-all">{{ workflowStore.selectedWorkflow.id }}</span> • Created: {{ new Date(workflowStore.selectedWorkflow.created_at).toLocaleString() }}
        </p>
      </div>

      <div class="flex items-center gap-4">
        <!-- Version Rollback Dropdown -->
        <div class="flex items-center gap-2">
          <label for="version-dropdown" class="text-brand-slate text-xs font-semibold uppercase tracking-wider select-none">Version:</label>
          <select 
            id="version-dropdown" 
            v-model="selectedVersion" 
            class="app-input w-auto !py-1.5 cursor-pointer bg-bg-deep"
            @change="rollbackToVersion"
          >
            <option 
              v-for="v in workflowStore.selectedWorkflow.version" 
              :key="v" 
              :value="v"
            >
              v{{ v }}
            </option>
          </select>
        </div>

        <button 
          @click="triggerRun" 
          class="btn-primary" 
          :disabled="workflowStore.loading"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>
          Run Workflow
        </button>
      </div>
    </div>

    <!-- Main Workspace Layout Grid -->
    <div class="grid grid-cols-1 lg:grid-cols-[3fr_2fr] gap-6">
      <!-- Left Column: DAG Graph -->
      <div class="flex flex-col gap-4">
        <div class="glass-card p-4 flex justify-between items-center select-none">
          <span class="font-bold text-sm tracking-wide text-white/95">DAG Visualization</span>
          <div class="flex flex-wrap items-center gap-3.5 text-xs text-brand-slate">
            <span class="flex items-center gap-1.5"><span class="w-2.5 h-2.5 rounded-full inline-block bg-brand-slate"></span> Pending</span>
            <span class="flex items-center gap-1.5"><span class="w-2.5 h-2.5 rounded-full inline-block bg-brand-primary animate-pulse"></span> Running</span>
            <span class="flex items-center gap-1.5"><span class="w-2.5 h-2.5 rounded-full inline-block bg-brand-green"></span> Success</span>
            <span class="flex items-center gap-1.5"><span class="w-2.5 h-2.5 rounded-full inline-block bg-brand-red"></span> Failed</span>
            <span class="flex items-center gap-2 border-l border-border-subtle pl-3.5">
              <span class="w-2 h-2 rounded-full inline-block" :class="connected ? 'bg-brand-green shadow-lg shadow-brand-green/50 animate-pulse' : 'bg-brand-red shadow-lg shadow-brand-red/50'"></span>
              {{ connected ? 'Live' : 'Offline' }}
            </span>
          </div>
        </div>
        <DagViewer 
          :definition="workflowStore.selectedWorkflow.definition" 
          :step-statuses="stepStatuses"
        />
      </div>

      <!-- Right Column: Runs & Logs -->
      <div class="flex flex-col gap-6">
        <!-- Runs History -->
        <div class="glass-card p-6 flex flex-col">
          <h2 class="text-base font-bold text-white/90 mb-4 select-none">Run History</h2>
          <RunHistoryPanel 
             :runs="runStore.runs" 
             :loading="runStore.loading"
             @select="handleSelectRun"
          />
        </div>

        <!-- Execution Step Logs -->
        <div class="glass-card p-6 flex flex-col" v-if="runStore.selectedRun">
          <div class="flex justify-between items-center mb-4 pb-4 border-b border-border-subtle/50">
            <div>
              <h2 class="text-base font-bold text-white/90">Execution Logs</h2>
              <p class="text-brand-slate text-xs mt-1 select-all">Run: {{ runStore.selectedRun.id }}</p>
            </div>
            <span 
              class="app-badge" 
              :class="`badge-${runStore.selectedRun.status.toLowerCase()}`"
            >
              {{ runStore.selectedRun.status }}
            </span>
          </div>
          <StepLogViewer :logs="runStore.runLogs" />
        </div>
        
        <div class="glass-card p-6 flex items-center justify-center text-brand-slate text-sm font-medium h-48 border-2 border-dashed border-border-subtle/70 rounded-lg text-center" v-else>
          Select a run from the history to view execution details and logs.
        </div>
      </div>
    </div>
  </div>

  <div class="max-w-[1450px] mx-auto px-6 py-16 text-center text-brand-slate font-medium animate-pulse" v-else-if="workflowStore.loading">
    Loading workflow details...
  </div>
  <div class="max-w-[1450px] mx-auto px-6 py-16 text-center text-brand-slate font-medium border border-border-subtle rounded-lg" v-else>
    Workflow not found or failed to load.
  </div>
</template>
