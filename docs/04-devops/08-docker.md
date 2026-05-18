# Step 08 — Dockerization: Multi-Stage Builds + docker-compose

## Goal

Containerize all services with multi-stage Dockerfiles and docker-compose orchestration. Use a base `docker-compose.yml` for infrastructure only, with override files for local dev and production.

> **Prerequisite**: [Step 07 — Market Service](../03-services/07-market-service.md)

---

## Step 8.1 — Per-Service Dockerfile

`docker/Dockerfile` (single file, builds via `SERVICE_NAME` arg):

```dockerfile
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git gcc musl-dev
WORKDIR /app
COPY go.mod go.sum* ./
COPY . .
RUN go mod tidy && go mod download
ARG SERVICE_NAME
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/service ./cmd/${SERVICE_NAME}

FROM gcr.io/distroless/static-debian12
COPY --from=builder /bin/service /service
ENTRYPOINT ["/service"]
```

---

## Step 8.2 — Base docker-compose.yml (Infrastructure Only)

`docker-compose.yml` defines only infrastructure — no service builds:

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: gotradex
    ports:
      - "5436:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  kafka:
    image: bitnamilegacy/kafka:3.7
    environment:
      - KAFKA_CFG_NODE_ID=0
      - KAFKA_CFG_PROCESS_ROLES=controller,broker
      - KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093
      - KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      - KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=0@kafka:9093
      - KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER
    ports:
      - "9092:9092"
```

---

## Step 8.3 — docker-compose.override.yml (Local Dev)

`docker-compose.override.yml` is **auto-loaded** by `docker compose up`. Builds all 5 services with port mappings:

```yaml
services:
  api-gateway:
    build:
      context: .
      dockerfile: docker/Dockerfile
      args:
        SERVICE_NAME: api-gateway
    ports:
      - "8080:8080"
    environment:
      - APP_PORT=8080
      - DATABASE_URL=postgres://postgres:postgres@localhost:5436/gotradex?sslmode=disable
      - KAFKA_BROKERS=localhost:9092
    depends_on:
      postgres:
        condition: service_healthy

  user-service:
    build:
      context: .
      dockerfile: docker/Dockerfile
      args:
        SERVICE_NAME: user-service
    ports:
      - "50051:50051"
    environment:
      - DATABASE_URL=postgres://postgres:postgres@localhost:5436/gotradex?sslmode=disable
    depends_on:
      postgres:
        condition: service_healthy

  # ... order-service, matching-engine, market-service similarly
```

---

## Step 8.4 — docker-compose.prod.yml (VPS/Production)

`docker-compose.prod.yml` is used for VPS/production. No port mappings, uses internal DNS:

```bash
# On VPS, deploy with:
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

---

## Step 8.5 — Start & Verify

```bash
# Start infrastructure only (postgres + kafka)
make docker-up

# Start everything (infra + all services, auto-loads override)
make dev-up

# Start for production (no port exposure)
make prod-up

# View logs
make docker-logs
make docker-logs-api

# Stop
make docker-stop
```

| Target | Description |
|--------|-------------|
| `make docker-up` | Infrastructure only (postgres + kafka) |
| `make dev-up` | All services via `docker compose up -d` |
| `make prod-up` | Production deploy via `docker-compose.prod.yml` |
| `make docker-stop` | Stop all services |
| `make docker-down` | Stop infrastructure |
| `make docker-logs` | Follow all logs |
| `make docker-logs-<svc>` | Follow specific service logs |
| `make docker-migrate` | Run migrations on docker postgres |

---

## Verification Checklist

- [ ] `make docker-up` starts postgres + kafka healthy
- [ ] `make dev-up` starts all 5 services with ports exposed
- [ ] `make prod-up` starts services without port exposure
- [ ] `make docker-stop` cleanly stops all services

> ➡️ Next: [Step 09 — CI/CD](../04-devops/09-cicd.md)