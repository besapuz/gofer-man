package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/besapuz/gofer-man/internal/domain"
)

// AccrualServiceInterface описывает поведение клиента начислений
type AccrualServiceInterface interface {
	GetAccrual(ctx context.Context, orderNumber string) (*domain.AccrualResponse, error)
}

// AccrualService предоставляет клиент для взаимодействия с внешней системой начислений
type AccrualService struct {
	baseURL    string
	client     *http.Client
	retryAfter time.Duration
}

// NewAccrualService создает новый экземпляр AccrualService
func NewAccrualService(baseURL string) *AccrualService {
	return &AccrualService{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		retryAfter: time.Second * 1,
	}
}

// GetAccrual получает информацию о начислении баллов для заказа из внешней системы
// Возвращает информацию о статусе обработки и начисленных баллах
func (s *AccrualService) GetAccrual(ctx context.Context, orderNumber string) (*domain.AccrualResponse, error) {
	url := fmt.Sprintf("%s/api/orders/%s", s.baseURL, orderNumber)
	maxRetry := 5
	for attempt := 0; attempt < maxRetry; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := s.client.Do(req)
		if err != nil {
			return nil, err
		}

		var result *domain.AccrualResponse
		var shouldRetry bool

		switch resp.StatusCode {
		case http.StatusOK:
			var accrualResp domain.AccrualResponse
			if err := json.NewDecoder(resp.Body).Decode(&accrualResp); err != nil {
				resp.Body.Close()
				return nil, err
			}
			result = &accrualResp
		case http.StatusNoContent:
			result = &domain.AccrualResponse{
				Order:  orderNumber,
				Status: "REGISTERED",
			}
		case http.StatusTooManyRequests:
			shouldRetry = true
		default:
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
		}

		resp.Body.Close()

		// Если результат получен и повтор не нужен — возвращаем
		if !shouldRetry {
			return result, nil
		}

		// Ждём перед повтором
		select {
		case <-time.After(s.retryAfter):
			// продолжаем цикл
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, fmt.Errorf("превышено количество попыток получения данных из системы начислений")
}
