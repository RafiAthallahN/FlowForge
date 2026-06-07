<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useWorkflowStore } from '../stores/workflows'
import HealthPanel from '../components/HealthPanel.vue'

const workflowStore = useWorkflowStore()
const page = ref(1)

// New Workflow form fields
const showCreateModal = ref(false)
const newId = ref('')
const newName = ref('')
const newDefinition = ref(JSON.stringify({
  steps: [
    {
      id: "step-1",
      type: "delay",
      config: 10000000 // 10ms delay in nanoseconds (duration)
    },
    {
      id: "step-2",
      type: "default",
      depends_on: ["step-1"]
    }
  ]
}, null, 2))

const createError = ref<string | null>(null)

onMounted(async () => {
  await workflowStore.fetchWorkflows(page.value, 10)
})

// Client-side DAG Topological check helper before submission
function validateDagLocally(defStr: string): string | null {
  try {
    const parsed = JSON.parse(defStr)
    if (!parsed || !Array.isArray(parsed.steps)) {
      return "Definition must contain a 'steps' array."
    }
    
    // Check circular dependencies (topological sort)
    const inDegree = new Map<string, number>()
    const adjList = new Map<string, string[]>()
    const allSteps = new Set<string>()

    for (const step of parsed.steps) {
      if (!step.id) return "Every step must have an 'id'."
      allSteps.add(step.id)
      inDegree.set(step.id, 0)
      adjList.set(step.id, [])
    }

    for (const step of parsed.steps) {
      const deps = step.depends_on || []
      for (const dep of deps) {
        if (!allSteps.has(dep)) {
          return `Dependency '${dep}' for step '${step.id}' does not exist.`
        }
        adjList.get(dep)!.push(step.id)
        inDegree.set(step.id, inDegree.get(step.id)! + 1)
      }
    }

    // Queue for BFS
    const queue: string[] = []
    for (const id of allSteps) {
      if (inDegree.get(id) === 0) {
        queue.push(id)
      }
    }

    let visitedCount = 0
    while (queue.length > 0) {
      const u = queue.shift()!
      visitedCount++
      for (const v of adjList.get(u) || []) {
        inDegree.set(v, inDegree.get(v)! - 1)
        if (inDegree.get(v) === 0) {
          queue.push(v)
        }
      }
    }

    if (visitedCount !== allSteps.size) {
      return "Circular dependency detected. Workflow definition must be a valid DAG."
    }

    return null
  } catch (err: any) {
    return "Invalid JSON syntax: " + err.message
  }
}

async function handleCreateWorkflow() {
  createError.value = null
  
  // Local validation
  const validationError = validateDagLocally(newDefinition.value)
  if (validationError) {
    createError.value = validationError
    return
  }

  try {
    await workflowStore.createWorkflow({
      id: newId.value,
      name: newName.value,
      definition: newDefinition.value
    })
    
    // Reset fields
    newId.value = ''
    newName.value = ''
    showCreateModal.value = false
    
    // Refresh workflows list
    await workflowStore.fetchWorkflows(page.value, 10)
  } catch (err: any) {
    createError.value = err.response?.data?.error || err.message
  }
}
</script>

<template>
  <div class="max-w-[1400px] mx-auto px-6 py-8 animate-fade-in flex flex-col gap-8">
    <!-- Top Health Panel -->
    <div class="flex flex-col gap-4">
      <h2 class="text-lg font-bold text-white/90 uppercase tracking-wider select-none">Platform Health Summary</h2>
      <HealthPanel />
    </div>

    <!-- Workflows Section -->
    <div class="glass-card p-6 flex flex-col gap-6">
      <div class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 border-b border-border-subtle pb-6">
        <div>
          <h1 class="text-2xl font-extrabold tracking-tight">Workflows</h1>
          <p class="text-brand-slate text-sm mt-1 select-none">Manage and monitor your pipeline definitions</p>
        </div>
        <button @click="showCreateModal = true" class="btn-primary">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
          Create Workflow
        </button>
      </div>

      <!-- Workflows List -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5" v-if="workflowStore.workflows.length > 0">
        <div 
          v-for="wf in workflowStore.workflows" 
          :key="wf.id" 
          class="glass-card bg-[#111123]/35 p-6 flex flex-col justify-between h-[210px] hover:-translate-y-1 hover:bg-[#191932]/50 transition-all duration-300"
        >
          <div>
            <div class="flex justify-between items-start gap-3">
              <h3 class="font-bold text-base truncate flex-1 text-white/90">{{ wf.name }}</h3>
              <span class="app-badge badge-pending shrink-0">v{{ wf.version }}</span>
            </div>
            <p class="text-brand-slate text-xs font-mono mt-3 select-all">ID: {{ wf.id }}</p>
            <p class="text-brand-slate/50 text-[11px] mt-1 select-none">Updated: {{ new Date(wf.updated_at).toLocaleString() }}</p>
          </div>

          <div class="flex justify-end items-center mt-4 border-t border-border-subtle/50 pt-4">
            <router-link :to="`/workflows/${wf.id}`" class="btn-secondary btn-sm select-none">
              Open Details
            </router-link>
          </div>
        </div>
      </div>

      <div class="py-16 text-center text-brand-slate select-none text-sm font-medium animate-pulse" v-else-if="workflowStore.loading">
        Loading workflows...
      </div>

      <div class="py-16 text-center text-brand-slate select-none text-sm font-medium border-2 border-dashed border-border-subtle rounded-lg" v-else>
        No workflows found. Create one to get started!
      </div>
    </div>

    <!-- Create Workflow Modal -->
    <div class="fixed inset-0 bg-[#05050f]/85 backdrop-blur-sm flex items-center justify-center z-50 p-6" v-if="showCreateModal">
      <div class="glass-card w-full max-w-[650px] p-8 animate-fade-in flex flex-col gap-6">
        <div class="flex justify-between items-center border-b border-border-subtle pb-4">
          <h2 class="text-xl font-bold">Create Workflow Definition</h2>
          <button @click="showCreateModal = false" class="text-brand-slate hover:text-white transition-colors cursor-pointer">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
          </button>
        </div>

        <form @submit.prevent="handleCreateWorkflow" class="flex flex-col gap-4">
          <div class="flex flex-col gap-1.5">
            <label for="wf-id" class="text-xs font-semibold text-brand-slate uppercase tracking-wider">Workflow ID</label>
            <input 
              id="wf-id" 
              type="text" 
              v-model="newId" 
              class="app-input" 
              placeholder="e.g. process-orders" 
              required
            />
          </div>

          <div class="flex flex-col gap-1.5">
            <label for="wf-name" class="text-xs font-semibold text-brand-slate uppercase tracking-wider">Workflow Name</label>
            <input 
              id="wf-name" 
              type="text" 
              v-model="newName" 
              class="app-input" 
              placeholder="e.g. Order Processing Pipeline" 
              required
            />
          </div>

          <div class="flex flex-col gap-1.5">
            <label for="wf-definition" class="text-xs font-semibold text-brand-slate uppercase tracking-wider">Workflow Definition (JSON DAG)</label>
            <textarea 
              id="wf-definition" 
              v-model="newDefinition" 
              class="app-input font-mono text-xs leading-relaxed min-h-[180px] max-h-[300px] resize-y bg-[#090919]" 
              rows="8" 
              required
            ></textarea>
          </div>

          <div class="p-3.5 bg-brand-red/10 border border-brand-red/20 text-brand-red text-xs rounded-md font-medium" v-if="createError">
            {{ createError }}
          </div>

          <div class="flex justify-end gap-3 mt-4 border-t border-border-subtle/50 pt-4">
            <button type="button" @click="showCreateModal = false" class="btn-secondary">
              Cancel
            </button>
            <button type="submit" class="btn-primary">
              Create Definition
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
