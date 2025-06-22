package balance

import (
	"context"
	"fmt"
	"time"

	"github.com/antonminaichev/gophermart-loyalty/internal/types/balance"
)

type Service struct {
	b BalanceRepository
	w WithdrawalRepository
}

func NewService(b BalanceRepository, w WithdrawalRepository) *Service {
	return &Service{b: b, w: w}
}

func (s *Service) ListBalance(ctx context.Context, userID int64) (*balance.BalanceDTO, error) {
	current, withdrawn, err := s.b.GetBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &balance.BalanceDTO{
		Current:   current,
		Withdrawn: withdrawn,
	}, nil
}

func (s *Service) WithdrawBalance(ctx context.Context, userID int64, req *balance.WithdrawRequest) error {
	current, _, err := s.b.GetBalance(ctx, userID)
	if err != nil {
		fmt.Println("Something Went wrong")
		return err
	}
	if req.Sum > current {
		return fmt.Errorf("insufficient funds")
	}
	withdrawal := &balance.Withdrawal{
		UserID:      userID,
		OrderNumber: req.Order,
		Amount:      req.Sum,
		CreatedAt:   time.Now().UTC(),
	}
	return s.w.CreateWithdrawal(ctx, withdrawal)
}

func (s *Service) ListWithdrawals(ctx context.Context, userID int64) ([]balance.Withdrawal, error) {
	return s.w.ListWithdrawalsByUser(ctx, userID)
}
