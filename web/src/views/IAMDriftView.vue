<template>
  <div class="iam-drift-view">
    <header class="page-header">
      <h1 class="page-title">IAM Drift Detection</h1>
      <p class="page-subtitle">Track drift between Crossplane and AWS IAM resources</p>
    </header>

    <!-- Summary Cards -->
    <div class="summary-cards">
      <div class="summary-card">
        <div class="card-value">{{ summary.totalRoles }}</div>
        <div class="card-label">Total Roles</div>
      </div>
      <div class="summary-card warning">
        <div class="card-value">{{ summary.rolesWithDrift }}</div>
        <div class="card-label">Roles with Drift</div>
      </div>
      <div class="summary-card danger">
        <div class="card-value">{{ summary.extraPrivileges }}</div>
        <div class="card-label">Extra Privileges</div>
      </div>
      <div class="summary-card">
        <div class="card-value">{{ summary.pausedResources }}</div>
        <div class="card-label">Paused Resources</div>
      </div>
    </div>

    <!-- Drift Table -->
    <div class="drift-table">
      <div class="table-header">
        <h3>IAM Resources</h3>
        <button class="scan-btn" @click="triggerScan">
          🔄 Trigger Scan
        </button>
      </div>
      <table>
        <thead>
          <tr>
            <th>Resource</th>
            <th>Tenant</th>
            <th>Type</th>
            <th>Drift Status</th>
            <th>Severity</th>
            <th>Paused</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="resource in iamResources" :key="resource.id">
            <td class="resource-name">{{ resource.name }}</td>
            <td>{{ resource.tenant }}</td>
            <td>{{ resource.type }}</td>
            <td>
              <span class="drift-status" :class="resource.driftType">
                {{ resource.driftType === 'none' ? 'No Drift' : resource.driftType }}
              </span>
            </td>
            <td>
              <span class="severity" :class="resource.severity" v-if="resource.severity">
                {{ resource.severity }}
              </span>
              <span v-else>-</span>
            </td>
            <td>
              <span v-if="resource.isPaused" class="paused-badge">Yes</span>
              <span v-else>No</span>
            </td>
            <td>
              <button class="action-btn" @click="showDiff(resource)" :disabled="resource.driftType === 'none'">
                View Diff
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <p class="note">
      💡 Full IAM drift detection will be implemented in Phase 4
    </p>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref } from 'vue'

interface IAMResource {
  id: string
  name: string
  tenant: string
  type: 'Role' | 'Policy'
  driftType: 'none' | 'extra_privileges' | 'missing_privileges' | 'mixed'
  severity: 'critical' | 'high' | 'medium' | 'low' | null
  isPaused: boolean
}

export default defineComponent({
  name: 'IAMDriftView',
  setup() {
    const summary = ref({
      totalRoles: 4,
      rolesWithDrift: 2,
      extraPrivileges: 1,
      pausedResources: 1,
    })

    // Mock data - will be replaced with API calls
    const iamResources = ref<IAMResource[]>([
      { id: '1', name: 'tenant-alpha-lambda-role', tenant: 'alpha', type: 'Role', driftType: 'extra_privileges', severity: 'critical', isPaused: false },
      { id: '2', name: 'tenant-alpha-s3-policy', tenant: 'alpha', type: 'Policy', driftType: 'none', severity: null, isPaused: false },
      { id: '3', name: 'tenant-beta-data-role', tenant: 'beta', type: 'Role', driftType: 'none', severity: null, isPaused: true },
      { id: '4', name: 'tenant-beta-dynamodb-policy', tenant: 'beta', type: 'Policy', driftType: 'missing_privileges', severity: 'medium', isPaused: false },
    ])

    const triggerScan = () => {
      alert('Scan triggered! (Will be implemented in Phase 4)')
    }

    const showDiff = (resource: IAMResource) => {
      alert(`Showing diff for ${resource.name} (Will be implemented in Phase 4)`)
    }

    return {
      summary,
      iamResources,
      triggerScan,
      showDiff,
    }
  },
})
</script>

<style lang="scss" scoped>
.iam-drift-view {
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

.summary-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.summary-card {
  background: var(--bg-secondary, #1e293b);
  border-radius: 0.75rem;
  border: 1px solid var(--border-color, #334155);
  padding: 1.5rem;
  text-align: center;

  .card-value {
    font-size: 2.5rem;
    font-weight: 700;
    color: var(--text-primary, #e2e8f0);
  }

  .card-label {
    font-size: 0.875rem;
    color: var(--text-secondary, #94a3b8);
    margin-top: 0.5rem;
  }

  &.warning .card-value {
    color: #f59e0b;
  }

  &.danger .card-value {
    color: #ef4444;
  }
}

.drift-table {
  background: var(--bg-secondary, #1e293b);
  border-radius: 0.75rem;
  border: 1px solid var(--border-color, #334155);
  overflow: hidden;
  margin-bottom: 2rem;

  .table-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 1.5rem;
    border-bottom: 1px solid var(--border-color, #334155);

    h3 {
      margin: 0;
      font-size: 1rem;
      color: var(--text-primary, #e2e8f0);
    }
  }

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
}

.resource-name {
  font-weight: 500;
  color: var(--text-primary, #e2e8f0);
}

.drift-status {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 0.375rem;
  font-size: 0.75rem;
  font-weight: 500;

  &.none {
    background: rgba(34, 197, 94, 0.1);
    color: #22c55e;
  }

  &.extra_privileges {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
  }

  &.missing_privileges {
    background: rgba(245, 158, 11, 0.1);
    color: #f59e0b;
  }

  &.mixed {
    background: rgba(139, 92, 246, 0.1);
    color: #8b5cf6;
  }
}

.severity {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: capitalize;

  &.critical {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
  }

  &.high {
    background: rgba(249, 115, 22, 0.1);
    color: #f97316;
  }

  &.medium {
    background: rgba(245, 158, 11, 0.1);
    color: #f59e0b;
  }

  &.low {
    background: rgba(59, 130, 246, 0.1);
    color: #3b82f6;
  }
}

.paused-badge {
  color: #8b5cf6;
  font-weight: 600;
}

.scan-btn,
.action-btn {
  padding: 0.5rem 1rem;
  background: var(--bg-tertiary, #0f172a);
  border: 1px solid var(--border-color, #334155);
  border-radius: 0.375rem;
  color: var(--text-primary, #e2e8f0);
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover:not(:disabled) {
    background: var(--bg-hover, #334155);
    border-color: var(--accent-primary, #3b82f6);
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.note {
  text-align: center;
  color: var(--text-secondary, #94a3b8);
  padding: 1rem;
  background: var(--bg-secondary, #1e293b);
  border-radius: 0.5rem;
}
</style>

