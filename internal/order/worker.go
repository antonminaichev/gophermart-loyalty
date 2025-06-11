package order

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type AccrualResponse struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual"`
}

type AccrualClient interface {
	GetOrder(ctx context.Context, number string) (*AccrualResponse, error)
}

type Updater interface {
	UpdateFromAccrual(ctx context.Context, number, status string, accrual *float64) error
}

type HTTPAccrualClient struct {
	Client         *http.Client
	AccrualAddress string
}

func (c *HTTPAccrualClient) GetOrder(ctx context.Context, number string) (*AccrualResponse, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.AccrualAddress, number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("too many requests (429) for order %s", number)
	case http.StatusNotFound:
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var ar AccrualResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("decode body: %w", err)
	}
	return &ar, nil
}

func workerLoop(
	ctx context.Context,
	id int,
	accrualClient AccrualClient,
	jobs <-chan string,
	svc Updater,
) {
	log.Printf("[Worker %d] Запущен", id)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[Worker %d] Завершение по сигналу контекста", id)
			return

		case number, ok := <-jobs:
			if !ok {
				log.Printf("[Worker %d] jobs-канал закрыт — выхожу", id)
				return
			}

			ar, err := accrualClient.GetOrder(ctx, number)
			if err != nil {
				log.Printf("[Worker %d] Ошибка запроса для %s: %v", id, number, err)
				continue
			}
			if ar == nil || ar.Order == "" {
				log.Printf("[Worker %d] accrual не вернул данные по заказу %s (ar: %#v)", id, number, ar)
				continue
			}

			if err := svc.UpdateFromAccrual(ctx, ar.Order, ar.Status, ar.Accrual); err != nil {
				log.Printf("[Worker %d] Ошибка обновления заказа %s: %v", id, ar.Order, err)
				continue
			}
			log.Printf("[Worker %d] Заказ %s: status=%s accrual=%v", id, ar.Order, ar.Status, ar.Accrual)
		}
	}
}

func DispatcherLoop(
	ctx context.Context,
	accrualClient AccrualClient,
	svc *Service,
	workerCount int,
	interval time.Duration,
) {
	jobs := make(chan string, workerCount*3)

	for i := 1; i <= workerCount; i++ {
		go workerLoop(ctx, i, accrualClient, jobs, svc)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Println("[Dispatcher] Стартовал DispatcherLoop")
	for {
		select {
		case <-ctx.Done():
			log.Println("[Dispatcher] Останов по сигналу контекста, закрываю jobs")
			close(jobs)
			return
		case <-ticker.C:
			orders, err := svc.ListForPolling(ctx)
			if err != nil {
				log.Printf("[Dispatcher] Ошибка ListForPolling: %v", err)
				continue
			}
			if len(orders) == 0 {
				log.Println("[Dispatcher] Нет заказов для опроса")
				continue
			}
			log.Printf("[Dispatcher] Найдено %d заказов для опроса", len(orders))
			for _, o := range orders {
				select {
				case jobs <- o.Number:
				default:
					log.Printf("[Dispatcher] jobs-канал переполнен, пропускаю заказ %s на этот цикл", o.Number)
				}
			}
		}
	}
}
