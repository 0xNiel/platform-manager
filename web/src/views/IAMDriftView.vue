<template>
  <div class="iam-drift-view">
    <div class="header">
      <h1>IAM Drift Detection</h1>
      <button 
        @click="triggerScan" 
        :disabled="scanning"
        class="scan-button"
      >
        {{ scanning ? 'Scanning...' : 'Trigger Scan' }}
      </button>
    </div>

    <div v-if="loading" class="loading">
      <div class="spinner"></div>
      <p>Loading drift data...</p>
    </div>

    <div v-else-if="error" class="error">
      <p>{{ error }}</p>
    </div>

    <div v-else class="drift-content">
      <!-- Platform-wide Summary -->
      <div class="summary-card">
        <h2>Platform Summary</h2>
        <div class="summary-stats">
          <div class="stat">
            <span class="stat-label">Total Roles</span>
            <span class="stat-value">{{ platformSummary.totalRoles }}</span>
          </div>
          <div class="stat">
            <span class="stat-label">Total Policies</span>
            <span class="stat-value">{{ platformSummary.totalPolicies }}</span>
          </div>
          <div class="stat critical">
            <span class="stat-label">Critical Drifts</span>
            <span class="stat-value">{{ platformSummary.criticalDrifts }}</span>
          </div>
          <div class="stat high">
            <span class="stat-label">High Drifts</span>
            <span class="stat-value">{{ platformSummary.highDrifts }}</span>
          </div>
          <div class="stat warning">
            <span class="stat-label">Warning Drifts</span>
            <span class="stat-value">{{ platformSummary.warningDrifts }}</span>
          </div>
        </div>
        <div class="last-checked">
          Last scanned: {{ formatTime(lastScanTime) }}
        </div>
      </div>

      <!-- Tenant Drift List -->
      <div class="tenants-section">
        <h2>Tenants</h2>
        <div v-if="tenants.length === 0" class="no-data">
          No tenant drift data available. Trigger a scan to start monitoring.
        </div>
        <div v-else class="tenant-cards">
          <div 
            v-for="tenant in tenants" 
            :key="tenant.tenantName"
            class="tenant-card"
            :class="{ 'has-drift': tenant.rolesWithDrift > 0 || tenant.policiesWithDrift > 0 }"
            @click="selectTenant(tenant.tenantName)"
          >
            <div class="tenant-header">
              <h3>{{ tenant.tenantName }}</h3>
              <span 
                v-if="tenant.rolesWithDrift > 0 || tenant.policiesWithDrift > 0" 
                class="drift-badge"
              >
                {{ tenant.rolesWithDrift + tenant.policiesWithDrift }} drifts
              </span>
            </div>
            <div class="tenant-stats">
              <div class="stat-row">
                <span>Roles: {{ tenant.rolesWithDrift }} / {{ tenant.totalRoles }}</span>
                <span>Policies: {{ tenant.policiesWithDrift }} / {{ tenant.totalPolicies }}</span>
              </div>
              <div class="severity-row">
                <span class="critical">⚠️ {{ tenant.criticalDrifts }}</span>
                <span class="high">⚡ {{ tenant.highDrifts }}</span>
                <span class="warning">⚠ {{ tenant.warningDrifts }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Detailed Drift View (when tenant selected) -->
      <div v-if="selectedTenant" class="drift-details">
        <div class="details-header">
          <h2>Drift Details: {{ selectedTenant }}</h2>
          <button @click="selectedTenant = null" class="close-button">Close</button>
        </div>

        <div v-if="selectedTenantDetails.length === 0" class="no-drift">
          ✅ No drift detected for this tenant
        </div>

        <div v-else class="drift-items">
          <div 
            v-for="(drift, index) in selectedTenantDetails" 
            :key="index"
            class="drift-item"
            :class="`severity-${drift.severity}`"
          >
            <div class="drift-item-header">
              <div class="resource-info">
                <span class="resource-type">{{ drift.resourceType }}</span>
                <span class="resource-name">{{ drift.resourceName }}</span>
              </div>
              <span class="severity-badge" :class="`severity-${drift.severity}`">
                {{ drift.severity }}
              </span>
            </div>

            <div v-if="drift.details && drift.details.length > 0" class="drift-findings">
              <h4>Findings:</h4>
              <div 
                v-for="(detail, detailIndex) in drift.details" 
                :key="detailIndex"
                class="finding"
              >
                <div class="finding-header">
                  <span class="finding-type">{{ detail.type }}</span>
                  <span class="finding-severity">{{ detail.severity }}</span>
                </div>
                <p class="finding-message">{{ detail.message }}</p>
                <div v-if="detail.path" class="finding-path">
                  Path: <code>{{ detail.path }}</code>
                </div>
                <div v-if="detail.diff" class="finding-diff">
                  <pre>{{ detail.diff }}</pre>
                </div>
              </div>
            </div>

            <div class="drift-timestamp">
              Checked: {{ formatTime(drift.checkedAt) }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, onMounted } from 'vue'
import axios from 'axios'

interface TenantDriftSummary {
  tenantName: string
  totalRoles: number
  totalPolicies: number
  rolesWithDrift: number
  policiesWithDrift: number
  criticalDrifts: number
  highDrifts: number
  warningDrifts: number
  lastChecked: string
}

interface PlatformDriftSummary {
  totalRoles: number
  totalPolicies: number
  rolesWithDrift: number
  policiesWithDrift: number
  criticalDrifts: number
  highDrifts: number
  warningDrifts: number
  lastChecked: string
}

interface DriftResult {
  resourceType: string
  resourceName: string
  resourceArn?: string
  tenantName: string
  hasDrift: boolean
  severity: string
  driftTypes?: string[]
  details?: Array<{
    type: string
    severity: string
    path?: string
    message: string
    diff?: string
  }>
  checkedAt: string
  error?: string
}

export default defineComponent({
  name: 'IAMDriftView',
  setup() {
    const loading = ref(true)
    const error = ref('')
    const scanning = ref(false)
    const platformSummary = ref<PlatformDriftSummary>({
      totalRoles: 0,
      totalPolicies: 0,
      rolesWithDrift: 0,
      policiesWithDrift: 0,
      criticalDrifts: 0,
      highDrifts: 0,
      warningDrifts: 0,
      lastChecked: '',
    })
    const tenants = ref<TenantDriftSummary[]>([])
    const lastScanTime = ref<string>('')
    const selectedTenant = ref<string | null>(null)
    const selectedTenantDetails = ref<DriftResult[]>([])

    const API_BASE = 'http://localhost:9080/api/v1'

    const loadDriftData = async () => {
      try {
        loading.value = true
        error.value = ''

        // Load platform summary
        const platformResponse = await axios.get(`${API_BASE}/iam/drift/platform`)
        platformSummary.value = platformResponse.data.summary || {}
        lastScanTime.value = platformResponse.data.lastScanTime || ''

        // Load tenant summaries
        const tenantsResponse = await axios.get(`${API_BASE}/iam/drift/tenants`)
        tenants.value = tenantsResponse.data.tenants || []
      } catch (err: unknown) {
        const errorMessage = axios.isAxiosError(err) 
          ? err.response?.data?.message || err.message 
          : 'Failed to load drift data'
        error.value = errorMessage
        console.error('Failed to load drift data:', err)
      } finally {
        loading.value = false
      }
    }

    const triggerScan = async () => {
      try {
        scanning.value = true
        await axios.post(`${API_BASE}/iam/drift/scan`)
        
        // Wait a moment then reload
        setTimeout(loadDriftData, 2000)
      } catch (err: unknown) {
        const errorMessage = axios.isAxiosError(err)
          ? err.response?.data?.message || err.message
          : 'Failed to trigger scan'
        error.value = errorMessage
        console.error('Failed to trigger scan:', err)
      } finally {
        scanning.value = false
      }
    }

    const selectTenant = async (tenantName: string) => {
      selectedTenant.value = tenantName
      
      try {
        const response = await axios.get(`${API_BASE}/iam/drift/tenants/${tenantName}`)
        selectedTenantDetails.value = response.data.details || []
      } catch (err: unknown) {
        console.error('Failed to load tenant details:', err)
        selectedTenantDetails.value = []
      }
    }

    const formatTime = (timestamp: string) => {
      if (!timestamp) return 'Never'
      const date = new Date(timestamp)
      return date.toLocaleString()
    }

    onMounted(() => {
      loadDriftData()
      // Auto-refresh every 30 seconds
      const interval = setInterval(loadDriftData, 30000)
      return () => clearInterval(interval)
    })

    return {
      loading,
      error,
      scanning,
      platformSummary,
      tenants,
      lastScanTime,
      selectedTenant,
      selectedTenantDetails,
      loadDriftData,
      triggerScan,
      selectTenant,
      formatTime,
    }
  },
})
</script>

<style scoped lang="scss">
.iam-drift-view {
  max-width: 1400px;
  margin: 0 auto;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;

  h1 {
    font-size: 2rem;
    font-weight: 700;
    color: var(--text-primary, #e2e8f0);
    margin: 0;
  }
}

.scan-button {
  padding: 0.75rem 1.5rem;
  background: var(--accent-primary, #3b82f6);
  color: white;
  border: none;
  border-radius: 0.5rem;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 600;
  transition: all 0.2s ease;

  &:hover:not(:disabled) {
    background: var(--accent-hover, #2563eb);
    transform: translateY(-1px);
  }

  &:disabled {
    background: var(--bg-tertiary, #334155);
    cursor: not-allowed;
    opacity: 0.5;
  }
}

.loading, .error, .no-data {
  text-align: center;
  padding: 3rem;
  color: var(--text-secondary, #94a3b8);
}

.spinner {
  width: 50px;
  height: 50px;
  border: 4px solid var(--bg-tertiary, #334155);
  border-top: 4px solid var(--accent-primary, #3b82f6);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 1.5rem;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.summary-card {
  background: var(--bg-secondary, #1e293b);
  border: 1px solid var(--border-color, #334155);
  border-radius: 0.75rem;
  padding: 1.5rem;
  margin-bottom: 2rem;

  h2 {
    margin: 0 0 1.5rem 0;
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--text-primary, #e2e8f0);
  }
}

.summary-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 1rem;
  margin-bottom: 1rem;
}

.stat {
  display: flex;
  flex-direction: column;
  padding: 1rem;
  background: var(--bg-tertiary, #0f172a);
  border-radius: 0.5rem;
  border-left: 4px solid var(--accent-primary, #3b82f6);

  &.critical {
    border-left-color: #ef4444;
    background: rgba(239, 68, 68, 0.1);
  }

  &.high {
    border-left-color: #f59e0b;
    background: rgba(245, 158, 11, 0.1);
  }

  &.warning {
    border-left-color: #fbbf24;
    background: rgba(251, 191, 36, 0.1);
  }
}

.stat-label {
  font-size: 0.75rem;
  color: var(--text-secondary, #94a3b8);
  margin-bottom: 0.25rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-value {
  font-size: 1.875rem;
  font-weight: 700;
  color: var(--text-primary, #e2e8f0);
}

.last-checked {
  font-size: 0.875rem;
  color: var(--text-secondary, #94a3b8);
  margin-top: 0.75rem;
}

.tenants-section {
  margin-bottom: 2rem;

  h2 {
    margin-bottom: 1.5rem;
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--text-primary, #e2e8f0);
  }
}

.tenant-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1rem;
}

.tenant-card {
  background: var(--bg-secondary, #1e293b);
  border: 1px solid var(--border-color, #334155);
  border-radius: 0.75rem;
  padding: 1.5rem;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    transform: translateY(-2px);
    border-color: var(--accent-primary, #3b82f6);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  }

  &.has-drift {
    border-left: 4px solid #ef4444;
  }
}

.tenant-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;

  h3 {
    margin: 0;
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary, #e2e8f0);
  }
}

.drift-badge {
  background: #ef4444;
  color: white;
  padding: 0.25rem 0.75rem;
  border-radius: 1rem;
  font-size: 0.75rem;
  font-weight: 700;
}

.tenant-stats {
  .stat-row, .severity-row {
    display: flex;
    justify-content: space-between;
    margin-bottom: 0.5rem;
    font-size: 0.875rem;
    color: var(--text-secondary, #94a3b8);
  }

  .severity-row {
    span {
      font-weight: 700;

      &.critical { color: #ef4444; }
      &.high { color: #f59e0b; }
      &.warning { color: #fbbf24; }
    }
  }
}

.drift-details {
  background: var(--bg-secondary, #1e293b);
  border: 1px solid var(--border-color, #334155);
  border-radius: 0.75rem;
  padding: 1.5rem;
}

.details-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;

  h2 {
    margin: 0;
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--text-primary, #e2e8f0);
  }
}

.close-button {
  padding: 0.5rem 1rem;
  background: var(--bg-tertiary, #334155);
  color: var(--text-primary, #e2e8f0);
  border: none;
  border-radius: 0.5rem;
  cursor: pointer;
  font-size: 0.875rem;
  transition: all 0.2s ease;

  &:hover {
    background: var(--bg-hover, #475569);
  }
}

.drift-items {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.drift-item {
  border: 1px solid var(--border-color, #334155);
  border-radius: 0.5rem;
  padding: 1rem;
  background: var(--bg-tertiary, #0f172a);

  &.severity-critical {
    border-left: 4px solid #ef4444;
  }

  &.severity-high {
    border-left: 4px solid #f59e0b;
  }

  &.severity-warning {
    border-left: 4px solid #fbbf24;
  }
}

.drift-item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.resource-info {
  display: flex;
  gap: 0.75rem;
  align-items: center;

  .resource-type {
    background: rgba(59, 130, 246, 0.2);
    padding: 0.25rem 0.5rem;
    border-radius: 0.25rem;
    font-size: 0.75rem;
    font-weight: 700;
    color: #60a5fa;
    text-transform: uppercase;
  }

  .resource-name {
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary, #e2e8f0);
  }
}

.severity-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 0.25rem;
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;

  &.severity-critical {
    background: #ef4444;
    color: white;
  }

  &.severity-high {
    background: #f59e0b;
    color: white;
  }

  &.severity-warning {
    background: #fbbf24;
    color: #1e293b;
  }
}

.drift-findings {
  margin-top: 1rem;

  h4 {
    margin: 0 0 0.75rem 0;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-secondary, #94a3b8);
    text-transform: uppercase;
  }
}

.finding {
  background: rgba(15, 23, 42, 0.5);
  padding: 0.75rem;
  border-radius: 0.375rem;
  margin-bottom: 0.75rem;
  border: 1px solid var(--border-color, #334155);
}

.finding-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 0.5rem;

  .finding-type {
    font-size: 0.75rem;
    font-weight: 700;
    color: #60a5fa;
    text-transform: uppercase;
  }

  .finding-severity {
    font-size: 0.75rem;
    color: var(--text-secondary, #94a3b8);
  }
}

.finding-message {
  margin: 0.5rem 0;
  font-size: 0.875rem;
  color: var(--text-primary, #e2e8f0);
  line-height: 1.5;
}

.finding-path {
  margin-top: 0.5rem;
  font-size: 0.75rem;
  color: var(--text-secondary, #94a3b8);

  code {
    background: var(--bg-secondary, #1e293b);
    padding: 0.125rem 0.375rem;
    border-radius: 0.25rem;
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    color: #22d3ee;
  }
}

.finding-diff {
  margin-top: 0.75rem;

  pre {
    background: var(--bg-secondary, #1e293b);
    padding: 0.75rem;
    border-radius: 0.375rem;
    font-size: 0.75rem;
    overflow-x: auto;
    color: var(--text-primary, #e2e8f0);
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  }
}

.drift-timestamp {
  margin-top: 0.75rem;
  font-size: 0.75rem;
  color: var(--text-secondary, #64748b);
}

.no-drift {
  text-align: center;
  padding: 3rem;
  font-size: 1.125rem;
  color: #22c55e;
}
</style>

