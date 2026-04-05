# /kafka-test — Kafka Producer/Consumer Round-Trip Test

Test that Kafka producer and consumer work end-to-end. Requires Kafka running via `make docker-up`.

## Prerequisites

```bash
make docker-up
# Wait 10 seconds for Kafka to be ready
sleep 10
```

## Steps

### Step 1: Verify Kafka is running

```bash
docker compose exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092
```

- Done when: Returns broker API versions without error

### Step 2: Create test topics

```bash
docker compose exec kafka kafka-topics --create \
  --if-not-exists \
  --topic test-orders \
  --bootstrap-server localhost:9092 \
  --partitions 4 \
  --replication-factor 1

docker compose exec kafka kafka-topics --create \
  --if-not-exists \
  --topic test-trades \
  --bootstrap-server localhost:9092 \
  --partitions 4 \
  --replication-factor 1
```

### Step 3: Write round-trip test

Create `pkg/kafka/roundtrip_test.go`:

```go
func TestProducerConsumer_RoundTrip(t *testing.T) {
    // Start consumer in goroutine
    // Publish message to test-orders
    // Wait for consumer to receive
    // Assert message matches
}
```

### Step 4: Run the test

```bash
go test ./pkg/kafka/... -v -run TestProducerConsumer_RoundTrip -timeout 30s
```

### Step 5: Verify retry logic (optional)

```bash
# Kill a consumer mid-test, verify producer retries
# Restart consumer, verify no message loss
```

## Done When

- Test passes: message published → consumer receives it
- Consumer commits offset only after handler returns success
- Producer retries on failure (max 3 attempts)

## Cleanup

```bash
docker compose exec kafka kafka-topics --delete --topic test-orders --bootstrap-server localhost:9092
docker compose exec kafka kafka-topics --delete --topic test-trades --bootstrap-server localhost:9092
```
