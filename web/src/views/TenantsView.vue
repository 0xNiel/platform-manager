<template>
  <div class="tenants-view">
    <header class="page-header">
      <h1 class="page-title">Tenants</h1>
      <p class="page-subtitle">Overview of all platform tenants</p>
    </header>

    <div class="tenants-table">
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
import { defineComponent, ref } from 'vue'
import { useRouter } from 'vue-router'

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

export default defineComponent({
  name: 'TenantsView',
  setup() {
    const router = useRouter()

    // Mock data - will be replaced with API calls
    const tenants = ref<Tenant[]>([
      {
        id: 'alpha',
        name: 'Tenant Alpha',
        icon: '🤖',
        namespaces: ['tenant-alpha'],
        status: 'warning',
        resources: { total: 18, failed: 1 },
        iamDrift: 1,
        argoApps: { total: 3, synced: 2 },
        cpu: '450m',
        memory: '512Mi',
      },
      {
        id: 'beta',
        name: 'Tenant Beta',
        icon: '📊',
        namespaces: ['tenant-beta'],
        status: 'healthy',
        resources: { total: 12, failed: 0 },
        iamDrift: 0,
        argoApps: { total: 2, synced: 2 },
        cpu: '320m',
        memory: '384Mi',
      },
      {
        id: 'gamma',
        name: 'Tenant Gamma',
        icon: '⚙️',
        namespaces: ['tenant-gamma'],
        status: 'critical',
        resources: { total: 8, failed: 2 },
        iamDrift: 1,
        argoApps: { total: 3, synced: 1 },
        cpu: '180m',
        memory: '256Mi',
      },
    ])

    const goToTenant = (id: string) => {
      router.push(`/tenants/${id}`)
    }

    return {
      tenants,
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

