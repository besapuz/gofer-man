package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/besapuz/gofer-man/internal/domain"
	"github.com/besapuz/gofer-man/internal/mocks"
	"github.com/besapuz/gofer-man/internal/repository"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOrderService_UploadOrder(t *testing.T) {
	ctx := context.Background()

	mockOrderRepo := new(mocks.OrderRepositoryInterface)
	mockBalanceRepo := new(mocks.BalanceRepositoryInterface)
	mockAccrual := new(mocks.AccrualServiceInterface)

	service := NewOrderService(mockOrderRepo, mockBalanceRepo, mockAccrual)

	t.Run("invalid Luhn checksum", func(t *testing.T) {
		created, err := service.UploadOrder(ctx, 1, "12345678902")
		require.False(t, created)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid order number")
	})

	t.Run("non-numeric order number", func(t *testing.T) {
		created, err := service.UploadOrder(ctx, 1, "abc123")
		require.False(t, created)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid order number")
	})

	t.Run("valid order - created successfully", func(t *testing.T) {
		mockOrderRepo.On("CreateOrder", ctx, mock.AnythingOfType("*domain.Order")).Run(func(args mock.Arguments) {
			order := args.Get(1).(*domain.Order)
			require.Equal(t, 1, order.UserID)
			require.Equal(t, "4561261212345467", order.Number)
			require.Equal(t, "NEW", order.Status)
		}).Return(nil).Once()

		created, err := service.UploadOrder(ctx, 1, "4561261212345467")
		require.NoError(t, err)
		require.True(t, created) // ← тут теперь должно быть true

		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("order already exists - same user", func(t *testing.T) {
		mockOrderRepo.On("CreateOrder", ctx, mock.Anything).Return(repository.ErrOrderExists).Once()
		mockOrderRepo.On("GetOrderByNumber", ctx, "4561261212345467").Return(&domain.Order{
			ID:     1,
			UserID: 1,
			Number: "4561261212345467",
		}, nil).Once()

		created, err := service.UploadOrder(ctx, 1, "4561261212345467")
		require.False(t, created)
		require.NoError(t, err)

		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("order already exists - different user", func(t *testing.T) {
		mockOrderRepo.On("CreateOrder", ctx, mock.Anything).Return(repository.ErrOrderExists).Once()
		mockOrderRepo.On("GetOrderByNumber", ctx, "4561261212345467").Return(&domain.Order{
			ID:     1,
			UserID: 2,
			Number: "4561261212345467",
		}, nil).Once()

		created, err := service.UploadOrder(ctx, 1, "4561261212345467")
		require.False(t, created)
		require.Error(t, err)
		require.Contains(t, err.Error(), "order taken by another user")

		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("error on GetOrderByNumber after ErrOrderExists", func(t *testing.T) {
		mockOrderRepo.On("CreateOrder", ctx, mock.Anything).Return(repository.ErrOrderExists).Once()
		mockOrderRepo.On("GetOrderByNumber", ctx, "4561261212345467").Return((*domain.Order)(nil), errors.New("db error")).Once()

		created, err := service.UploadOrder(ctx, 1, "4561261212345467")
		require.False(t, created)
		require.Error(t, err)
		require.Contains(t, err.Error(), "get order by number")

		mockOrderRepo.AssertExpectations(t)
	})
}

func TestOrderService_GetUserOrders(t *testing.T) {
	ctx := context.Background()

	mockOrderRepo := new(mocks.OrderRepositoryInterface)
	mockBalanceRepo := new(mocks.BalanceRepositoryInterface)
	mockAccrual := new(mocks.AccrualServiceInterface)

	service := NewOrderService(mockOrderRepo, mockBalanceRepo, mockAccrual)

	t.Run("returns user orders successfully", func(t *testing.T) {
		expectedOrders := []*domain.Order{
			{ID: 1, UserID: 1, Number: "1234567890", Status: "PROCESSED"},
		}

		mockOrderRepo.On("GetOrdersByUserID", ctx, 1).Return(expectedOrders, nil).Once()

		orders, err := service.GetUserOrders(ctx, 1)
		require.NoError(t, err)
		require.Equal(t, expectedOrders, orders)

		mockOrderRepo.AssertExpectations(t)
	})
}

func TestOrderService_ProcessOrders(t *testing.T) {
	ctx := context.Background()

	mockOrderRepo := new(mocks.OrderRepositoryInterface)
	mockBalanceRepo := new(mocks.BalanceRepositoryInterface)
	mockAccrual := new(mocks.AccrualServiceInterface)

	service := NewOrderService(mockOrderRepo, mockBalanceRepo, mockAccrual)

	t.Run("processes unprocessed orders", func(t *testing.T) {
		unprocessedOrders := []*domain.Order{
			{ID: 1, UserID: 1, Number: "4561261212345467", Status: "NEW"},
		}

		// Получаем заказы
		mockOrderRepo.On("GetUnprocessedOrders", ctx, 100).Return(unprocessedOrders, nil).Once()

		// Обновляем статус на PROCESSING
		mockOrderRepo.On("UpdateOrderAccrual", ctx, 1, "PROCESSING", 0.0).Return(nil).Once()

		// Запрос к внешнему сервису
		mockAccrual.On("GetAccrual", ctx, "4561261212345467").Return(&domain.AccrualResponse{
			Order:   "4561261212345467",
			Status:  "PROCESSED",
			Accrual: 500.0,
		}, nil).Once()

		// Начисление баланса
		mockBalanceRepo.On("AddAccrual", ctx, 1, 500.0).Return(nil).Once()

		// Финальное обновление заказа
		mockOrderRepo.On("UpdateOrderAccrual", ctx, 1, "PROCESSED", 500.0).Return(nil).Once()

		err := service.ProcessOrders(ctx)
		require.NoError(t, err)

		mockOrderRepo.AssertExpectations(t)
		mockAccrual.AssertExpectations(t)
		mockBalanceRepo.AssertExpectations(t)
	})

	t.Run("accrual service returns error - continue processing", func(t *testing.T) {
		orders := []*domain.Order{
			{ID: 1, Number: "4561261212345467", Status: "NEW", UserID: 1},
		}

		mockOrderRepo.On("GetUnprocessedOrders", ctx, 100).Return(orders, nil).Once()
		mockOrderRepo.On("UpdateOrderAccrual", ctx, 1, "PROCESSING", 0.0).Return(nil).Once()
		mockAccrual.On("GetAccrual", ctx, "4561261212345467").Return((*domain.AccrualResponse)(nil), errors.New("timeout")).Once()

		// Должно продолжить без паники
		err := service.ProcessOrders(ctx)
		require.NoError(t, err) // Ошибка логируется, но не возвращается
	})

	t.Run("accrual service returns invalid status", func(t *testing.T) {
		orders := []*domain.Order{
			{ID: 1, Number: "4561261212345467", Status: "NEW", UserID: 1},
		}

		mockOrderRepo.On("GetUnprocessedOrders", ctx, 100).Return(orders, nil).Once()
		mockOrderRepo.On("UpdateOrderAccrual", ctx, 1, "PROCESSING", 0.0).Return(nil).Once()
		mockAccrual.On("GetAccrual", ctx, "4561261212345467").Return(&domain.AccrualResponse{
			Order:   "4561261212345467",
			Status:  "INVALID",
			Accrual: 0,
		}, nil).Once()
		mockOrderRepo.On("UpdateOrderAccrual", ctx, 1, "INVALID", 0.0).Return(nil).Once()

		err := service.ProcessOrders(ctx)
		require.NoError(t, err)

		mockOrderRepo.AssertExpectations(t)
		mockAccrual.AssertExpectations(t)
	})

	t.Run("error updating order accrual - continue", func(t *testing.T) {
		orders := []*domain.Order{
			{ID: 1, Number: "4561261212345467", Status: "NEW", UserID: 1},
		}

		mockOrderRepo.On("GetUnprocessedOrders", ctx, 100).Return(orders, nil).Once()
		mockOrderRepo.On("UpdateOrderAccrual", ctx, 1, "PROCESSING", 0.0).Return(nil).Once()
		mockAccrual.On("GetAccrual", ctx, "4561261212345467").Return(&domain.AccrualResponse{
			Order:   "4561261212345467",
			Status:  "PROCESSED",
			Accrual: 100,
		}, nil).Once()
		mockBalanceRepo.On("AddAccrual", ctx, 1, 100.0).Return(errors.New("balance update failed")).Once() // ошибка, но продолжаем
		mockOrderRepo.On("UpdateOrderAccrual", ctx, 1, "PROCESSED", 100.0).Return(errors.New("update failed")).Once()

		// Должно продолжить без остановки
		err := service.ProcessOrders(ctx)
		require.NoError(t, err)
	})

	t.Run("order already in PROCESSING - skip to accrual check", func(t *testing.T) {
		orders := []*domain.Order{
			{ID: 1, Number: "4561261212345467", Status: "PROCESSING", UserID: 1},
		}

		mockOrderRepo.On("GetUnprocessedOrders", ctx, 100).Return(orders, nil).Once()
		mockAccrual.On("GetAccrual", ctx, "4561261212345467").Return(&domain.AccrualResponse{
			Order:   "4561261212345467",
			Status:  "PROCESSED",
			Accrual: 200,
		}, nil).Once()
		mockBalanceRepo.On("AddAccrual", ctx, 1, 200.0).Return(nil).Once()
		mockOrderRepo.On("UpdateOrderAccrual", ctx, 1, "PROCESSED", 200.0).Return(nil).Once()

		err := service.ProcessOrders(ctx)
		require.NoError(t, err)
	})
}

func TestOrderService_GetOrderByNumber(t *testing.T) {
	ctx := context.Background()

	mockOrderRepo := new(mocks.OrderRepositoryInterface)
	mockBalanceRepo := new(mocks.BalanceRepositoryInterface)
	mockAccrual := new(mocks.AccrualServiceInterface)

	service := NewOrderService(mockOrderRepo, mockBalanceRepo, mockAccrual)

	t.Run("returns order by number", func(t *testing.T) {
		expectedOrder := &domain.Order{ID: 1, Number: "4561261212345467", UserID: 1}
		mockOrderRepo.On("GetOrderByNumber", ctx, "4561261212345467").Return(expectedOrder, nil).Once()

		order, err := service.GetOrderByNumber(ctx, "4561261212345467")
		require.NoError(t, err)
		require.Equal(t, expectedOrder, order)
	})

	t.Run("error from repository", func(t *testing.T) {
		mockOrderRepo.On("GetOrderByNumber", ctx, "4561261212345467").Return((*domain.Order)(nil), errors.New("db error")).Once()

		order, err := service.GetOrderByNumber(ctx, "4561261212345467")
		require.Error(t, err)
		require.Nil(t, order)
	})
}

// Тест для mapAccrualStatusToInternal — граничные случаи
func Test_mapAccrualStatusToInternal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"REGISTERED", "PROCESSING"},
		{"PROCESSING", "PROCESSING"},
		{"PROCESSED", "PROCESSED"},
		{"INVALID", "INVALID"},
		{"UNKNOWN", "PROCESSING"}, // default
		{"", "PROCESSING"},        // пустая строка
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%s", tt.input), func(t *testing.T) {
			result := mapAccrualStatusToInternal(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
