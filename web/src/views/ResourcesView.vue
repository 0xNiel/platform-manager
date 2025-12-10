<template>
  <div class="resources-view">
    <header class="page-header">
      <h1 class="page-title">Resources</h1>
      <p class="page-subtitle">All platform resources across tenants</p>
    </header>

    <!-- Filters -->
    <div class="filters">
      <select v-model="filters.tenant" class="filter-select">
        <option value="">All Tenants</option>
        <option value="alpha">Tenant Alpha</option>
        <option value="beta">Tenant Beta</option>
        <option value="gamma">Tenant Gamma</option>
      </select>

      <select v-model="filters.state" class="filter-select">
        <option value="">All States</option>
        <option value="ready">Ready</option>
        <option value="failed">Failed</option>
        <option value="waiting">Waiting</option>
        <option value="unknown">Unknown</option>
        <option value="paused">Paused</option>
      </select>

      <select v-model="filters.kind" class="filter-select">
        <option value="">All Kinds</option>
        <option value="Deployment">Deployment</option>
        <option value="Role">IAM Role</option>
        <option value="Policy">IAM Policy</option>
        <option value="Application">Argo Application</option>
      </select>

      <input
        v-model="filters.search"
        type="text"
        placeholder="Search resources..."
        class="filter-input"
      />
    </div>

    <!-- Resources Table -->
    <div class="resources-table">
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Kind</th>
            <th>Namespace</th>
            <th>Tenant</th>
            <th>State</th>
            <th>Age</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="resource in filteredResources" :key="resource.id">
            <td class="resource-name">{{ resource.name }}</td>
            <td>{{ resource.kind }}</td>
            <td>{{ resource.namespace || '-' }}</td>
            <td>{{ resource.tenant }}</td>
            <td>
              <span class="state-badge" :class="resource.state">
                {{ resource.state }}
              </span>
            </td>
            <td>{{ resource.age }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed } from 'vue'

interface Resource {
  id: string
  name: string
  kind: string
  namespace: string
  tenant: string
  state: 'ready' | 'failed' | 'waiting' | 'unknown' | 'paused'
  age: string
}

export default defineComponent({
  name: 'ResourcesView',
  setup() {
    const filters = ref({
      tenant: '',
      state: '',
      kind: '',
      search: '',
    })

    // Mock data - will be replaced with API calls
    const resources = ref<Resource[]>([
      { id: '1', name: 'ml-training-service', kind: 'Deployment', namespace: 'tenant-alpha', tenant: 'alpha', state: 'ready', age: '5d' },
      { id: '2', name: 'inference-api', kind: 'Deployment', namespace: 'tenant-alpha', tenant: 'alpha', state: 'ready', age: '5d' },
      { id: '3', name: 'tenant-alpha-lambda-role', kind: 'Role', namespace: '', tenant: 'alpha', state: 'ready', age: '10d' },
      { id: '4', name: 'tenant-alpha-s3-policy', kind: 'Policy', namespace: '', tenant: 'alpha', state: 'ready', age: '10d' },
      { id: '5', name: 'alpha-ml-platform', kind: 'Application', namespace: 'argocd', tenant: 'alpha', state: 'ready', age: '5d' },
      { id: '6', name: 'failing-deployment', kind: 'Deployment', namespace: 'tenant-alpha', tenant: 'alpha', state: 'failed', age: '1h' },
      { id: '7', name: 'etl-processor', kind: 'Deployment', namespace: 'tenant-beta', tenant: 'beta', state: 'ready', age: '3d' },
      { id: '8', name: 'data-api', kind: 'Deployment', namespace: 'tenant-beta', tenant: 'beta', state: 'ready', age: '3d' },
      { id: '9', name: 'tenant-beta-data-role', kind: 'Role', namespace: '', tenant: 'beta', state: 'paused', age: '8d' },
      { id: '10', name: 'crashloop-pod', kind: 'Pod', namespace: 'tenant-beta', tenant: 'beta', state: 'failed', age: '2h' },
    ])

    const filteredResources = computed(() => {
      return resources.value.filter((r) => {
        if (filters.value.tenant && r.tenant !== filters.value.tenant) return false
        if (filters.value.state && r.state !== filters.value.state) return false
        if (filters.value.kind && r.kind !== filters.value.kind) return false
        if (filters.value.search && !r.name.toLowerCase().includes(filters.value.search.toLowerCase())) return false
        return true
      })
    })

    return {
      filters,
      filteredResources,
    }
  },
})
</script>

<style lang="scss" scoped>
.resources-view {
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

.filters {
  display: flex;
  gap: 1rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.filter-select,
.filter-input {
  padding: 0.5rem 1rem;
  background: var(--bg-secondary, #1e293b);
  border: 1px solid var(--border-color, #334155);
  border-radius: 0.5rem;
  color: var(--text-primary, #e2e8f0);
  font-size: 0.875rem;

  &:focus {
    outline: none;
    border-color: var(--accent-primary, #3b82f6);
  }
}

.filter-input {
  min-width: 200px;
}

.resources-table {
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

  tr:last-child td {
    border-bottom: none;
  }

  tr:hover {
    background: var(--bg-hover, #334155);
  }
}

.resource-name {
  font-weight: 500;
  color: var(--text-primary, #e2e8f0);
}

.state-badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: capitalize;

  &.ready {
    background: rgba(34, 197, 94, 0.1);
    color: #22c55e;
  }

  &.failed {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
  }

  &.waiting {
    background: rgba(245, 158, 11, 0.1);
    color: #f59e0b;
  }

  &.unknown {
    background: rgba(107, 114, 128, 0.1);
    color: #6b7280;
  }

  &.paused {
    background: rgba(139, 92, 246, 0.1);
    color: #8b5cf6;
  }
}
</style>

