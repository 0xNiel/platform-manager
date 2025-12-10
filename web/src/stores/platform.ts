// web/src/stores/platform.ts
// Pinia store for platform state
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api, type PlatformHealth, type TenantHealth, type Tenant } from '@/api/client'

export const usePlatformStore = defineStore('platform', () => {
  // State
  const platformHealth = ref<PlatformHealth | null>(null)
  const tenants = ref<Tenant[]>([])
  const tenantHealthMap = ref<Map<string, TenantHealth>>(new Map())
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Getters
  const overallStatus = computed(() => {
    if (!platformHealth.value) return 'unknown'
    return platformHealth.value.overallHealth
  })

  const failedResourceCount = computed(() => {
    if (!platformHealth.value) return 0
    return platformHealth.value.resourceStates.failed
  })

  const iamDriftCount = computed(() => {
    if (!platformHealth.value) return 0
    return platformHealth.value.iamDrift.rolesWithDrift
  })

  // Actions
  async function fetchPlatformHealth() {
    loading.value = true
    error.value = null
    try {
      platformHealth.value = await api.getPlatformHealth()
    } catch (e) {
      error.value = 'Failed to fetch platform health'
      console.error(e)
    } finally {
      loading.value = false
    }
  }

  async function fetchTenants() {
    loading.value = true
    error.value = null
    try {
      tenants.value = await api.getTenants()
    } catch (e) {
      error.value = 'Failed to fetch tenants'
      console.error(e)
    } finally {
      loading.value = false
    }
  }

  async function fetchTenantHealth(tenantId: string) {
    try {
      const health = await api.getTenantHealth(tenantId)
      tenantHealthMap.value.set(tenantId, health)
    } catch (e) {
      console.error(`Failed to fetch health for tenant ${tenantId}:`, e)
    }
  }

  function getTenantHealth(tenantId: string): TenantHealth | undefined {
    return tenantHealthMap.value.get(tenantId)
  }

  // Polling for real-time updates
  let pollInterval: ReturnType<typeof setInterval> | null = null

  function startPolling(intervalMs = 30000) {
    if (pollInterval) return
    
    // Initial fetch
    fetchPlatformHealth()
    fetchTenants()
    
    // Poll for updates
    pollInterval = setInterval(() => {
      fetchPlatformHealth()
    }, intervalMs)
  }

  function stopPolling() {
    if (pollInterval) {
      clearInterval(pollInterval)
      pollInterval = null
    }
  }

  return {
    // State
    platformHealth,
    tenants,
    tenantHealthMap,
    loading,
    error,
    
    // Getters
    overallStatus,
    failedResourceCount,
    iamDriftCount,
    
    // Actions
    fetchPlatformHealth,
    fetchTenants,
    fetchTenantHealth,
    getTenantHealth,
    startPolling,
    stopPolling,
  }
})

