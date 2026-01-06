# Security Mitigations Implementation Guide

This document provides detailed implementation steps for remediating the vulnerabilities identified in the Security Review.

---

## Priority 1: Immediate Fixes

### 1. Upgrade Go Runtime

**Issue:** Go 1.25.4 contains crypto/x509 vulnerabilities (GO-2025-4175, GO-2025-4155)

**Solution:**

```bash
# Update go.mod
go mod edit -go=1.25.5

# Update go toolchain if needed
go install golang.org/dl/go1.25.5@latest
go1.25.5 download

# Verify
go version

# Run tests
go test ./...

# Verify vulnerabilities are resolved
govulncheck ./...
```

**Files to Update:**
- `go.mod` - Update Go version directive
- `Dockerfile` - Update builder image to `golang:1.25.5` or later
- CI/CD configs - Update Go version

---

### 2. Fix WebSocket Origin Validation

**Issue:** WebSocket upgrader accepts connections from any origin

**Current Code (INSECURE):**
```go
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true  // ⚠️ INSECURE
    },
}
```

**Secure Implementation:**

Create a new configuration structure for allowed origins:

**File:** `internal/terminal/config.go` (add to existing file or types.go)

```go
// TerminalSecurityConfig holds security settings for terminal
type TerminalSecurityConfig struct {
    AllowedOrigins []string
    DevMode        bool
}

// DefaultSecurityConfig returns secure defaults
func DefaultSecurityConfig() TerminalSecurityConfig {
    return TerminalSecurityConfig{
        AllowedOrigins: []string{
            // No origins allowed by default - must be configured
        },
        DevMode: false,
    }
}

// IsOriginAllowed checks if an origin is in the allowed list
func (c *TerminalSecurityConfig) IsOriginAllowed(origin string) bool {
    if c.DevMode {
        return true // Allow all in dev mode
    }
    
    for _, allowed := range c.AllowedOrigins {
        if origin == allowed {
            return true
        }
    }
    return false
}
```

**File:** `internal/api/handlers/terminal.go`

Update the TerminalHandler struct:

```go
type TerminalHandler struct {
    manager        *terminal.Manager
    clientset      *kubernetes.Clientset
    restConfig     *rest.Config
    logger         logr.Logger
    securityConfig terminal.TerminalSecurityConfig  // ADD THIS
}
```

Update NewTerminalHandler:

```go
func NewTerminalHandler(
    k8sClient client.Client, 
    terminalMgr *terminal.Manager, 
    logger logr.Logger,
    securityConfig terminal.TerminalSecurityConfig,  // ADD THIS
) (*TerminalHandler, error) {
    // ... existing code ...
    
    return &TerminalHandler{
        manager:        terminalMgr,
        clientset:      clientset,
        restConfig:     config,
        logger:         logger.WithName("terminal-handler"),
        securityConfig: securityConfig,  // ADD THIS
    }, nil
}
```

Create upgrader dynamically:

```go
// createUpgrader creates a WebSocket upgrader with proper origin validation
func (h *TerminalHandler) createUpgrader() websocket.Upgrader {
    return websocket.Upgrader{
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
        CheckOrigin: func(r *http.Request) bool {
            origin := r.Header.Get("Origin")
            allowed := h.securityConfig.IsOriginAllowed(origin)
            
            if !allowed {
                h.logger.Info("rejected websocket connection from unauthorized origin",
                    "origin", origin,
                    "remoteAddr", r.RemoteAddr)
            }
            
            return allowed
        },
    }
}
```

Update HandleWebSocket:

```go
func (h *TerminalHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "sessionId")
    
    session, err := h.manager.GetSession(sessionID)
    if err != nil {
        h.logger.Error(err, "session not found for websocket", "sessionId", sessionID)
        http.Error(w, "Session not found", http.StatusNotFound)
        return
    }
    
    // Create upgrader with security config
    upgrader := h.createUpgrader()
    
    // Upgrade to WebSocket
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        h.logger.Error(err, "failed to upgrade to websocket", "sessionId", sessionID)
        return
    }
    
    // ... rest of existing code ...
}
```

**Environment Configuration:**

Add to deployment config or environment:

```yaml
env:
  - name: TERMINAL_ALLOWED_ORIGINS
    value: "https://platform.yourcompany.com,https://platform-staging.yourcompany.com"
  - name: DEV_MODE
    value: "false"  # true only in development
```

**Main.go Update:**

```go
// Parse allowed origins from environment
allowedOriginsStr := os.Getenv("TERMINAL_ALLOWED_ORIGINS")
allowedOrigins := []string{}
if allowedOriginsStr != "" {
    allowedOrigins = strings.Split(allowedOriginsStr, ",")
}

devMode := os.Getenv("DEV_MODE") == "true"

securityConfig := terminal.TerminalSecurityConfig{
    AllowedOrigins: allowedOrigins,
    DevMode:        devMode,
}

// Pass to terminal handler
terminalHandler, err := handlers.NewTerminalHandler(
    mgr.GetClient(),
    terminalManager,
    setupLog,
    securityConfig,
)
```

---

### 3. Fix Authentication Dev Mode Bypass

**Issue:** X-Dev-Role header allows authentication bypass

**File:** `internal/api/middleware/auth.go`

**Secure Implementation:**

```go
// ExtractUser middleware extracts user info from OAuth2Proxy headers
func ExtractUser(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := UserInfo{
            Username: r.Header.Get("X-Auth-Request-User"),
            Email:    r.Header.Get("X-Auth-Request-Email"),
            Groups:   parseGroups(r.Header.Get("X-Auth-Request-Groups")),
        }

        // If no user from headers, handle based on environment
        if user.Username == "" {
            // Check if dev mode is explicitly enabled
            devMode := os.Getenv("DEV_MODE") == "true"
            
            if devMode {
                // Development mode: allow dev header
                if devRole := r.Header.Get("X-Dev-Role"); devRole != "" {
                    // Validate role is a real role
                    if isValidRole(Role(devRole)) {
                        user.Username = "dev-user"
                        user.Email = "dev@localhost"
                        user.Role = Role(devRole)
                        
                        // Log dev mode usage prominently
                        log.Printf("⚠️  DEV MODE: Using X-Dev-Role header with role: %s", devRole)
                    } else {
                        user.Username = "anonymous"
                        user.Role = RoleReadOnly
                    }
                } else {
                    user.Username = "dev-anonymous"
                    user.Role = RoleReadOnly
                }
            } else {
                // Production: no authentication means read-only anonymous
                user.Username = "anonymous"
                user.Role = RoleReadOnly
                
                // Log authentication failures for monitoring
                log.Printf("⚠️  Unauthenticated request from %s to %s", 
                    r.RemoteAddr, r.URL.Path)
            }
        } else {
            // Map groups to role
            user.Role = mapGroupsToRole(user.Groups)
        }

        ctx := context.WithValue(r.Context(), userInfoKey, user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// isValidRole checks if a role string is valid
func isValidRole(role Role) bool {
    switch role {
    case RoleAdmin, RoleInfra, RoleML, RoleReadOnly:
        return true
    default:
        return false
    }
}
```

**Nginx/Reverse Proxy Configuration:**

Add to production nginx config to block dev headers:

```nginx
# Block development headers in production
proxy_set_header X-Dev-Role "";

# Only allow OAuth2 Proxy headers
proxy_pass_header X-Auth-Request-User;
proxy_pass_header X-Auth-Request-Email;
proxy_pass_header X-Auth-Request-Groups;
```

---

### 4. Update Dockerfile.toolbox

**Issue:** Outdated dependencies with potential vulnerabilities

**File:** `Dockerfile.toolbox`

**Updated Version:**

```dockerfile
# Dockerfile.toolbox
# Toolbox container for web terminal sessions
# Includes kubectl, aws CLI, and common debugging tools
# Multi-architecture support (amd64/arm64)

FROM alpine:3.21

# Install base tools
RUN apk add --no-cache \
    bash \
    curl \
    wget \
    git \
    vim \
    nano \
    jq \
    yq \
    bind-tools \
    net-tools \
    iputils \
    tcpdump \
    ca-certificates \
    openssl \
    python3 \
    py3-pip

# Detect architecture and set ARCH variable
ARG TARGETARCH
RUN echo "Building for architecture: ${TARGETARCH}"

# Install kubectl (architecture-aware) - UPDATED VERSION
ARG KUBECTL_VERSION=v1.33.0
RUN ARCH=${TARGETARCH:-amd64} && \
    curl -LO "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${ARCH}/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/

# Install AWS CLI - PINNED VERSION
ARG AWS_CLI_VERSION=1.32.0
RUN pip3 install --no-cache-dir awscli==${AWS_CLI_VERSION} --break-system-packages

# Install Helm (architecture-aware) - UPDATED VERSION
ARG HELM_VERSION=v3.16.3
RUN ARCH=${TARGETARCH:-amd64} && \
    curl -LO "https://get.helm.sh/helm-${HELM_VERSION}-linux-${ARCH}.tar.gz" && \
    tar -zxvf helm-${HELM_VERSION}-linux-${ARCH}.tar.gz && \
    mv linux-${ARCH}/helm /usr/local/bin/ && \
    rm -rf linux-${ARCH} helm-${HELM_VERSION}-linux-${ARCH}.tar.gz

# Install ArgoCD CLI (architecture-aware) - UPDATED VERSION
ARG ARGOCD_VERSION=v2.13.1
RUN ARCH=${TARGETARCH:-amd64} && \
    curl -sSL -o /usr/local/bin/argocd "https://github.com/argoproj/argo-cd/releases/download/${ARGOCD_VERSION}/argocd-linux-${ARCH}" && \
    chmod +x /usr/local/bin/argocd

# Create non-root user
RUN addgroup -g 1000 toolbox && \
    adduser -D -u 1000 -G toolbox toolbox && \
    mkdir -p /home/toolbox && \
    chown -R toolbox:toolbox /home/toolbox

# Set up shell environment
COPY --chown=toolbox:toolbox <<'EOF' /home/toolbox/.bashrc
# Bash configuration for toolbox
export PS1='[\u@$TENANT_ID \W]\$ '
export EDITOR=vim

# Aliases
alias k=kubectl
alias ll='ls -alh'
alias kgp='kubectl get pods'
alias kgs='kubectl get svc'
alias kgd='kubectl get deployments'
alias logs='kubectl logs'
alias describe='kubectl describe'

# Functions
kdebug() {
  kubectl run debug-$RANDOM --rm -it --image=alpine:3.21 -- /bin/sh
}

# Show current context
echo "Terminal Session: $TENANT_ID"
echo "User: $USERNAME"
echo ""
echo "Available tools:"
echo "  - kubectl (k) ${KUBECTL_VERSION}"
echo "  - aws cli ${AWS_CLI_VERSION}"
echo "  - helm ${HELM_VERSION}"
echo "  - argocd ${ARGOCD_VERSION}"
echo "  - git, jq, yq, curl, wget"
echo ""
echo "Type 'exit' to close this session"
echo ""
EOF

# Switch to non-root user
USER toolbox
WORKDIR /home/toolbox

# Default command (will be overridden by pod spec)
CMD ["/bin/bash"]
```

---

### 5. Fix NPM Vulnerabilities

**Commands to Run:**

```bash
cd /Users/odnielgonzalez/Documents/2-WorkStuff--ai-platform-in-go/platform-manager/web

# 1. Update package-lock.json with safe fixes
npm audit fix

# 2. Review changes
git diff package-lock.json

# 3. Test the application
npm run build
npm run serve

# 4. For breaking changes (in separate commit):
# Review each update carefully
npm audit fix --force

# 5. Alternative: Update specific packages manually
npm update cross-spawn@latest
npm update postcss@latest
npm update webpack-dev-server@latest

# 6. Verify fixes
npm audit

# 7. Run tests
npm run test:unit

# 8. Check for runtime issues
npm run serve
# Test all functionality in browser
```

**Expected Results:**
- Cross-spawn updated to >= 6.0.6
- PostCSS updated to >= 8.4.31
- webpack-dev-server updated to > 5.2.0
- vue-template-compiler issues resolved

---

### 6. Implement CORS Restrictions

**File:** `internal/api/middleware/cors.go`

**Secure Implementation:**

```go
package middleware

import (
    "net/http"
    "os"
    "strings"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
    AllowedOrigins []string
    DevMode        bool
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
    allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
    allowedOrigins := []string{}
    if allowedOriginsStr != "" {
        for _, origin := range strings.Split(allowedOriginsStr, ",") {
            allowedOrigins = append(allowedOrigins, strings.TrimSpace(origin))
        }
    }
    
    return CORSConfig{
        AllowedOrigins: allowedOrigins,
        DevMode:        os.Getenv("DEV_MODE") == "true",
    }
}

// CORS adds CORS headers to responses
func CORS(config CORSConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            
            // In dev mode, allow all origins
            if config.DevMode {
                w.Header().Set("Access-Control-Allow-Origin", "*")
            } else if isAllowedOrigin(origin, config.AllowedOrigins) {
                // In production, only allow configured origins
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Vary", "Origin")
            }
            
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
            w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Auth-Request-User, X-Auth-Request-Email, X-Auth-Request-Groups")
            w.Header().Set("Access-Control-Allow-Credentials", "true")
            w.Header().Set("Access-Control-Max-Age", "300")

            // Handle preflight requests
            if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusNoContent)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// isAllowedOrigin checks if an origin is in the allowed list
func isAllowedOrigin(origin string, allowedOrigins []string) bool {
    for _, allowed := range allowedOrigins {
        if origin == allowed {
            return true
        }
    }
    return false
}
```

**Update server.go:**

```go
// Get CORS config
corsConfig := middleware.DefaultCORSConfig()

// Apply CORS middleware
r.Use(middleware.CORS(corsConfig))
```

---

## Priority 2: High Priority Fixes

### 7. Add Rate Limiting to WebSocket

**New File:** `internal/terminal/ratelimit.go`

```go
package terminal

import (
    "sync"
    "time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
    tokens         int
    maxTokens      int
    refillRate     int           // tokens per second
    lastRefill     time.Time
    mu             sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate int) *RateLimiter {
    return &RateLimiter{
        tokens:     maxTokens,
        maxTokens:  maxTokens,
        refillRate: refillRate,
        lastRefill: time.Now(),
    }
}

// Allow checks if an action is allowed
func (rl *RateLimiter) Allow() bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    // Refill tokens based on time passed
    now := time.Now()
    elapsed := now.Sub(rl.lastRefill)
    tokensToAdd := int(elapsed.Seconds()) * rl.refillRate
    
    if tokensToAdd > 0 {
        rl.tokens += tokensToAdd
        if rl.tokens > rl.maxTokens {
            rl.tokens = rl.maxTokens
        }
        rl.lastRefill = now
    }
    
    // Check if we have tokens available
    if rl.tokens > 0 {
        rl.tokens--
        return true
    }
    
    return false
}

// Wait waits until an action is allowed
func (rl *RateLimiter) Wait() {
    for !rl.Allow() {
        time.Sleep(100 * time.Millisecond)
    }
}
```

**Update Session struct in types.go:**

```go
type Session struct {
    // ... existing fields ...
    RateLimiter *RateLimiter  // ADD THIS
}
```

**Update terminal handler:**

```go
// In CreateSession
session := &Session{
    // ... existing fields ...
    RateLimiter: NewRateLimiter(100, 10), // 100 tokens, refill 10/sec
}

// In HandleWebSocket goroutine for reading:
go func() {
    defer stdinWriter.Close()
    for {
        select {
        case <-done:
            return
        case <-ctx.Done():
            return
        default:
        }

        _, message, err := conn.ReadMessage()
        if err != nil {
            // ... existing error handling ...
            return
        }
        
        // ADD RATE LIMITING
        if !session.RateLimiter.Allow() {
            h.logger.Info("rate limit exceeded", "sessionId", sessionID)
            // Optionally send a message to user
            conn.WriteMessage(websocket.TextMessage, 
                []byte("\r\n⚠️  Rate limit exceeded. Please slow down.\r\n"))
            time.Sleep(100 * time.Millisecond)
            continue
        }
        
        h.manager.UpdateActivity(sessionID)
        _, err = stdinWriter.Write(message)
        if err != nil {
            h.logger.Error(err, "stdin write error", "sessionId", sessionID)
            return
        }
    }
}()
```

---

### 8. Enhanced Session Security

**Update Session struct:**

```go
type Session struct {
    ID              string
    Username        string
    TenantID        string
    Namespace       string
    PodName         string
    CreatedAt       time.Time
    LastActivity    time.Time
    IdleTimeout     time.Duration
    Done            chan struct{}
    CommandRecorder *CommandRecorder
    
    // WebSocket fields
    Conn          *websocket.Conn
    ExecContext   context.Context
    ExecCancel    context.CancelFunc
    ExecMu        sync.Mutex
    
    // Security fields
    RateLimiter   *RateLimiter
    CreatedByIP   string          // ADD: Track originating IP
    CreatedByUA   string          // ADD: Track User-Agent
    SessionToken  string          // ADD: Separate token for WebSocket auth
}
```

**Update CreateSession:**

```go
func (h *TerminalHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
    user := middleware.UserFromContext(r.Context())
    
    // Extract IP and User-Agent
    clientIP := r.RemoteAddr
    if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
        clientIP = strings.Split(forwarded, ",")[0]
    }
    userAgent := r.Header.Get("User-Agent")
    
    // Generate session token
    sessionToken := generateSecureToken()
    
    opts := terminal.SessionOptions{
        Username:     user.Username,
        CreatedByIP:  clientIP,
        CreatedByUA:  userAgent,
        SessionToken: sessionToken,
    }
    
    session, err := h.manager.CreateSession(r.Context(), opts)
    // ... rest of existing code ...
    
    response := map[string]interface{}{
        "sessionId":    session.ID,
        "sessionToken": sessionToken,  // Send to client
        "podName":      session.PodName,
        "namespace":    session.Namespace,
        "createdAt":    session.CreatedAt,
        "wsUrl":        fmt.Sprintf("/api/v1/terminal/sessions/%s/ws?token=%s", 
                                     session.ID, sessionToken),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func generateSecureToken() string {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)
}
```

**Update HandleWebSocket:**

```go
func (h *TerminalHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "sessionId")
    
    // Get session token from query parameter
    token := r.URL.Query().Get("token")
    
    session, err := h.manager.GetSession(sessionID)
    if err != nil {
        h.logger.Error(err, "session not found", "sessionId", sessionID)
        http.Error(w, "Session not found", http.StatusNotFound)
        return
    }
    
    // Validate session token
    if token != session.SessionToken {
        h.logger.Error(nil, "invalid session token", "sessionId", sessionID)
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Validate IP and User-Agent (optional, for extra security)
    clientIP := r.RemoteAddr
    if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
        clientIP = strings.Split(forwarded, ",")[0]
    }
    
    if session.CreatedByIP != clientIP {
        h.logger.Info("websocket connection from different IP",
            "sessionId", sessionID,
            "originalIP", session.CreatedByIP,
            "newIP", clientIP)
        // Optionally reject or allow with warning
    }
    
    // ... rest of existing code ...
}
```

---

## Testing Checklist

After implementing these mitigations:

```bash
# 1. Test Go vulnerability fix
govulncheck ./...

# 2. Test npm vulnerability fix
cd web && npm audit

# 3. Build and test locally
make docker-build
make deploy

# 4. Test WebSocket origin validation
# Use browser dev tools to test different origins

# 5. Test dev mode is properly disabled
curl -H "X-Dev-Role: admin" http://localhost:8080/api/v1/health

# 6. Test CORS with unauthorized origin
curl -H "Origin: https://malicious.com" http://localhost:8080/api/v1/health

# 7. Test rate limiting
# Flood WebSocket with messages

# 8. Test session token validation
# Try connecting to WebSocket without token

# 9. Run integration tests
make test

# 10. Verify all security headers
curl -I http://localhost:8080/api/v1/health
```

---

## Deployment Checklist

Before deploying to production:

- [ ] Update Go to 1.25.5+
- [ ] Update npm packages
- [ ] Update Dockerfile.toolbox with new versions
- [ ] Configure ALLOWED_ORIGINS environment variable
- [ ] Set DEV_MODE=false in production
- [ ] Configure TERMINAL_ALLOWED_ORIGINS
- [ ] Update nginx config to block dev headers
- [ ] Run full security test suite
- [ ] Update documentation
- [ ] Notify team of changes
- [ ] Monitor logs for security events

---

## Rollback Plan

If issues arise after deployment:

```bash
# Quick rollback
git revert <commit-hash>
kubectl rollout undo deployment/platform-manager

# Or deploy previous version
kubectl set image deployment/platform-manager \
  manager=platform-manager:previous-version
```

---

## Monitoring and Alerting

Add monitoring for:

1. Failed WebSocket origin validations
2. Rate limit exceedances
3. Invalid session token attempts
4. Dev mode usage (should be zero in production)
5. CORS violations
6. Authentication failures

Example Prometheus alerts:

```yaml
- alert: DevModeInProduction
  expr: dev_mode_enabled == 1
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "Development mode enabled in production"

- alert: WebSocketOriginRejections
  expr: rate(websocket_origin_rejections_total[5m]) > 10
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High rate of WebSocket origin rejections"

- alert: SessionTokenFailures
  expr: rate(session_token_validation_failures_total[5m]) > 5
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High rate of session token validation failures"
```

---

## Documentation Updates

Update these files:

1. **README.md** - Add security section
2. **DEPLOYMENT.md** - Add security configuration
3. **DEVELOPMENT.md** - Document dev mode usage
4. **API.md** - Update WebSocket endpoint documentation

---

## Training and Communication

1. Brief development team on security changes
2. Update onboarding documentation
3. Schedule security training session
4. Create incident response runbook
5. Document escalation procedures

---

**Last Updated:** January 6, 2026  
**Next Review:** As needed based on deployment

