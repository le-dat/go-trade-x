# Step 02 — API Gateway: Gin REST + JWT Auth + Rate Limiting

## Goal

Build the API Gateway that handles client requests with JWT authentication and rate limiting.

> **Prerequisite**: [Step 01 — Infrastructure](../01-infrastructure/01-system-overview.md)

---

## Endpoints

```
POST /api/v1/auth/register  → UserService.Register (gRPC)
POST /api/v1/auth/login     → UserService.Login (gRPC)
POST /api/v1/orders         → OrderService.PlaceOrder (gRPC) [JWT required]
GET  /api/v1/orders/:id     → OrderService.GetOrder (gRPC) [JWT required]
GET  /healthz
```

---

## Middleware Stack (applied in order)

1. `Recovery` — catch panics, return 500
2. `Logger` — zap structured request logging (method, path, status, latency)
3. `RateLimiter` — token bucket per IP (use `golang.org/x/time/rate`)
4. `Auth` — validate JWT, inject `userID` into context (protected routes only)

---

## Step 2.1 — Install gRPC Toolchain

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/cmd/protoc-gen-grpc-gateway@latest
go install github.com/envoyproxy/protoc-gen-validate@latest
```

---

## Step 2.2 — gRPC Client Factory

`cmd/api-gateway/clients/grpc.go`:

```go
func NewGrpcClients(ctx context.Context) (*GrpcClients, error) {
    userConn, err := grpc.DialContext(ctx, "localhost:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, fmt.Errorf("connect to user service: %w", err)
    }

    orderConn, err := grpc.DialContext(ctx, "localhost:50052",
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, fmt.Errorf("connect to order service: %w", err)
    }

    return &GrpcClients{
        UserService:  userpb.NewUserServiceClient(userConn),
        OrderService: orderpb.NewOrderServiceClient(orderConn),
    }, nil
}
```

---

## Step 2.3 — Middleware

`cmd/api-gateway/middleware/auth.go`:

```go
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "missing token"})
            return
        }

        claims, err := ValidateToken(strings.TrimPrefix(token, "Bearer "), jwtSecret)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
            return
        }

        c.Set("userID", claims.UserID)
        c.Next()
    }
}
```

---

## Step 2.4 — Start & Test

```bash
make run-api
curl http://localhost:8080/healthz
# → {"status":"ok"}

# Without token (should 401)
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/orders
# → 401
```

---

## Verification Checklist

- [ ] Gateway starts on port 8080
- [ ] `/healthz` returns `{"status":"ok"}`
- [ ] Auth middleware rejects requests without JWT
- [ ] Rate limiter enforces limits

> ➡️ Next: [Step 03 — User Service](../03-services/03-user-service.md)