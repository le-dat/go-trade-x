# Step 13 — Security: Authentication, Authorization, Data Protection

## Goal

Document security measures and best practices for GoTradeX.

---

## Authentication

### JWT

- **Algorithm**: HS256
- **Expiry**: 24 hours
- **Claims**: `userID`, `exp`, `iat`
- **Secret**: Minimum 256-bit key, stored in secrets manager

### Token Validation

```go
func ValidateToken(tokenString string, secret string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(secret), nil
    })
    if err != nil {
        return nil, err
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, errors.New("invalid token")
    }

    return claims, nil
}
```

---

## Authorization

### Protected Routes

All `/api/v1/orders/*` routes require valid JWT in `Authorization: Bearer <token>` header.

### Service-to-Service

- gRPC calls use `insecure.NewCredentials()` in development
- In production: mTLS with certificate rotation

---

## Data Protection

### Database

- `SELECT FOR UPDATE` for balance mutations (prevents race conditions)
- All connections over TLS in production
- Connection pooling with limited max connections

### Kafka

- TLS for broker communication in production
- No sensitive data in message keys (use `userID`, not full payload)

### WebSocket

- Client write timeout: 10 seconds
- Max message size: 512 bytes
- Silent unregister on write error (don't expose internal errors)

---

## Security Checklist

### Development

- [ ] Never commit `.env` files
- [ ] Use environment variables for all secrets
- [ ] Run `golangci-lint` before push

### Production

- [ ] TLS termination at load balancer
- [ ] Secrets stored in Vault/AWS Secrets Manager
- [ ] Network segmentation (private subnets)
- [ ] Regular dependency scanning (`go mod verify`, `trivy`)
- [ ] Rate limiting enabled on API Gateway
- [ ] Idempotency keys prevent replay attacks

---

## Secrets Management

```bash
# NEVER do this
JWT_SECRET=my-super-secret-key  # commit to git = instant revoke

# DO this
JWT_SECRET=${JWT_SECRET}  # read from environment
```

In Kubernetes:
```yaml
env:
  - name: JWT_SECRET
    valueFrom:
      secretKeyRef:
        name: gotradex-secrets
        key: jwt-secret
```

---

## Rate Limiting

`golang.org/x/time/rate` — token bucket per IP:

```go
func RateLimiterMiddleware(rps float64, burst int) gin.HandlerFunc {
    limiter := rate.NewLimiter(rate.Limit(rps), burst)
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.AbortWithStatusJSON(429, gin.H{"error": "rate limit exceeded"})
            return
        }
        c.Next()
    }
}
```

Applied at 100 req/min per IP on API Gateway.