# Step 07 — Market Service: WebSocket Hub + Kafka Consumer

## Goal

Implement the WebSocket hub that broadcasts trade events to subscribed clients.

> **Prerequisite**: [Step 06 — Matching Engine](./06-matching-engine.md)

---

## WebSocket Hub Design

`internal/market/hub.go`:

```go
type Hub struct {
    clients    map[string]map[*Client]bool  // symbol → clients
    register   chan *Client
    unregister chan *Client
    broadcast  chan TradeEvent
    mu         sync.RWMutex
}

func (h *Hub) Run(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case client := <-h.register:
            h.mu.Lock()
            if h.clients[client.symbol] == nil {
                h.clients[client.symbol] = make(map[*Client]bool)
            }
            h.clients[client.symbol][client] = true
            h.mu.Unlock()
        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client.symbol][client]; ok {
                delete(h.clients[client.symbol], client)
                close(client.send)
            }
            h.mu.Unlock()
        case trade := <-h.broadcast:
            h.mu.RLock()
            clients := h.clients[trade.Symbol]
            h.mu.RUnlock()
            msg, _ := json.Marshal(trade)
            for client := range clients {
                select {
                case client.send <- msg:
                default:
                    h.unregister <- client
                }
            }
        }
    }
}
```

---

## Client Connection

`internal/market/client.go`:

```go
const (
    writeWait      = 10 * time.Second
    pongWait       = 30 * time.Second
    maxMessageSize = 512
)

func (c *Client) Read() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()
    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })
    for {
        _, _, err := c.conn.ReadMessage()
        if err != nil {
            break
        }
    }
}

func (c *Client) Write() {
    ticker := time.NewTicker(pongWait)
    defer ticker.Stop()
    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
                return
            }
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

---

## WebSocket Upgrade

`cmd/market-service/main.go`:

```go
http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
    symbol := r.URL.Query().Get("symbol")
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), symbol: symbol}
    hub.register <- client
    go client.Write()
    go client.Read()
})
```

---

## Wire Kafka Consumer

```go
go func() {
    consumer.Run(ctx, "trades", func(ctx context.Context, msg kafka.Message) error {
        var trade TradeEvent
        json.Unmarshal(msg.Value, &trade)
        hub.broadcast <- trade
        return nil
    })
}()
```

---

## Client Usage

```javascript
const ws = new WebSocket("ws://localhost:8081/ws?symbol=BTC/USD");
ws.onmessage = (event) => {
  const trade = JSON.parse(event.data);
  console.log(`Trade: ${trade.qty} BTC @ $${trade.price}`);
};
```

---

## Verification Checklist

- [ ] Client connects with `ws://localhost:8081/ws?symbol=BTC/USD`
- [ ] Ping/pong keepalive works (30s interval)
- [ ] Client receives trade < 100ms after match
- [ ] Silent unregister on write error

> ➡️ Next: [Step 08 — Dockerization](../devops/08-docker.md)