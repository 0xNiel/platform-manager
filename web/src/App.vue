<template>
  <div id="platform-manager" class="platform-manager">
    <nav class="nav" v-if="!isEmbedded">
      <div class="nav-brand">
        <span class="nav-logo">⚡</span>
        <span class="nav-title">Platform Manager</span>
      </div>
      <div class="nav-links">
        <router-link to="/" class="nav-link">Dashboard</router-link>
        <router-link to="/tenants" class="nav-link">Tenants</router-link>
        <router-link to="/resources" class="nav-link">Resources</router-link>
        <router-link to="/iam" class="nav-link">IAM Drift</router-link>
        <router-link to="/troubleshooting" class="nav-link">Troubleshooting</router-link>
        <!-- Terminal Button -->
        <button
          v-if="terminalStore.canUseTerminal && terminalStore.isTerminalEnabled"
          class="nav-link terminal-btn"
          :class="{ active: terminalStore.isOpen, connected: terminalStore.isConnected }"
          @click="terminalStore.toggleTerminal"
          :title="terminalStore.isOpen ? 'Close Terminal' : 'Open Terminal'"
        >
          <span class="terminal-icon">💻</span>
          <span v-if="terminalStore.isConnected" class="terminal-status">●</span>
        </button>
      </div>
    </nav>
    <main class="main-content" :class="{ 'terminal-open': terminalStore.isOpen }">
      <router-view />
    </main>
    <!-- Toast notifications -->
    <ToastContainer />
    <!-- Terminal Drawer -->
    <TerminalDrawer />
  </div>
</template>

<script lang="ts">
import { defineComponent, computed, onMounted } from 'vue'
import ToastContainer from './components/ToastContainer.vue'
import TerminalDrawer from './components/TerminalDrawer.vue'
import { useTerminalStore } from './stores/terminal'

export default defineComponent({
  name: 'PlatformManager',
  components: {
    ToastContainer,
    TerminalDrawer,
  },
  setup() {
    // Check if running embedded in single-spa shell
    const isEmbedded = computed(() => !!window.singleSpaNavigate)

    // Terminal store for adjusting content height
    const terminalStore = useTerminalStore()

    // Fetch terminal capabilities on mount
    onMounted(async () => {
      await terminalStore.fetchCapabilities()
    })

    return {
      isEmbedded,
      terminalStore,
    }
  },
})
</script>

<style lang="scss">
.platform-manager {
  font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto,
    'Helvetica Neue', Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  min-height: 100vh;
  background-color: var(--bg-primary, #0f172a);
  color: var(--text-primary, #e2e8f0);
}

.nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 2rem;
  background-color: var(--bg-secondary, #1e293b);
  border-bottom: 1px solid var(--border-color, #334155);
}

.nav-brand {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.nav-logo {
  font-size: 1.5rem;
}

.nav-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary, #e2e8f0);
}

.nav-links {
  display: flex;
  gap: 1.5rem;
}

.nav-link {
  color: var(--text-secondary, #94a3b8);
  text-decoration: none;
  font-weight: 500;
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  transition: all 0.2s ease;
  background: transparent;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1rem;
  font-family: inherit;

  &:hover {
    color: var(--text-primary, #e2e8f0);
    background-color: var(--bg-hover, #334155);
  }

  &.router-link-active {
    color: var(--accent-primary, #3b82f6);
    background-color: var(--accent-bg, rgba(59, 130, 246, 0.1));
  }
}

.terminal-btn {
  position: relative;

  .terminal-icon {
    font-size: 1.25rem;
    line-height: 1;
  }

  .terminal-status {
    position: absolute;
    top: 8px;
    right: 8px;
    font-size: 8px;
    color: #10b981;
    animation: pulse-glow 2s infinite;
  }

  &.active {
    background-color: rgba(16, 185, 129, 0.15);
    color: #10b981;

    &:hover {
      background-color: rgba(16, 185, 129, 0.25);
    }
  }

  &.connected {
    .terminal-icon {
      filter: drop-shadow(0 0 4px rgba(16, 185, 129, 0.6));
    }
  }
}

@keyframes pulse-glow {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.main-content {
  padding: 2rem;
  transition: margin-bottom 0.3s ease;
  
  &.terminal-open {
    margin-bottom: 300px; // Matches default terminal height
  }
}
</style>

