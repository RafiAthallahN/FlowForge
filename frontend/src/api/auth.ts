import client from './client'
import type { LoginRequest, RegisterRequest } from '../types'

export const authApi = {
  async login(data: LoginRequest): Promise<{ token: string }> {
    const res = await client.post('/auth/login', data)
    return res.data
  },

  async register(data: RegisterRequest): Promise<any> {
    const res = await client.post('/auth/register', data)
    return res.data
  },
}
