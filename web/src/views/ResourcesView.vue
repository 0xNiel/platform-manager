<!-- web/src/views/ResourcesView.vue -->
<template>
  <div class="resources-view">
    <div class="header">
      <h1>Resources</h1>
      <p class="subtitle">Manage platform resources with actions</p>
    </div>

    <!-- Filters -->
    <div class="filters">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Search resources..."
        class="search-input"
      />
      <select v-model="selectedState" class="filter-select">
        <option value="">All States</option>
        <option value="Ready">Ready</option>
        <option value="Failed">Failed</option>
        <option value="Waiting">Waiting</option>
        <option value="Paused">Paused</option>
      </select>
      <select v-model="selectedKind" class="filter-select">
        <option value="">All Kinds</option>
        <option value="Application">ArgoCD Applications</option>
        <option value="Role">IAM Roles</option>
        <option value="Policy">IAM Policies</option>
        <option value="Deployment">Deployments</option>
      </select>
    </div>

    <!-- Resources List -->
    <div v-if="loading" class="loading">Loading resources...</div>
    
    <div v-else-if="error" class="error-message">
      {{ error }}
    </div>

    <div v-else class="resources-grid">
      <div
        v-for="resource in filteredResources"
        :key="`${resource.kind}-${resource.namespace}-${resource.name}`"
        class="resource-card"
      >
        <div class="resource-header">
          <div class="resource-info">
            <span :class="['resource-badge', `badge--${resource.category.toLowerCase()}`]">
              {{ resource.category }}
            </span>
            <h3 class="resource-name">{{ resource.name }}</h3>
            <span class="resource-kind">{{ resource.kind }}</span>
          </div>
          <span :class="['status-badge', `status--${resource.state.toLowerCase()}`]">
            {{ resource.state }}
          </span>
        </div>

        <div class="resource-details">
          <div v-if="resource.namespace" class="detail">
            <span class="label">Namespace:</span>
            <span class="value">{{ resource.namespace }}</span>
          </div>
          <div v-if="resource.tenantRef" class="detail">
            <span class="label">Tenant:</span>
            <span class="value">{{ resource.tenantRef }}</span>
          </div>
          <div v-if="resource.message" class="detail message">
            {{ resource.message }}
          </div>
        </div>

        <!-- Action Buttons -->
        <div class="actions">
          <!-- ArgoCD Actions -->
          <template v-if="resource.category === 'ArgoCD'">
            <ActionButton
              label="Refresh"
              icon="🔄"
              variant="success"
              @click="() => handleRefreshArgo(resource)"
            />
            <ActionButton
              label="Sync"
              icon="🔁"
              variant="primary"
              confirm-message="Are you sure you want to sync this application?"
              @click="() => handleSyncArgo(resource)"
            />
          </template>

          <!-- Crossplane Actions -->
          <template v-if="resource.category === 'Crossplane'">
            <ActionButton
              v-if="resource.state !== 'Paused'"
              label="Pause"
              icon="⏸"
              variant="warning"
              @click="() => handlePauseCrossplane(resource)"
            />
            <ActionButton
              v-else
              label="Unpause"
              icon="▶"
              variant="success"
              @click="() => handleUnpauseCrossplane(resource)"
            />
            <ActionButton
              label="Reconcile"
              icon="⚡"
              variant="primary"
              @click="() => handleReconcileCrossplane(resource)"
            />
          </template>

          <!-- Delete Action (Admin only) -->
          <ActionButton
            v-if="canDelete"
            label="Delete"
            icon="🗑"
            variant="danger"
            @click="() => handleDelete(resource)"
          />
        </div>
      </div>
    </div>

    <div v-if="!loading && filteredResources.length === 0" class="empty-state">
      <p>No resources found matching your filters.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { api, type CrossplaneResource } from '../api/client'
import ActionButton from '../components/ActionButton.vue'
import { useToast } from '../composables/useToast'

const { success, error: showError } = useToast()

// Types
interface Resource {
  name: string
  kind: string
  namespace?: string
  tenantRef?: string
  category: string
  state: string
  message?: string
  group?: string
  version?: string
}

interface ApiResource {
  spec?: {
    name?: string
    kind?: string
    namespace?: string
    tenantRef?: string
    category?: string
    group?: string
    version?: string
  }
  metadata?: {
    name?: string
  }
  status?: {
    state?: string
    message?: string
  }
}

// State
const resources = ref<Resource[]>([])
const loading = ref(true)
const error = ref('')
const searchQuery = ref('')
const selectedState = ref('')
const selectedKind = ref('')

// Role-based permissions (would come from auth context in production)
const canDelete = ref(true) // TODO: Get from user role

// Computed
const filteredResources = computed(() => {
  let filtered = resources.value

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    filtered = filtered.filter(
      (r) =>
        r.name.toLowerCase().includes(query) ||
        r.kind.toLowerCase().includes(query) ||
        r.namespace?.toLowerCase().includes(query)
    )
  }

  if (selectedState.value) {
    filtered = filtered.filter((r) => r.state === selectedState.value)
  }

  if (selectedKind.value) {
    filtered = filtered.filter((r) => r.kind === selectedKind.value)
  }

  return filtered
})

// Methods
const loadResources = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await api.getResources() as { resources?: ApiResource[] }
    const data = response.resources || []
    resources.value = data.map((r) => ({
      name: r.spec?.name || r.metadata?.name || 'Unknown',
      kind: r.spec?.kind || 'Unknown',
      namespace: r.spec?.namespace,
      tenantRef: r.spec?.tenantRef,
      category: r.spec?.category || 'Kubernetes',
      state: r.status?.state || 'Unknown',
      message: r.status?.message,
      group: r.spec?.group,
      version: r.spec?.version,
    }))
  } catch (e) {
    const err = e as Error
    error.value = `Failed to load resources: ${err.message}`
  } finally {
    loading.value = false
  }
}

// Action Handlers
const handleRefreshArgo = async (resource: Resource) => {
  try {
    await api.refreshArgoApp(resource.name, resource.namespace || 'argocd')
    success(`Refreshed ${resource.name}`)
    await loadResources()
  } catch (e) {
    const err = e as { response?: { data?: { error?: string } }; message: string }
    showError(`Failed to refresh: ${err.response?.data?.error || err.message}`)
  }
}

const handleSyncArgo = async (resource: Resource) => {
  try {
    await api.syncArgoApp({
      name: resource.name,
      namespace: resource.namespace || 'argocd',
      prune: false,
      dryRun: false,
    })
    success(`Sync initiated for ${resource.name}`)
    await loadResources()
  } catch (e) {
    const err = e as { response?: { data?: { error?: string } }; message: string }
    showError(`Failed to sync: ${err.response?.data?.error || err.message}`)
  }
}

const handlePauseCrossplane = async (resource: Resource) => {
  try {
    const crossplaneResource: CrossplaneResource = {
      group: resource.group || 'iam.aws.upbound.io',
      version: resource.version || 'v1beta1',
      kind: resource.kind,
      namespace: resource.namespace,
      name: resource.name,
    }
    await api.pauseCrossplaneResource(crossplaneResource)
    success(`Paused ${resource.name}`)
    await loadResources()
  } catch (e) {
    const err = e as { response?: { data?: { error?: string } }; message: string }
    showError(`Failed to pause: ${err.response?.data?.error || err.message}`)
  }
}

const handleUnpauseCrossplane = async (resource: Resource) => {
  try {
    const crossplaneResource: CrossplaneResource = {
      group: resource.group || 'iam.aws.upbound.io',
      version: resource.version || 'v1beta1',
      kind: resource.kind,
      namespace: resource.namespace,
      name: resource.name,
    }
    await api.unpauseCrossplaneResource(crossplaneResource)
    success(`Unpaused ${resource.name}`)
    await loadResources()
  } catch (e) {
    const err = e as { response?: { data?: { error?: string } }; message: string }
    showError(`Failed to unpause: ${err.response?.data?.error || err.message}`)
  }
}

const handleReconcileCrossplane = async (resource: Resource) => {
  try {
    const crossplaneResource: CrossplaneResource = {
      group: resource.group || 'iam.aws.upbound.io',
      version: resource.version || 'v1beta1',
      kind: resource.kind,
      namespace: resource.namespace,
      name: resource.name,
    }
    await api.reconcileCrossplaneResource(crossplaneResource)
    success(`Reconciliation requested for ${resource.name}`)
    await loadResources()
  } catch (e) {
    const err = e as { response?: { data?: { error?: string } }; message: string }
    showError(`Failed to reconcile: ${err.response?.data?.error || err.message}`)
  }
}

const handleDelete = async (resource: Resource) => {
  const confirmed = prompt(
    `Type "${resource.name}" to confirm deletion:`,
    ''
  )
  
  if (confirmed !== resource.name) {
    showError('Deletion cancelled: confirmation did not match')
    return
  }

  try {
    await api.deleteResource({
      group: resource.group || 'apps',
      version: resource.version || 'v1',
      kind: resource.kind,
      namespace: resource.namespace,
      name: resource.name,
      confirm: resource.name,
    })
    success(`Deleted ${resource.name}`)
    await loadResources()
  } catch (e) {
    const err = e as { response?: { data?: { error?: string } }; message: string }
    showError(`Failed to delete: ${err.response?.data?.error || err.message}`)
  }
}

onMounted(() => {
  loadResources()
})
</script>

<style scoped lang="scss">
.resources-view {
  max-width: 1400px;
  margin: 0 auto;
}

.header {
  margin-bottom: 2rem;

  h1 {
    font-size: 2rem;
    font-weight: 700;
    margin-bottom: 0.5rem;
    color: #e2e8f0;
  }

  .subtitle {
    color: #94a3b8;
  }
}

.filters {
  display: flex;
  gap: 1rem;
  margin-bottom: 2rem;
  flex-wrap: wrap;
}

.search-input,
.filter-select {
  padding: 0.75rem 1rem;
  border: 1px solid #334155;
  border-radius: 0.5rem;
  background-color: #1e293b;
  color: #e2e8f0;
  font-size: 0.875rem;

  &:focus {
    outline: none;
    border-color: #3b82f6;
  }
}

.search-input {
  flex: 1;
  min-width: 200px;
}

.resources-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 1.5rem;
}

.resource-card {
  background-color: #1e293b;
  border: 1px solid #334155;
  border-radius: 0.75rem;
  padding: 1.5rem;
  transition: all 0.2s;

  &:hover {
    border-color: #3b82f6;
    box-shadow: 0 4px 6px -1px rgba(59, 130, 246, 0.1);
  }
}

.resource-header {
  display: flex;
  justify-content: space-between;
  align-items: start;
  margin-bottom: 1rem;
}

.resource-info {
  flex: 1;
}

.resource-badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 0.25rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  margin-bottom: 0.5rem;

  &.badge--argocd {
    background-color: rgba(239, 68, 68, 0.2);
    color: #fca5a5;
  }

  &.badge--crossplane {
    background-color: rgba(59, 130, 246, 0.2);
    color: #93c5fd;
  }

  &.badge--kubernetes {
    background-color: rgba(16, 185, 129, 0.2);
    color: #6ee7b7;
  }
}

.resource-name {
  font-size: 1.125rem;
  font-weight: 600;
  color: #e2e8f0;
  margin-bottom: 0.25rem;
}

.resource-kind {
  font-size: 0.875rem;
  color: #94a3b8;
}

.status-badge {
  padding: 0.375rem 0.75rem;
  border-radius: 0.375rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;

  &.status--ready {
    background-color: rgba(16, 185, 129, 0.2);
    color: #6ee7b7;
  }

  &.status--failed {
    background-color: rgba(239, 68, 68, 0.2);
    color: #fca5a5;
  }

  &.status--waiting {
    background-color: rgba(245, 158, 11, 0.2);
    color: #fcd34d;
  }

  &.status--paused {
    background-color: rgba(148, 163, 184, 0.2);
    color: #cbd5e1;
  }

  &.status--unknown {
    background-color: rgba(100, 116, 139, 0.2);
    color: #94a3b8;
  }
}

.resource-details {
  margin-bottom: 1rem;
  font-size: 0.875rem;

  .detail {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 0.5rem;

    .label {
      color: #94a3b8;
      font-weight: 500;
    }

    .value {
      color: #e2e8f0;
    }

    &.message {
      color: #94a3b8;
      font-style: italic;
    }
  }
}

.actions {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.loading,
.empty-state {
  text-align: center;
  padding: 3rem;
  color: #94a3b8;
}

.error-message {
  padding: 1rem;
  background-color: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 0.5rem;
  color: #fca5a5;
}
</style>
