import client from './client'
import type { WorkflowRun, ExecutionLog, HealthMetrics } from '../types'

export const runsApi = {
  async list(workflowId?: string, page = 1, limit = 10): Promise<{ runs: WorkflowRun[]; total: number }> {
    const params: Record<string, any> = { page, limit }
    if (workflowId) params.workflow_id = workflowId
    const res = await client.get('/runs', { params })
    return res.data
  },

  async get(id: string): Promise<{ run: WorkflowRun; logs: ExecutionLog[] }> {
    const res = await client.get(`/runs/${id}`)
    return res.data
  },

  async getHealthMetrics(): Promise<HealthMetrics> {
    const res = await client.get('/health/metrics')
    return res.data
  },
}
