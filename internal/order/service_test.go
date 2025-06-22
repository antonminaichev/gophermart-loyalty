package order

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/antonminaichev/gophermart-loyalty/internal/types/order"
	"github.com/stretchr/testify/assert"
)

type mockRepo struct {
	findOrderByNumberFn    func(ctx context.Context, number string) (*order.Order, error)
	createOrderFn          func(ctx context.Context, o *order.Order) error
	updateOrderFn          func(ctx context.Context, o *order.Order) error
	listOrdersByUserFn     func(ctx context.Context, userID int64) ([]order.Order, error)
	listOrdersForPollingFn func(ctx context.Context) ([]order.Order, error)
	updateOrderUpdateFn    func(ctx context.Context, number, status string, accrual *float64, processedAt *time.Time) error
}

func (m *mockRepo) FindOrderByNumber(ctx context.Context, number string) (*order.Order, error) {
	return m.findOrderByNumberFn(ctx, number)
}
func (m *mockRepo) CreateOrder(ctx context.Context, o *order.Order) error {
	return m.createOrderFn(ctx, o)
}
func (m *mockRepo) ListOrdersByUser(ctx context.Context, userID int64) ([]order.Order, error) {
	return m.listOrdersByUserFn(ctx, userID)
}
func (m *mockRepo) ListOrdersForPolling(ctx context.Context) ([]order.Order, error) {
	return m.listOrdersForPollingFn(ctx)
}
func (m *mockRepo) UpdateOrderUpdate(ctx context.Context, number, status string, accrual *float64, processedAt *time.Time) error {
	return m.updateOrderUpdateFn(ctx, number, status, accrual, processedAt)
}
func (m *mockRepo) UpdateOrder(ctx context.Context, o *order.Order) error {
	return m.updateOrderFn(ctx, o)
}

func TestSubmitOrderInvalidNumber(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)
	err := svc.SubmitOrder(context.Background(), 1, "12345")
	assert.Equal(t, ErrInvalidNumber, err)
}

func TestSubmitOrderAlreadyExists(t *testing.T) {
	repo := &mockRepo{
		findOrderByNumberFn: func(ctx context.Context, number string) (*order.Order, error) {
			return &order.Order{UserID: 1, Number: number}, nil
		},
	}
	svc := NewService(repo)
	err := svc.SubmitOrder(context.Background(), 1, "12345678903")
	assert.Equal(t, ErrOrderAlreadyExists, err)
}

func TestSubmitOrderConflict(t *testing.T) {
	repo := &mockRepo{
		findOrderByNumberFn: func(ctx context.Context, number string) (*order.Order, error) {
			return &order.Order{UserID: 2, Number: number}, nil
		},
	}
	svc := NewService(repo)
	err := svc.SubmitOrder(context.Background(), 1, "12345678903")
	assert.Equal(t, ErrOrderConflict, err)
}

func TestSubmitOrderNewOrder(t *testing.T) {
	repo := &mockRepo{
		findOrderByNumberFn: func(ctx context.Context, number string) (*order.Order, error) {
			return nil, sql.ErrNoRows // <---- вот это правильно!
		},
		createOrderFn: func(ctx context.Context, o *order.Order) error {
			return nil
		},
	}
	svc := NewService(repo)
	err := svc.SubmitOrder(context.Background(), 1, "79927398713")
	assert.Equal(t, ErrOrderAccepted, err)
}

func TestUpdateFromAccrualProcessed(t *testing.T) {
	var called bool
	repo := &mockRepo{
		updateOrderUpdateFn: func(ctx context.Context, number, status string, accrual *float64, processedAt *time.Time) error {
			called = true
			assert.NotNil(t, processedAt)
			return nil
		},
	}
	svc := NewService(repo)
	err := svc.UpdateFromAccrual(context.Background(), "order1", "PROCESSED", nil)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestListOrders(t *testing.T) {
	repo := &mockRepo{
		listOrdersByUserFn: func(ctx context.Context, userID int64) ([]order.Order, error) {
			return []order.Order{{UserID: userID, Number: "1"}}, nil
		},
	}
	svc := NewService(repo)
	orders, err := svc.ListOrders(context.Background(), 1)
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
}
