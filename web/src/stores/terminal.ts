// Terminal store - manages terminal sessions and state
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import apiClient from '@/api/client'

export interface TerminalSession {
  sessionId: string
  tenantId: string
  podName: string
  namespace: string
  createdAt: string
  wsUrl: string
}

export interface TerminalCapabilities {
  enabled: boolean
  hasAccess: boolean
  idleTimeout: number
  maxSessions: number
}

export const useTerminalStore = defineStore('terminal', () => {
  // State
  const isOpen = ref(false)
  const isConnected = ref(false)
  const isConnecting = ref(false)
  const currentSession = ref<TerminalSession | null>(null)
  const capabilities = ref<TerminalCapabilities | null>(null)
  const error = ref<string | null>(null)
  const height = ref(300) // Terminal height in pixels

  // Computed
  const canUseTerminal = computed(() => capabilities.value?.hasAccess ?? false)
  const isTerminalEnabled = computed(() => capabilities.value?.enabled ?? false)

  // Actions
  async function fetchCapabilities() {
    try {
      const response = await apiClient.get('/terminal/config')
      capabilities.value = response.data
      return capabilities.value
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch capabilities'
      console.error('Failed to fetch terminal capabilities:', err)
      error.value = errorMessage
      return null
    }
  }

  async function createSession() {
    isConnecting.value = true
    error.value = null

    try {
      // No tenant ID needed - the terminal is a shared toolbox
      const response = await apiClient.post('/terminal/sessions', {})
      currentSession.value = response.data
      return response.data
    } catch (err) {
      let errorMessage = 'Failed to create session'
      if (err && typeof err === 'object' && 'response' in err) {
        const axiosError = err as { response?: { data?: { error?: string } } }
        errorMessage = axiosError.response?.data?.error || errorMessage
      } else if (err instanceof Error) {
        errorMessage = err.message
      }
      console.error('Failed to create terminal session:', err)
      error.value = errorMessage
      throw err
    } finally {
      isConnecting.value = false
    }
  }

  async function deleteSession() {
    if (!currentSession.value) return

    try {
      await apiClient.delete(`/terminal/sessions/${currentSession.value.sessionId}`)
      currentSession.value = null
      isConnected.value = false
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to delete session'
      console.error('Failed to delete terminal session:', err)
      error.value = errorMessage
    }
  }

  function openTerminal() {
    isOpen.value = true
  }

  function closeTerminal() {
    isOpen.value = false
  }

  function toggleTerminal() {
    isOpen.value = !isOpen.value
  }

  function setHeight(newHeight: number) {
    height.value = Math.max(200, Math.min(800, newHeight))
  }

  function setConnected(connected: boolean) {
    isConnected.value = connected
  }

  function clearError() {
    error.value = null
  }

  return {
    // State
    isOpen,
    isConnected,
    isConnecting,
    currentSession,
    capabilities,
    error,
    height,
    // Computed
    canUseTerminal,
    isTerminalEnabled,
    // Actions
    fetchCapabilities,
    createSession,
    deleteSession,
    openTerminal,
    closeTerminal,
    toggleTerminal,
    setHeight,
    setConnected,
    clearError,
  }
})

