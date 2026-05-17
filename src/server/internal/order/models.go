package order

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrInsufficientBalance = errors.New("insufficient balance")

type OrderStatus string

const (
	StatusPending   OrderStatus = "PENDING"
	StatusPartial  OrderStatus = "PARTIAL"
	StatusFilled   OrderStatus = "FILLED"
	StatusCancelled OrderStatus = "CANCELLED"
	StatusRejected  OrderStatus = "REJECTED"
)

type OrderSide string
type OrderType string

const (
	SideBuy  OrderSide = "BUY"
	SideSell OrderSide = "SELL"

	TypeLimit  OrderType = "LIMIT"
	TypeMarket OrderType = "MARKET"
)

type Order struct {
	ID             uuid.UUID  `json:"id"`
	IdempotencyKey string     `json:"idempotency_key"`
	UserID         uuid.UUID  `json:"user_id"`
	Symbol         string     `json:"symbol"`
	Side           OrderSide  `json:"side"`
	Type           OrderType  `json:"type"`
	Price          string     `json:"price"`
	Quantity       string     `json:"quantity"`
	FilledQty      string     `json:"filled_qty"`
	Status         OrderStatus `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type OutboxMessage struct {
	ID        uuid.UUID `json:"id"`
	Topic     string    `json:"topic"`
	Key       string    `json:"key"`
	Payload   []byte    `json:"payload"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

const OutboxStatusPending   = "PENDING"
const OutboxStatusProcessed = "PROCESSED"
const OutboxStatusFailed    = "FAILED"

// UserServiceClient defines the interface for calling User Service from Order Service.
type UserServiceClient interface {
	DeductBalance(ctx context.Context, userID, asset string, amount decimal.Decimal) error
	CreditBalance(ctx context.Context, userID, asset string, amount decimal.Decimal) error
}