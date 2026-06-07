<script setup lang="ts">
import { computed } from 'vue'
import { VueFlow, type Node, type Edge } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'

interface StepDef {
  id: string
  type?: string
  depends_on?: string[]
}

interface WorkflowDef {
  steps: StepDef[]
}

const props = defineProps<{
  definition: string
  stepStatuses?: Map<string, string>
}>()

// Deep, modern status colors for dark-mode cards
const statusColors: Record<string, string> = {
  Pending: '#1e293b', // slate-800
  Running: '#1e3a8a', // blue-900
  Success: '#064e3b', // green-900
  Failed: '#7f1d1d', // red-900
}

const statusBorderColors: Record<string, string> = {
  Pending: '#475569', // slate-600
  Running: '#3b82f6', // blue-500
  Success: '#22c55e', // green-500
  Failed: '#ef4444', // red-500
}

const parsedDef = computed<WorkflowDef>(() => {
  try {
    return JSON.parse(props.definition)
  } catch {
    return { steps: [] }
  }
})

const nodes = computed<Node[]>(() => {
  const steps = parsedDef.value.steps || []
  const cols = Math.ceil(Math.sqrt(steps.length))

  return steps.map((step, i) => {
    const status = props.stepStatuses?.get(step.id) || 'Pending'
    const col = i % cols
    const row = Math.floor(i / cols)

    return {
      id: step.id,
      position: { x: col * 220 + 40, y: row * 120 + 40 },
      data: {
        label: step.id,
        type: step.type || 'default',
        status,
      },
      style: {
        background: statusColors[status] || statusColors.Pending,
        color: '#e8e8ff',
        border: `2px solid ${statusBorderColors[status] || statusBorderColors.Pending}`,
        borderRadius: '8px',
        padding: '14px 20px',
        fontSize: '12px',
        fontWeight: '700',
        fontFamily: "'Inter', sans-serif",
        boxShadow: status === 'Running'
          ? `0 0 24px rgba(59, 130, 246, 0.4)`
          : '0 4px 12px rgba(0,0,0,0.5)',
        transition: 'all 0.3s ease',
        minWidth: '135px',
        textAlign: 'center',
      },
    }
  })
})

const edges = computed<Edge[]>(() => {
  const steps = parsedDef.value.steps || []
  const result: Edge[] = []

  for (const step of steps) {
    for (const dep of step.depends_on || []) {
      result.push({
        id: `${dep}->${step.id}`,
        source: dep,
        target: step.id,
        animated: true,
        style: {
          stroke: '#6366f1',
          strokeWidth: 2,
        },
      })
    }
  }

  return result
})
</script>

<template>
  <div class="w-full h-[500px] rounded-lg overflow-hidden border border-border-subtle bg-[#0a0a1e]/60 shadow-lg">
    <VueFlow
      :nodes="nodes"
      :edges="edges"
      :default-viewport="{ zoom: 1, x: 0, y: 0 }"
      fit-view-on-init
      class="w-full h-full"
    >
      <Background />
      <Controls />
    </VueFlow>
  </div>
</template>

<style scoped>
:deep(.vue-flow__background) {
  background: rgba(10, 10, 30, 0.8);
}

:deep(.vue-flow__controls) {
  background: rgba(17, 17, 35, 0.85);
  border: 1px solid rgba(139, 92, 246, 0.15);
  border-radius: 6px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
}

:deep(.vue-flow__controls-button) {
  background: rgba(20, 20, 50, 0.8);
  border-bottom: 1px solid rgba(139, 92, 246, 0.15);
  color: #e8e8ff;
  fill: #e8e8ff;
}

:deep(.vue-flow__controls-button:hover) {
  background: rgba(40, 40, 90, 0.7);
}
</style>
