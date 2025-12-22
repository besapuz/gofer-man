package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/besapuz/gofer-man/internal/domain"
	"github.com/besapuz/gofer-man/internal/repository"
	"github.com/besapuz/gofer-man/pkg/luhn"
)

// OrderService предоставляет бизнес-логику для работы с заказами
type OrderService struct {
	orderRepo   *repository.OrderRepository
	balanceRepo *repository.BalanceRepository
	accrual     *AccrualService
}

// NewOrderService создает новый экземпляр OrderService
func NewOrderService(orderRepo *repository.OrderRepository, balanceRepo *repository.BalanceRepository, accrual *AccrualService) *OrderService {
	return &OrderService{
		orderRepo:   orderRepo,
		balanceRepo: balanceRepo,
		accrual:     accrual,
	}
}

// UploadOrder загружает номер заказа от пользователя
// Проверяет номер с помощью алгоритма Луна и создает запись заказа
// Возвращает статус обработки заказа и возможную ошибку
func (s *OrderService) UploadOrder(ctx context.Context, userID int, orderNumber string) error {
	// Валидируем номер заказа
	if !luhn.Valid(orderNumber) {
		return fmt.Errorf("invalid order number")
	}

	// Проверяем, что номер заказа состоит только из цифр
	if _, err := strconv.ParseInt(orderNumber, 10, 64); err != nil {
		return fmt.Errorf("invalid order number")
	}

	// Создаем заказ
	order := &domain.Order{
		UserID: userID,
		Number: orderNumber,
		Status: "NEW",
	}

	err := s.orderRepo.CreateOrder(ctx, order)
	if err != nil {
		// Обрабатываем ошибку существования заказа
		if errors.Is(err, repository.ErrOrderExists) {
			// Проверяем, чей это заказ
			existingOrder, err := s.orderRepo.GetOrderByNumber(ctx, orderNumber)
			if err != nil {
				return fmt.Errorf("get order by number: %w", err)
			}

			if existingOrder != nil && existingOrder.UserID == userID {
				// Заказ уже загружен этим пользователем
				return nil
			} else {
				// Заказ загружен другим пользователем
				return fmt.Errorf("order taken by another user")
			}
		}
		return err
	}

	return nil
}

// GetUserOrders возвращает список заказов пользователя
// Заказы сортируются по времени загрузки (от новых к старым)
// Возвращает список заказов и возможную ошибку
func (s *OrderService) GetUserOrders(ctx context.Context, userID int) ([]*domain.Order, error) {
	orders, err := s.orderRepo.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user orders: %w", err)
	}

	return orders, nil
}

func (s *OrderService) ProcessOrders(ctx context.Context) error {
	// Получаем необработанные заказы (ограничиваем количество для одного цикла)
	orders, err := s.orderRepo.GetUnprocessedOrders(ctx, 100)
	if err != nil {
		return fmt.Errorf("get unprocessed orders: %w", err)
	}

	// Обрабатываем каждый заказ
	for _, order := range orders {
		// Обновляем статус на PROCESSING если был NEW
		if order.Status == "NEW" {
			err = s.orderRepo.UpdateOrderAccrual(ctx, order.ID, "PROCESSING", 0)
			if err != nil {
				// Логируем ошибку и продолжаем обработку других заказов
				fmt.Printf("Error updating order status to PROCESSING: %v\n", err)
				continue
			}
		}

		// Получаем информацию о начислении из внешней системы
		accrualResp, err := s.accrual.GetAccrual(ctx, order.Number)
		if err != nil {
			// Если превышен лимит запросов или временная ошибка, оставляем как PROCESSING
			fmt.Printf("Error getting accrual for order %s: %v\n", order.Number, err)
			continue
		}

		// Маппим статусы внешней системы на внутренние статусы
		internalStatus := mapAccrualStatusToInternal(accrualResp.Status)

		// Если заказ обработан и есть начисление, обновляем баланс
		if internalStatus == "PROCESSED" && accrualResp.Accrual > 0 {
			err = s.balanceRepo.AddAccrual(ctx, order.UserID, accrualResp.Accrual)
			if err != nil {
				fmt.Printf("Error adding accrual to balance for user %d: %v\n", order.UserID, err)
				// Продолжаем обновлять статус заказа даже если начисление не удалось
			}
		}

		// Обновляем статус заказа и начисление
		err = s.orderRepo.UpdateOrderAccrual(ctx, order.ID, internalStatus, accrualResp.Accrual)
		if err != nil {
			fmt.Printf("Error updating order accrual for order %d: %v\n", order.ID, err)
			continue
		}
	}

	return nil
}

// mapAccrualStatusToInternal преобразует статусы внешней системы начисления во внутренние статусы
func mapAccrualStatusToInternal(accrualStatus string) string {
	switch accrualStatus {
	case "REGISTERED":
		return "PROCESSING"
	case "INVALID":
		return "INVALID"
	case "PROCESSING":
		return "PROCESSING"
	case "PROCESSED":
		return "PROCESSED"
	default:
		return "PROCESSING" // По умолчанию считаем что в обработке
	}
}

// GetOrderByNumber возвращает заказ по номеру
func (s *OrderService) GetOrderByNumber(ctx context.Context, orderNumber string) (*domain.Order, error) {
	return s.orderRepo.GetOrderByNumber(ctx, orderNumber)
}
