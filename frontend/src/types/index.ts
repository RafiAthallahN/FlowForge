export interface Tenant {
  id: string
  tenant_id: string
  name: string
}

export interface User {
  id: string
  tenant_id: string
  email: string
  role: 'Admin' | 'Editor' | 'Viewer'
}

export interface Workflow {
  id: string
  tenant_id: string
  version: number
  name: string
  definition: string
  created_at: string
  updated_at: string
}

export interface WorkflowRun {
  id: string
  tenant_id: string
  workflow_id: string
  workflow_version: number
  status: 'Pending' | 'Running' | 'Success' | 'Failed'
  started_at: string | null
  completed_at: string | null
  error_message: string | null
  created_at: string
}

export interface ExecutionLog {
  id: string
  tenant_id: string
  workflow_run_id: string
  step_id: string
  status: string
  retry_count: number
  duration_ms: number
  log_output: string
  failure_reason?: string
  suggested_fix?: string
  created_at: string
}

export interface StepEvent {
  run_id: string
  step_id: string
  status: string
  log_line?: string
  duration_ms?: number
  failure_reason?: string
  suggested_fix?: string
}

export interface HealthMetrics {
  active_runs: number
  total_runs: number
  success_count: number
  failure_count: number
  success_rate: number
  failure_rate: number
  avg_duration_ms: number
}

export interface LoginRequest {
  email: string
  password: string
  tenant_id: string
}

export interface RegisterRequest extends LoginRequest {
  role: 'Admin' | 'Editor' | 'Viewer'
}
