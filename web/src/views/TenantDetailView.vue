<template>
  <div class="tenant-detail">
    <header class="page-header">
      <router-link to="/tenants" class="back-link">← Back to Tenants</router-link>
      <div class="header-content">
        <h1 class="page-title">{{ tenant?.name || 'Loading...' }}</h1>
        <span class="status-badge" :class="tenant?.status">
          {{ tenant?.status }}
        </span>
      </div>
    </header>

    <!-- Tabs -->
    <div class="tabs">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        class="tab"
        :class="{ active: activeTab === tab.id }"
        @click="activeTab = tab.id"
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- Tab Content -->
    <div class="tab-content">
      <!-- Overview Tab -->
      <div v-if="activeTab === 'overview'" class="overview-tab">
        <div class="cards-grid">
          <div class="card">
            <div class="card-header">Resources</div>
            <div class="card-body">
              <div class="metric-large">{{ tenant?.resources.total }}</div>
              <div class="metric-breakdown">
                <span class="ready">{{ tenant?.resources.ready }} ready</span>
                <span class="failed">{{ tenant?.resources.failed }} failed</span>
              </div>
            </div>
          </div>
          <div class="card">
            <div class="card-header">CPU Usage</div>
            <div class="card-body">
              <div class="metric-large">{{ tenant?.cpu }}</div>
            </div>
          </div>
          <div class="card">
            <div class="card-header">Memory Usage</div>
            <div class="card-body">
              <div class="metric-large">{{ tenant?.memory }}</div>
            </div>
          </div>
          <div class="card">
            <div class="card-header">IAM Drift</div>
            <div class="card-body">
              <div class="metric-large" :class="{ danger: tenant?.iamDrift > 0 }">
                {{ tenant?.iamDrift }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Resources Tab -->
      <div v-if="activeTab === 'resources'" class="resources-tab">
        <div v-if="loadingResources" class="loading">Loading resources...</div>
        <div v-else-if="resources.length === 0" class="empty-state">
          <p>No resources found for this tenant.</p>
        </div>
        <div v-else class="resources-list">
          <div v-for="resource in resources" :key="`${resource.kind}-${resource.name}`" class="resource-item">
            <div class="resource-info">
              <span :class="['resource-badge', `badge--${resource.category.toLowerCase()}`]">
                {{ resource.category }}
              </span>
              <div class="resource-details">
                <h4>{{ resource.name }}</h4>
                <p class="resource-meta">{{ resource.kind }} • {{ resource.namespace || 'cluster-scoped' }}</p>
              </div>
            </div>
            <span :class="['status-indicator', resource.state.toLowerCase()]">
              {{ resource.state }}
            </span>
          </div>
        </div>
      </div>

      <!-- IAM Tab -->
      <div v-if="activeTab === 'iam'" class="iam-tab">
        <div v-if="loadingIam" class="loading">Loading IAM drift...</div>
        <div v-else-if="iamDriftItems.length === 0" class="empty-state">
          <p>✓ No IAM drift detected for this tenant.</p>
        </div>
        <div v-else class="iam-list">
          <div v-for="item in iamDriftItems" :key="`${item.type}-${item.name}`" class="iam-item">
            <div class="iam-info">
              <span :class="['iam-badge', item.type === 'role' ? 'badge--role' : 'badge--policy']">
                {{ item.type }}
              </span>
              <div class="iam-details">
                <h4>{{ item.name }}</h4>
                <p class="iam-meta">{{ item.driftCount }} drift(s) detected</p>
              </div>
            </div>
            <span class="status-indicator warning">Drift</span>
          </div>
        </div>
      </div>

      <!-- GitOps Tab -->
      <div v-if="activeTab === 'gitops'" class="gitops-tab">
        <div v-if="loadingGitOps" class="loading">Loading ArgoCD applications...</div>
        <div v-else-if="argoApps.length === 0" class="empty-state">
          <p>No ArgoCD applications found for this tenant.</p>
        </div>
        <div v-else class="gitops-list">
          <div v-for="app in argoApps" :key="app.name" class="gitops-item">
            <div class="gitops-info">
              <span class="gitops-badge">ArgoCD</span>
              <div class="gitops-details">
                <h4>{{ app.name }}</h4>
                <p class="gitops-meta">{{ app.namespace }}</p>
              </div>
            </div>
            <div class="gitops-status">
              <span :class="['status-indicator', app.syncStatus.toLowerCase()]">
                {{ app.syncStatus }}
              </span>
              <span :class="['health-indicator', app.healthStatus.toLowerCase()]">
                {{ app.healthStatus }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Terminal Tab -->
      <div v-if="activeTab === 'terminal'" class="terminal-tab">
        <p class="placeholder">
          🚧 Web terminal feature coming soon!<br><br>
          This will provide direct kubectl access to tenant namespaces.<br>
          In the meantime, use: <code>kubectl -n {{ tenant?.id }}</code>
        </p>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, onMounted, watch } from 'vue'
import axios from 'axios'

interface TenantDetail {
  id: string
  name: string
  status: 'healthy' | 'warning' | 'critical'
  resources: {
    total: number
    ready: number
    failed: number
  }
  cpu: string
  memory: string
  iamDrift: number
}

interface Resource {
  name: string
  kind: string
  namespace: string
  category: string
  state: string
}

interface IAMDriftItem {
  type: 'role' | 'policy'
  name: string
  driftCount: number
}

interface ArgoApp {
  name: string
  namespace: string
  syncStatus: string
  healthStatus: string
}

export default defineComponent({
  name: 'TenantDetailView',
  props: {
    id: {
      type: String,
      required: true,
    },
  },
  setup(props) {
    const activeTab = ref('overview')
    const tenant = ref<TenantDetail | null>(null)
    const loading = ref(true)
    const error = ref<string | null>(null)
    
    // Resources tab
    const resources = ref<Resource[]>([])
    const loadingResources = ref(false)
    
    // IAM tab
    const iamDriftItems = ref<IAMDriftItem[]>([])
    const loadingIam = ref(false)
    
    // GitOps tab
    const argoApps = ref<ArgoApp[]>([])
    const loadingGitOps = ref(false)

    const tabs = [
      { id: 'overview', label: 'Overview' },
      { id: 'resources', label: 'Resources' },
      { id: 'iam', label: 'IAM' },
      { id: 'gitops', label: 'GitOps' },
      { id: 'terminal', label: 'Terminal' },
    ]

    const loadTenant = async () => {
      try {
        loading.value = true
        error.value = null

        // Fetch tenant info
        const tenantResponse = await axios.get(`http://localhost:9080/api/v1/tenants/${props.id}`)
        const tenantData = tenantResponse.data

        // Fetch tenant health
        const healthResponse = await axios.get(`http://localhost:9080/api/v1/health/tenants/${props.id}`)
        const healthData = healthResponse.data

        // Determine status
        let status: 'healthy' | 'warning' | 'critical' = 'healthy'
        if (healthData.overallHealth === 'Unknown') {
          status = 'warning'
        } else if (healthData.kubernetesResources?.failed > 0 || healthData.crossplaneResources?.failed > 0) {
          const totalFailed = (healthData.kubernetesResources?.failed || 0) + (healthData.crossplaneResources?.failed || 0)
          status = totalFailed > 1 ? 'critical' : 'warning'
        } else if (healthData.iamDrift?.rolesWithDrift > 0 || healthData.iamDrift?.policiesWithDrift > 0 || healthData.argo?.outOfSync > 0) {
          status = 'warning'
        }

        // Calculate ready resources
        const totalResources = (healthData.kubernetesResources?.total || 0) + (healthData.crossplaneResources?.total || 0)
        const readyResources = (healthData.kubernetesResources?.ready || 0) + (healthData.crossplaneResources?.ready || 0)
        const failedResources = (healthData.kubernetesResources?.failed || 0) + (healthData.crossplaneResources?.failed || 0)

        // Fetch metrics
        let cpu = '0m'
        let memory = '0Mi'
        try {
          const metricsResponse = await axios.get(`http://localhost:9080/api/v1/metrics/tenants/${props.id}`)
          const metricsData = metricsResponse.data
          
          cpu = `${Math.round((metricsData.totalCpuCores || 0) * 1000)}m`
          memory = `${Math.round(metricsData.totalMemoryMb || 0)}Mi`
        } catch (err) {
          console.warn('Failed to fetch metrics:', err)
        }

        // Calculate IAM drift count
        const iamDrift = (healthData.iamDrift?.rolesWithDrift || 0) + (healthData.iamDrift?.policiesWithDrift || 0)

        tenant.value = {
          id: tenantData.name,
          name: tenantData.displayName || tenantData.name,
          status,
          resources: {
            total: totalResources,
            ready: readyResources,
            failed: failedResources,
          },
          cpu,
          memory,
          iamDrift,
        }

      } catch (err) {
        console.error('Failed to load tenant:', err)
        error.value = err instanceof Error ? err.message : 'Failed to load tenant details'
      } finally {
        loading.value = false
      }
    }

    const loadResources = async () => {
      try {
        loadingResources.value = true
        const response = await axios.get(`http://localhost:9080/api/v1/resources`)
        
        // ResourceSummary CRs have data in .spec
        interface ResourceSummaryItem {
          spec: {
            name: string
            kind: string
            namespace?: string
            category: string
            tenantRef: string
          }
          status?: {
            state?: string
          }
        }
        
        resources.value = response.data.resources
          .filter((r: ResourceSummaryItem) => r.spec.tenantRef === props.id)
          .map((r: ResourceSummaryItem) => ({
            name: r.spec.name,
            kind: r.spec.kind,
            namespace: r.spec.namespace || 'cluster-scoped',
            category: r.spec.category,
            state: r.status?.state || 'Ready',
          }))
          .sort((a: Resource, b: Resource) => {
            // Sort by category, then kind, then name
            if (a.category !== b.category) return a.category.localeCompare(b.category)
            if (a.kind !== b.kind) return a.kind.localeCompare(b.kind)
            return a.name.localeCompare(b.name)
          })
      } catch (err) {
        console.error('Failed to load resources:', err)
      } finally {
        loadingResources.value = false
      }
    }

    const loadIamDrift = async () => {
      try {
        loadingIam.value = true
        const response = await axios.get(`http://localhost:9080/api/v1/iam/drift/tenants/${props.id}`)
        
        interface DriftDetail {
          resourceType: string
          resourceName: string
          hasDrift: boolean
          severity: string
        }
        
        // The response has tenant summary with drift details
        const tenantData = response.data.tenant
        
        // API returns "drifts" (lowercase) not "Drifts"
        const drifts: DriftDetail[] = tenantData?.drifts || []
        
        // Create map to group by resource
        const resourceMap = new Map<string, { type: string; name: string; count: number }>()
        
        drifts.forEach((drift: DriftDetail) => {
          const key = `${drift.resourceType}-${drift.resourceName}`
          if (resourceMap.has(key)) {
            const existing = resourceMap.get(key)!
            existing.count++
          } else {
            resourceMap.set(key, {
              type: drift.resourceType.toLowerCase(),
              name: drift.resourceName,
              count: 1,
            })
          }
        })
        
        // Convert map to array
        iamDriftItems.value = Array.from(resourceMap.values()).map((item) => ({
          type: item.type as 'role' | 'policy',
          name: item.name,
          driftCount: item.count,
        }))
      } catch (err) {
        console.error('Failed to load IAM drift:', err)
        // If endpoint returns 404 or error, just show empty (no drift)
        iamDriftItems.value = []
      } finally {
        loadingIam.value = false
      }
    }

    const loadArgoApps = async () => {
      try {
        loadingGitOps.value = true
        const response = await axios.get(`http://localhost:9080/api/v1/resources`)
        
        interface ResourceSummaryItem {
          spec: {
            name: string
            namespace: string
            kind: string
            category: string
            tenantRef: string
          }
          status?: {
            state?: string
            message?: string
          }
        }
        
        argoApps.value = response.data.resources
          .filter((r: ResourceSummaryItem) => 
            r.spec.kind === 'Application' && 
            r.spec.category === 'ArgoCD' && 
            r.spec.tenantRef === props.id
          )
          .map((app: ResourceSummaryItem) => {
            // Parse status.message which has format: "Sync: Synced, Health: Healthy"
            const message = app.status?.message || ''
            
            // Extract sync status (default to Unknown if not found)
            let syncStatus = 'Unknown'
            const syncMatch = message.match(/Sync:\s*(\w+)/)
            if (syncMatch) {
              syncStatus = syncMatch[1]
            }
            
            // Extract health status (default to Unknown if not found)
            let healthStatus = 'Unknown'
            const healthMatch = message.match(/Health:\s*(\w+)/)
            if (healthMatch) {
              healthStatus = healthMatch[1]
            }
            
            return {
              name: app.spec.name,
              namespace: app.spec.namespace,
              syncStatus,
              healthStatus,
            }
          })
          .sort((a: ArgoApp, b: ArgoApp) => a.name.localeCompare(b.name))
      } catch (err) {
        console.error('Failed to load ArgoCD apps:', err)
      } finally {
        loadingGitOps.value = false
      }
    }

    // Watch for tab changes to load data on demand
    watch(activeTab, (newTab) => {
      if (newTab === 'resources' && resources.value.length === 0) {
        loadResources()
      } else if (newTab === 'iam' && iamDriftItems.value.length === 0 && tenant.value && tenant.value.iamDrift > 0) {
        loadIamDrift()
      } else if (newTab === 'gitops' && argoApps.value.length === 0) {
        loadArgoApps()
      }
    })

    onMounted(() => {
      loadTenant()
      
      // Refresh every 30 seconds
      const interval = setInterval(loadTenant, 30000)
      
      return () => clearInterval(interval)
    })

    return {
      tenant,
      loading,
      error,
      activeTab,
      tabs,
      resources,
      loadingResources,
      iamDriftItems,
      loadingIam,
      argoApps,
      loadingGitOps,
    }
  },
})
</script>

<style lang="scss" scoped>
.tenant-detail {
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 2rem;
}

.back-link {
  color: var(--text-secondary, #94a3b8);
  text-decoration: none;
  font-size: 0.875rem;
  display: inline-block;
  margin-bottom: 0.5rem;

  &:hover {
    color: var(--text-primary, #e2e8f0);
  }
}

.header-content {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.page-title {
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary, #e2e8f0);
  margin: 0;
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

.tabs {
  display: flex;
  gap: 0.5rem;
  border-bottom: 1px solid var(--border-color, #334155);
  margin-bottom: 2rem;
}

.tab {
  padding: 0.75rem 1.5rem;
  background: transparent;
  border: none;
  color: var(--text-secondary, #94a3b8);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  transition: all 0.2s ease;

  &:hover {
    color: var(--text-primary, #e2e8f0);
  }

  &.active {
    color: var(--accent-primary, #3b82f6);
    border-bottom-color: var(--accent-primary, #3b82f6);
  }
}

.cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1.5rem;
}

.card {
  background: var(--bg-secondary, #1e293b);
  border-radius: 0.75rem;
  border: 1px solid var(--border-color, #334155);
  overflow: hidden;
}

.card-header {
  padding: 1rem 1.5rem;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-secondary, #94a3b8);
  border-bottom: 1px solid var(--border-color, #334155);
}

.card-body {
  padding: 1.5rem;
}

.metric-large {
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary, #e2e8f0);

  &.danger {
    color: #ef4444;
  }
}

.metric-breakdown {
  display: flex;
  gap: 1rem;
  margin-top: 0.5rem;
  font-size: 0.875rem;

  .ready {
    color: #22c55e;
  }

  .failed {
    color: #ef4444;
  }
}

.placeholder {
  padding: 3rem;
  text-align: center;
  color: var(--text-secondary, #94a3b8);
  background: var(--bg-secondary, #1e293b);
  border-radius: 0.75rem;
  border: 1px dashed var(--border-color, #334155);
  
  code {
    background: rgba(100, 116, 139, 0.2);
    padding: 0.25rem 0.5rem;
    border-radius: 0.25rem;
    font-family: monospace;
    color: var(--text-primary, #e2e8f0);
  }
}

.loading {
  padding: 2rem;
  text-align: center;
  color: var(--text-secondary, #94a3b8);
}

.empty-state {
  padding: 3rem;
  text-align: center;
  color: var(--text-secondary, #94a3b8);
  background: var(--bg-secondary, #1e293b);
  border-radius: 0.75rem;
  border: 1px dashed var(--border-color, #334155);
}

// Resources tab styles
.resources-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.resource-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  background: var(--bg-secondary, #1e293b);
  border-radius: 0.5rem;
  border: 1px solid var(--border-color, #334155);
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--accent-primary, #3b82f6);
  }
}

.resource-info {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.resource-details {
  h4 {
    margin: 0;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-primary, #e2e8f0);
  }

  .resource-meta {
    margin: 0.25rem 0 0 0;
    font-size: 0.75rem;
    color: var(--text-secondary, #94a3b8);
  }
}

.resource-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 0.375rem;
  font-size: 0.75rem;
  font-weight: 600;
  white-space: nowrap;

  &.badge--argocd {
    background: rgba(241, 90, 34, 0.1);
    color: #f15a22;
  }

  &.badge--crossplane {
    background: rgba(255, 193, 7, 0.1);
    color: #ffc107;
  }

  &.badge--iam {
    background: rgba(156, 39, 176, 0.1);
    color: #9c27b0;
  }

  &.badge--kubernetes {
    background: rgba(33, 150, 243, 0.1);
    color: #2196f3;
  }
}

.status-indicator {
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  white-space: nowrap;

  &.ready,
  &.healthy,
  &.synced {
    background: rgba(34, 197, 94, 0.1);
    color: #22c55e;
  }

  &.failed,
  &.degraded {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
  }

  &.warning,
  &.outofsync {
    background: rgba(245, 158, 11, 0.1);
    color: #f59e0b;
  }

  &.unknown {
    background: rgba(148, 163, 184, 0.1);
    color: #94a3b8;
  }
}

// IAM tab styles
.iam-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.iam-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  background: var(--bg-secondary, #1e293b);
  border-radius: 0.5rem;
  border: 1px solid var(--border-color, #334155);
  transition: all 0.2s ease;

  &:hover {
    border-color: #9c27b0;
  }
}

.iam-info {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.iam-details {
  h4 {
    margin: 0;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-primary, #e2e8f0);
  }

  .iam-meta {
    margin: 0.25rem 0 0 0;
    font-size: 0.75rem;
    color: var(--text-secondary, #94a3b8);
  }
}

.iam-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 0.375rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  white-space: nowrap;

  &.badge--role {
    background: rgba(156, 39, 176, 0.1);
    color: #9c27b0;
  }

  &.badge--policy {
    background: rgba(233, 30, 99, 0.1);
    color: #e91e63;
  }
}

// GitOps tab styles
.gitops-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.gitops-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  background: var(--bg-secondary, #1e293b);
  border-radius: 0.5rem;
  border: 1px solid var(--border-color, #334155);
  transition: all 0.2s ease;

  &:hover {
    border-color: #f15a22;
  }
}

.gitops-info {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.gitops-details {
  h4 {
    margin: 0;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-primary, #e2e8f0);
  }

  .gitops-meta {
    margin: 0.25rem 0 0 0;
    font-size: 0.75rem;
    color: var(--text-secondary, #94a3b8);
  }
}

.gitops-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 0.375rem;
  font-size: 0.75rem;
  font-weight: 600;
  background: rgba(241, 90, 34, 0.1);
  color: #f15a22;
  white-space: nowrap;
}

.gitops-status {
  display: flex;
  gap: 0.5rem;
}

.health-indicator {
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  white-space: nowrap;

  &.healthy {
    background: rgba(34, 197, 94, 0.1);
    color: #22c55e;
  }

  &.degraded {
    background: rgba(245, 158, 11, 0.1);
    color: #f59e0b;
  }

  &.unknown {
    background: rgba(148, 163, 184, 0.1);
    color: #94a3b8;
  }
}
</style>

