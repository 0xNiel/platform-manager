<template>
  <div class="terminal-drawer" :class="{ open: terminalStore.isOpen }" :style="{ height: `${terminalStore.height}px` }">
    <!-- Resizer handle -->
    <div class="terminal-resizer" @mousedown="startResize" />

    <!-- Terminal header -->
    <div class="terminal-header">
      <div class="terminal-header-left">
        <span class="terminal-icon">💻</span>
        <span class="terminal-title">Web Terminal</span>
        <span v-if="terminalStore.currentSession" class="terminal-session">
          {{ terminalStore.currentSession.podName }}
        </span>
        <span v-if="terminalStore.isConnected" class="terminal-status connected">●</span>
        <span v-else-if="terminalStore.isConnecting" class="terminal-status connecting">●</span>
        <span v-else class="terminal-status disconnected">●</span>
      </div>
      <div class="terminal-header-right">
        <button v-if="!terminalStore.currentSession" class="terminal-btn" @click="connect" :disabled="terminalStore.isConnecting">
          <span v-if="terminalStore.isConnecting">Connecting...</span>
          <span v-else>Connect</span>
        </button>
        <button v-else class="terminal-btn danger" @click="disconnect">
          Disconnect
        </button>
        <button class="terminal-btn" @click="clearTerminal" :disabled="!terminal">
          Clear
        </button>
        <button class="terminal-btn" @click="terminalStore.closeTerminal">
          Close
        </button>
      </div>
    </div>

    <!-- Terminal content -->
    <div class="terminal-content">
      <div v-if="!terminalStore.canUseTerminal" class="terminal-message">
        <p>⚠️ You don't have permission to use the terminal.</p>
        <p class="terminal-message-hint">Contact your administrator to request access.</p>
      </div>
      <div v-else-if="!terminalStore.isTerminalEnabled" class="terminal-message">
        <p>⚠️ Terminal feature is disabled.</p>
        <p class="terminal-message-hint">Terminal must be enabled by the platform administrator.</p>
      </div>
      <div v-else-if="error" class="terminal-message error">
        <p>❌ {{ error }}</p>
        <button class="terminal-btn" @click="retry">Retry</button>
      </div>
      <div v-else-if="!terminalStore.currentSession" class="terminal-message">
        <p>Click "Connect" to start a terminal session.</p>
        <p class="terminal-message-hint">A shared toolbox pod with kubectl, helm, and other tools.</p>
      </div>
      <div v-else ref="terminalContainer" class="terminal-container" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebLinksAddon } from 'xterm-addon-web-links'
import { useTerminalStore } from '@/stores/terminal'
import 'xterm/css/xterm.css'

const terminalStore = useTerminalStore()

// Terminal instance
const terminal = ref<Terminal | null>(null)
const fitAddon = ref<FitAddon | null>(null)
const terminalContainer = ref<HTMLElement | null>(null)
const ws = ref<WebSocket | null>(null)
const resizeObserver = ref<ResizeObserver | null>(null)

// UI state
const error = ref<string | null>(null)
const isDisconnecting = ref(false)

// Resizing
const isResizing = ref(false)
const startY = ref(0)
const startHeight = ref(0)

// Initialize terminal
function initTerminal() {
  if (!terminalContainer.value) {
    console.log('[Terminal] Cannot initialize: no container')
    return
  }
  
  // If terminal already exists, don't reinitialize
  if (terminal.value) {
    console.log('[Terminal] Terminal already exists')
    return
  }

  console.log('[Terminal] Initializing new terminal instance')

  // Create terminal instance - don't set rows, let fitAddon handle sizing
  terminal.value = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e293b',
      foreground: '#e2e8f0',
      cursor: '#3b82f6',
      selection: 'rgba(59, 130, 246, 0.3)',
    },
    scrollback: 1000,
    convertEol: true,
  })

  // Add addons
  fitAddon.value = new FitAddon()
  terminal.value.loadAddon(fitAddon.value)
  terminal.value.loadAddon(new WebLinksAddon())

  // Open terminal in container
  terminal.value.open(terminalContainer.value)
  
  // Use requestAnimationFrame to ensure the container has been laid out
  requestAnimationFrame(() => {
    if (terminal.value && fitAddon.value) {
      try {
        fitAddon.value.fit()
      } catch (e) {
        // Ignore errors from fitting disposed terminal
      }
      console.log('[Terminal] Terminal initialized and fitted', {
        container: !!terminalContainer.value,
        terminal: !!terminal.value,
        rows: terminal.value?.rows,
        cols: terminal.value?.cols
      })
    }
  })
  
  // Set up ResizeObserver to handle container size changes
  if (terminalContainer.value) {
    resizeObserver.value = new ResizeObserver(() => {
      // Debounce the fit call and guard against disposed terminal
      requestAnimationFrame(() => {
        if (terminal.value && fitAddon.value) {
          try {
            fitAddon.value.fit()
          } catch (e) {
            // Ignore errors from fitting disposed terminal
          }
        }
      })
    })
    resizeObserver.value.observe(terminalContainer.value)
  }
  
  // Set up terminal input handler
  terminal.value.onData((data) => {
    if (ws.value?.readyState === WebSocket.OPEN) {
      // Send as binary data (ArrayBuffer)
      const encoder = new TextEncoder()
      ws.value.send(encoder.encode(data))
    } else {
      console.log('[Terminal] WebSocket not open, cannot send input. State:', ws.value?.readyState)
    }
  })
  
  // Focus the terminal so it can accept input
  terminal.value.focus()
  console.log('[Terminal] Terminal focused and ready for input')
}

// Connect to shared toolbox terminal
async function connect() {
  error.value = null

  try {
    // Close any existing WebSocket
    if (ws.value) {
      console.log('[Terminal] Closing existing WebSocket before new connection')
      ws.value.close()
      ws.value = null
    }
    
    // Dispose existing terminal to start fresh
    if (terminal.value) {
      console.log('[Terminal] Disposing existing terminal')
      // Clean up ResizeObserver first
      if (resizeObserver.value) {
        resizeObserver.value.disconnect()
        resizeObserver.value = null
      }
      // Clear addon reference before disposing terminal
      fitAddon.value = null
      try {
        terminal.value.dispose()
      } catch (e) {
        // Ignore addon disposal errors - not critical
      }
      terminal.value = null
    }
    
    // Create session - no tenant ID needed, uses shared toolbox
    const session = await terminalStore.createSession()
    
    // Wait for DOM to update so terminalContainer ref is available
    await nextTick()
    await nextTick() // Double nextTick to be safe
    
    // Initialize fresh terminal
    if (terminalContainer.value) {
      console.log('[Terminal] Initializing fresh terminal after session creation')
      initTerminal()
      terminal.value?.writeln('Connecting to ' + session.podName + '...')
    }

    // Log terminal state
    console.log('[Terminal] Terminal state before connection:', {
      exists: !!terminal.value,
      rows: terminal.value?.rows,
      cols: terminal.value?.cols,
      containerExists: !!terminalContainer.value
    })

    // Connect WebSocket
    await connectWebSocket(session.wsUrl)
  } catch (err) {
    let errorMessage = 'Failed to connect'
    if (err && typeof err === 'object' && 'response' in err) {
      const axiosError = err as { response?: { data?: { error?: string } } }
      errorMessage = axiosError.response?.data?.error || errorMessage
    } else if (err instanceof Error) {
      errorMessage = err.message
    }
    error.value = errorMessage
    terminal.value?.writeln(`\r\n\x1b[31mError: ${error.value}\x1b[0m`)
  }
}

// Connect WebSocket
async function connectWebSocket(wsUrl: string) {
  // Close any existing WebSocket
  if (ws.value) {
    ws.value.close()
    ws.value = null
  }

  // Get WebSocket URL - use the API base URL from environment or default
  const apiBase = process.env.VUE_APP_API_URL || 'http://localhost:9080/api/v1'
  const apiUrl = new URL(apiBase)
  const protocol = apiUrl.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = apiUrl.host // Use the API host, not window.location.host
  const wsFullUrl = `${protocol}//${host}${wsUrl}`

  console.log('[Terminal] Connecting WebSocket to:', wsFullUrl)

  ws.value = new WebSocket(wsFullUrl)
  
  // Set binary type to arraybuffer for consistent handling
  ws.value.binaryType = 'arraybuffer'

  ws.value.onopen = () => {
    console.log('[Terminal] WebSocket connected, terminal focused')
    terminalStore.setConnected(true)
    error.value = null
    
    // Focus terminal to accept input
    terminal.value?.focus()
    
    // Don't send initial newline - the shell will show its prompt automatically
    // Sending a newline causes a double prompt
  }

  ws.value.onmessage = (event) => {
    // Handle binary data from WebSocket
    if (event.data instanceof ArrayBuffer) {
      const decoder = new TextDecoder()
      const text = decoder.decode(event.data)
      if (terminal.value) {
        terminal.value.write(text)
      }
    } else if (event.data instanceof Blob) {
      // Convert Blob to text (shouldn't happen with binaryType = 'arraybuffer')
      event.data.text().then(text => {
        if (terminal.value) {
          terminal.value.write(text)
        }
      })
    } else {
      // String data
      if (terminal.value) {
        terminal.value.write(event.data)
      }
    }
  }

  ws.value.onerror = (err) => {
    console.error('[Terminal] WebSocket error:', err)
    error.value = 'WebSocket connection error'
    try {
      terminal.value?.writeln('\r\n\x1b[31mConnection error\x1b[0m')
    } catch (e) {
      // Ignore if terminal is disposed
    }
  }

  ws.value.onclose = (event) => {
    console.log('[Terminal] WebSocket closed:', event.code, event.reason)
    terminalStore.setConnected(false)
    // Only show "Connection closed" if not intentionally disconnecting and terminal exists
    if (!isDisconnecting.value && terminal.value) {
      try {
        terminal.value.writeln('\r\n\x1b[33mConnection closed\x1b[0m')
      } catch (e) {
        // Ignore errors if terminal is being disposed
      }
    }
  }
}

// Disconnect
async function disconnect() {
  console.log('[Terminal] Disconnecting...')
  
  // Set flag to prevent onclose from writing to disposed terminal
  isDisconnecting.value = true
  
  // Close WebSocket first
  if (ws.value) {
    ws.value.close()
    ws.value = null
  }
  
  // Delete session (cleanup pod)
  await terminalStore.deleteSession()
  
  // Clean up ResizeObserver first
  if (resizeObserver.value) {
    resizeObserver.value.disconnect()
    resizeObserver.value = null
  }
  
  // Clear addon reference before disposing terminal
  // (terminal.dispose() will handle addon cleanup internally)
  fitAddon.value = null
  
  // Dispose the terminal instance completely to reset state
  if (terminal.value) {
    try {
      terminal.value.dispose()
    } catch (e) {
      // Ignore addon disposal errors - not critical
    }
    terminal.value = null
  }
  
  // Reset state
  error.value = null
  isDisconnecting.value = false
  
  console.log('[Terminal] Disconnected and terminal disposed')
}

// Clear terminal
function clearTerminal() {
  terminal.value?.clear()
}

// Retry connection
function retry() {
  error.value = null
  connect()
}

// Resizing functions
function startResize(e: MouseEvent) {
  isResizing.value = true
  startY.value = e.clientY
  startHeight.value = terminalStore.height

  document.addEventListener('mousemove', doResize)
  document.addEventListener('mouseup', stopResize)
}

function doResize(e: MouseEvent) {
  if (!isResizing.value) return

  const delta = startY.value - e.clientY
  const newHeight = startHeight.value + delta
  terminalStore.setHeight(newHeight)
  
  // Refit terminal (with guard)
  nextTick(() => {
    if (terminal.value && fitAddon.value) {
      try {
        fitAddon.value.fit()
      } catch (e) {
        // Ignore errors from fitting disposed terminal
      }
    }
  })
}

function stopResize() {
  isResizing.value = false
  document.removeEventListener('mousemove', doResize)
  document.removeEventListener('mouseup', stopResize)
}

// Watch for terminal open/close
watch(() => terminalStore.isOpen, async (open) => {
  if (open) {
    await nextTick()
    
    // If there's an existing session but no terminal, we need to reinitialize
    if (terminalStore.currentSession) {
      // Reinitialize terminal if needed
      if (!terminal.value && terminalContainer.value) {
        console.log('[Terminal] Reinitializing terminal for existing session')
        initTerminal()
        terminal.value?.writeln('Reconnecting to session...')
      }
      
      // Reconnect WebSocket if not connected
      if (!ws.value || ws.value.readyState !== WebSocket.OPEN) {
        console.log('[Terminal] Reconnecting to existing session:', terminalStore.currentSession.sessionId)
        try {
          await connectWebSocket(terminalStore.currentSession.wsUrl)
        } catch (err) {
          console.error('[Terminal] Failed to reconnect:', err)
          // Clear the stale session and terminal
          await terminalStore.deleteSession()
          // Clean up in proper order
          if (resizeObserver.value) {
            resizeObserver.value.disconnect()
            resizeObserver.value = null
          }
          fitAddon.value = null
          if (terminal.value) {
            try {
              terminal.value.dispose()
            } catch (e) {
              // Ignore addon disposal errors - not critical
            }
            terminal.value = null
          }
          error.value = 'Failed to reconnect to previous session. Please start a new session.'
        }
      }
    } else {
      // No existing session - just initialize terminal if needed
      if (!terminal.value && terminalContainer.value) {
        initTerminal()
      }
    }
    
    // Fit terminal after open (with guard)
    if (terminal.value && fitAddon.value) {
      try {
        fitAddon.value.fit()
      } catch (e) {
        // Ignore errors from fitting disposed terminal
      }
    }
  } else {
    // When closing, disconnect WebSocket but keep the session for reconnection
    if (ws.value) {
      console.log('[Terminal] Closing WebSocket due to drawer close')
      ws.value.close()
      ws.value = null
    }
  }
})

// Lifecycle
onMounted(async () => {
  // Fetch terminal capabilities
  await terminalStore.fetchCapabilities()

  // Initialize terminal if drawer is open
  if (terminalStore.isOpen) {
    await nextTick()
    initTerminal()
  }
})

onBeforeUnmount(() => {
  // Set flag to prevent callbacks from writing to disposed terminal
  isDisconnecting.value = true
  
  if (ws.value) {
    ws.value.close()
  }
  if (resizeObserver.value) {
    resizeObserver.value.disconnect()
    resizeObserver.value = null
  }
  // Clear addon reference before disposing terminal
  fitAddon.value = null
  if (terminal.value) {
    try {
      terminal.value.dispose()
    } catch (e) {
      // Ignore addon disposal errors - not critical
    }
    terminal.value = null
  }
})
</script>

<style lang="scss" scoped>
.terminal-drawer {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background: #1e293b;
  border-top: 2px solid #334155;
  box-shadow: 0 -4px 6px -1px rgba(0, 0, 0, 0.1);
  transform: translateY(100%);
  transition: transform 0.3s ease;
  z-index: 1000;
  display: flex;
  flex-direction: column;

  &.open {
    transform: translateY(0);
  }
}

.terminal-resizer {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  cursor: ns-resize;
  background: transparent;
  
  &:hover {
    background: #3b82f6;
  }
}

.terminal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 1rem;
  background: #0f172a;
  border-bottom: 1px solid #334155;
}

.terminal-header-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.terminal-icon {
  font-size: 1.25rem;
}

.terminal-title {
  font-weight: 600;
  color: #e2e8f0;
}

.terminal-session {
  font-size: 0.875rem;
  color: #94a3b8;
  font-family: monospace;
}

.terminal-status {
  font-size: 0.75rem;
  
  &.connected {
    color: #10b981;
  }
  
  &.connecting {
    color: #f59e0b;
    animation: pulse 1.5s infinite;
  }
  
  &.disconnected {
    color: #6b7280;
  }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.terminal-header-right {
  display: flex;
  gap: 0.5rem;
}

.terminal-btn {
  padding: 0.5rem 1rem;
  background: #334155;
  color: #e2e8f0;
  border: none;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;

  &:hover:not(:disabled) {
    background: #475569;
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  &.danger {
    background: #dc2626;

    &:hover:not(:disabled) {
      background: #b91c1c;
    }
  }
}

.terminal-content {
  flex: 1;
  overflow: hidden;
  position: relative;
  padding: 0.5rem;
}

.terminal-container {
  width: 100%;
  height: 100%;
  // xterm.js requires the container to have no padding for proper fit calculations
}

.terminal-message {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  text-align: center;
  color: #94a3b8;

  p {
    margin: 0.5rem 0;
  }

  &.error {
    color: #f87171;
  }
}

.terminal-message-hint {
  font-size: 0.875rem;
  color: #64748b;
}
</style>

