package order

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/verno/gotradex/pkg/logger"
	"github.com/verno/gotradex/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// userIDKey is the gRPC metadata key for passing user ID from API gateway.
const userIDKey = "x-user-id"

func userIDFromContext(ctx context.Context) (uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.Nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	values := md.Get(userIDKey)
	if len(values) == 0 || values[0] == "" {
		return uuid.Nil, status.Error(codes.Unauthenticated, "missing user ID in metadata")
	}
	return uuid.Parse(values[0])
}

func validateSide(s string) bool {
	return s == "BUY" || s == "SELL"
}

func validateType(t string) bool {
	return t == "LIMIT" || t == "MARKET"
}

type Handler struct {
	proto.UnimplementedOrderServiceServer
	svc Service
	log *zap.Logger
}

func NewHandler(svc Service) *Handler {
	return &Handler{
		svc: svc,
		log: logger.Get().With(zap.String("service", "order")),
	}
}

func (h *Handler) PlaceOrder(ctx context.Context, req *proto.PlaceOrderRequest) (*proto.PlaceOrderResponse, error) {
	if req.Symbol == "" || req.Quantity == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol and quantity are required")
	}
	if !validateSide(req.Side) {
		return nil, status.Error(codes.InvalidArgument, "side must be BUY or SELL")
	}
	if !validateType(req.Type) {
		return nil, status.Error(codes.InvalidArgument, "type must be LIMIT or MARKET")
	}

	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	qty, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid quantity format")
	}

	if req.Type == "LIMIT" && req.Price == "" {
		return nil, status.Error(codes.InvalidArgument, "price is required for LIMIT orders")
	}

	orderReq := &PlaceOrderRequest{
		UserID:         userID,
		Symbol:         strings.TrimSpace(req.Symbol),
		Side:           OrderSide(req.Side),
		Type:           OrderType(req.Type),
		Price:          req.Price,
		Quantity:       qty,
		IdempotencyKey: req.IdempotencyKey,
	}

	resp, err := h.svc.PlaceOrder(ctx, orderReq)
	if err != nil {
		h.log.With(zap.Error(err)).Error("PlaceOrder failed")
		if errors.Is(err, ErrInsufficientBalance) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		if errors.Is(err, ErrInvalidSymbol) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, ErrInvalidQuantity) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &proto.PlaceOrderResponse{
		OrderId:   resp.OrderID,
		Status:    string(resp.Status),
		Symbol:    resp.Symbol,
		Side:      string(resp.Side),
		Type:      string(resp.Type),
		Price:     resp.Price,
		Quantity:  resp.Quantity,
		FilledQty: resp.FilledQty,
	}, nil
}

func (h *Handler) GetOrder(ctx context.Context, req *proto.GetOrderRequest) (*proto.GetOrderResponse, error) {
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}

	userID := uuid.Nil
	if uid, err := userIDFromContext(ctx); err == nil {
		userID = uid
	}

	o, err := h.svc.GetOrder(ctx, orderID, userID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &proto.GetOrderResponse{
		OrderId:   o.ID.String(),
		UserId:    o.UserID.String(),
		Symbol:    o.Symbol,
		Side:      string(o.Side),
		Type:      string(o.Type),
		Price:     o.Price,
		Quantity:  o.Quantity,
		FilledQty: o.FilledQty,
		Status:    string(o.Status),
	}, nil
}

func (h *Handler) CancelOrder(ctx context.Context, req *proto.CancelOrderRequest) (*proto.CancelOrderResponse, error) {
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}

	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.svc.CancelOrder(ctx, orderID, userID); err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		if errors.Is(err, ErrOnlyPendingCancel) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		if errors.Is(err, ErrInvalidQuantity) {
			return nil, status.Error(codes.Internal, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &proto.CancelOrderResponse{
		Success: true,
		OrderId: orderID.String(),
	}, nil
}

// Ensure Handler implements proto.UnimplementedOrderServiceServer
var _ proto.OrderServiceServer = (*Handler)(nil)