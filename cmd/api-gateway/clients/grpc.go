package clients

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/verno/gotradex/pkg/auth"
)

// UserServiceClient interface for User Service gRPC calls
type UserServiceClient interface {
	Register(ctx context.Context, email, password string) (*RegisterResponse, error)
	Login(ctx context.Context, email, password string) (*LoginResponse, error)
	GetBalance(ctx context.Context, userID string) (*GetBalanceResponse, error)
	DeductBalance(ctx context.Context, userID, asset string, amount float64) (*DeductBalanceResponse, error)
	CreditBalance(ctx context.Context, userID, asset string, amount float64) (*CreditBalanceResponse, error)
}

// OrderServiceClient interface for Order Service gRPC calls
type OrderServiceClient interface {
	PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*PlaceOrderResponse, error)
	GetOrder(ctx context.Context, orderID string) (*GetOrderResponse, error)
	CancelOrder(ctx context.Context, orderID string) (*CancelOrderResponse, error)
}

// Response types
type RegisterResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
}

type GetBalanceResponse struct {
	UserID    string             `json:"user_id"`
	Balances  []BalanceResponse  `json:"balances"`
}

type BalanceResponse struct {
	Asset     string  `json:"asset"`
	Available float64 `json:"available"`
	Locked    float64 `json:"locked"`
}

type DeductBalanceResponse struct {
	Success bool    `json:"success"`
	NewBalance float64 `json:"new_balance"`
}

type CreditBalanceResponse struct {
	Success bool    `json:"success"`
	NewBalance float64 `json:"new_balance"`
}

type PlaceOrderRequest struct {
	UserID          string
	Symbol          string
	Side            string
	Type            string
	Price           string
	Quantity        string
	IdempotencyKey  string
}

type PlaceOrderResponse struct {
	OrderID   string `json:"order_id"`
	Status    string `json:"status"`
	Symbol    string `json:"symbol"`
	Side      string `json:"side"`
	Type      string `json:"type"`
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
	FilledQty string `json:"filled_qty"`
}

type GetOrderResponse struct {
	OrderID   string `json:"order_id"`
	UserID    string `json:"user_id"`
	Symbol    string `json:"symbol"`
	Side      string `json:"side"`
	Type      string `json:"type"`
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
	FilledQty string `json:"filled_qty"`
	Status    string `json:"status"`
}

type CancelOrderResponse struct {
	Success  bool   `json:"success"`
	OrderID  string `json:"order_id"`
}

// NewGRPCClientFactory creates a client factory based on environment
func NewGRPCClientFactory(jwtManager *auth.JWTManager) (UserServiceClient, OrderServiceClient) {
	// In production, this would create real gRPC clients
	// For now, return mock clients since proto definitions don't exist
	return NewMockUserClient(jwtManager), NewMockOrderClient()
}

// GRPCClientFactory holds the gRPC client connections
type GRPCClientFactory struct {
	UserService  UserServiceClient
	OrderService OrderServiceClient
}

func NewGRPCClientFactoryWithTimeout(jwtSecret string, timeout time.Duration) *GRPCClientFactory {
	jwtManager := auth.NewJWTManager(jwtSecret, 24*time.Hour)
	return &GRPCClientFactory{
		UserService:  NewMockUserClient(jwtManager),
		OrderService: NewMockOrderClient(),
	}
}

// Ensure mock implementations implement the interfaces
var _ UserServiceClient = (*MockUserClient)(nil)
var _ OrderServiceClient = (*MockOrderClient)(nil)
