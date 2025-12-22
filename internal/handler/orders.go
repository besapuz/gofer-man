package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

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

	orderNumber := string(body)
	if orderNumber == "" {
		http.Error(w, "Order number required", http.StatusBadRequest)
		return
	}

	err = h.orderService.UploadOrder(r.Context(), userID, orderNumber)
	if err != nil {
		// Проверяем тип ошибки по тексту
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid order number") {
			http.Error(w, "Invalid order number", http.StatusUnprocessableEntity)
			return
		}

		if strings.Contains(errMsg, "order taken by another user") {
			http.Error(w, "Order already uploaded by another user", http.StatusConflict)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}

	// Если ошибки нет - заказ успешно создан
	w.WriteHeader(http.StatusAccepted)
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
