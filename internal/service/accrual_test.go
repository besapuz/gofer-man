package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/besapuz/gofer-man/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccrualService_GetAccrual(t *testing.T) {
	tests := []struct {
		name           string
		orderNumber    string
		statusCode     int
		responseBody   string
		expectedResp   *domain.AccrualResponse
		expectedError  string
		useContext     context.Context
		contextTimeout time.Duration
		expectRetry    bool
	}{
		{
			name:         "200 OK - valid accrual response",
			orderNumber:  "4561261212345467",
			statusCode:   http.StatusOK,
			responseBody: `{"order":"4561261212345467","status":"PROCESSED","accrual":500.5}`,
			expectedResp: &domain.AccrualResponse{
				Order:   "4561261212345467",
				Status:  "PROCESSED",
				Accrual: 500.5,
			},
		},
		{
			name:         "204 No Content - order registered, no data yet",
			orderNumber:  "4561261212345467",
			statusCode:   http.StatusNoContent,
			responseBody: "",
			expectedResp: &domain.AccrualResponse{
				Order:  "4561261212345467",
				Status: "REGISTERED",
			},
		},
		{
			name:         "429 Too Many Requests - retry once and succeed",
			orderNumber:  "4561261212345467",
			statusCode:   http.StatusOK,
			responseBody: `{"order":"4561261212345467","status":"PROCESSED","accrual":100}`,
			expectedResp: &domain.AccrualResponse{
				Order:   "4561261212345467",
				Status:  "PROCESSED",
				Accrual: 100,
			},
			expectRetry: true,
		},
		{
			name:          "500 Internal Server Error - unexpected status",
			orderNumber:   "4561261212345467",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `{"error":"internal error"}`,
			expectedError: "unexpected status: 500",
		},
		{
			name:          "404 Not Found - order unknown",
			orderNumber:   "999999",
			statusCode:    http.StatusNotFound,
			responseBody:  `{"error":"order not found"}`,
			expectedError: "unexpected status: 404",
		},
		{
			name:          "invalid JSON response - wrong type",
			orderNumber:   "4561261212345467",
			statusCode:    http.StatusOK,
			responseBody:  `{"order":"4561261212345467", "accrual": "not-a-number"}`,
			expectedError: "cannot unmarshal",
		},
		{
			name:          "empty order number - server returns 400 or 404",
			orderNumber:   "",
			statusCode:    http.StatusBadRequest,
			responseBody:  `{"error": "invalid order number"}`,
			expectedError: "unexpected status: 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			var lastReq *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				lastReq = r

				if r.Method != "GET" {
					t.Errorf("expected GET, got %s", r.Method)
				}
				if !strings.HasPrefix(r.URL.Path, "/api/orders/") {
					t.Errorf("expected path starting with /api/orders/, got %s", r.URL.Path)
				}

				// Для 429: первый вызов — 429, второй — успешный
				if tt.expectRetry && callCount == 1 {
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}

				w.WriteHeader(tt.statusCode)
				if tt.responseBody != "" {
					_, _ = w.Write([]byte(tt.responseBody))
				}
			}))
			defer server.Close()

			client := NewAccrualService(server.URL)

			// Контекст
			ctx := tt.useContext
			if ctx == nil {
				var cancel context.CancelFunc
				timeout := time.Second * 5
				if tt.contextTimeout > 0 {
					timeout = tt.contextTimeout
				}
				ctx, cancel = context.WithTimeout(context.Background(), timeout)
				defer cancel()
			}

			resp, err := client.GetAccrual(ctx, tt.orderNumber)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.expectedResp.Order, resp.Order)
			assert.Equal(t, tt.expectedResp.Status, resp.Status)
			if tt.expectedResp.Accrual > 0 {
				assert.InDelta(t, tt.expectedResp.Accrual, resp.Accrual, 0.001)
			}

			// Проверка количества вызовов
			expectedCalls := 1
			if tt.expectRetry {
				expectedCalls = 2
			}
			assert.Equal(t, expectedCalls, callCount)

			// Контекст в запросе — проверяем, что он не nil и был передан
			require.NotNil(t, lastReq)
			assert.NotNil(t, lastReq.Context(), "request must have context")
		})
	}
}

// Дополнительный тест: проверка, что таймаут клиента срабатывает
func TestAccrualService_ClientTimeout(t *testing.T) {
	// Сервер, который не отвечает сразу
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // дольше, чем Timeout клиента (10 сек? нет — у нас 10 сек, но меняем)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(domain.AccrualResponse{
			Order:   "4561261212345467",
			Status:  "PROCESSING",
			Accrual: 0,
		})
	}))
	defer server.Close()

	// Клиент с малым таймаутом
	client := &AccrualService{
		baseURL: server.URL,
		client: &http.Client{
			Timeout: 100 * time.Millisecond, // очень короткий таймаут
		},
		retryAfter: time.Millisecond * 10,
	}

	ctx := context.Background()
	_, err := client.GetAccrual(ctx, "4561261212345467")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Client.Timeout")
}

// Дополнительный тест: проверка, что некорректный URL вызывает ошибку
func TestAccrualService_InvalidURL(t *testing.T) {
	client := NewAccrualService("http://localhost:8081") // недоступный хост

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.GetAccrual(ctx, "4561261212345467")

	require.Error(t, err)
	// Ошибка может быть разной (connection refused и т.п.), но должна быть
	assert.Contains(t, err.Error(), "connection refused")
}
