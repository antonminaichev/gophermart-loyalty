package balance

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/antonminaichev/gophermart-loyalty/internal/middleware"
	"github.com/antonminaichev/gophermart-loyalty/internal/types/balance"
)

type stubBalanceRepo struct {
	current   float64
	withdrawn float64
	errGet    error
}

func (r *stubBalanceRepo) GetBalance(ctx context.Context, userID int64) (float64, float64, error) {
	if r.errGet != nil {
		return 0, 0, r.errGet
	}
	return r.current, r.withdrawn, nil
}

type stubWithdrawalRepo struct {
	withdrawals []balance.Withdrawal
	errCreate   error
	errList     error
}

func (r *stubWithdrawalRepo) CreateWithdrawal(ctx context.Context, w *balance.Withdrawal) error {
	if r.errCreate != nil {
		return r.errCreate
	}
	r.withdrawals = append(r.withdrawals, *w)
	return nil
}

func (r *stubWithdrawalRepo) ListWithdrawalsByUser(ctx context.Context, userID int64) ([]balance.Withdrawal, error) {
	if r.errList != nil {
		return nil, r.errList
	}
	return r.withdrawals, nil
}

func TestListBalance(t *testing.T) {
	balRepo := &stubBalanceRepo{current: 100, withdrawn: 30}
	withdrawRepo := &stubWithdrawalRepo{}

	svc := NewService(balRepo, withdrawRepo)

	bal, err := svc.ListBalance(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}

	if bal.Current != 100 {
		t.Errorf("expected current 100, got %f", bal.Current)
	}
	if bal.Withdrawn != 30 {
		t.Errorf("expected withdrawn 30, got %f", bal.Withdrawn)
	}
}

func TestListBalanceError(t *testing.T) {
	balRepo := &stubBalanceRepo{errGet: errors.New("db error")}
	withdrawRepo := &stubWithdrawalRepo{}

	svc := NewService(balRepo, withdrawRepo)

	_, err := svc.ListBalance(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWithdrawBalanceSuccess(t *testing.T) {
	balRepo := &stubBalanceRepo{current: 100}
	withdrawRepo := &stubWithdrawalRepo{}

	svc := NewService(balRepo, withdrawRepo)

	req := &balance.WithdrawRequest{Order: "12345", Sum: 50}

	err := svc.WithdrawBalance(context.Background(), 1, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(withdrawRepo.withdrawals) != 1 {
		t.Errorf("expected 1 withdrawal, got %d", len(withdrawRepo.withdrawals))
	}
	if withdrawRepo.withdrawals[0].Amount != 50 {
		t.Errorf("expected withdrawal amount 50, got %f", withdrawRepo.withdrawals[0].Amount)
	}
}

func TestWithdrawBalanceInsufficientFunds(t *testing.T) {
	balRepo := &stubBalanceRepo{current: 30}
	withdrawRepo := &stubWithdrawalRepo{}

	svc := NewService(balRepo, withdrawRepo)

	req := &balance.WithdrawRequest{Order: "12345", Sum: 50}

	err := svc.WithdrawBalance(context.Background(), 1, req)
	if err == nil || err.Error() != "insufficient funds" {
		t.Errorf("expected 'insufficient funds' error, got %v", err)
	}
}

func TestWithdrawBalanceCreateWithdrawalError(t *testing.T) {
	balRepo := &stubBalanceRepo{current: 100}
	withdrawRepo := &stubWithdrawalRepo{errCreate: errors.New("db create error")}

	svc := NewService(balRepo, withdrawRepo)

	req := &balance.WithdrawRequest{Order: "12345", Sum: 50}

	err := svc.WithdrawBalance(context.Background(), 1, req)
	if err == nil || err.Error() != "db create error" {
		t.Errorf("expected db create error, got %v", err)
	}
}

func TestListWithdrawals(t *testing.T) {
	withdrawRepo := &stubWithdrawalRepo{
		withdrawals: []balance.Withdrawal{
			{OrderNumber: "123", Amount: 10},
			{OrderNumber: "456", Amount: 20},
		},
	}
	balRepo := &stubBalanceRepo{}

	svc := NewService(balRepo, withdrawRepo)

	ws, err := svc.ListWithdrawals(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(ws) != 2 {
		t.Errorf("expected 2 withdrawals, got %d", len(ws))
	}
}

func TestListWithdrawalsError(t *testing.T) {
	withdrawRepo := &stubWithdrawalRepo{errList: errors.New("db list error")}
	balRepo := &stubBalanceRepo{}

	svc := NewService(balRepo, withdrawRepo)

	_, err := svc.ListWithdrawals(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func setupHandler() (*Handler, *stubBalanceRepo, *stubWithdrawalRepo) {
	balRepo := &stubBalanceRepo{}
	withdrawRepo := &stubWithdrawalRepo{}
	svc := NewService(balRepo, withdrawRepo)
	return NewHandler(svc), balRepo, withdrawRepo
}

func TestHandlerListBalance(t *testing.T) {
	handler, balRepo, _ := setupHandler()
	userID := int64(1)
	balRepo.current = 100
	balRepo.withdrawn = 20

	req := httptest.NewRequest(http.MethodGet, "/balance", nil)
	req = req.WithContext(middleware.ContextWithUserID(req.Context(), userID))

	w := httptest.NewRecorder()
	handler.GetBalance(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestHandlerWithdrawBalance(t *testing.T) {
	handler, balRepo, withdrawRepo := setupHandler()
	userID := int64(1)
	balRepo.current = 100

	body := `{"order":"12345","sum":50}`
	req := httptest.NewRequest(http.MethodPost, "/withdraw", strings.NewReader(body))
	req = req.WithContext(middleware.ContextWithUserID(req.Context(), userID))

	w := httptest.NewRecorder()
	handler.WithdrawBalance(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if len(withdrawRepo.withdrawals) != 1 {
		t.Errorf("expected 1 withdrawal, got %d", len(withdrawRepo.withdrawals))
	}
}

func TestHandlerWithdrawBalanceInsufficientFunds(t *testing.T) {
	handler, balRepo, _ := setupHandler()
	userID := int64(1)
	balRepo.current = 30

	body := `{"order":"12345","sum":50}`
	req := httptest.NewRequest(http.MethodPost, "/withdraw", strings.NewReader(body))
	req = req.WithContext(middleware.ContextWithUserID(req.Context(), userID))

	w := httptest.NewRecorder()
	handler.WithdrawBalance(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPaymentRequired {
		t.Errorf("expected status 402, got %d", resp.StatusCode)
	}
}

func TestHandlerListWithdrawals(t *testing.T) {
	handler, _, withdrawRepo := setupHandler()
	userID := int64(1)

	withdrawRepo.withdrawals = []balance.Withdrawal{
		{OrderNumber: "123", Amount: 10},
		{OrderNumber: "456", Amount: 20},
	}

	req := httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
	req = req.WithContext(middleware.ContextWithUserID(req.Context(), userID))

	w := httptest.NewRecorder()
	handler.ListWithdrawals(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}
