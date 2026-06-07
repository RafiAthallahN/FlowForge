import { defineStore } from 'pinia'
import { ref } from 'vue'
import { workflowsApi } from '../api/workflows'
import type { Workflow } from '../types'

export const useWorkflowStore = defineStore('workflows', () => {
  const workflows = ref<Workflow[]>([])
  const selectedWorkflow = ref<Workflow | null>(null)
  const totalCount = ref(0)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchWorkflows(page = 1, limit = 10) {
    loading.value = true
    error.value = null
    try {
      const data = await workflowsApi.list(page, limit)
      workflows.value = data.workflows || []
      totalCount.value = data.total
    } catch (e: any) {
      error.value = e.response?.data?.error || e.message
    } finally {
      loading.value = false
    }
  }

  async function fetchWorkflow(id: string, version?: number) {
    loading.value = true
    error.value = null
    try {
      selectedWorkflow.value = await workflowsApi.get(id, version)
    } catch (e: any) {
      error.value = e.response?.data?.error || e.message
    } finally {
      loading.value = false
    }
  }

  async function createWorkflow(data: { id: string; name: string; definition: string }) {
    const wf = await workflowsApi.create(data)
    workflows.value.unshift(wf)
    return wf
  }

  async function updateWorkflow(id: string, data: { name: string; definition: string }) {
    const wf = await workflowsApi.update(id, data)
    selectedWorkflow.value = wf
    return wf
  }

  async function rollbackWorkflow(id: string, version: number) {
    const wf = await workflowsApi.rollback(id, version)
    selectedWorkflow.value = wf
    return wf
  }

  async function runWorkflow(id: string) {
    loading.value = true
    error.value = null
    try {
      return await workflowsApi.run(id)
    } catch (e: any) {
      error.value = e.response?.data?.error || e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    workflows, selectedWorkflow, totalCount, loading, error,
    fetchWorkflows, fetchWorkflow, createWorkflow, updateWorkflow, rollbackWorkflow, runWorkflow
  }
})
