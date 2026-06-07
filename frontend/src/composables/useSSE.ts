import { ref, onUnmounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useRunStore } from '../stores/runs'
import type { StepEvent } from '../types'

export function useSSE() {
  const connected = ref(false)
  const error = ref<string | null>(null)
  let eventSource: EventSource | null = null

  function connect() {
    const authStore = useAuthStore()
    const runStore = useRunStore()

    if (!authStore.token) {
      error.value = 'Not authenticated'
      return
    }

    // Close existing connection
    disconnect()

    const isDev = typeof window !== 'undefined' && window.location.port === '5173'
    const url = isDev ? 'http://localhost:8080/api/events/stream' : '/api/events/stream'
    // Note: EventSource doesn't support custom headers natively.
    // We'll pass the token as a query parameter for SSE.
    // In production, use a cookie or a proxy.
    eventSource = new EventSource(`${url}?token=${authStore.token}`)

    eventSource.onopen = () => {
      connected.value = true
      error.value = null
    }

    eventSource.onmessage = (event) => {
      try {
        const stepEvent: StepEvent = JSON.parse(event.data)
        runStore.handleStepEvent(stepEvent)
      } catch {
        // Ignore parse errors (keepalive comments, etc.)
      }
    }

    eventSource.onerror = () => {
      connected.value = false
      error.value = 'SSE connection lost'
      // Auto-reconnect is handled by EventSource natively
    }
  }

  function disconnect() {
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
    connected.value = false
  }

  onUnmounted(() => {
    disconnect()
  })

  return { connected, error, connect, disconnect }
}
