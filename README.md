# GoTradeX

**Real-time Trading & Order Matching Platform**

A high-performance, event-driven cryptocurrency trading platform built with Go microservices. GoTradeX features a custom in-memory order matching engine, gRPC service communication, Kafka event streaming, and real-time WebSocket market data broadcasting.

---

## 🏗 Architecture

The platform is designed around a clean microservices architecture, decoupled by Kafka for high-throughput order processing and real-time market updates.

### Microservices

* **API Gateway (REST/Gin):** Serves as the entry point for all client requests. Handles authentication, routing, and rate limiting.
* **User Service (gRPC):** Manages user registration, authentication (JWT), and account balances (PostgreSQL).
* **Order Service (gRPC + Kafka):** Validates balances, persists orders, and publishes them to the Kafka `orders` topic.
* **Matching Engine (Kafka Consumer):** High-performance in-memory order book (Bids: Max-Heap, Asks: Min-Heap). Consumes matching requests and emits filled trades to the Kafka `trades` topic.
* **Market Service (Kafka Consumer + WebSocket):** Consumes trades from Kafka and broadcasts them in real-time to subscribed WebSocket clients.

### Communication Flow
```text
Client ──(REST)──► API Gateway ──(gRPC)──► Order Service ──(Produce)──► Kafka [orders]
                                                                             │
                                                                         (Consume)
                                                                             ▼
                                                                      Matching Engine
                                                                             │
                                                                         (Produce)
                                                                             ▼
Client ◄──(ws://)── Market Service ◄──(Consume)──────────────────────── Kafka [trades]
```

## 🛠 Tech Stack

* **Language:** Go 1.22+
* **Frameworks:** Gin (REST), gRPC/Protobuf
* **Message Broker:** Apache Kafka (segmentio/kafka-go)
* **Database:** PostgreSQL (jackc/pgx)
* **Caching & Rate Limiting:** Redis
* **Real-time Data:** WebSockets (gorilla/websocket)
* **Infrastructure:** Docker & Docker Compose
* **Observability:** Uber Zap (Structured Logging)

---

## 🚀 Quick Start

Ensure you have **Docker**, **Docker Compose**, and **Go 1.22+** installed on your machine.

### 1. Start Infrastructure
Start the required dependencies (PostgreSQL, Kafka, Zookeeper, Redis):
```bash
make docker-up
```

### 2. Run Database Migrations
Create the necessary tables for users, balances, and orders:
```bash
make migrate
```

### 3. Start the Microservices
Run the compiled services locally:
```bash
make run-all
```
*(Alternatively, you can build and run individual services from the `cmd/` directory.)*

---

## 📖 API Usage Examples

### 1. Register a new user
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"trader@example.com","password":"secret123"}'
```

### 2. Login
```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -d '{"email":"trader@example.com","password":"secret123"}' | jq -r .token)
```

### 3. Place a Limit Order
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "BTC/USD",
    "side": "BUY",
    "type": "LIMIT",
    "price": "42000.00",
    "quantity": "0.01",
    "idempotency_key": "order-001"
  }'
```

### 4. Listen to Market Trades (WebSocket)
Connect to the Market Service to receive real-time streams of matched trades:
```javascript
const ws = new WebSocket("ws://localhost:8081/ws?symbol=BTC/USD");

ws.onmessage = (event) => {
  const trade = JSON.parse(event.data);
  console.log(`Trade Executed: ${trade.qty} BTC @ $${trade.price}`);
};
```

---

## 📂 Repository Structure

```text
GoTradeX/
├── cmd/                # Entrypoints for microservices
│   ├── api-gateway/
│   ├── user-service/
│   ├── order-service/
│   ├── matching-engine/
│   └── market-service/
├── docs/               # Project documentation and roadmap
├── internal/           # Private application code (domain logic)
├── pkg/                # Shared libraries (logger, db, kafka, auth)
├── proto/              # gRPC Protobuf definitions
├── migrations/         # SQL migration scripts
├── docker/             # Dockerfiles for each service
├── docker-compose.yml  # Local infrastructure orchestration
└── Makefile            # Common build and run commands
```


