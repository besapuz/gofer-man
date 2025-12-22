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

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var accrualResp domain.AccrualResponse
		if err := json.NewDecoder(resp.Body).Decode(&accrualResp); err != nil {
			return nil, err
		}
		return &accrualResp, nil

	case http.StatusNoContent:
		return &domain.AccrualResponse{
			Order:  orderNumber,
			Status: "REGISTERED",
		}, nil

	case http.StatusTooManyRequests:
		// Ждем и повторяем запрос
		time.Sleep(s.retryAfter)
		return s.GetAccrual(ctx, orderNumber)

	default:
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
}
