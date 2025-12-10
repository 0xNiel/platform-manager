<template>
  <div class="dashboard">
    <header class="dashboard-header">
      <h1 class="dashboard-title">Platform Overview</h1>
      <p class="dashboard-subtitle">30,000-foot view of your infrastructure</p>
    </header>

    <!-- Status Bar -->
    <div class="status-bar">
      <div class="status-indicator" :class="overallStatus">
        <span class="status-dot"></span>
        <span class="status-text">{{ overallStatusText }}</span>
      </div>
      <div class="status-stats">
        <div class="stat">
          <span class="stat-value">{{ stats.tenants }}</span>
          <span class="stat-label">Tenants</span>
        </div>
        <div class="stat">
          <span class="stat-value">{{ stats.resources }}</span>
          <span class="stat-label">Resources</span>
        </div>
      </div>
    </div>

    <!-- Health Cards Grid -->
    <div class="cards-grid">
      <!-- Resource State Card -->
      <div class="card">
        <div class="card-header">
          <h3 class="card-title">Resource States</h3>
        </div>
        <div class="card-body">
          <div class="state-list">
            <div class="state-item ready">
              <span class="state-count">{{ resourceStates.ready }}</span>
              <span class="state-label">Ready</span>
            </div>
            <div class="state-item failed">
              <span class="state-count">{{ resourceStates.failed }}</span>
              <span class="state-label">Failed</span>
            </div>
            <div class="state-item waiting">
              <span class="state-count">{{ resourceStates.waiting }}</span>
              <span class="state-label">Waiting</span>
            </div>
            <div class="state-item unknown">
              <span class="state-count">{{ resourceStates.unknown }}</span>
              <span class="state-label">Unknown</span>
            </div>
            <div class="state-item paused">
              <span class="state-count">{{ resourceStates.paused }}</span>
              <span class="state-label">Paused</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Crossplane Health Card -->
      <div class="card">
        <div class="card-header">
          <h3 class="card-title">Crossplane Health</h3>
        </div>
        <div class="card-body">
          <div class="metric-row">
            <span class="metric-label">Compositions</span>
            <span class="metric-value">{{ crossplane.compositions }}</span>
          </div>
          <div class="metric-row">
            <span class="metric-label">Claims</span>
            <span class="metric-value">{{ crossplane.claims }}</span>
          </div>
          <div class="metric-row">
            <span class="metric-label">XRs</span>
            <span class="metric-value">{{ crossplane.xrs }}</span>
          </div>
          <div class="metric-row warning" v-if="crossplane.failed > 0">
            <span class="metric-label">Failed</span>
            <span class="metric-value">{{ crossplane.failed }}</span>
          </div>
        </div>
      </div>

      <!-- ArgoCD Health Card -->
      <div class="card">
        <div class="card-header">
          <h3 class="card-title">GitOps / ArgoCD</h3>
        </div>
        <div class="card-body">
          <div class="metric-row">
            <span class="metric-label">Total Apps</span>
            <span class="metric-value">{{ argo.totalApps }}</span>
          </div>
          <div class="metric-row success">
            <span class="metric-label">Synced</span>
            <span class="metric-value">{{ argo.synced }}</span>
          </div>
          <div class="metric-row warning" v-if="argo.outOfSync > 0">
            <span class="metric-label">Out of Sync</span>
            <span class="metric-value">{{ argo.outOfSync }}</span>
          </div>
          <div class="metric-row danger" v-if="argo.degraded > 0">
            <span class="metric-label">Degraded</span>
            <span class="metric-value">{{ argo.degraded }}</span>
          </div>
        </div>
      </div>

      <!-- IAM Drift Card -->
      <div class="card">
        <div class="card-header">
          <h3 class="card-title">IAM Drift</h3>
        </div>
        <div class="card-body">
          <div class="metric-row">
            <span class="metric-label">Total Roles</span>
            <span class="metric-value">{{ iamDrift.totalRoles }}</span>
          </div>
          <div class="metric-row danger" v-if="iamDrift.rolesWithDrift > 0">
            <span class="metric-label">Roles with Drift</span>
            <span class="metric-value">{{ iamDrift.rolesWithDrift }}</span>
          </div>
          <div class="metric-row danger" v-if="iamDrift.extraPrivileges > 0">
            <span class="metric-label">Extra Privileges</span>
            <span class="metric-value">{{ iamDrift.extraPrivileges }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Quick Actions -->
    <div class="quick-actions">
      <router-link to="/tenants" class="action-btn">
        View All Tenants →
      </router-link>
      <router-link to="/resources?state=failed" class="action-btn danger">
        View Failed Resources →
      </router-link>
      <router-link to="/iam" class="action-btn warning">
        Review IAM Drift →
      </router-link>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, computed, ref } from 'vue'

export default defineComponent({
  name: 'DashboardView',
  setup() {
    // Mock data - will be replaced with API calls
    const stats = ref({
      tenants: 3,
      resources: 42,
    })

    const resourceStates = ref({
      ready: 35,
      failed: 3,
      waiting: 2,
      unknown: 1,
      paused: 1,
    })

    const crossplane = ref({
      compositions: 12,
      claims: 28,
      xrs: 28,
      failed: 2,
    })

    const argo = ref({
      totalApps: 8,
      synced: 6,
      outOfSync: 2,
      degraded: 0,
    })

    const iamDrift = ref({
      totalRoles: 15,
      rolesWithDrift: 2,
      extraPrivileges: 1,
    })

    const overallStatus = computed(() => {
      if (resourceStates.value.failed > 0 || iamDrift.value.extraPrivileges > 0) {
        return 'danger'
      }
      if (resourceStates.value.waiting > 0 || argo.value.outOfSync > 0) {
        return 'warning'
      }
      return 'healthy'
    })

    const overallStatusText = computed(() => {
      switch (overallStatus.value) {
        case 'danger':
          return 'Critical Issues Detected'
        case 'warning':
          return 'Attention Required'
        default:
          return 'All Systems Operational'
      }
    })

    return {
      stats,
      resourceStates,
      crossplane,
      argo,
      iamDrift,
      overallStatus,
      overallStatusText,
    }
  },
})
</script>

<style lang="scss" scoped>
.dashboard {
  max-width: 1400px;
  margin: 0 auto;
}

.dashboard-header {
  margin-bottom: 2rem;
}

.dashboard-title {
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary, #e2e8f0);
  margin: 0 0 0.5rem 0;
}

.dashboard-subtitle {
  color: var(--text-secondary, #94a3b8);
  margin: 0;
}

.status-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.5rem;
  background: var(--bg-secondary, #1e293b);
  border-radius: 0.75rem;
  margin-bottom: 2rem;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 0.75rem;

  .status-dot {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    animation: pulse 2s infinite;
  }

  &.healthy .status-dot {
    background: #22c55e;
    box-shadow: 0 0 8px #22c55e;
  }

  &.warning .status-dot {
    background: #f59e0b;
    box-shadow: 0 0 8px #f59e0b;
  }

  &.danger .status-dot {
    background: #ef4444;
    box-shadow: 0 0 8px #ef4444;
  }

  .status-text {
    font-weight: 600;
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.status-stats {
  display: flex;
  gap: 2rem;
}

.stat {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-primary, #e2e8f0);
}

.stat-label {
  font-size: 0.875rem;
  color: var(--text-secondary, #94a3b8);
}

.cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.card {
  background: var(--bg-secondary, #1e293b);
  border-radius: 0.75rem;
  border: 1px solid var(--border-color, #334155);
  overflow: hidden;
}

.card-header {
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--border-color, #334155);
}

.card-title {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary, #e2e8f0);
}

.card-body {
  padding: 1.5rem;
}

.state-list {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
}

.state-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  background: var(--bg-tertiary, #0f172a);
  min-width: 70px;

  .state-count {
    font-size: 1.5rem;
    font-weight: 700;
  }

  .state-label {
    font-size: 0.75rem;
    color: var(--text-secondary, #94a3b8);
    text-transform: uppercase;
  }

  &.ready .state-count {
    color: #22c55e;
  }
  &.failed .state-count {
    color: #ef4444;
  }
  &.waiting .state-count {
    color: #f59e0b;
  }
  &.unknown .state-count {
    color: #6b7280;
  }
  &.paused .state-count {
    color: #8b5cf6;
  }
}

.metric-row {
  display: flex;
  justify-content: space-between;
  padding: 0.5rem 0;
  border-bottom: 1px solid var(--border-color, #334155);

  &:last-child {
    border-bottom: none;
  }

  .metric-label {
    color: var(--text-secondary, #94a3b8);
  }

  .metric-value {
    font-weight: 600;
    color: var(--text-primary, #e2e8f0);
  }

  &.success .metric-value {
    color: #22c55e;
  }
  &.warning .metric-value {
    color: #f59e0b;
  }
  &.danger .metric-value {
    color: #ef4444;
  }
}

.quick-actions {
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
}

.action-btn {
  padding: 0.75rem 1.5rem;
  background: var(--bg-secondary, #1e293b);
  border: 1px solid var(--border-color, #334155);
  border-radius: 0.5rem;
  color: var(--text-primary, #e2e8f0);
  text-decoration: none;
  font-weight: 500;
  transition: all 0.2s ease;

  &:hover {
    background: var(--bg-hover, #334155);
    border-color: var(--accent-primary, #3b82f6);
  }

  &.danger {
    border-color: #ef4444;
    color: #ef4444;

    &:hover {
      background: rgba(239, 68, 68, 0.1);
    }
  }

  &.warning {
    border-color: #f59e0b;
    color: #f59e0b;

    &:hover {
      background: rgba(245, 158, 11, 0.1);
    }
  }
}
</style>

