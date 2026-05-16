package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/verno/gotradex/pkg/logger"
	"github.com/verno/gotradex/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	proto.UnimplementedUserServiceServer
	svc Service
	log *zap.Logger
}

func NewHandler(svc Service) *Handler {
	return &Handler{
		svc: svc,
		log: logger.Get().With(zap.String("service", "user")),
	}
}

func (h *Handler) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	u, err := h.svc.Register(ctx, req.Email, req.Password)
	if err != nil {
		h.log.With(zap.Error(err)).Error("Register failed")
		if err == ErrEmailAlreadyExists {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &proto.RegisterResponse{
		UserId: u.ID.String(),
		Email:  u.Email,
	}, nil
}

func (h *Handler) Login(ctx context.Context, req *proto.LoginRequest) (*proto.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	token, u, err := h.svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		h.log.With(zap.Error(err)).Error("Login failed")
		if err == ErrInvalidCredentials {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &proto.LoginResponse{
		Token:  token,
		UserId: u.ID.String(),
		Email:  u.Email,
	}, nil
}

func (h *Handler) GetBalance(ctx context.Context, req *proto.GetBalanceRequest) (*proto.GetBalanceResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	balances, err := h.svc.GetBalance(ctx, userID)
	if err != nil {
		h.log.With(zap.Error(err)).Error("GetBalance failed")
		return nil, status.Error(codes.Internal, err.Error())
	}

	var protoBalances []*proto.Balance
	for _, b := range balances {
		protoBalances = append(protoBalances, &proto.Balance{
			Asset:     b.Asset,
			Available: b.Available,
			Locked:    b.Locked,
		})
	}

	return &proto.GetBalanceResponse{
		UserId:   userID.String(),
		Balances: protoBalances,
	}, nil
}

func (h *Handler) DeductBalance(ctx context.Context, req *proto.DeductBalanceRequest) (*proto.DeductBalanceResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}

	if err := h.svc.DeductBalance(ctx, userID, req.Asset, req.Amount); err != nil {
		h.log.With(zap.Error(err)).Error("DeductBalance failed")
		if err == ErrInsufficientBalance {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	newBalance, err := h.svc.GetBalance(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var available float64
	for _, b := range newBalance {
		if b.Asset == req.Asset {
			available = b.Available
			break
		}
	}

	return &proto.DeductBalanceResponse{
		Success:    true,
		NewBalance: available,
	}, nil
}

func (h *Handler) CreditBalance(ctx context.Context, req *proto.CreditBalanceRequest) (*proto.CreditBalanceResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}

	if err := h.svc.CreditBalance(ctx, userID, req.Asset, req.Amount); err != nil {
		h.log.With(zap.Error(err)).Error("CreditBalance failed")
		return nil, status.Error(codes.Internal, err.Error())
	}

	newBalance, err := h.svc.GetBalance(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var available float64
	for _, b := range newBalance {
		if b.Asset == req.Asset {
			available = b.Available
			break
		}
	}

	return &proto.CreditBalanceResponse{
		Success:    true,
		NewBalance: available,
	}, nil
}
