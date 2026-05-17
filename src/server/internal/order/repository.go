package order

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var (
	ErrOrderNotFound = errors.New("order not found")
)

type Repository interface {
	CreateWithTx(ctx context.Context, tx pgx.Tx, order *Order) error
	InsertOutboxWithTx(ctx context.Context, tx pgx.Tx, msg *OutboxMessage) error
	GetByIdempotencyKey(ctx context.Context, key string) (*Order, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Order, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status OrderStatus) error
	GetPendingOutbox(ctx context.Context, limit int) ([]OutboxMessage, error)
	MarkOutboxProcessed(ctx context.Context, id uuid.UUID) error
	MarkOutboxFailed(ctx context.Context, id uuid.UUID) error
	BeginTx(ctx context.Context) (pgx.Tx, error)
	// DeductBalanceTx deducts balance within a transaction using SELECT FOR UPDATE.
	// Returns the new available balance after deduction. Returns error if insufficient.
	DeductBalanceTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string, amount decimal.Decimal) (decimal.Decimal, error)
	// CreditBalanceTx credits balance within a transaction.
	CreditBalanceTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string, amount decimal.Decimal) error
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, order *Order) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO orders (id, idempotency_key, user_id, symbol, side, type, price, quantity, filled_qty, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		order.ID, order.IdempotencyKey, order.UserID, order.Symbol, order.Side, order.Type,
		order.Price, order.Quantity, order.FilledQty, order.Status, order.CreatedAt, order.UpdatedAt,
	)
	return err
}

func (r *postgresRepository) GetByIdempotencyKey(ctx context.Context, key string) (*Order, error) {
	var o Order
	err := r.db.QueryRow(ctx,
		`SELECT id, idempotency_key, user_id, symbol, side, type, price, quantity, filled_qty, status, created_at, updated_at
		 FROM orders WHERE idempotency_key = $1`,
		key,
	).Scan(&o.ID, &o.IdempotencyKey, &o.UserID, &o.Symbol, &o.Side, &o.Type, &o.Price, &o.Quantity, &o.FilledQty, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	return &o, err
}

func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	var o Order
	err := r.db.QueryRow(ctx,
		`SELECT id, idempotency_key, user_id, symbol, side, type, price, quantity, filled_qty, status, created_at, updated_at
		 FROM orders WHERE id = $1`,
		id,
	).Scan(&o.ID, &o.IdempotencyKey, &o.UserID, &o.Symbol, &o.Side, &o.Type, &o.Price, &o.Quantity, &o.FilledQty, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	return &o, err
}

func (r *postgresRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status OrderStatus) error {
	_, err := r.db.Exec(ctx,
		`UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3`,
		status, time.Now(), id,
	)
	return err
}

func (r *postgresRepository) GetPendingOutbox(ctx context.Context, limit int) ([]OutboxMessage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, topic, key, payload, status, created_at FROM outbox WHERE status = $1 ORDER BY created_at ASC LIMIT $2`,
		OutboxStatusPending, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []OutboxMessage
	for rows.Next() {
		var m OutboxMessage
		if err := rows.Scan(&m.ID, &m.Topic, &m.Key, &m.Payload, &m.Status, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *postgresRepository) InsertOutboxWithTx(ctx context.Context, tx pgx.Tx, msg *OutboxMessage) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO outbox (id, topic, key, payload, status, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		msg.ID, msg.Topic, msg.Key, msg.Payload, msg.Status, msg.CreatedAt,
	)
	return err
}

func (r *postgresRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func (r *postgresRepository) MarkOutboxProcessed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE outbox SET status = $1 WHERE id = $2`, OutboxStatusProcessed, id)
	return err
}

func (r *postgresRepository) MarkOutboxFailed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE outbox SET status = $1 WHERE id = $2`, OutboxStatusFailed, id)
	return err
}

func (r *postgresRepository) DeductBalanceTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string, amount decimal.Decimal) (decimal.Decimal, error) {
	var available decimal.Decimal
	err := tx.QueryRow(ctx,
		`SELECT available FROM balances WHERE user_id = $1 AND asset = $2 FOR UPDATE`,
		userID, asset,
	).Scan(&available)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return decimal.Zero, ErrInsufficientBalance
		}
		return decimal.Zero, err
	}

	if available.LessThan(amount) {
		return decimal.Zero, ErrInsufficientBalance
	}

	newAvailable := available.Sub(amount)
	_, err = tx.Exec(ctx,
		`UPDATE balances SET available = $1 WHERE user_id = $2 AND asset = $3`,
		newAvailable, userID, asset,
	)
	if err != nil {
		return decimal.Zero, err
	}
	return newAvailable, nil
}

func (r *postgresRepository) CreditBalanceTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string, amount decimal.Decimal) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO balances (user_id, asset, available, locked) VALUES ($1, $2, $3, 0)
		 ON CONFLICT (user_id, asset) DO UPDATE SET available = balances.available + $3`,
		userID, asset, amount,
	)
	return err
}