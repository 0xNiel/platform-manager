<!-- web/src/components/ActionButton.vue -->
<template>
  <button
    :class="['action-button', `action-button--${variant}`, { 'is-loading': loading, 'is-disabled': disabled }]"
    :disabled="disabled || loading"
    @click="handleClick"
  >
    <span v-if="loading" class="spinner"></span>
    <span v-else class="icon">{{ icon }}</span>
    <span class="label">{{ label }}</span>
  </button>
</template>

<script setup lang="ts">
import { ref } from 'vue'

interface Props {
  label: string
  icon?: string
  variant?: 'primary' | 'danger' | 'warning' | 'success'
  disabled?: boolean
  confirmMessage?: string
}

const props = withDefaults(defineProps<Props>(), {
  icon: '▶',
  variant: 'primary',
  disabled: false,
})

const emit = defineEmits<{
  click: []
}>()

const loading = ref(false)

const handleClick = async () => {
  if (props.confirmMessage) {
    if (!confirm(props.confirmMessage)) {
      return
    }
  }

  loading.value = true
  try {
    emit('click')
  } finally {
    // Keep loading state - parent will reset it
  }
}

defineExpose({
  setLoading: (value: boolean) => {
    loading.value = value
  },
})
</script>

<style scoped lang="scss">
.action-button {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;

  &--primary {
    background-color: #3b82f6;
    color: white;

    &:hover:not(.is-disabled):not(.is-loading) {
      background-color: #2563eb;
    }
  }

  &--success {
    background-color: #10b981;
    color: white;

    &:hover:not(.is-disabled):not(.is-loading) {
      background-color: #059669;
    }
  }

  &--warning {
    background-color: #f59e0b;
    color: white;

    &:hover:not(.is-disabled):not(.is-loading) {
      background-color: #d97706;
    }
  }

  &--danger {
    background-color: #ef4444;
    color: white;

    &:hover:not(.is-disabled):not(.is-loading) {
      background-color: #dc2626;
    }
  }

  &.is-disabled,
  &.is-loading {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .spinner {
    width: 1rem;
    height: 1rem;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  .icon {
    font-size: 1rem;
  }

  .label {
    white-space: nowrap;
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>

