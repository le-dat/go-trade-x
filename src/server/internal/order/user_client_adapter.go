package order

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/verno/gotradex/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserServiceClientAdapter wraps the generated proto UserServiceClient
// to implement the order.Service's UserServiceClient interface.
type UserServiceClientAdapter struct {
	client proto.UserServiceClient
}

func NewUserServiceClientAdapter(client proto.UserServiceClient) *UserServiceClientAdapter {
	return &UserServiceClientAdapter{client: client}
}

func (a *UserServiceClientAdapter) DeductBalance(ctx context.Context, userID, asset string, amount decimal.Decimal) error {
	_, err := a.client.DeductBalance(ctx, &proto.DeductBalanceRequest{
		UserId: userID,
		Asset:  asset,
		Amount: amount.InexactFloat64(),
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.FailedPrecondition {
			return ErrInsufficientBalance
		}
		return err
	}
	return nil
}

func (a *UserServiceClientAdapter) CreditBalance(ctx context.Context, userID, asset string, amount decimal.Decimal) error {
	_, err := a.client.CreditBalance(ctx, &proto.CreditBalanceRequest{
		UserId: userID,
		Asset:  asset,
		Amount: amount.InexactFloat64(),
	})
	return err
}