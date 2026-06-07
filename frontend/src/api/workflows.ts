import client from './client'
import type { Workflow, WorkflowRun } from '../types'

export const workflowsApi = {
  async list(page = 1, limit = 10): Promise<{ workflows: Workflow[]; total: number }> {
    const res = await client.get('/workflows', { params: { page, limit } })
    return res.data
  },

  async get(id: string, version?: number): Promise<Workflow> {
    const params = version ? { version } : {}
    const res = await client.get(`/workflows/${id}`, { params })
    return res.data
  },

  async create(data: { id: string; name: string; definition: string }): Promise<Workflow> {
    const res = await client.post('/workflows', data)
    return res.data
  },

  async update(id: string, data: { name: string; definition: string }): Promise<Workflow> {
    const res = await client.put(`/workflows/${id}`, data)
    return res.data
  },

  async rollback(id: string, version: number): Promise<Workflow> {
    const res = await client.post(`/workflows/${id}/rollback`, { version })
    return res.data
  },

  async run(id: string): Promise<WorkflowRun> {
    const res = await client.post(`/workflows/${id}/run`)
    return res.data
  },
}
