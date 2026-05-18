# Step 14 — Nginx: Reverse Proxy & TLS Termination

## Goal

Use nginx as the reverse proxy in front of GoTradeX services for production deployment. Handles TLS termination, load balancing, and WebSocket upstream routing.

> **Prerequisite**: [Step 11 — Deployment](./11-deploy.md)

---

## Responsibilities

| Role | Details |
|------|---------|
| TLS Termination | HTTPS → HTTP on port 443, redirect HTTP → HTTPS on port 80 |
| Reverse Proxy | Forward REST/gRPC to API Gateway, WebSocket to Market Service |
| Load Balancing | Round-robin across multiple API Gateway instances |
| Rate Limiting | Additional layer at nginx level ( complements app-level rate limiter) |
| Static Assets | Serve Swagger UI (`/swagger/*`) |

---

## Config: `nginx.conf`

```nginx
worker_processes auto;
error_log /var/log/nginx/error.log warn;

events {
    worker_connections 4096;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    # Logging
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';
    access_log /var/log/nginx/access.log main;

    # Gzip compression
    gzip on;
    gzip_types application/json text/plain application/javascript;

    # Rate limiting zones
    limit_req_zone $binary_remote_addr zone=api:10m rate=100r/s;
    limit_req_zone $binary_remote_addr zone=auth:10m rate=5r/s;

    # Buffers
    proxy_buffer_size 128k;
    proxy_buffers 4 256k;
    proxy_busy_buffers_size 256k;

    # Timeouts
    proxy_connect_timeout 30s;
    proxy_send_timeout 30s;
    proxy_read_timeout 30s;

    # API Gateway upstream (load balance across instances)
    upstream api_gateway {
        server 127.0.0.1:8080;
        # Add more instances for horizontal scaling:
        # server 127.0.0.1:8081;
        # server 127.0.0.1:8082;
        keepalive 64;
    }

    # Market Service WebSocket upstream
    upstream market_ws {
        server 127.0.0.1:8081;
        keepalive 64;
    }

    # HTTP → HTTPS redirect
    server {
        listen 80;
        server_name gotradex.com;
        return 301 https://$server_name$request_uri;
    }

    # Main HTTPS server
    server {
        listen 443 ssl http2;
        server_name gotradex.com;

        # TLS certificate (use certbot / Let's Encrypt in production)
        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
        ssl_prefer_server_ciphers off;
        ssl_session_cache shared:SSL:10m;
        ssl_session_timeout 1d;

        # Security headers
        add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
        add_header X-Frame-Options SAMEORIGIN;
        add_header X-Content-Type-Options nosniff;

        # Swagger UI (static files from api-gateway container)
        location /swagger/ {
            proxy_pass http://api_gateway/swagger/;
            proxy_set_header Host $host;
        }

        # Auth endpoints — stricter rate limiting
        location /api/v1/auth/ {
            limit_req zone=auth burst=10 nodelay;
            proxy_pass http://api_gateway;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # Protected API endpoints
        location /api/v1/ {
            limit_req zone=api burst=50 nodelay;
            proxy_pass http://api_gateway;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            # Pass JWT to upstream (nginx does NOT validate — api-gateway does)
            proxy_set_header Authorization $http_authorization;
        }

        # Health check endpoint (no rate limiting)
        location /healthz {
            proxy_pass http://api_gateway;
            proxy_set_header Host $host;
            access_log off;
        }

        # Market Service WebSocket
        location /ws {
            proxy_pass http://market_ws;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_read_timeout 86400;
            proxy_send_timeout 86400;
        }

        # Default — 404
        location / {
            return 404 '{"error":"not found"}';
            add_header Content-Type application/json;
        }
    }
}
```

---

## gRPC Proxy (for User/Order Services)

For gRPC services (User Service on port 50051, Order Service on port 50052), nginx must handle HTTP/2:

```nginx
# gRPC upstreams
upstream user_service {
    server 127.0.0.1:50051;
}

upstream order_service {
    server 127.0.0.1:50052;
}

# gRPC server block
server {
    listen 50051 http2;
    server_name gotradex.com;

    # gRPC-specific headers
    location / {
        grpc_pass grpc://user_service;
    }
}
```

> **Note**: nginx `grpc_pass` requires the upstream to support HTTP/2. For production gRPC load balancing, consider Envoy or Linkerd instead of nginx.

---

## Docker: Nginx as Sidecar

For containerized deployment, run nginx as a sidecar container alongside API Gateway:

```yaml
# docker-compose.prod.yml excerpt
services:
  api-gateway:
    # ... existing config ...
    expose:
      - "8080"

  nginx:
    image: nginx:alpine
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    ports:
      - "80:80"
      - "443:443"
    depends_on:
      - api-gateway
```

---

## Health Check

```bash
# HTTP health check
curl -f https://gotradex.com/healthz

# gRPC health check
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

---

## Verification Checklist

- [ ] HTTP (port 80) redirects to HTTPS (port 443)
- [ ] `/swagger/*` serves Swagger UI
- [ ] `/api/v1/auth/*` has stricter rate limiting (5 r/s vs 100 r/s)
- [ ] WebSocket `/ws` upgrades correctly and stays open
- [ ] TLS handshake completes with valid cipher suite
- [ ] Upstream headers (`X-Real-IP`, `X-Forwarded-For`) are passed correctly
- [ ] `nginx -t` passes with no configuration errors

---

## Performance Tuning

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| `worker_processes` | `auto` | Use all CPU cores |
| `worker_connections` | `4096` | Handle high concurrent WebSocket connections |
| `keepalive` | `64` | Reuse connections to upstream |
| `proxy_read_timeout` | `86400` | WebSocket long-lived connections (24h) |
| `gzip` | `on` | Compress JSON responses |
| `ssl_session_cache` | `10m` | Reduce TLS handshake overhead |

> ➡️ Next: [Step 15 — Monitoring & Alerting](./15-monitoring.md) *(future)*