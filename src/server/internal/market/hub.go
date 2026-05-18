package market

import (
	"encoding/json"
	"sync"

	"go.uber.org/zap"
)

// TradeEvent represents a trade that is broadcast to clients.
type TradeEvent struct {
	Symbol    string `json:"symbol"`
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
	BuyOrder  string `json:"buy_order"`
	SellOrder string `json:"sell_order"`
}

// Hub maintains the set of active clients and broadcasts messages to clients.
type Hub struct {
	clients map[string]map[*Client]bool

	register   chan *Client
	unregister chan *Client
	broadcast  chan TradeEvent

	mu  sync.RWMutex
	log *zap.Logger
}

// NewHub creates a new Hub.
func NewHub(log *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan TradeEvent, 256),
		log:        log,
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run(ctx <-chan struct{}) {
	for {
		select {
		case <-ctx:
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

			if len(clients) == 0 {
				continue
			}

			msg, err := json.Marshal(trade)
			if err != nil {
				h.log.With(zap.Error(err)).Warn("failed to marshal trade event")
				continue
			}

			for client := range clients {
				select {
				case client.send <- msg:
				default:
					// Slow client - unregister
					h.mu.Lock()
					if _, ok := h.clients[client.symbol][client]; ok {
						delete(h.clients[client.symbol], client)
						close(client.send)
					}
					h.mu.Unlock()
				}
			}
		}
	}
}

// Broadcast sends a trade event to all subscribers of the symbol.
func (h *Hub) Broadcast(trade TradeEvent) {
	h.broadcast <- trade
}

// Subscribe registers a client for a symbol.
func (h *Hub) Subscribe(client *Client) {
	h.register <- client
}

// Unsubscribe unregisters a client.
func (h *Hub) Unsubscribe(client *Client) {
	h.unregister <- client
}