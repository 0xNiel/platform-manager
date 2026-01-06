# Phase 6 Session Cleanup Fix

## Issue

After successfully using the terminal once, subsequent connection attempts (to the same or different tenants) would fail silently:
- "New Session" button would disappear after disconnect
- Terminal would show "Waiting for connection..." but never connect
- No error messages were displayed

## Root Causes

### 1. Missing Error State Reset
- After disconnecting, the `error` state wasn't cleared
- If an error occurred during the previous session, it would persist
- The error wouldn't be displayed because the terminal was in a disconnected state

### 2. Stale Session Reconnection Attempts
- When reopening the drawer, it tried to auto-reconnect to the previous session
- The previous session had been deleted on the backend
- The reconnection would fail silently, leaving the UI stuck

### 3. No Error Feedback on Connection Failure
- When `connectToTenant()` was called but the session creation failed, the error wasn't always displayed
- The error state wasn't reset before attempting a new connection

## Fixes Applied

### Fix 1: Clear Error State on Disconnect

```typescript
async function disconnect() {
  if (ws.value) {
    ws.value.close()
    ws.value = null
  }
  await terminalStore.deleteSession()
  terminal.value?.clear()
  terminal.value?.writeln('Platform Manager Web Terminal')
  terminal.value?.writeln('Disconnected. Click "New Session" to start a new session.')
  terminal.value?.writeln('')
  
  // Reset error state
  error.value = null
}
```

### Fix 2: Handle Stale Session Reconnection Gracefully

```typescript
watch(() => terminalStore.isOpen, async (open) => {
  if (open) {
    // ...
    
    // Auto-connect if there's an existing session
    if (terminalStore.currentSession && !ws.value) {
      console.log('Reconnecting to existing session:', terminalStore.currentSession.sessionId)
      try {
        await connectWebSocket(terminalStore.currentSession.wsUrl)
      } catch (err) {
        console.error('Failed to reconnect:', err)
        // Clear the stale session
        await terminalStore.deleteSession()
        error.value = 'Failed to reconnect to previous session. Please start a new session.'
      }
    }
  }
  // ...
})
```

### Fix 3: Reset Error Before New Connection

```typescript
async function connectToTenant(tenantId: string) {
  showTenantSelector.value = false
  error.value = null // Reset error state

  try {
    // Create session
    const session = await terminalStore.createSession(tenantId)
    // Connect WebSocket
    await connectWebSocket(session.wsUrl)
  } catch (err) {
    // ... error handling ...
  }
}
```

## Files Modified

- `web/src/components/TerminalDrawer.vue`:
  - Added `error.value = null` to `disconnect()`
  - Added try-catch to auto-reconnect logic in `watch()`
  - Added `error.value = null` at start of `connectToTenant()`

## Testing

1. **Build Frontend**:
   ```bash
   cd web && npm run build
   ```

2. **Refresh Browser** (Cmd+R or Ctrl+R)

3. **Test Scenario 1 - Initial Connection**:
   - Open terminal drawer
   - Click "New Session"
   - Select tenant-alpha
   - Verify connection works

4. **Test Scenario 2 - Disconnect and Reconnect to Same Tenant**:
   - Click "Disconnect"
   - Click "New Session"
   - Select tenant-alpha again
   - Verify connection works

5. **Test Scenario 3 - Connect to Different Tenants**:
   - Connect to tenant-alpha
   - Disconnect
   - Click "New Session"
   - Select tenant-beta
   - Verify connection works

6. **Test Scenario 4 - Close and Reopen Drawer**:
   - Connect to a tenant
   - Close the drawer (without disconnecting)
   - Reopen the drawer
   - Verify it reconnects OR shows "New Session" button

## Expected Behavior After Fix

- ✅ "New Session" button always visible when not connected
- ✅ Can connect to any tenant after disconnecting
- ✅ Can switch between tenants
- ✅ Error messages displayed when connection fails
- ✅ Clean state after each disconnect
- ✅ No stale sessions blocking new connections

## Next Steps

After verifying all scenarios work:
1. Add Prometheus metrics for terminal sessions
2. Enhance auditing with command recording
3. Consider adding session timeout warnings

