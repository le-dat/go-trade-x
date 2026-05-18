package clients

import (
	"context"
	"os"
	"time"

	"github.com/verno/gotradex/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

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
	GetOrder(ctx context.Context, orderID string, userID string) (*GetOrderResponse, error)
	CancelOrder(ctx context.Context, orderID string, userID string) (*CancelOrderResponse, error)
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
	orderServiceAddr := os.Getenv("ORDER_SERVICE_ADDR")
	if orderServiceAddr == "" {
		orderServiceAddr = "localhost:50052"
	}
	orderClient, _ := newGRPCOrderClient(orderServiceAddr)
	return &GRPCClientFactory{
		UserService:  NewMockUserClient(jwtManager),
		OrderService: orderClient,
	}
}

// Ensure mock implementations implement the interfaces
var _ UserServiceClient = (*MockUserClient)(nil)
var _ OrderServiceClient = (*MockOrderClient)(nil)

// grpcOrderClient wraps the generated proto OrderServiceClient with proper error handling.
type grpcOrderClient struct {
	conn   *grpc.ClientConn
	client proto.OrderServiceClient
}

func newGRPCOrderClient(addr string) (*grpcOrderClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &grpcOrderClient{
		conn:   conn,
		client: proto.NewOrderServiceClient(conn),
	}, nil
}

func (c *grpcOrderClient) PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*PlaceOrderResponse, error) {
	md := metadata.Pairs("x-user-id", req.UserID)
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := c.client.PlaceOrder(ctx, &proto.PlaceOrderRequest{
		UserId:         req.UserID,
		Symbol:         req.Symbol,
		Side:           req.Side,
		Type:           req.Type,
		Price:          req.Price,
		Quantity:       req.Quantity,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	return &PlaceOrderResponse{
		OrderID:   resp.OrderId,
		Status:    resp.Status,
		Symbol:    resp.Symbol,
		Side:      resp.Side,
		Type:      resp.Type,
		Price:     resp.Price,
		Quantity:  resp.Quantity,
		FilledQty: resp.FilledQty,
	}, nil
}

func (c *grpcOrderClient) GetOrder(ctx context.Context, orderID string, userID string) (*GetOrderResponse, error) {
	md := metadata.Pairs("x-user-id", userID)
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := c.client.GetOrder(ctx, &proto.GetOrderRequest{OrderId: orderID})
	if err != nil {
		return nil, err
	}
	return &GetOrderResponse{
		OrderID:   resp.OrderId,
		UserID:    resp.UserId,
		Symbol:    resp.Symbol,
		Side:      resp.Side,
		Type:      resp.Type,
		Price:     resp.Price,
		Quantity:  resp.Quantity,
		FilledQty: resp.FilledQty,
		Status:    resp.Status,
	}, nil
}

func (c *grpcOrderClient) CancelOrder(ctx context.Context, orderID string, userID string) (*CancelOrderResponse, error) {
	md := metadata.Pairs("x-user-id", userID)
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := c.client.CancelOrder(ctx, &proto.CancelOrderRequest{OrderId: orderID})
	if err != nil {
		return nil, err
	}
	return &CancelOrderResponse{
		Success: resp.Success,
		OrderID: resp.OrderId,
	}, nil
}

func (c *grpcOrderClient) Close() error {
	return c.conn.Close()
}

// Ensure grpcOrderClient implements OrderServiceClient
var _ OrderServiceClient = (*grpcOrderClient)(nil)
