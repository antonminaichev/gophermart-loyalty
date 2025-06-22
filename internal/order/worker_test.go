package order

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/antonminaichev/gophermart-loyalty/internal/types/order"
	"github.com/stretchr/testify/assert"
)

type mockAccrualClient struct {
	mu       sync.Mutex
	requests []string
	respMap  map[string]*AccrualResponse
	errMap   map[string]error
}

func (m *mockAccrualClient) GetOrder(ctx context.Context, number string) (*AccrualResponse, error) {
	m.mu.Lock()
	m.requests = append(m.requests, number)
	m.mu.Unlock()
	if err, ok := m.errMap[number]; ok {
		return nil, err
	}
	if resp, ok := m.respMap[number]; ok {
		return resp, nil
	}
	return nil, nil
}

func newMockAccrualClient() *mockAccrualClient {
	return &mockAccrualClient{
		respMap: make(map[string]*AccrualResponse),
		errMap:  make(map[string]error),
	}
}

type mockService struct {
	updatedOrders []string
	updateErr     error
	lastStatus    string
	lastAccrual   *float64
	mu            sync.Mutex
}

func (m *mockService) UpdateFromAccrual(ctx context.Context, number, status string, accrual *float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updatedOrders = append(m.updatedOrders, number)
	m.lastStatus = status
	m.lastAccrual = accrual
	return m.updateErr
}

func (m *mockService) ListForPolling(ctx context.Context) ([]order.Order, error) {
	return []order.Order{}, nil
}

// -------- Тесты --------

func TestWorkerLoop_Success(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mAccrual := newMockAccrualClient()
	accrualVal := 100.0
	mAccrual.respMap["12345"] = &AccrualResponse{
		Order:   "12345",
		Status:  "PROCESSED",
		Accrual: &accrualVal,
	}
	jobs := make(chan string, 1)
	jobs <- "12345"
	close(jobs)

	mSvc := &mockService{}

	workerLoop(ctx, 1, mAccrual, jobs, mSvc)

	assert.Equal(t, []string{"12345"}, mSvc.updatedOrders)
	assert.Equal(t, "PROCESSED", mSvc.lastStatus)
	assert.NotNil(t, mSvc.lastAccrual)
	assert.Equal(t, 100.0, *mSvc.lastAccrual)
}

func TestWorkerLoop_ErrorFromClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mAccrual := newMockAccrualClient()
	mAccrual.errMap["54321"] = errors.New("connection error")

	jobs := make(chan string, 1)
	jobs <- "54321"
	close(jobs)

	mSvc := &mockService{}

	workerLoop(ctx, 2, mAccrual, jobs, mSvc)

	assert.Empty(t, mSvc.updatedOrders)
}

func TestWorkerLoop_EmptyResponse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mAccrual := newMockAccrualClient()
	mAccrual.respMap["99999"] = nil

	jobs := make(chan string, 1)
	jobs <- "99999"
	close(jobs)

	mSvc := &mockService{}

	workerLoop(ctx, 3, mAccrual, jobs, mSvc)

	assert.Empty(t, mSvc.updatedOrders)
}

func TestWorkerLoop_ResponseWithoutOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mAccrual := newMockAccrualClient()
	mAccrual.respMap["88888"] = &AccrualResponse{
		Order:   "",
		Status:  "PROCESSING",
		Accrual: nil,
	}

	jobs := make(chan string, 1)
	jobs <- "88888"
	close(jobs)

	mSvc := &mockService{}

	workerLoop(ctx, 4, mAccrual, jobs, mSvc)

	assert.Empty(t, mSvc.updatedOrders)
}

func TestWorkerLoop_UpdateError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mAccrual := newMockAccrualClient()
	mAccrual.respMap["22222"] = &AccrualResponse{
		Order:   "22222",
		Status:  "INVALID",
		Accrual: nil,
	}
	jobs := make(chan string, 1)
	jobs <- "22222"
	close(jobs)

	mSvc := &mockService{updateErr: errors.New("db error")}

	workerLoop(ctx, 5, mAccrual, jobs, mSvc)

	assert.Equal(t, []string{"22222"}, mSvc.updatedOrders)
}
