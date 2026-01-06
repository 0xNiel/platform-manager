# Phase 6 Terminal TTY Size Fix

## Issue

Bash prompt not displaying on initial connection. The prompt would only show after:
1. Closing and reopening the terminal drawer
2. Never show again on subsequent connections

## Root Cause

The backend was not providing terminal size information to the Kubernetes exec stream. When you execute a command in a TTY (terminal) session, the shell needs to know the terminal dimensions (width and height in characters) to properly format output and display prompts.

### Technical Details

The `remotecommand.StreamOptions` struct has an optional `TerminalSizeQueue` field:

```go
type StreamOptions struct {
    Stdin             io.Reader
    Stdout            io.Writer
    Stderr            io.Writer
    Tty               bool
    TerminalSizeQueue TerminalSizeQueue  // <-- This was missing
}
```

When `Tty: true` is set but `TerminalSizeQueue` is `nil`, the terminal doesn't know its dimensions and bash may not display the prompt properly or format output correctly.

### Why It Worked After Reopening

When you closed and reopened the drawer:
1. The frontend sent a newline character (`\n`)
2. This triggered bash to display output
3. Bash would show the prompt after receiving input
4. But this was a workaround, not a proper fix

## The Fix

Added `TerminalSizeQueue` to the `StreamWithContext` call with default terminal dimensions (80x24 characters):

```go
// Create terminal size queue with default size
terminalSizeQueue := &terminalSizeQueue{
    conn: conn,
    defaultSize: &remotecommand.TerminalSize{
        Width:  80,
        Height: 24,
    },
}

// Execute with pipes
err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
    Stdin:             stdinReader,
    Stdout:            stdoutWriter,
    Stderr:            stdoutWriter,
    Tty:               true,
    TerminalSizeQueue: terminalSizeQueue, // <-- Added this
})
```

The `terminalSizeQueue` type implements the `remotecommand.TerminalSizeQueue` interface:

```go
// terminalSizeQueue implements remotecommand.TerminalSizeQueue
type terminalSizeQueue struct {
    conn        *websocket.Conn
    defaultSize *remotecommand.TerminalSize
}

func (t *terminalSizeQueue) Next() *remotecommand.TerminalSize {
    // Return default size
    // TODO: Handle resize messages from client in future enhancement
    return t.defaultSize
}
```

## Files Modified

- `internal/api/handlers/terminal.go`:
  - Added `TerminalSizeQueue` initialization in `HandleWebSocket()`
  - Used existing `terminalSizeQueue` type with default size (80x24)

## Testing

1. **Build Backend**:
   ```bash
   cd /Users/odnielgonzalez/Documents/2-WorkStuff--ai-platform-in-go/platform-manager
   make build
   ```

2. **Restart Manager**:
   ```bash
   pkill -f "bin/manager"
   ./bin/manager \
     --metrics-bind-address=:8081 \
     --health-probe-bind-address=:8082 \
     --leader-elect=false \
     --api-bind-address=:9080 \
     --enable-terminal=true \
     --terminal-namespace=toolbox-sessions \
     --terminal-image=platform-manager-toolbox:dev \
     --terminal-idle-timeout=10m
   ```

3. **Test in Browser**:
   - Refresh the page (Cmd+R or Ctrl+R)
   - Click "Terminal" in navbar
   - Click "New Session"
   - Select any tenant
   - **Prompt should appear immediately** (e.g., `toolbox@toolbox-alpha-xxxxx:~$`)
   - Type commands and verify they work
   - Disconnect and reconnect to verify it works consistently

## Expected Behavior After Fix

- ✅ Bash prompt appears immediately on first connection
- ✅ No need to close/reopen the drawer
- ✅ Works consistently across multiple connections
- ✅ Works for all tenants
- ✅ Terminal output is properly formatted

## Additional Context

### Standard Terminal Sizes

The default size of 80x24 is a standard terminal dimension:
- **Width**: 80 characters (columns)
- **Height**: 24 lines (rows)

This is based on the historical VT100 terminal standard and is still widely used as a default.

### Future Enhancements

The `terminalSizeQueue` implementation currently returns a static size. In the future, we can enhance this to:

1. **Dynamic Resizing**: Handle terminal resize events from the frontend
2. **Custom Sizes**: Allow users to configure their preferred terminal size
3. **Responsive**: Calculate size based on browser window dimensions

This would involve:
- Sending resize events via WebSocket from the frontend
- Implementing a channel-based size queue that updates dynamically
- Using xterm.js's `onResize` event to detect frontend terminal size changes

Example future implementation:

```go
type terminalSizeQueue struct {
    resizeChan chan remotecommand.TerminalSize
}

func (t *terminalSizeQueue) Next() *remotecommand.TerminalSize {
    size := <-t.resizeChan
    return &size
}
```

And in the frontend:

```typescript
terminal.onResize((dimensions) => {
  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({
      type: 'resize',
      width: dimensions.cols,
      height: dimensions.rows,
    }))
  }
})
```

## Related Issues and Fixes

This fix is part of a series of WebSocket terminal fixes:

1. **Auth Fix**: Moved WebSocket endpoint outside auth middleware
   - Doc: `PHASE6_WEBSOCKET_FIX.md`

2. **I/O Fix**: Fixed binary encoding and concurrent streaming
   - Doc: `PHASE6_WEBSOCKET_INPUT_FIX.md`

3. **Session Cleanup Fix**: Fixed stale session reconnection
   - Doc: `PHASE6_SESSION_CLEANUP_FIX.md`

4. **TTY Size Fix**: Added terminal size queue (this document)
   - Doc: `PHASE6_TTY_SIZE_FIX.md`

## Summary

The terminal now properly initializes with terminal dimensions, allowing bash to correctly display prompts and format output from the very first connection. No workarounds or multiple attempts needed!

