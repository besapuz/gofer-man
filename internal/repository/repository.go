package repository

import (
	"context"
	"errors"

	"github.com/besapuz/gofer-man/internal/domain"
)

// Общие ошибки уровня репозитория
var (
	// ErrUserExists указывает, что пользователь с данным логином уже существует
	ErrUserExists = errors.New("user already exists")

	// ErrUserNotFound указывает, что пользователь с указанными данными не найден
	ErrUserNotFound = errors.New("user not found")

	// ErrOrderExists указывает, что заказ с данным номером уже существует
	ErrOrderExists = errors.New("order already exists")

	// ErrOrderNotFound указывает, что заказ не найден по заданным критериям
	ErrOrderNotFound = errors.New("order not found")

	// ErrInsufficientFunds указывает, что у пользователя недостаточно средств для списания
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// OrderRepositoryInterface описывает поведение репозитория заказов
type OrderRepositoryInterface interface {
	CreateOrder(ctx context.Context, order *domain.Order) error
	GetOrderByNumber(ctx context.Context, number string) (*domain.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int) ([]*domain.Order, error)
	GetUnprocessedOrders(ctx context.Context, limit int) ([]*domain.Order, error)
	UpdateOrderAccrual(ctx context.Context, id int, status string, accrual float64) error
}

// BalanceRepositoryInterface описывает поведение репозитория баланса
type BalanceRepositoryInterface interface {
	AddAccrual(ctx context.Context, userID int, accrual float64) error
	GetBalance(ctx context.Context, userID int) (*domain.Balance, error)
	Withdraw(ctx context.Context, userID int, orderNumber string, amount float64) error
	GetWithdrawals(ctx context.Context, userID int) ([]*domain.Withdrawal, error)
}
