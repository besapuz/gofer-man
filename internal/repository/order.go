package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/besapuz/gofer-man/internal/domain"
	"github.com/lib/pq"
)

const (
	pgUniqueViolationCode = "23505"
)

// OrderRepository предоставляет методы доступа к данным для сущностей заказов
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository создает новый экземпляр OrderRepository
func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `INSERT INTO orders (user_id, number, status) VALUES ($1, $2, $3) 
              RETURNING id, uploaded_at`
	err = tx.QueryRowContext(ctx, query, order.UserID, order.Number, order.Status).
		Scan(&order.ID, &order.UploadedAt)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pgUniqueViolationCode {
			// ВНЕШНИЙ SELECT — не в транзакции!
			var existingUserID int
			err := r.db.QueryRowContext(ctx, "SELECT user_id FROM orders WHERE number = $1", order.Number).
				Scan(&existingUserID)
			if err != nil {
				if err == sql.ErrNoRows {
					return fmt.Errorf("order number exists but not found in DB — possible inconsistency")
				}
				return fmt.Errorf("failed to check existing order: %w", err)
			}
			if existingUserID == order.UserID {
				return ErrOrderExists
			}
			return ErrOrderTakenByOther
		}
		return fmt.Errorf("insert order: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetOrdersByUserID возвращает список заказов для указанного пользователя
func (r *OrderRepository) GetOrdersByUserID(ctx context.Context, userID int) ([]*domain.Order, error) {
	query := `SELECT id, number, status, accrual, uploaded_at, processed_at 
              FROM orders WHERE user_id = $1 
              ORDER BY uploaded_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		order := &domain.Order{}
		var processedAt sql.NullTime
		err := rows.Scan(
			&order.ID,
			&order.Number,
			&order.Status,
			&order.Accrual,
			&order.UploadedAt,
			&processedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		if processedAt.Valid {
			order.ProcessedAt = &processedAt.Time
		}
		orders = append(orders, order)
	}
	// Проверяем, не произошла ли ошибка при итерации
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return orders, nil
}

// GetUnprocessedOrders возвращает список необработанных заказов
func (r *OrderRepository) GetUnprocessedOrders(ctx context.Context, limit int) ([]*domain.Order, error) {
	query := `SELECT id, user_id, number, status FROM orders 
              WHERE status IN ('NEW', 'PROCESSING') 
              ORDER BY uploaded_at ASC LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query unprocessed orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		order := &domain.Order{}
		err := rows.Scan(&order.ID, &order.UserID, &order.Number, &order.Status)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, order)
	}
	// Проверяем ошибку итерации
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return orders, nil
}

// UpdateOrderAccrual обновляет информацию о начислении баллов для заказа
func (r *OrderRepository) UpdateOrderAccrual(ctx context.Context, orderID int, status string, accrual float64) error {
	query := `UPDATE orders SET status = $1, accrual = $2, processed_at = $3 
              WHERE id = $4`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, status, accrual, now, orderID)
	if err != nil {
		return fmt.Errorf("update order accrual: %w", err)
	}
	return nil
}

// GetOrderByNumber возвращает заказ по номеру
func (r *OrderRepository) GetOrderByNumber(ctx context.Context, number string) (*domain.Order, error) {
	var order domain.Order
	var processedAt sql.NullTime

	query := `SELECT id, user_id, number, status, accrual, uploaded_at, processed_at 
              FROM orders WHERE number = $1`

	err := r.db.QueryRowContext(ctx, query, number).Scan(
		&order.ID,
		&order.UserID,
		&order.Number,
		&order.Status,
		&order.Accrual,
		&order.UploadedAt,
		&processedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get order by number: %w", err)
	}

	if processedAt.Valid {
		order.ProcessedAt = &processedAt.Time
	}

	return &order, nil
}
