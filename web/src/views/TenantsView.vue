<template>
  <div class="tenants-view">
    <header class="page-header">
      <h1 class="page-title">Tenants</h1>
      <p class="page-subtitle">Overview of all platform tenants</p>
    </header>

    <div v-if="loading" class="loading">
      <div class="spinner"></div>
      <p>Loading tenants...</p>
    </div>

    <div v-else-if="error" class="error">
      <p>{{ error }}</p>
    </div>

    <div v-else-if="tenants.length === 0" class="empty">
      <p>No tenants found</p>
    </div>

    <div v-else class="tenants-table">
      <table>
        <thead>
          <tr>
            <th>Tenant</th>
            <th>Status</th>
            <th>Resources</th>
            <th>Failed</th>
            <th>IAM Drift</th>
            <th>Argo Apps</th>
            <th>CPU / Memory</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="tenant in tenants"
            :key="tenant.id"
            class="tenant-row"
            @click="goToTenant(tenant.id)"
          >
            <td>
              <div class="tenant-name">
                <span class="tenant-icon">{{ tenant.icon }}</span>
                <div>
                  <div class="name">{{ tenant.name }}</div>
                  <div class="namespaces">{{ tenant.namespaces.join(', ') }}</div>
                </div>
              </div>
            </td>
            <td>
              <span class="status-badge" :class="tenant.status">
                {{ tenant.status }}
              </span>
            </td>
            <td>{{ tenant.resources.total }}</td>
            <td>
              <span :class="{ 'text-danger': tenant.resources.failed > 0 }">
                {{ tenant.resources.failed }}
              </span>
            </td>
            <td>
              <span :class="{ 'text-danger': tenant.iamDrift > 0 }">
                {{ tenant.iamDrift }}
              </span>
            </td>
            <td>
              <span class="argo-status">
                {{ tenant.argoApps.synced }}/{{ tenant.argoApps.total }}
              </span>
            </td>
            <td>
              <div class="resource-usage">
                <span>{{ tenant.cpu }}</span>
                <span class="separator">/</span>
                <span>{{ tenant.memory }}</span>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'

interface Tenant {
  id: string
  name: string
  icon: string
  namespaces: string[]
  status: 'healthy' | 'warning' | 'critical'
  resources: {
    total: number
    failed: number
  }
  iamDrift: number
  argoApps: {
    total: number
    synced: number
  }
  cpu: string
  memory: string
}

interface PlatformTenant {
  name: string
  displayName: string
  health: string
  failedResources: number
  totalResources: number
  iamDriftCount: number
  argoOutOfSync: number
}

interface MetricsResponse {
  totalCpuCores?: number
  totalMemoryMb?: number
}

interface TenantHealthResponse {
  argo?: {
    totalApps?: number
    synced?: number
  }
}

// Tenant icons mapping
const TENANT_ICONS: Record<string, string> = {
  'tenant-alpha': '🤖',
  'tenant-beta': '📊',
  'tenant-gamma': '⚙️',
}

export default defineComponent({
  name: 'TenantsView',
  setup() {
    const router = useRouter()
    const tenants = ref<Tenant[]>([])
    const loading = ref(true)
    const error = ref<string | null>(null)

    const fetchTenants = async () => {
      try {
        loading.value = true
        error.value = null

        // Fetch platform health which includes tenant summaries
        const healthResponse = await axios.get('http://localhost:9080/api/v1/health/platform')
        const platformHealth = healthResponse.data

        // Transform the data to match our UI structure
        const transformedTenants = platformHealth.tenants.map((tenant: PlatformTenant) => {
          // Determine status based on health and failures
          let status: 'healthy' | 'warning' | 'critical' = 'healthy'
          if (tenant.health === 'Unknown') {
            status = 'warning'
          } else if (tenant.failedResources > 0) {
            status = tenant.failedResources > 1 ? 'critical' : 'warning'
          } else if (tenant.iamDriftCount > 0 || tenant.argoOutOfSync > 0) {
            status = 'warning'
          }

          return {
            id: tenant.name,
            name: tenant.displayName || tenant.name,
            icon: TENANT_ICONS[tenant.name] || '🏢',
            namespaces: [tenant.name], // Could be fetched from tenant details if needed
            status,
            resources: {
              total: tenant.totalResources,
              failed: tenant.failedResources,
            },
            iamDrift: tenant.iamDriftCount,
            argoApps: {
              total: platformHealth.argoSummary?.totalApps || 0,
              synced: platformHealth.argoSummary?.synced || 0,
            },
            cpu: '0m', // Will be populated from metrics if available
            memory: '0Mi',
          }
        })

        // Sort tenants alphabetically by name for consistent display order
        tenants.value = transformedTenants.sort((a: Tenant, b: Tenant) => a.name.localeCompare(b.name))

        // Fetch metrics for each tenant
        await Promise.all(
          tenants.value.map(async (tenant) => {
            try {
              const metricsResponse = await axios.get<MetricsResponse>(
                `http://localhost:9080/api/v1/metrics/tenants/${tenant.id}`
              )
              const metrics = metricsResponse.data
              
              // Format CPU in millicores
              const cpuCores = metrics.totalCpuCores || 0
              tenant.cpu = `${Math.round(cpuCores * 1000)}m`
              
              // Format memory in Mi
              const memoryMb = metrics.totalMemoryMb || 0
              tenant.memory = `${Math.round(memoryMb)}Mi`
            } catch (err) {
              // If metrics fail, keep default values
              console.warn(`Failed to fetch metrics for ${tenant.id}:`, err)
            }
          })
        )

        // Fetch ArgoCD apps per tenant
        for (const tenant of tenants.value) {
          try {
            const healthResponse = await axios.get<TenantHealthResponse>(
              `http://localhost:9080/api/v1/health/tenants/${tenant.id}`
            )
            const tenantHealth = healthResponse.data
            
            if (tenantHealth.argo) {
              tenant.argoApps = {
                total: tenantHealth.argo.totalApps || 0,
                synced: tenantHealth.argo.synced || 0,
              }
            }
          } catch (err) {
            console.warn(`Failed to fetch ArgoCD info for ${tenant.id}:`, err)
          }
        }

      } catch (err) {
        console.error('Failed to fetch tenants:', err)
        error.value = err instanceof Error ? err.message : 'Failed to load tenants'
      } finally {
        loading.value = false
      }
    }

    const goToTenant = (id: string) => {
      router.push(`/tenants/${id}`)
    }

    onMounted(() => {
      fetchTenants()
      
      // Refresh every 30 seconds
      const interval = setInterval(fetchTenants, 30000)
      
      return () => clearInterval(interval)
    })

    return {
      tenants,
      loading,
      error,
      goToTenant,
    }
  },
})
</script>

<style lang="scss" scoped>
.tenants-view {
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 2rem;
}

.page-title {
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary, #e2e8f0);
  margin: 0 0 0.5rem 0;
}

.page-subtitle {
  color: var(--text-secondary, #94a3b8);
  margin: 0;
}

.loading,
.error,
.empty {
  text-align: center;
  padding: 4rem 2rem;
  background: var(--bg-secondary, #1e293b);
  border-radius: 0.75rem;
  border: 1px solid var(--border-color, #334155);
}

.loading {
  .spinner {
    width: 48px;
    height: 48px;
    border: 4px solid var(--border-color, #334155);
    border-top-color: var(--primary, #3b82f6);
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin: 0 auto 1rem;
  }

  p {
    color: var(--text-secondary, #94a3b8);
    margin: 0;
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.error {
  p {
    color: #ef4444;
    margin: 0;
  }
}

.empty {
  p {
    color: var(--text-secondary, #94a3b8);
    margin: 0;
  }
}

.tenants-table {
  background: var(--bg-secondary, #1e293b);
  border-radius: 0.75rem;
  border: 1px solid var(--border-color, #334155);
  overflow: hidden;

  table {
    width: 100%;
    border-collapse: collapse;
  }

  th {
    text-align: left;
    padding: 1rem 1.5rem;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    color: var(--text-secondary, #94a3b8);
    background: var(--bg-tertiary, #0f172a);
    border-bottom: 1px solid var(--border-color, #334155);
  }

  td {
    padding: 1rem 1.5rem;
    border-bottom: 1px solid var(--border-color, #334155);
  }

  .tenant-row {
    cursor: pointer;
    transition: background 0.2s ease;

    &:hover {
      background: var(--bg-hover, #334155);
    }

    &:last-child td {
      border-bottom: none;
    }
  }
}

.tenant-name {
  display: flex;
  align-items: center;
  gap: 0.75rem;

  .tenant-icon {
    font-size: 1.5rem;
  }

  .name {
    font-weight: 600;
    color: var(--text-primary, #e2e8f0);
  }

  .namespaces {
    font-size: 0.75rem;
    color: var(--text-secondary, #94a3b8);
  }
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: capitalize;

  &.healthy {
    background: rgba(34, 197, 94, 0.1);
    color: #22c55e;
  }

  &.warning {
    background: rgba(245, 158, 11, 0.1);
    color: #f59e0b;
  }

  &.critical {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
  }
}

.text-danger {
  color: #ef4444;
  font-weight: 600;
}

.argo-status {
  font-family: monospace;
}

.resource-usage {
  font-family: monospace;
  font-size: 0.875rem;

  .separator {
    color: var(--text-secondary, #94a3b8);
    margin: 0 0.25rem;
  }
}
</style>

