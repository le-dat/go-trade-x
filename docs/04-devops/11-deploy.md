# Step 11 — Deployment: Production Checklist

## Goal

Document production deployment requirements and checklist.

> **Prerequisite**: [Step 10 — Environment & Config](./10-env.md)

---

## Production Requirements

### Infrastructure

- PostgreSQL 16+ with connection pooling (PgBouncer or pgpool-II)
- Kafka cluster (3+ brokers for production)
- Redis 7+ for rate limiting
- Docker Swarm or Kubernetes for orchestration

### Security

- TLS termination at load balancer
- Secrets management (Vault, AWS Secrets Manager, etc.)
- Network segmentation (services should not expose ports directly)
- Regular security scanning of dependencies

### Monitoring

- Prometheus metrics endpoint on each service
- Grafana dashboards for:
  - Request latency p50, p95, p99
  - Error rates
  - Kafka consumer lag
  - Order book depth
  - WebSocket connection count
- Alerting on:
  - Error rate > 1%
  - Kafka consumer lag > 1000 messages
  - Matching engine latency p99 > 10ms

---

## Deployment Steps

### 1. Build Images

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml build
```

### 2. Run Migrations

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml exec postgres psql -U postgres -d gotradex -f /path/to/migrations/*.up.sql
```

### 3. Start Services

```bash
make prod-up
```

### 4. Verify

```bash
docker compose ps
curl http://localhost:8080/healthz
```

---

## Health Check Endpoints

| Service | Endpoint |
|---------|----------|
| API Gateway | `GET /healthz` |
| User Service | gRPC `/healthz` |
| Order Service | gRPC `/healthz` |
| Market Service | `GET /healthz` (HTTP) |

---

## Rollback Procedure

1. Re-tag previous image: `docker tag gotradex:prev gotradex:current`
2. Restart service: `docker compose -f docker-compose.yml -f docker-compose.prod.yml restart <service>`
3. Verify: `curl http://localhost:8080/healthz`

---

## Makefile Deployment Targets

```bash
make prod-up          # Start all services for production (no port exposure)
make prod-stop        # Stop all production services
make docker-logs      # Follow logs
make docker-logs-<svc> # Follow specific service logs
```

---

## Verification Checklist

- [ ] All services start and report healthy
- [ ] Migrations run successfully
- [ ] End-to-end order flow works
- [ ] Monitoring dashboards show data
- [ ] Alerts configured and tested

> ➡️ Next: [Step 12 — Architecture Decisions](../05-architecture/12-architecture.md)