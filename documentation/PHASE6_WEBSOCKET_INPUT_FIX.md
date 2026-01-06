# Phase 6 WebSocket Input/Output Fix

## Issue

Terminal connected but was not receiving any output and could not accept input. Connection would timeout after ~30 seconds.

## Root Causes

### 1. Multiple `onData` Handlers
- The `terminal.value.onData()` handler was being attached **every time** the WebSocket connected (in `ws.value.onopen`)
- When closing and reopening the drawer, this resulted in multiple duplicate handlers
- Each keystroke would be sent multiple times to the WebSocket

### 2. Text vs Binary Encoding
- Frontend was sending keystrokes as plain text strings
- Backend expects binary data (as specified in `HandleWebSocket`)
- Mismatch caused input to be ignored

### 3. WebSocket Not Properly Cleaned Up
- When closing the terminal drawer, the WebSocket remained connected
- No cleanup handler for the `watch(() => terminalStore.isOpen)` watcher
- Stale connections accumulated

## Fixes Applied

### Fix 1: Single onData Handler in initTerminal
Moved the `terminal.value.onData()` setup to the `initTerminal()` function, so it's only attached **once** when the terminal is first created:

```typescript
function initTerminal() {
  // ... terminal creation ...
  
  // Set up terminal input handler ONCE during initialization
  terminal.value.onData((data) => {
    if (ws.value?.readyState === WebSocket.OPEN) {
      // Send as binary data (ArrayBuffer)
      const encoder = new TextEncoder()
      ws.value.send(encoder.encode(data))
    }
  })
}
```

### Fix 2: Binary Data Encoding for Input
Changed from sending plain strings to encoding as binary data using `TextEncoder`:

```typescript
// Before
ws.value.send(data)

// After
const encoder = new TextEncoder()
ws.value.send(encoder.encode(data))
```

### Fix 3: Binary Data Decoding for Output
Updated the `ws.value.onmessage` handler to properly handle binary data from the backend:

```typescript
ws.value.onmessage = (event) => {
  // Handle binary data from WebSocket
  if (event.data instanceof ArrayBuffer) {
    const decoder = new TextDecoder()
    const text = decoder.decode(event.data)
    terminal.value?.write(text)
  } else if (event.data instanceof Blob) {
    // Convert Blob to text
    event.data.text().then(text => {
      terminal.value?.write(text)
    })
  } else {
    // String data (fallback)
    terminal.value?.write(event.data)
  }
}
```

### Fix 4: WebSocket Cleanup on Drawer Close
Added proper cleanup in the `watch()` for `terminalStore.isOpen`:

```typescript
watch(() => terminalStore.isOpen, async (open) => {
  if (open) {
    // ... connection logic ...
  } else {
    // When closing, disconnect WebSocket but keep the session for reconnection
    if (ws.value) {
      console.log('Closing WebSocket due to drawer close')
      ws.value.close()
      ws.value = null
    }
  }
})
```

### Fix 5: Auto-Reconnect on Drawer Reopen
When the drawer is reopened, automatically reconnect to the existing session if available:

```typescript
if (open) {
  await nextTick()
  if (!terminal.value) {
    initTerminal()
  }
  fitAddon.value?.fit()
  
  // Auto-connect if there's an existing session
  if (terminalStore.currentSession && !ws.value) {
    console.log('Reconnecting to existing session:', terminalStore.currentSession.sessionId)
    await connectWebSocket(terminalStore.currentSession.wsUrl)
  } else if (!terminalStore.currentSession) {
    // Show tenant selector if no session exists
    showTenantSelector.value = true
  }
}
```

## Files Modified

- `web/src/components/TerminalDrawer.vue`:
  - Moved `onData` handler to `initTerminal()`
  - Added binary encoding/decoding for WebSocket messages
  - Added cleanup logic for drawer close
  - Added auto-reconnect logic for drawer open

## Testing

1. **Build Frontend**:
   ```bash
   cd web && npm run build
   ```

2. **Test Scenario 1 - Initial Connection**:
   - Open terminal drawer
   - Select a tenant
   - Verify prompt appears immediately
   - Type commands and verify they work

3. **Test Scenario 2 - Close and Reopen**:
   - Close the terminal drawer
   - Reopen it
   - Verify it reconnects automatically
   - Verify commands still work

4. **Test Scenario 3 - Multiple Open/Close Cycles**:
   - Repeat closing and reopening multiple times
   - Verify no duplicate keystrokes
   - Verify clean connections each time

## Expected Behavior After Fix

- ✅ Shell prompt appears immediately after connection
- ✅ Keyboard input is sent correctly to the backend
- ✅ Shell output is displayed in real-time
- ✅ Drawer can be closed and reopened without issues
- ✅ No duplicate handlers or stale connections
- ✅ WebSocket properly cleaned up on close

## Next Steps

After verifying the terminal works:
1. Add Prometheus metrics for terminal sessions
2. Enhance auditing with command recording
3. Consider adding session persistence/recovery

