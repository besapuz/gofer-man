package handler

import (
	"encoding/json"
	"net/http"

	"github.com/besapuz/gofer-man/internal/domain"
	"github.com/besapuz/gofer-man/internal/service"
)

// BalanceHandler обрабатывает HTTP запросы для работы с балансом
type BalanceHandler struct {
	balanceService *service.BalanceService
}

// NewBalanceHandler создает новый экземпляр BalanceHandler
func NewBalanceHandler(balanceService *service.BalanceService) *BalanceHandler {
	return &BalanceHandler{balanceService: balanceService}
}

// GetBalance обрабатывает получение текущего баланса пользователя
// GET /api/user/balance
func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	balance, err := h.balanceService.GetBalance(r.Context(), userID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

// Withdraw обрабатывает запрос на списание баллов
// POST /api/user/balance/withdraw
func (h *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	var req domain.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.balanceService.Withdraw(r.Context(), userID, &req)
	if err != nil {
		switch err.Error() {
		case "invalid order number":
			http.Error(w, "Invalid order number", http.StatusUnprocessableEntity)
		case "insufficient funds":
			http.Error(w, "Insufficient funds", http.StatusPaymentRequired)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetWithdrawals обрабатывает получение истории списаний
// GET /api/user/withdrawals
func (h *BalanceHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	withdrawals, err := h.balanceService.GetWithdrawals(r.Context(), userID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(withdrawals)
}
