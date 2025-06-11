package balance

import (
	"context"

	"github.com/antonminaichev/gophermart-loyalty/internal/types/balance"
)

type BalanceRepository interface {
	GetBalance(ctx context.Context, userID int64) (current float64, withdrawn float64, err error)
}

type WithdrawalRepository interface {
	CreateWithdrawal(ctx context.Context, w *balance.Withdrawal) error
	ListWithdrawalsByUser(ctx context.Context, userID int64) ([]balance.Withdrawal, error)
}
