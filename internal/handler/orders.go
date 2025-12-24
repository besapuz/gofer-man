package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/besapuz/gofer-man/internal/repository"
	"github.com/besapuz/gofer-man/internal/service"
)

// OrderHandler обрабатывает HTTP запросы для работы с заказами
type OrderHandler struct {
	orderService *service.OrderService
}

// NewOrderHandler создает новый экземпляр OrderHandler
func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

// UploadOrder обрабатывает загрузку номера заказа
// POST /api/user/orders
func (h *OrderHandler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	// Читаем номер заказа
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	orderNumber := strings.TrimSpace(string(body))
	if orderNumber == "" {
		http.Error(w, "Order number required", http.StatusBadRequest)
		return
	}

	// Получаем флаг — был ли заказ создан впервые
	created, err := h.orderService.UploadOrder(r.Context(), userID, orderNumber)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "invalid order number"):
			http.Error(w, "Invalid order number", http.StatusUnprocessableEntity)
		case errors.Is(err, repository.ErrOrderTakenByOther):
			http.Error(w, "Order already uploaded by another user", http.StatusConflict)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Если заказ был только что создан — 202
	if created {
		w.WriteHeader(http.StatusAccepted) // 202
	} else {
		w.WriteHeader(http.StatusOK) // 200
	}
}

// GetOrders обрабатывает получение списка заказов пользователя
// GET /api/user/orders
func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	orders, err := h.orderService.GetUserOrders(r.Context(), userID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}
