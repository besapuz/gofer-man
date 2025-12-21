package service

import (
	"context"
	"errors"

	"github.com/besapuz/gofer-man/internal/domain"
	"github.com/besapuz/gofer-man/internal/repository"
	"github.com/besapuz/gofer-man/pkg/luhn"
)

// BalanceService предоставляет бизнес-логику для работы с балансом пользователей
type BalanceService struct {
	balanceRepo *repository.BalanceRepository
	orderRepo   *repository.OrderRepository
}

// NewBalanceService создает новый экземпляр BalanceService
func NewBalanceService(balanceRepo *repository.BalanceRepository, orderRepo *repository.OrderRepository) *BalanceService {
	return &BalanceService{
		balanceRepo: balanceRepo,
		orderRepo:   orderRepo,
	}
}

// GetBalance возвращает текущий баланс пользователя
// Включает текущий баланс и сумму использованных баллов
func (s *BalanceService) GetBalance(ctx context.Context, userID int) (*domain.Balance, error) {
	return s.balanceRepo.GetBalance(ctx, userID)
}

// Withdraw выполняет списание баллов с баланса пользователя
// Проверяет корректность номера заказа и достаточность средств
func (s *BalanceService) Withdraw(ctx context.Context, userID int, req *domain.WithdrawRequest) error {
	// Валидируем номер заказа
	if !luhn.Valid(req.Order) {
		return errors.New("invalid order number")
	}

	// Валидируем сумму
	if req.Sum <= 0 {
		return errors.New("sum must be positive")
	}

	// Проверяем, существует ли уже заказ для списания
	// (В реальной системе мы бы проверили таблицу заказов)

	// Выполняем списание
	err := s.balanceRepo.Withdraw(ctx, userID, req.Order, req.Sum)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientFunds) {
			return errors.New("insufficient funds")
		}
		return err
	}

	return nil
}

// GetWithdrawals возвращает историю списаний пользователя
// Списания сортируются по времени (от новых к старым)
func (s *BalanceService) GetWithdrawals(ctx context.Context, userID int) ([]*domain.Withdrawal, error) {
	return s.balanceRepo.GetWithdrawals(ctx, userID)
}

// AddAccrual добавляет начисление баллов к балансу пользователя
func (s *BalanceService) AddAccrual(ctx context.Context, userID int, amount float64) error {
	return s.balanceRepo.AddAccrual(ctx, userID, amount)
}
