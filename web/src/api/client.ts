// web/src/api/client.ts
// API client for Platform Manager backend
import axios, { type AxiosInstance } from 'axios'

// Create axios instance with defaults
const apiClient: AxiosInstance = axios.create({
  baseURL: process.env.VUE_APP_API_URL || 'http://localhost:9080/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor for adding auth headers if needed
apiClient.interceptors.request.use(
  (config) => {
    // OAuth2Proxy handles auth at gateway level
    // Headers like X-Auth-Request-User are set by the gateway
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor for error handling
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response) {
      // Handle specific error codes
      switch (error.response.status) {
        case 401:
          console.error('Unauthorized - redirecting to login')
          // Gateway handles auth redirect
          break
        case 403:
          console.error('Forbidden - insufficient permissions')
          break
        case 500:
          console.error('Server error:', error.response.data)
          break
      }
    }
    return Promise.reject(error)
  }
)

// API types
export interface PlatformHealth {
  overallHealth: 'healthy' | 'warning' | 'critical'
  resourceStates: ResourceStateCounts
  crossplane: CrossplaneSummary
  argo: ArgoSummary
  iamDrift: IAMDriftSummary
}

export interface ResourceStateCounts {
  ready: number
  failed: number
  waiting: number
  unknown: number
  paused: number
  total: number
}

export interface CrossplaneSummary {
  compositions: number
  claims: number
  xrs: number
  failed: number
}

export interface ArgoSummary {
  totalApps: number
  synced: number
  outOfSync: number
  healthy: number
  degraded: number
}

export interface IAMDriftSummary {
  totalRoles: number
  rolesWithDrift: number
  extraPrivileges: number
  missingPrivileges: number
}

export interface Tenant {
  id: string
  name: string
  namespaces: string[]
  status: 'healthy' | 'warning' | 'critical'
}

export interface TenantHealth {
  tenantRef: string
  overallHealth: string
  crossplaneResources: ResourceStateCounts
  kubernetesResources: ResourceStateCounts
  iamDrift: IAMDriftSummary
  argo: ArgoSummary
  cpuUsage: string
  memoryUsage: string
}

// Action types
export interface ActionResponse {
  success: boolean
  message: string
  details?: Record<string, unknown>
}

export interface CrossplaneResource {
  group: string
  version: string
  kind: string
  namespace?: string
  name: string
}

export interface ArgoSyncOptions {
  name: string
  namespace: string
  prune?: boolean
  dryRun?: boolean
}

export interface DeleteResourceOptions extends CrossplaneResource {
  confirm: string
}

// API methods
export const api = {
  // Health endpoints
  async getPlatformHealth(): Promise<PlatformHealth> {
    const response = await apiClient.get('/health/platform')
    return response.data
  },

  async getTenantHealthList(): Promise<TenantHealth[]> {
    const response = await apiClient.get('/health/tenants')
    return response.data
  },

  async getTenantHealth(id: string): Promise<TenantHealth> {
    const response = await apiClient.get(`/health/tenants/${id}`)
    return response.data
  },

  // Tenant endpoints
  async getTenants(): Promise<Tenant[]> {
    const response = await apiClient.get('/tenants')
    return response.data
  },

  async getTenant(id: string): Promise<Tenant> {
    const response = await apiClient.get(`/tenants/${id}`)
    return response.data
  },

  // Resource endpoints
  async getResources(params?: {
    tenant?: string
    state?: string
    kind?: string
    search?: string
  }): Promise<{ resources: unknown[]; total: number }> {
    const response = await apiClient.get('/resources', { params })
    return response.data
  },

  // IAM endpoints
  async getIAMDriftSummary(): Promise<IAMDriftSummary> {
    const response = await apiClient.get('/iam/drift')
    return response.data
  },

  async triggerIAMScan(): Promise<void> {
    await apiClient.post('/iam/scan')
  },

  // Action endpoints
  async syncArgoApp(options: ArgoSyncOptions): Promise<ActionResponse> {
    const response = await apiClient.post('/actions/argo/sync', options)
    return response.data
  },

  async refreshArgoApp(name: string, namespace: string): Promise<ActionResponse> {
    const response = await apiClient.post('/actions/argo/refresh', { name, namespace })
    return response.data
  },

  async pauseCrossplaneResource(resource: CrossplaneResource): Promise<ActionResponse> {
    const response = await apiClient.post('/actions/crossplane/pause', resource)
    return response.data
  },

  async unpauseCrossplaneResource(resource: CrossplaneResource): Promise<ActionResponse> {
    const response = await apiClient.post('/actions/crossplane/unpause', resource)
    return response.data
  },

  async reconcileCrossplaneResource(resource: CrossplaneResource): Promise<ActionResponse> {
    const response = await apiClient.post('/actions/crossplane/reconcile', resource)
    return response.data
  },

  async deleteResource(options: DeleteResourceOptions): Promise<ActionResponse> {
    const response = await apiClient.delete('/actions/resources', { data: options })
    return response.data
  },
}

export default apiClient

