<template>
  <div class="troubleshooting-view">
    <div class="header">
      <h1>Troubleshooting</h1>
      <div class="actions">
        <button @click="refreshFindings" class="btn btn-primary" :disabled="loading">
          <span v-if="loading">⟳ Scanning...</span>
          <span v-else>↻ Refresh</span>
        </button>
        <button @click="triggerScan" class="btn btn-secondary" :disabled="scanning">
          <span v-if="scanning">⚡ Scanning...</span>
          <span v-else>⚡ Trigger Scan</span>
        </button>
      </div>
    </div>

    <!-- Summary Cards -->
    <div class="summary-cards" v-if="summary">
      <div class="card summary-card">
        <div class="card-header">
          <h3>Total Findings</h3>
        </div>
        <div class="card-body">
          <div class="metric-large">{{ summary.totalFindings }}</div>
          <div class="metric-label">Active Issues</div>
        </div>
      </div>

      <div class="card summary-card critical">
        <div class="card-header">
          <h3>Critical</h3>
        </div>
        <div class="card-body">
          <div class="metric-large">{{ summary.critical }}</div>
          <div class="metric-label">Requires Immediate Attention</div>
        </div>
      </div>

      <div class="card summary-card high">
        <div class="card-header">
          <h3>High</h3>
        </div>
        <div class="card-body">
          <div class="metric-large">{{ summary.high }}</div>
          <div class="metric-label">Important Issues</div>
        </div>
      </div>

      <div class="card summary-card medium">
        <div class="card-header">
          <h3>Medium</h3>
        </div>
        <div class="card-body">
          <div class="metric-large">{{ summary.medium }}</div>
          <div class="metric-label">Moderate Issues</div>
        </div>
      </div>

      <div class="card summary-card low">
        <div class="card-header">
          <h3>Low</h3>
        </div>
        <div class="card-body">
          <div class="metric-large">{{ summary.low }}</div>
          <div class="metric-label">Minor Issues</div>
        </div>
      </div>
    </div>

    <!-- Filters -->
    <div class="filters">
      <div class="filter-group">
        <label>Severity:</label>
        <select v-model="selectedSeverity" @change="applyFilters">
          <option value="">All</option>
          <option value="critical">Critical</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
          <option value="info">Info</option>
        </select>
      </div>

      <div class="filter-group">
        <label>Tenant:</label>
        <select v-model="selectedTenant" @change="applyFilters">
          <option value="">All Tenants</option>
          <option v-for="tenant in tenants" :key="tenant" :value="tenant">
            {{ tenant }}
          </option>
        </select>
      </div>

      <div class="filter-group">
        <label>Search:</label>
        <input
          type="text"
          v-model="searchQuery"
          @input="applyFilters"
          placeholder="Search findings..."
        />
      </div>
    </div>

    <!-- Top Findings Section -->
    <div class="section" v-if="summary && summary.topFindings && summary.topFindings.length > 0">
      <h2>🔥 Top Issues</h2>
      <div class="findings-list">
        <div
          v-for="finding in summary.topFindings.slice(0, 5)"
          :key="finding.id"
          class="finding-card"
          :class="'severity-' + finding.severity"
        >
          <div class="finding-header">
            <div class="finding-severity">
              <span class="severity-badge" :class="finding.severity">
                {{ finding.severity.toUpperCase() }}
              </span>
            </div>
            <div class="finding-title">
              <h3>{{ finding.title }}</h3>
              <div class="finding-meta">
                <span class="rule-name">{{ finding.ruleName }}</span>
                <span class="resource-ref">{{ formatResourceRef(finding.resourceRef) }}</span>
                <span v-if="finding.tenantName" class="tenant-name">{{ finding.tenantName }}</span>
              </div>
            </div>
            <div class="finding-actions">
              <button @click="toggleFinding(finding.id)" class="btn btn-sm">
                {{ expandedFindings.has(finding.id) ? '▼' : '▶' }}
              </button>
            </div>
          </div>

          <div v-if="expandedFindings.has(finding.id)" class="finding-details">
            <div class="detail-section">
              <h4>Description</h4>
              <p>{{ finding.message }}</p>
            </div>

            <div class="detail-section" v-if="finding.recommendation">
              <h4>💡 Recommendation</h4>
              <p class="recommendation">{{ finding.recommendation }}</p>
            </div>

            <div class="detail-section">
              <h4>Details</h4>
              <div class="detail-grid">
                <div class="detail-item">
                  <span class="label">First Seen:</span>
                  <span class="value">{{ formatTimestamp(finding.firstSeen) }}</span>
                </div>
                <div class="detail-item">
                  <span class="label">Last Seen:</span>
                  <span class="value">{{ formatTimestamp(finding.lastSeen) }}</span>
                </div>
                <div class="detail-item">
                  <span class="label">Occurrences:</span>
                  <span class="value">{{ finding.occurrences }}</span>
                </div>
                <div class="detail-item">
                  <span class="label">Status:</span>
                  <span class="value">{{ finding.status }}</span>
                </div>
              </div>
            </div>

            <div class="finding-actions-bar">
              <button @click="viewResource(finding)" class="btn btn-sm btn-primary">
                View Resource
              </button>
              <button @click="resolveFinding(finding.id)" class="btn btn-sm btn-success">
                Mark Resolved
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- All Findings -->
    <div class="section">
      <h2>All Findings ({{ filteredFindings.length }})</h2>
      
      <div v-if="loading" class="loading">
        <div class="spinner"></div>
        <p>Loading findings...</p>
      </div>

      <div v-else-if="filteredFindings.length === 0" class="empty-state">
        <div class="empty-icon">✓</div>
        <h3>No Issues Found</h3>
        <p>All systems are operating normally</p>
      </div>

      <div v-else class="findings-list">
        <div
          v-for="finding in paginatedFindings"
          :key="finding.id"
          class="finding-card"
          :class="'severity-' + finding.severity"
        >
          <div class="finding-header">
            <div class="finding-severity">
              <span class="severity-badge" :class="finding.severity">
                {{ finding.severity.toUpperCase() }}
              </span>
            </div>
            <div class="finding-title">
              <h3>{{ finding.title }}</h3>
              <div class="finding-meta">
                <span class="rule-name">{{ finding.ruleName }}</span>
                <span class="resource-ref">{{ formatResourceRef(finding.resourceRef) }}</span>
                <span v-if="finding.tenantName" class="tenant-name">{{ finding.tenantName }}</span>
              </div>
            </div>
            <div class="finding-actions">
              <button @click="toggleFinding(finding.id)" class="btn btn-sm">
                {{ expandedFindings.has(finding.id) ? '▼' : '▶' }}
              </button>
            </div>
          </div>

          <div v-if="expandedFindings.has(finding.id)" class="finding-details">
            <div class="detail-section">
              <h4>Description</h4>
              <p>{{ finding.message }}</p>
            </div>

            <div class="detail-section" v-if="finding.recommendation">
              <h4>💡 Recommendation</h4>
              <p class="recommendation">{{ finding.recommendation }}</p>
            </div>

            <div class="detail-section">
              <h4>Details</h4>
              <div class="detail-grid">
                <div class="detail-item">
                  <span class="label">First Seen:</span>
                  <span class="value">{{ formatTimestamp(finding.firstSeen) }}</span>
                </div>
                <div class="detail-item">
                  <span class="label">Last Seen:</span>
                  <span class="value">{{ formatTimestamp(finding.lastSeen) }}</span>
                </div>
                <div class="detail-item">
                  <span class="label">Occurrences:</span>
                  <span class="value">{{ finding.occurrences }}</span>
                </div>
                <div class="detail-item">
                  <span class="label">Status:</span>
                  <span class="value">{{ finding.status }}</span>
                </div>
              </div>
            </div>

            <div class="finding-actions-bar">
              <button @click="viewResource(finding)" class="btn btn-sm btn-primary">
                View Resource
              </button>
              <button @click="resolveFinding(finding.id)" class="btn btn-sm btn-success">
                Mark Resolved
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="pagination">
        <button @click="currentPage--" :disabled="currentPage === 1" class="btn btn-sm">
          Previous
        </button>
        <span class="page-info">Page {{ currentPage }} of {{ totalPages }}</span>
        <button @click="currentPage++" :disabled="currentPage === totalPages" class="btn btn-sm">
          Next
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, onMounted, onUnmounted } from 'vue'
import apiClient from '../api/client'

interface ResourceRef {
  group?: string
  version: string
  kind: string
  namespace?: string
  name: string
  uid?: string
}

interface Finding {
  id: string
  ruleId: string
  ruleName: string
  severity: string
  status: string
  title: string
  message: string
  recommendation?: string
  resourceRef: ResourceRef
  tenantName?: string
  firstSeen: string
  lastSeen: string
  occurrences: number
  metadata?: Record<string, string>
}

interface Summary {
  totalFindings: number
  critical: number
  high: number
  medium: number
  low: number
  info: number
  byTenant: Array<{
    tenantName: string
    total: number
    critical: number
    high: number
    medium: number
    low: number
    info: number
  }>
  topFindings: Finding[]
  lastEvaluation: string
}

export default defineComponent({
  name: 'TroubleshootingView',
  setup() {
    
    const summary = ref<Summary | null>(null)
    const findings = ref<Finding[]>([])
    const loading = ref(false)
    const scanning = ref(false)
    const expandedFindings = ref(new Set<string>())
    
    // Filters
    const selectedSeverity = ref('')
    const selectedTenant = ref('')
    const searchQuery = ref('')
    
    // Pagination
    const currentPage = ref(1)
    const itemsPerPage = 20
    
    // Auto-refresh
    let refreshInterval: number | null = null

    const tenants = computed(() => {
      if (!summary.value) return []
      return summary.value.byTenant.map(t => t.tenantName)
    })

    const filteredFindings = computed(() => {
      let result = findings.value

      // Filter by severity
      if (selectedSeverity.value) {
        result = result.filter(f => f.severity === selectedSeverity.value)
      }

      // Filter by tenant
      if (selectedTenant.value) {
        result = result.filter(f => f.tenantName === selectedTenant.value)
      }

      // Filter by search
      if (searchQuery.value) {
        const query = searchQuery.value.toLowerCase()
        result = result.filter(f =>
          f.title.toLowerCase().includes(query) ||
          f.message.toLowerCase().includes(query) ||
          f.ruleName.toLowerCase().includes(query) ||
          f.resourceRef.name.toLowerCase().includes(query)
        )
      }

      return result
    })

    const totalPages = computed(() => {
      return Math.ceil(filteredFindings.value.length / itemsPerPage)
    })

    const paginatedFindings = computed(() => {
      const start = (currentPage.value - 1) * itemsPerPage
      const end = start + itemsPerPage
      return filteredFindings.value.slice(start, end)
    })

    const fetchSummary = async () => {
      try {
        const response = await apiClient.get(`/troubleshooting/summary`)
        summary.value = response.data
      } catch (error) {
        console.error('Failed to fetch troubleshooting summary:', error)
      }
    }

    const fetchFindings = async () => {
      try {
        loading.value = true
        const response = await apiClient.get(`/troubleshooting/findings`)
        findings.value = response.data.findings || []
      } catch (error) {
        console.error('Failed to fetch findings:', error)
        findings.value = []
      } finally {
        loading.value = false
      }
    }

    const refreshFindings = async () => {
      await Promise.all([fetchSummary(), fetchFindings()])
    }

    const triggerScan = async () => {
      try {
        scanning.value = true
        await apiClient.post(`/troubleshooting/scan`)
        // Wait a bit for scan to complete, then refresh
        setTimeout(async () => {
          await refreshFindings()
          scanning.value = false
        }, 3000)
      } catch (error) {
        console.error('Failed to trigger scan:', error)
        scanning.value = false
      }
    }

    const toggleFinding = (id: string) => {
      if (expandedFindings.value.has(id)) {
        expandedFindings.value.delete(id)
      } else {
        expandedFindings.value.add(id)
      }
      expandedFindings.value = new Set(expandedFindings.value)
    }

    const resolveFinding = async (id: string) => {
      try {
        await apiClient.post(`/troubleshooting/findings/${id}/resolve`)
        await refreshFindings()
      } catch (error) {
        console.error('Failed to resolve finding:', error)
      }
    }

    const viewResource = (finding: Finding) => {
      const ref = finding.resourceRef
      console.log('View resource:', ref)
      // TODO: Navigate to resource detail view
    }

    const formatResourceRef = (ref: ResourceRef): string => {
      if (ref.namespace) {
        return `${ref.kind}/${ref.namespace}/${ref.name}`
      }
      return `${ref.kind}/${ref.name}`
    }

    const formatTimestamp = (timestamp: string): string => {
      if (!timestamp) return 'N/A'
      const date = new Date(timestamp)
      return date.toLocaleString()
    }

    const applyFilters = () => {
      currentPage.value = 1
    }

    onMounted(() => {
      refreshFindings()
      // Auto-refresh every 30 seconds
      refreshInterval = window.setInterval(refreshFindings, 30000)
    })

    onUnmounted(() => {
      if (refreshInterval) {
        clearInterval(refreshInterval)
      }
    })

    return {
      summary,
      findings,
      loading,
      scanning,
      expandedFindings,
      selectedSeverity,
      selectedTenant,
      searchQuery,
      currentPage,
      totalPages,
      tenants,
      filteredFindings,
      paginatedFindings,
      refreshFindings,
      triggerScan,
      toggleFinding,
      resolveFinding,
      viewResource,
      formatResourceRef,
      formatTimestamp,
      applyFilters,
    }
  },
})
</script>

<style scoped>
.troubleshooting-view {
  padding: 2rem;
  background: var(--bg-primary, #0f172a);
  color: var(--text-primary, #e2e8f0);
  min-height: 100vh;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
}

.header h1 {
  font-size: 2rem;
  font-weight: 600;
  color: var(--text-primary, #e2e8f0);
}

.actions {
  display: flex;
  gap: 1rem;
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 0.375rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-primary {
  background: #3b82f6;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #2563eb;
}

.btn-secondary {
  background: var(--bg-secondary, #1e293b);
  color: var(--text-primary, #e2e8f0);
  border: 1px solid var(--border-color, #334155);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--bg-tertiary, #334155);
}

.btn-sm {
  padding: 0.25rem 0.75rem;
  font-size: 0.875rem;
}

.btn-success {
  background: #10b981;
  color: white;
}

.btn-success:hover:not(:disabled) {
  background: #059669;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.summary-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.card {
  background: var(--bg-secondary, #1e293b);
  border: 1px solid var(--border-color, #334155);
  border-radius: 0.5rem;
  padding: 1.5rem;
}

.summary-card .card-header h3 {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-secondary, #94a3b8);
  margin: 0;
}

.summary-card .card-body {
  margin-top: 1rem;
}

.metric-large {
  font-size: 2.5rem;
  font-weight: 700;
  color: var(--text-primary, #e2e8f0);
}

.metric-label {
  font-size: 0.875rem;
  color: var(--text-secondary, #94a3b8);
  margin-top: 0.5rem;
}

.summary-card.critical {
  border-left: 4px solid #ef4444;
}

.summary-card.high {
  border-left: 4px solid #f97316;
}

.summary-card.medium {
  border-left: 4px solid #eab308;
}

.summary-card.low {
  border-left: 4px solid #3b82f6;
}

.filters {
  display: flex;
  gap: 1rem;
  margin-bottom: 2rem;
  flex-wrap: wrap;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.filter-group label {
  font-weight: 500;
  color: var(--text-secondary, #94a3b8);
}

.filter-group select,
.filter-group input {
  padding: 0.5rem;
  background: var(--bg-secondary, #1e293b);
  border: 1px solid var(--border-color, #334155);
  border-radius: 0.375rem;
  color: var(--text-primary, #e2e8f0);
}

.section {
  margin-bottom: 2rem;
}

.section h2 {
  font-size: 1.5rem;
  font-weight: 600;
  margin-bottom: 1rem;
  color: var(--text-primary, #e2e8f0);
}

.findings-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.finding-card {
  background: var(--bg-secondary, #1e293b);
  border: 1px solid var(--border-color, #334155);
  border-radius: 0.5rem;
  padding: 1rem;
}

.finding-card.severity-critical {
  border-left: 4px solid #ef4444;
}

.finding-card.severity-high {
  border-left: 4px solid #f97316;
}

.finding-card.severity-medium {
  border-left: 4px solid #eab308;
}

.finding-card.severity-low {
  border-left: 4px solid #3b82f6;
}

.finding-card.severity-info {
  border-left: 4px solid #6b7280;
}

.finding-header {
  display: flex;
  gap: 1rem;
  align-items: start;
}

.finding-severity {
  flex-shrink: 0;
}

.severity-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 0.25rem;
  font-size: 0.75rem;
  font-weight: 600;
}

.severity-badge.critical {
  background: #ef4444;
  color: white;
}

.severity-badge.high {
  background: #f97316;
  color: white;
}

.severity-badge.medium {
  background: #eab308;
  color: black;
}

.severity-badge.low {
  background: #3b82f6;
  color: white;
}

.severity-badge.info {
  background: #6b7280;
  color: white;
}

.finding-title {
  flex: 1;
}

.finding-title h3 {
  font-size: 1rem;
  font-weight: 600;
  margin: 0 0 0.5rem 0;
  color: var(--text-primary, #e2e8f0);
}

.finding-meta {
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
  font-size: 0.875rem;
  color: var(--text-secondary, #94a3b8);
}

.rule-name,
.resource-ref,
.tenant-name {
  padding: 0.125rem 0.5rem;
  background: var(--bg-tertiary, #334155);
  border-radius: 0.25rem;
}

.finding-actions {
  flex-shrink: 0;
}

.finding-details {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-color, #334155);
}

.detail-section {
  margin-bottom: 1rem;
}

.detail-section h4 {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-secondary, #94a3b8);
  margin-bottom: 0.5rem;
}

.detail-section p {
  color: var(--text-primary, #e2e8f0);
  line-height: 1.6;
}

.recommendation {
  background: var(--bg-tertiary, #334155);
  padding: 0.75rem;
  border-radius: 0.375rem;
  border-left: 3px solid #10b981;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.75rem;
}

.detail-item {
  display: flex;
  justify-content: space-between;
  padding: 0.5rem;
  background: var(--bg-tertiary, #334155);
  border-radius: 0.25rem;
}

.detail-item .label {
  font-weight: 500;
  color: var(--text-secondary, #94a3b8);
}

.detail-item .value {
  color: var(--text-primary, #e2e8f0);
}

.finding-actions-bar {
  display: flex;
  gap: 0.5rem;
  margin-top: 1rem;
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 1rem;
  margin-top: 2rem;
}

.page-info {
  color: var(--text-secondary, #94a3b8);
}

.loading {
  text-align: center;
  padding: 3rem;
}

.spinner {
  border: 4px solid var(--bg-tertiary, #334155);
  border-top: 4px solid #3b82f6;
  border-radius: 50%;
  width: 40px;
  height: 40px;
  animation: spin 1s linear infinite;
  margin: 0 auto 1rem;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.empty-state {
  text-align: center;
  padding: 3rem;
}

.empty-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
}

.empty-state h3 {
  color: var(--text-primary, #e2e8f0);
  margin-bottom: 0.5rem;
}

.empty-state p {
  color: var(--text-secondary, #94a3b8);
}
</style>

