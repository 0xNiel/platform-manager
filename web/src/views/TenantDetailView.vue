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
        <p class="placeholder">Resource list will be implemented in Phase 2</p>
      </div>

      <!-- IAM Tab -->
      <div v-if="activeTab === 'iam'" class="iam-tab">
        <p class="placeholder">IAM drift details will be implemented in Phase 4</p>
      </div>

      <!-- GitOps Tab -->
      <div v-if="activeTab === 'gitops'" class="gitops-tab">
        <p class="placeholder">ArgoCD applications will be shown here</p>
      </div>

      <!-- Terminal Tab -->
      <div v-if="activeTab === 'terminal'" class="terminal-tab">
        <p class="placeholder">Web terminal will be implemented in Phase 6</p>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'

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

export default defineComponent({
  name: 'TenantDetailView',
  props: {
    id: {
      type: String,
      required: true,
    },
  },
  setup(props) {
    const route = useRoute()
    const activeTab = ref('overview')

    const tabs = [
      { id: 'overview', label: 'Overview' },
      { id: 'resources', label: 'Resources' },
      { id: 'iam', label: 'IAM' },
      { id: 'gitops', label: 'GitOps' },
      { id: 'terminal', label: 'Terminal' },
    ]

    // Mock data - will be replaced with API calls
    const tenant = ref<TenantDetail | null>(null)

    const loadTenant = () => {
      // Simulate API call
      const mockTenants: Record<string, TenantDetail> = {
        alpha: {
          id: 'alpha',
          name: 'Tenant Alpha',
          status: 'warning',
          resources: { total: 18, ready: 17, failed: 1 },
          cpu: '450m',
          memory: '512Mi',
          iamDrift: 1,
        },
        beta: {
          id: 'beta',
          name: 'Tenant Beta',
          status: 'healthy',
          resources: { total: 12, ready: 12, failed: 0 },
          cpu: '320m',
          memory: '384Mi',
          iamDrift: 0,
        },
        gamma: {
          id: 'gamma',
          name: 'Tenant Gamma',
          status: 'critical',
          resources: { total: 8, ready: 6, failed: 2 },
          cpu: '180m',
          memory: '256Mi',
          iamDrift: 1,
        },
      }

      tenant.value = mockTenants[props.id] || null
    }

    onMounted(() => {
      loadTenant()
    })

    return {
      tenant,
      activeTab,
      tabs,
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
}
</style>

