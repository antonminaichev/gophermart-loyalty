package storage

import (
	"context"

	"github.com/antonminaichev/gophermart-loyalty/internal/types/balance"
	"github.com/antonminaichev/gophermart-loyalty/internal/types/order"
	"github.com/antonminaichev/gophermart-loyalty/internal/types/user"
)

// UserRepository отвечает за операции над пользователями.
type UserRepository interface {
	Create(ctx context.Context, u *user.User) error
	FindByLogin(ctx context.Context, login string) (*user.User, error)
}

// OrderRepository отвечает за операции над заказами.
type OrderRepository interface {
	CreateOrder(ctx context.Context, o *order.Order) error
	ListOrdersByUser(ctx context.Context, userID int64) ([]order.Order, error)
	UpdateOrder(ctx context.Context, o *order.Order) error
	FindOrderByNumber(ctx context.Context, number string) (*order.Order, error)
}

// WithdrawalRepository отвечает за списания.
type WithdrawalRepository interface {
	CreateWithdrawal(ctx context.Context, w *balance.Withdrawal) error
	ListWithdrawalsByUser(ctx context.Context, userID int64) ([]balance.Withdrawal, error)
}

// BalanceRepository умеет считать текущий и уже списанный баланс.
type BalanceRepository interface {
	GetBalance(ctx context.Context, userID int64) (current float64, withdrawn float64, err error)
}

// Storage объединяет все репозитории.
type Storage interface {
	UserRepository
	OrderRepository
	WithdrawalRepository
	BalanceRepository

	// Для управления соединением
	Ping(ctx context.Context) error
	Close() error
}
