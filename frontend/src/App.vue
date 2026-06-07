<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'

const authStore = useAuthStore()
const router = useRouter()

const showNav = computed(() => authStore.isAuthenticated)

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>

<template>
  <!-- Global Navigation Header -->
  <header class="sticky top-0 z-50 w-full bg-[#070714]/80 backdrop-blur-md border-b border-border-subtle" v-if="showNav">
    <div class="max-w-[1400px] mx-auto px-6 py-4 flex justify-between items-center">
      <router-link to="/" class="flex items-center gap-2 select-none">
        <span class="text-xl font-extrabold tracking-tight text-gradient">FlowForge</span>
      </router-link>

      <div class="flex items-center gap-4">
        <!-- Tenant / Role Info -->
        <div class="flex flex-col text-right">
          <span class="text-sm font-semibold text-white/90">{{ authStore.userId }}</span>
          <span class="text-xs text-brand-slate select-none">
            Tenant: <strong class="text-white">{{ authStore.tenantId }}</strong> • Role: {{ authStore.role }}
          </span>
        </div>

        <button @click="handleLogout" class="btn-secondary btn-sm hover:!border-brand-red/50 hover:!text-brand-red">
          Sign Out
        </button>
      </div>
    </div>
  </header>

  <!-- Router view main slot -->
  <main class="flex-1 w-full flex flex-col">
    <router-view />
  </main>
</template>

<style>
@import './assets/main.css';
</style>
