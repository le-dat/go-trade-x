# Step 08 — Dockerization: Multi-Stage Builds + docker-compose

## Goal

Containerize all services with multi-stage Dockerfiles and docker-compose orchestration.

> **Prerequisite**: [Step 07 — Market Service](../03-services/07-market-service.md)

---

## Step 8.1 — Per-Service Dockerfile Pattern

`docker/api-gateway/Dockerfile`:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/service ./cmd/api-gateway

FROM gcr.io/distroless/static-debian12
COPY --from=builder /bin/service /service
ENTRYPOINT ["/service"]
```

Apply same pattern to:
- `docker/user-service/Dockerfile`
- `docker/order-service/Dockerfile`
- `docker/matching-engine/Dockerfile`
- `docker/market-service/Dockerfile`

---

## Step 8.2 — docker-compose.yml

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: trading
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5436:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  kafka:
    image: confluentinc/cp-kafka:7.6.0
    ports:
      - "9092:9092"
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:29092,PLAINTEXT_HOST://localhost:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: 0
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: 1
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
    depends_on:
      - zookeeper

  zookeeper:
    image: confluentinc/cp-zookeeper:7.6.0
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  api-gateway:
    build: ./docker/api-gateway
    ports:
      - "8080:8080"
    depends_on:
      - user-service
      - order-service

  user-service:
    build: ./docker/user-service
    ports:
      - "50051:50051"
    depends_on:
      - postgres

  order-service:
    build: ./docker/order-service
    ports:
      - "50052:50052"
    depends_on:
      - postgres
      - kafka
      - user-service

  matching-engine:
    build: ./docker/matching-engine
    depends_on:
      - kafka

  market-service:
    build: ./docker/market-service
    ports:
      - "8081:8081"
    depends_on:
      - kafka

volumes:
  postgres_data:
```

---

## Step 8.3 — Start & Verify

```bash
make docker-up
docker compose ps

# Verify all services healthy
curl http://localhost:8080/healthz
# → {"status":"ok"}
```

---

## Verification Checklist

- [ ] `make docker-up` starts all services
- [ ] All services report healthy
- [ ] End-to-end order flow works

> ➡️ Next: [Step 09 — CI/CD](../04-devops/09-cicd.md)