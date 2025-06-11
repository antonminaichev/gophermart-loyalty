package order

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/antonminaichev/gophermart-loyalty/internal/types/order"
	"github.com/antonminaichev/gophermart-loyalty/internal/util/luna"
)

var (
	ErrInvalidNumber      = errors.New("invalid order number")
	ErrOrderAlreadyExists = errors.New("order already uploaded by this user")
	ErrOrderConflict      = errors.New("order already uploaded by another user")
	ErrOrderAccepted      = errors.New("order accepted")
)

type Service struct {
	repo OrderRepository
}

func NewService(r OrderRepository) *Service {
	return &Service{repo: r}
}

func (s *Service) SubmitOrder(ctx context.Context, userID int64, number string) error {
	if !luna.Validate(number) {
		return ErrInvalidNumber
	}
	existing, err := s.repo.FindOrderByNumber(ctx, number)
	if err == nil {
		if existing.UserID == userID {
			return ErrOrderAlreadyExists
		}
		return ErrOrderConflict
	}
	if err != sql.ErrNoRows {
		return err
	}
	o := &order.Order{
		UserID:     userID,
		Number:     number,
		Status:     order.StatusNew,
		UploadedAt: time.Now().UTC(),
	}
	if err := s.repo.CreateOrder(ctx, o); err != nil {
		return err
	}
	return ErrOrderAccepted
}

func (s *Service) ListOrders(ctx context.Context, userID int64) ([]order.Order, error) {
	return s.repo.ListOrdersByUser(ctx, userID)
}

func (s *Service) ListForPolling(ctx context.Context) ([]order.Order, error) {
	return s.repo.ListOrdersForPolling(ctx)
}

func (s *Service) UpdateFromAccrual(ctx context.Context, number string, status string, accrual *float64) error {
	var processedAt *time.Time
	if status == "PROCESSED" || status == "INVALID" {
		now := time.Now().UTC()
		processedAt = &now
	}
	return s.repo.UpdateOrderUpdate(ctx, number, status, accrual, processedAt)
}
