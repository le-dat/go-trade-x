package order

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var (
	ErrOnlyPendingCancel  = errors.New("only PENDING orders can be cancelled")
	ErrInvalidQuantity   = errors.New("invalid quantity")
	ErrInvalidSymbol     = errors.New("invalid symbol format")
)

type Service interface {
	PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*PlaceOrderResponse, error)
	GetOrder(ctx context.Context, orderID, userID uuid.UUID) (*Order, error)
	CancelOrder(ctx context.Context, orderID, userID uuid.UUID) error
}

type service struct {
	repo       Repository
	userClient UserServiceClient
	log        *zap.Logger
}

func NewService(repo Repository, userClient UserServiceClient, log *zap.Logger) Service {
	return &service{
		repo:       repo,
		userClient: userClient,
		log:        log,
	}
}

type PlaceOrderRequest struct {
	UserID         uuid.UUID
	Symbol         string
	Side           OrderSide
	Type           OrderType
	Price          string
	Quantity       decimal.Decimal
	IdempotencyKey string
}

type PlaceOrderResponse struct {
	OrderID   string
	Status    OrderStatus
	Symbol    string
	Side      OrderSide
	Type      OrderType
	Price     string
	Quantity  string
	FilledQty string
}

func (s *service) PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*PlaceOrderResponse, error) {
	if req.Quantity.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidQuantity
	}

	// 1. Idempotency check — return existing order if duplicate
	existing, err := s.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
	if err == nil {
		return &PlaceOrderResponse{
			OrderID:   existing.ID.String(),
			Status:    existing.Status,
			Symbol:    existing.Symbol,
			Side:      existing.Side,
			Type:      existing.Type,
			Price:     existing.Price,
			Quantity:  existing.Quantity,
			FilledQty: existing.FilledQty,
		}, nil
	}
	if !errors.Is(err, ErrOrderNotFound) {
		return nil, err
	}

	// 2. Validate symbol
	asset := quoteAsset(req.Symbol, req.Side)
	if asset == "" {
		return nil, ErrInvalidSymbol
	}

	// 3. Atomic DB transaction: deduct balance + write order + write outbox
	now := time.Now()
	order := &Order{
		ID:             uuid.New(),
		IdempotencyKey: req.IdempotencyKey,
		UserID:         req.UserID,
		Symbol:         req.Symbol,
		Side:           req.Side,
		Type:           req.Type,
		Price:          req.Price,
		Quantity:       req.Quantity.String(),
		FilledQty:      "0",
		Status:         StatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	outboxPayload, _ := json.Marshal(map[string]interface{}{
		"order_id":    order.ID.String(),
		"user_id":     order.UserID.String(),
		"symbol":      order.Symbol,
		"side":        order.Side,
		"type":        order.Type,
		"price":       order.Price,
		"quantity":    order.Quantity,
		"idempotency": order.IdempotencyKey,
		"event":       "ORDER_PLACED",
		"timestamp":   now.Unix(),
	})

	outboxMsg := &OutboxMessage{
		ID:        uuid.New(),
		Topic:     "orders",
		Key:       order.UserID.String(),
		Payload:   outboxPayload,
		Status:    OutboxStatusPending,
		CreatedAt: now,
	}

	err = s.withTransaction(ctx, func(tx pgx.Tx) error {
		// Deduct balance within the same transaction using SELECT FOR UPDATE
		if _, err := s.repo.DeductBalanceTx(ctx, tx, req.UserID, asset, req.Quantity); err != nil {
			return err
		}
		if err := s.repo.CreateWithTx(ctx, tx, order); err != nil {
			return err
		}
		return s.repo.InsertOutboxWithTx(ctx, tx, outboxMsg)
	})

	if err != nil {
		s.log.With(zap.Error(err)).Error("PlaceOrder transaction failed",
			zap.String("symbol", req.Symbol),
			zap.String("side", string(req.Side)),
			zap.Stringer("quantity", req.Quantity),
		)
		return nil, err
	}

	s.log.Info("Order placed",
		zap.String("order_id", order.ID.String()),
		zap.String("symbol", order.Symbol),
		zap.String("side", string(order.Side)),
	)

	return &PlaceOrderResponse{
		OrderID:   order.ID.String(),
		Status:    order.Status,
		Symbol:    order.Symbol,
		Side:      order.Side,
		Type:      order.Type,
		Price:     order.Price,
		Quantity:  order.Quantity,
		FilledQty: order.FilledQty,
	}, nil
}

func (s *service) GetOrder(ctx context.Context, orderID, userID uuid.UUID) (*Order, error) {
	o, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	// Only enforce ownership when userID is not Nil
	if userID != uuid.Nil && o.UserID != userID {
		return nil, ErrOrderNotFound
	}
	return o, nil
}

func (s *service) CancelOrder(ctx context.Context, orderID, userID uuid.UUID) error {
	o, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	if userID != uuid.Nil && o.UserID != userID {
		return ErrOrderNotFound
	}
	if o.Status != StatusPending {
		return ErrOnlyPendingCancel
	}

	qty, err := decimal.NewFromString(o.Quantity)
	if err != nil {
		s.log.With(zap.Error(err)).Error("failed to parse quantity on cancel",
			zap.String("order_id", orderID.String()),
			zap.String("quantity", o.Quantity),
		)
		return ErrInvalidQuantity
	}

	asset := quoteAsset(o.Symbol, o.Side)

	// Refund balance within a transaction
	err = s.withTransaction(ctx, func(tx pgx.Tx) error {
		return s.repo.CreditBalanceTx(ctx, tx, userID, asset, qty)
	})
	if err != nil {
		s.log.With(
			zap.Error(err),
			zap.String("order_id", orderID.String()),
			zap.String("asset", asset),
			zap.Stringer("quantity", qty),
		).Error("failed to credit balance on cancel — order cancelled but refund failed")
		return err
	}

	if err := s.repo.UpdateStatus(ctx, orderID, StatusCancelled); err != nil {
		return err
	}

	s.log.Info("Order cancelled", zap.String("order_id", orderID.String()))
	return nil
}

// withTransaction executes fn within a database transaction, rolling back on error.
func (s *service) withTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// quoteAsset returns the quote currency for BUY, base for SELL.
// Returns empty string for malformed symbols.
func quoteAsset(symbol string, side OrderSide) string {
	parts := splitSymbol(symbol)
	if parts[0] == "" || parts[1] == "" {
		return ""
	}
	if side == SideSell {
		return parts[0]
	}
	return parts[1]
}

// splitSymbol splits "BTC/USD" into [base, quote].
// Returns ["",""] if separator not found.
func splitSymbol(symbol string) [2]string {
	var base, quote string
	for i := 0; i < len(symbol); i++ {
		if symbol[i] == '/' {
			base = symbol[:i]
			quote = symbol[i+1:]
			return [2]string{base, quote}
		}
	}
	return [2]string{"", ""}
}