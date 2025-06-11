package order

import (
	"context"
	"time"

	"github.com/antonminaichev/gophermart-loyalty/internal/types/order"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, o *order.Order) error
	ListOrdersByUser(ctx context.Context, userID int64) ([]order.Order, error)
	UpdateOrder(ctx context.Context, o *order.Order) error
	FindOrderByNumber(ctx context.Context, number string) (*order.Order, error)
	ListOrdersForPolling(ctx context.Context) ([]order.Order, error)
	UpdateOrderUpdate(ctx context.Context, number string, status string, accrualAmount *float64, processedAt *time.Time) error
}
