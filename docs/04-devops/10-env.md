# Step 10 — Environment & Config: env vars for all services

## Goal

Document all environment variables needed across services.

> **Prerequisite**: [Step 09 — CI/CD](./09-cicd.md)

---

## API Gateway

```bash
API_GATEWAY_PORT=8080
USER_SERVICE_PORT=50051
ORDER_SERVICE_PORT=50052
JWT_SECRET=your-secret-key
JWT_EXPIRY=24h
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=trading
REDIS_HOST=localhost
REDIS_PORT=6379
GO_ENV=development
```

---

## User Service

```bash
USER_SERVICE_PORT=50051
DATABASE_URL=postgresql://postgres:postgres@localhost:5436/trading?sslmode=disable
JWT_SECRET=your-secret-key
JWT_EXPIRY=24h
GO_ENV=development
```

---

## Order Service

```bash
ORDER_SERVICE_PORT=50052
DATABASE_URL=postgresql://postgres:postgres@localhost:5436/trading?sslmode=disable
USER_SERVICE_ADDR=localhost:50051
KAFKA_BROKER=localhost:9092
KAFKA_ORDERS_TOPIC=orders
GO_ENV=development
```

---

## Matching Engine

```bash
KAFKA_BROKER=localhost:9092
KAFKA_ORDERS_TOPIC=orders
KAFKA_TRADES_TOPIC=trades
GO_ENV=development
```

---

## Market Service

```bash
MARKET_SERVICE_WS_PORT=8081
KAFKA_BROKER=localhost:9092
KAFKA_TRADES_TOPIC=trades
GO_ENV=development
```

---

## .env.example (root)

```bash
# API Gateway
API_GATEWAY_PORT=8080

# Services
USER_SERVICE_PORT=50051
ORDER_SERVICE_PORT=50052
MARKET_SERVICE_WS_PORT=8081

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=trading
DATABASE_URL=postgresql://postgres:postgres@localhost:5436/trading?sslmode=disable

# Kafka
KAFKA_BROKER=localhost:9092
KAFKA_ORDERS_TOPIC=orders
KAFKA_TRADES_TOPIC=trades

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Auth
JWT_SECRET=your-secret-key
JWT_EXPIRY=24h

# Go
GO_ENV=development
```

> ➡️ Next: [Step 11 — Deployment](../04-devops/11-deploy.md)