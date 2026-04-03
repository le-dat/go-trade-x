package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type Balance struct {
	UserID    uuid.UUID
	Asset     string
	Available float64
	Locked    float64
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error)
	GetBalances(ctx context.Context, userID uuid.UUID) ([]Balance, error)
	GetAvailableBalance(ctx context.Context, userID uuid.UUID, asset string) (float64, error)
	DeductBalance(ctx context.Context, userID uuid.UUID, asset string, amount float64) error
	CreditBalance(ctx context.Context, userID uuid.UUID, asset string, amount float64) error
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CreateUser(ctx context.Context, email, passwordHash string) (*User, error) {
	user := &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, created_at) VALUES ($1, $2, $3, $4)`,
		user.ID, user.Email, user.PasswordHash, user.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, err
	}

	// Initialize default balances for common assets
	defaultAssets := []string{"USD", "BTC", "ETH"}
	for _, asset := range defaultAssets {
		_, err := r.db.Exec(ctx,
			`INSERT INTO balances (user_id, asset, available, locked) VALUES ($1, $2, 0, 0)`,
			user.ID, asset,
		)
		if err != nil && !isUniqueViolation(err) {
			return nil, err
		}
	}

	return user, nil
}

func (r *postgresRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *postgresRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	var user User
	err := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *postgresRepository) GetBalances(ctx context.Context, userID uuid.UUID) ([]Balance, error) {
	rows, err := r.db.Query(ctx,
		`SELECT user_id, asset, available, locked FROM balances WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []Balance
	for rows.Next() {
		var b Balance
		if err := rows.Scan(&b.UserID, &b.Asset, &b.Available, &b.Locked); err != nil {
			return nil, err
		}
		balances = append(balances, b)
	}
	return balances, rows.Err()
}

func (r *postgresRepository) GetAvailableBalance(ctx context.Context, userID uuid.UUID, asset string) (float64, error) {
	var available float64
	err := r.db.QueryRow(ctx,
		`SELECT available FROM balances WHERE user_id = $1 AND asset = $2`,
		userID, asset,
	).Scan(&available)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return available, nil
}

func (r *postgresRepository) DeductBalance(ctx context.Context, userID uuid.UUID, asset string, amount float64) error {
	result, err := r.db.Exec(ctx,
		`UPDATE balances SET available = available - $1 WHERE user_id = $2 AND asset = $3 AND available >= $1`,
		amount, userID, asset,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("insufficient balance or asset not found")
	}
	return nil
}

func (r *postgresRepository) CreditBalance(ctx context.Context, userID uuid.UUID, asset string, amount float64) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO balances (user_id, asset, available, locked) VALUES ($1, $2, $3, 0)
		 ON CONFLICT (user_id, asset) DO UPDATE SET available = balances.available + $3`,
		userID, asset, amount,
	)
	return err
}

func isUniqueViolation(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate key") || contains(err.Error(), "unique constraint"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
