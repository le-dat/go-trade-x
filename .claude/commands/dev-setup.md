# /dev-setup — Start Local Development Infrastructure

Start Docker Compose services (Postgres, Kafka, Redis) and verify all are healthy.

## Steps

### Step 1: Check Docker is running

```bash
docker info > /dev/null 2>&1 && echo "Docker: OK" || echo "Docker: NOT RUNNING"
```

- Done when: Docker is running

### Step 2: Start infrastructure

```bash
cd /home/verno/projects/personal/learn/claude-claw/GoTradeX
make docker-up
```

- Done when: All containers started

### Step 3: Wait for services to be healthy

```bash
sleep 10 && docker compose ps
```

### Step 4: Verify Postgres

```bash
docker compose exec postgres pg_isready -U postgres && echo "Postgres: OK"
```

### Step 5: Verify Kafka

```bash
docker compose exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092 && echo "Kafka: OK"
```

### Step 6: Verify Redis (if configured)

```bash
docker compose exec redis redis-cli ping && echo "Redis: OK"
```

### Step 7: Run migrations

```bash
make migrate
```

## Summary

Print health status table:

```
| Service | Status |
|---------|--------|
| postgres | ✅ Healthy |
| kafka | ✅ Healthy |
| redis | ✅ Healthy |
| api-gateway | ✅ Built |
| user-service | ✅ Built |
| order-service | ✅ Built |
| matching-engine | ✅ Built |
| market-service | ✅ Built |
```

## Next

After setup, start individual services with:
- `make run-api` — API Gateway on :8080
- `make run-user` — User Service on :50051
- `make run-order` — Order Service on :50052
- `make run-market` — Market Service on :8081 (WebSocket)
