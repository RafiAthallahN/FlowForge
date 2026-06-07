import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '../api/auth'
import type { LoginRequest, RegisterRequest } from '../types'

interface TokenPayload {
  user_id: string
  tenant_id: string
  role: string
  exp: number
}

function decodeToken(token: string): TokenPayload | null {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return payload
  } catch {
    return null
  }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('ff_token'))
  const decoded = computed(() => token.value ? decodeToken(token.value) : null)
  const isAuthenticated = computed(() => !!token.value && !!decoded.value)
  const userId = computed(() => decoded.value?.user_id ?? '')
  const tenantId = computed(() => decoded.value?.tenant_id ?? '')
  const role = computed(() => decoded.value?.role ?? '')

  async function login(data: LoginRequest) {
    const res = await authApi.login(data)
    token.value = res.token
    localStorage.setItem('ff_token', res.token)
  }

  async function register(data: RegisterRequest) {
    return authApi.register(data)
  }

  function logout() {
    token.value = null
    localStorage.removeItem('ff_token')
  }

  return { token, isAuthenticated, userId, tenantId, role, login, register, logout }
})
