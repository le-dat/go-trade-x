package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/verno/gotradex/pkg/auth"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

type Service interface {
	Register(ctx context.Context, email, password string) (*User, error)
	Login(ctx context.Context, email, password string) (string, *User, error)
	GetBalance(ctx context.Context, userID uuid.UUID) ([]Balance, error)
	DeductBalance(ctx context.Context, userID uuid.UUID, asset string, amount float64) error
	CreditBalance(ctx context.Context, userID uuid.UUID, asset string, amount float64) error
}

type service struct {
	repo     Repository
	jwtMgr   *auth.JWTManager
	jwtExpiry time.Duration
}

func NewService(repo Repository, jwtMgr *auth.JWTManager, jwtExpiry time.Duration) Service {
	return &service{
		repo:     repo,
		jwtMgr:   jwtMgr,
		jwtExpiry: jwtExpiry,
	}
}

func (s *service) Register(ctx context.Context, email, password string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.CreateUser(ctx, email, string(hash))
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *service) Login(ctx context.Context, email, password string) (string, *User, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	token, err := s.jwtMgr.GenerateToken(user.ID.String(), user.Email)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

func (s *service) GetBalance(ctx context.Context, userID uuid.UUID) ([]Balance, error) {
	return s.repo.GetBalances(ctx, userID)
}

func (s *service) DeductBalance(ctx context.Context, userID uuid.UUID, asset string, amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}

	available, err := s.repo.GetAvailableBalance(ctx, userID, asset)
	if err != nil {
		return err
	}
	if available < amount {
		return ErrInsufficientBalance
	}

	return s.repo.DeductBalance(ctx, userID, asset, amount)
}

func (s *service) CreditBalance(ctx context.Context, userID uuid.UUID, asset string, amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	return s.repo.CreditBalance(ctx, userID, asset, amount)
}
