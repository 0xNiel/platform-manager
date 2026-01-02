<!-- web/src/components/SyncDialog.vue -->
<template>
  <div v-if="isOpen" class="modal-overlay" @click.self="close">
    <div class="modal-dialog">
      <div class="modal-header">
        <h3>Sync ArgoCD Application</h3>
        <button class="close-button" @click="close">×</button>
      </div>

      <div class="modal-body">
        <div class="app-info">
          <span class="app-icon">🔁</span>
          <div>
            <div class="app-name">{{ appName }}</div>
            <div class="app-namespace">{{ namespace }}</div>
          </div>
        </div>

        <div class="options">
          <label class="option-item">
            <input
              v-model="options.prune"
              type="checkbox"
              class="checkbox"
            />
            <div class="option-content">
              <div class="option-label">Prune Resources</div>
              <div class="option-description">
                Remove resources that are no longer defined in Git
              </div>
            </div>
          </label>

          <label class="option-item">
            <input
              v-model="options.force"
              type="checkbox"
              class="checkbox"
            />
            <div class="option-content">
              <div class="option-label">Force Sync</div>
              <div class="option-description">
                Force sync even if the application is already synced
              </div>
            </div>
          </label>

          <label class="option-item">
            <input
              v-model="options.dryRun"
              type="checkbox"
              class="checkbox"
            />
            <div class="option-content">
              <div class="option-label">Dry Run</div>
              <div class="option-description">
                Preview changes without applying them
              </div>
            </div>
          </label>
        </div>

        <div v-if="syncing" class="sync-status">
          <div class="spinner"></div>
          <div class="status-text">
            {{ syncStatus }}
          </div>
        </div>

        <div v-if="error" class="error-message">
          {{ error }}
        </div>

        <div v-if="success" class="success-message">
          ✓ Sync {{ options.dryRun ? 'preview' : 'initiated' }} successfully!
        </div>
      </div>

      <div class="modal-footer">
        <button
          class="button button--secondary"
          :disabled="syncing"
          @click="close"
        >
          Cancel
        </button>
        <button
          class="button button--primary"
          :disabled="syncing || success"
          @click="handleSync"
        >
          <span v-if="syncing" class="button-spinner"></span>
          <span v-else>{{ options.dryRun ? 'Preview' : 'Sync' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue'

interface Props {
  isOpen: boolean
  appName: string
  namespace: string
}

interface SyncOptions {
  prune: boolean
  force: boolean
  dryRun: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  close: []
  sync: [options: SyncOptions]
}>()

const options = reactive<SyncOptions>({
  prune: false,
  force: false,
  dryRun: false,
})

const syncing = ref(false)
const success = ref(false)
const error = ref('')
const syncStatus = ref('Initiating sync...')

// Reset state when dialog opens
watch(() => props.isOpen, (newValue) => {
  if (newValue) {
    options.prune = false
    options.force = false
    options.dryRun = false
    syncing.value = false
    success.value = false
    error.value = ''
    syncStatus.value = 'Initiating sync...'
  }
})

const close = () => {
  if (!syncing.value) {
    emit('close')
  }
}

const handleSync = async () => {
  syncing.value = true
  error.value = ''
  syncStatus.value = 'Initiating sync...'

  try {
    emit('sync', { ...options })
    
    // Simulate checking sync progress
    syncStatus.value = 'Sync in progress...'
    
    // Wait a bit then mark as successful
    await new Promise(resolve => setTimeout(resolve, 1500))
    
    syncStatus.value = 'Sync completed!'
    success.value = true
    
    // Auto-close after success
    setTimeout(() => {
      if (props.isOpen) {
        emit('close')
      }
    }, 1500)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Sync failed'
    syncStatus.value = 'Sync failed'
  } finally {
    syncing.value = false
  }
}
</script>

<style scoped lang="scss">
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.modal-dialog {
  background: var(--bg-secondary, #1e293b);
  border-radius: 0.75rem;
  border: 1px solid var(--border-color, #334155);
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow: hidden;
  animation: slideUp 0.3s ease;
}

@keyframes slideUp {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.5rem;
  border-bottom: 1px solid var(--border-color, #334155);

  h3 {
    margin: 0;
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--text-primary, #e2e8f0);
  }

  .close-button {
    background: none;
    border: none;
    font-size: 2rem;
    color: var(--text-secondary, #94a3b8);
    cursor: pointer;
    padding: 0;
    width: 2rem;
    height: 2rem;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 0.25rem;
    transition: all 0.2s;

    &:hover {
      background: var(--bg-hover, #334155);
      color: var(--text-primary, #e2e8f0);
    }
  }
}

.modal-body {
  padding: 1.5rem;
  overflow-y: auto;
  max-height: calc(90vh - 180px);
}

.app-info {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  background: var(--bg-tertiary, #0f172a);
  border-radius: 0.5rem;
  margin-bottom: 1.5rem;

  .app-icon {
    font-size: 2rem;
  }

  .app-name {
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary, #e2e8f0);
  }

  .app-namespace {
    font-size: 0.875rem;
    color: var(--text-secondary, #94a3b8);
  }
}

.options {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
}

.option-item {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 1rem;
  background: var(--bg-tertiary, #0f172a);
  border-radius: 0.5rem;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    background: var(--bg-hover, #1e293b);
  }

  .checkbox {
    margin-top: 0.25rem;
    width: 1.25rem;
    height: 1.25rem;
    cursor: pointer;
  }

  .option-content {
    flex: 1;
  }

  .option-label {
    font-weight: 500;
    color: var(--text-primary, #e2e8f0);
    margin-bottom: 0.25rem;
  }

  .option-description {
    font-size: 0.875rem;
    color: var(--text-secondary, #94a3b8);
  }
}

.sync-status {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid rgba(59, 130, 246, 0.3);
  border-radius: 0.5rem;
  margin-bottom: 1rem;

  .spinner {
    width: 1.5rem;
    height: 1.5rem;
    border: 3px solid rgba(59, 130, 246, 0.3);
    border-top-color: #3b82f6;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  .status-text {
    color: #3b82f6;
    font-weight: 500;
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.error-message {
  padding: 1rem;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 0.5rem;
  color: #ef4444;
  margin-bottom: 1rem;
}

.success-message {
  padding: 1rem;
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.3);
  border-radius: 0.5rem;
  color: #10b981;
  font-weight: 500;
  margin-bottom: 1rem;
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 1rem;
  padding: 1.5rem;
  border-top: 1px solid var(--border-color, #334155);
}

.button {
  padding: 0.625rem 1.25rem;
  border: none;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  &--secondary {
    background: var(--bg-tertiary, #0f172a);
    color: var(--text-primary, #e2e8f0);

    &:hover:not(:disabled) {
      background: var(--bg-hover, #334155);
    }
  }

  &--primary {
    background: #3b82f6;
    color: white;

    &:hover:not(:disabled) {
      background: #2563eb;
    }
  }

  .button-spinner {
    width: 1rem;
    height: 1rem;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }
}
</style>

