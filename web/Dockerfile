# Multi-stage Dockerfile for Vue.js Micro-Frontend (Platform Manager MFE)
# Supports multi-architecture builds (amd64/arm64)
# This builds the app as a single-spa MFE to be loaded by a gateway/shell app

# Stage 1: Build the MFE
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package.json package-lock.json ./

# Install dependencies
# Using --prefer-offline and --no-audit for faster builds
RUN npm ci --prefer-offline --no-audit

# Copy source code
COPY . .

# Build the application as MFE (SystemJS module for single-spa)
# This creates the MFE bundle in /app/dist
ENV BUILD_MODE=mfe
RUN npm run build

# Stage 2: Production nginx server (simple static file server)
FROM nginx:1.25-alpine

# Install curl for healthcheck
RUN apk add --no-cache curl

# Copy custom nginx configuration (simple static file server)
COPY <<'EOF' /etc/nginx/conf.d/default.conf
server {
    listen 8080;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript 
               application/x-javascript application/xml+rss 
               application/javascript application/json application/manifest+json;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;

    # CORS headers for MFE (allow gateway to load this)
    add_header Access-Control-Allow-Origin "*" always;
    add_header Access-Control-Allow-Methods "GET, OPTIONS" always;
    add_header Access-Control-Allow-Headers "Content-Type" always;

    # Serve MFE bundle and assets
    location / {
        # No SPA routing needed - just serve files
        try_files $uri =404;
        
        # Allow range requests for better performance
        add_header Accept-Ranges bytes;
    }

    # Cache static assets aggressively
    location ~* \.(js|css|map)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
        # Keep CORS headers
        add_header Access-Control-Allow-Origin "*" always;
    }

    # Serve source maps (development only, remove in production)
    location ~* \.map$ {
        add_header Access-Control-Allow-Origin "*" always;
    }

    # Health check endpoint
    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }

    # Manifest endpoint (for gateway to discover MFE metadata)
    location /manifest.json {
        add_header Content-Type application/json;
        add_header Access-Control-Allow-Origin "*" always;
    }
}
EOF

# Copy built MFE assets from builder stage
COPY --from=builder /app/dist /usr/share/nginx/html

# Create a manifest file for the gateway to discover this MFE
RUN cat > /usr/share/nginx/html/manifest.json <<'EOF'
{
  "name": "@platform/manager-mfe",
  "version": "0.1.0",
  "main": "/js/app.js",
  "type": "module",
  "format": "system"
}
EOF

# Run nginx as non-root user
RUN chown -R nginx:nginx /usr/share/nginx/html && \
    chown -R nginx:nginx /var/cache/nginx && \
    chown -R nginx:nginx /var/log/nginx && \
    chown -R nginx:nginx /etc/nginx/conf.d && \
    touch /var/run/nginx.pid && \
    chown -R nginx:nginx /var/run/nginx.pid

USER nginx

# Expose port 8080 (non-privileged port)
EXPOSE 8080

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
