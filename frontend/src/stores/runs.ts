import { defineStore } from 'pinia'
import { ref } from 'vue'
import { runsApi } from '../api/runs'
import type { WorkflowRun, ExecutionLog, HealthMetrics, StepEvent } from '../types'

export const useRunStore = defineStore('runs', () => {
  const runs = ref<WorkflowRun[]>([])
  const selectedRun = ref<WorkflowRun | null>(null)
  const runLogs = ref<ExecutionLog[]>([])
  const healthMetrics = ref<HealthMetrics | null>(null)
  const totalCount = ref(0)
  const loading = ref(false)
  const error = ref<string | null>(null)

  // SSE live step events for real-time DAG updates
  const liveStepEvents = ref<Map<string, StepEvent>>(new Map())

  async function fetchRuns(workflowId?: string, page = 1, limit = 10) {
    loading.value = true
    error.value = null
    try {
      const data = await runsApi.list(workflowId, page, limit)
      runs.value = data.runs || []
      totalCount.value = data.total
    } catch (e: any) {
      error.value = e.response?.data?.error || e.message
    } finally {
      loading.value = false
    }
  }

  async function fetchRun(id: string) {
    loading.value = true
    error.value = null
    try {
      const data = await runsApi.get(id)
      selectedRun.value = data.run
      runLogs.value = data.logs || []
    } catch (e: any) {
      error.value = e.response?.data?.error || e.message
    } finally {
      loading.value = false
    }
  }

  async function fetchHealthMetrics() {
    try {
      healthMetrics.value = await runsApi.getHealthMetrics()
    } catch (e: any) {
      error.value = e.response?.data?.error || e.message
    }
  }

  function handleStepEvent(event: StepEvent) {
    liveStepEvents.value.set(event.step_id, event)
    // Trigger reactivity
    liveStepEvents.value = new Map(liveStepEvents.value)
  }

  function clearLiveEvents() {
    liveStepEvents.value = new Map()
  }

  return {
    runs, selectedRun, runLogs, healthMetrics, totalCount, loading, error,
    liveStepEvents,
    fetchRuns, fetchRun, fetchHealthMetrics, handleStepEvent, clearLiveEvents
  }
})
