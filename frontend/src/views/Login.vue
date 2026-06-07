<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const authStore = useAuthStore()
const router = useRouter()

const isRegister = ref(false)
const email = ref('')
const password = ref('')
const tenantId = ref('')
const role = ref<'Admin' | 'Editor' | 'Viewer'>('Editor')

const error = ref<string | null>(null)
const loading = ref(false)

async function handleSubmit() {
  error.value = null
  loading.value = true

  try {
    if (isRegister.value) {
      await authStore.register({
        email: email.value,
        password: password.value,
        tenant_id: tenantId.value,
        role: role.value
      })
      isRegister.value = false
      alert('Registration successful! Please login.')
    } else {
      await authStore.login({
        email: email.value,
        password: password.value,
        tenant_id: tenantId.value
      })
      router.push('/')
    }
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-[85vh] w-full flex items-center justify-center p-4">
    <div class="glass-card p-8 w-full max-w-[420px] flex flex-col gap-6 animate-fade-in">
      <div class="text-center">
        <h1 class="text-3xl font-extrabold tracking-tight text-gradient">FlowForge</h1>
        <p class="text-brand-slate text-sm mt-2">
          {{ isRegister ? 'Create a tenant user account' : 'Sign in to access your workflows' }}
        </p>
      </div>

      <form @submit.prevent="handleSubmit" class="flex flex-col gap-4">
        <!-- Tenant ID -->
        <div class="flex flex-col gap-1.5">
          <label for="tenant-id" class="text-xs font-semibold text-brand-slate uppercase tracking-wider">Tenant ID</label>
          <input 
            id="tenant-id" 
            type="text" 
            v-model="tenantId" 
            class="app-input" 
            placeholder="e.g. tenant-a" 
            required
          />
        </div>

        <!-- Email -->
        <div class="flex flex-col gap-1.5">
          <label for="email" class="text-xs font-semibold text-brand-slate uppercase tracking-wider">Email Address</label>
          <input 
            id="email" 
            type="email" 
            v-model="email" 
            class="app-input" 
            placeholder="name@domain.com" 
            required
          />
        </div>

        <!-- Password -->
        <div class="flex flex-col gap-1.5">
          <label for="password" class="text-xs font-semibold text-brand-slate uppercase tracking-wider">Password</label>
          <input 
            id="password" 
            type="password" 
            v-model="password" 
            class="app-input" 
            placeholder="••••••••" 
            required
          />
        </div>

        <!-- Role Select (Register only) -->
        <div class="flex flex-col gap-1.5" v-if="isRegister">
          <label for="role" class="text-xs font-semibold text-brand-slate uppercase tracking-wider">Account Role</label>
          <select id="role" v-model="role" class="app-input cursor-pointer bg-bg-deep">
            <option value="Admin">Admin (Full Control)</option>
            <option value="Editor">Editor (Read/Write)</option>
            <option value="Viewer">Viewer (Read Only)</option>
          </select>
        </div>

        <!-- Error Alert -->
        <div class="p-3.5 bg-brand-red/10 border border-brand-red/20 text-brand-red text-xs rounded-md font-medium" v-if="error">
          {{ error }}
        </div>

        <!-- Submit Button -->
        <button type="submit" class="btn-primary w-full justify-center mt-2" :disabled="loading">
          {{ loading ? 'Processing...' : (isRegister ? 'Register Account' : 'Sign In') }}
        </button>
      </form>

      <!-- Toggle Links -->
      <div class="text-center text-sm">
        <a @click="isRegister = !isRegister" class="cursor-pointer text-brand-purple hover:text-white underline font-semibold transition-colors">
          {{ isRegister ? 'Already have an account? Sign In' : "Don't have an account? Register" }}
        </a>
      </div>
    </div>
  </div>
</template>
