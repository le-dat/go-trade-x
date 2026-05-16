package clients

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/verno/gotradex/pkg/auth"
)

var (
	ErrMockInvalidCredentials = errors.New("invalid credentials")
	ErrMockUserNotFound       = errors.New("user not found")
	ErrMockInsufficientBalance = errors.New("insufficient balance")
	ErrMockOrderNotFound      = errors.New("order not found")
)

// MockUserClient implements UserServiceClient with mock responses
type MockUserClient struct {
	jwtManager *auth.JWTManager
}

func NewMockUserClient(jwtManager *auth.JWTManager) *MockUserClient {
	return &MockUserClient{
		jwtManager: jwtManager,
	}
}

func (m *MockUserClient) Register(ctx context.Context, email, password string) (*RegisterResponse, error) {
	// Mock implementation - in production this would call UserService via gRPC
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	userID := uuid.New().String()
	return &RegisterResponse{
		UserID: userID,
		Email:  email,
	}, nil
}

func (m *MockUserClient) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	// Mock implementation - in production this would call UserService via gRPC
	if email == "" || password == "" {
		return nil, ErrMockInvalidCredentials
	}

	// For demo purposes, accept any non-empty credentials
	userID := uuid.New().String()
	token, err := m.jwtManager.GenerateToken(userID, email)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:  token,
		UserID: userID,
		Email:  email,
	}, nil
}

func (m *MockUserClient) GetBalance(ctx context.Context, userID string) (*GetBalanceResponse, error) {
	// Mock implementation - return some demo balances
	return &GetBalanceResponse{
		UserID: userID,
		Balances: []BalanceResponse{
			{Asset: "USD", Available: 10000.0, Locked: 0},
			{Asset: "BTC", Available: 0.5, Locked: 0},
		},
	}, nil
}

func (m *MockUserClient) DeductBalance(ctx context.Context, userID, asset string, amount float64) (*DeductBalanceResponse, error) {
	// Mock implementation
	if amount > 10000 {
		return nil, ErrMockInsufficientBalance
	}
	return &DeductBalanceResponse{
		Success:    true,
		NewBalance: 10000.0 - amount,
	}, nil
}

func (m *MockUserClient) CreditBalance(ctx context.Context, userID, asset string, amount float64) (*CreditBalanceResponse, error) {
	// Mock implementation
	return &CreditBalanceResponse{
		Success:    true,
		NewBalance: 10000.0 + amount,
	}, nil
}

// MockOrderClient implements OrderServiceClient with mock responses
type MockOrderClient struct{}

func NewMockOrderClient() *MockOrderClient {
	return &MockOrderClient{}
}

func (m *MockOrderClient) PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*PlaceOrderResponse, error) {
	// Mock implementation - in production this would call OrderService via gRPC
	if req.UserID == "" || req.Symbol == "" || req.Quantity == "" {
		return nil, errors.New("invalid request")
	}

	orderID := uuid.New().String()
	return &PlaceOrderResponse{
		OrderID:   orderID,
		Status:    "PENDING",
		Symbol:    req.Symbol,
		Side:      req.Side,
		Type:      req.Type,
		Price:     req.Price,
		Quantity:  req.Quantity,
		FilledQty: "0",
	}, nil
}

func (m *MockOrderClient) GetOrder(ctx context.Context, orderID string) (*GetOrderResponse, error) {
	// Mock implementation
	if orderID == "" {
		return nil, ErrMockOrderNotFound
	}

	return &GetOrderResponse{
		OrderID:   orderID,
		UserID:    uuid.New().String(),
		Symbol:    "BTC/USD",
		Side:      "BUY",
		Type:      "LIMIT",
		Price:     "42000.00",
		Quantity:  "0.01",
		FilledQty: "0",
		Status:    "PENDING",
	}, nil
}

func (m *MockOrderClient) CancelOrder(ctx context.Context, orderID string) (*CancelOrderResponse, error) {
	// Mock implementation
	if orderID == "" {
		return nil, ErrMockOrderNotFound
	}

	return &CancelOrderResponse{
		Success: true,
		OrderID: orderID,
	}, nil
}
