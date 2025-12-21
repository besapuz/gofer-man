package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/besapuz/gofer-man/internal/domain"
)

// BalanceRepository предоставляет методы доступа к данным для сущностей баланса
type BalanceRepository struct {
	db *sql.DB
}

// NewBalanceRepository создает новый экземпляр BalanceRepository
func NewBalanceRepository(db *sql.DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

// GetBalance возвращает текущий баланс пользователя
func (r *BalanceRepository) GetBalance(ctx context.Context, userID int) (*domain.Balance, error) {
	query := `SELECT current, withdrawn FROM balance WHERE user_id = $1`
	balance := &domain.Balance{UserID: userID}

	err := r.db.QueryRowContext(ctx, query, userID).Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		if err == sql.ErrNoRows {
			// Создаем нулевой баланс, если он не существует
			return r.createBalance(ctx, userID)
		}
		return nil, fmt.Errorf("get balance: %w", err)
	}

	return balance, nil
}

// createBalance создает запись баланса для нового пользователя
func (r *BalanceRepository) createBalance(ctx context.Context, userID int) (*domain.Balance, error) {
	query := `INSERT INTO balance (user_id, current, withdrawn) VALUES ($1, 0, 0) 
              RETURNING current, withdrawn`

	balance := &domain.Balance{UserID: userID}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		return nil, fmt.Errorf("create balance: %w", err)
	}

	return balance, nil
}

// AddAccrual добавляет начисление баллов к балансу пользователя
func (r *BalanceRepository) AddAccrual(ctx context.Context, userID int, amount float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Обновляем или создаем запись баланса
	query := `INSERT INTO balance (user_id, current, withdrawn) 
              VALUES ($1, $2, 0) 
              ON CONFLICT (user_id) 
              DO UPDATE SET current = balance.current + EXCLUDED.current,
                           updated_at = CURRENT_TIMESTAMP
              RETURNING current`

	var newBalance float64
	err = tx.QueryRowContext(ctx, query, userID, amount).Scan(&newBalance)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	return tx.Commit()
}

// Withdraw выполняет списание средств с баланса пользователя
func (r *BalanceRepository) Withdraw(ctx context.Context, userID int, orderNumber string, sum float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Проверяем баланс
	var currentBalance float64
	err = tx.QueryRowContext(ctx,
		"SELECT current FROM balance WHERE user_id = $1 FOR UPDATE",
		userID).Scan(&currentBalance)

	if err != nil {
		if err == sql.ErrNoRows {
			currentBalance = 0
		} else {
			return fmt.Errorf("get balance for update: %w", err)
		}
	}

	if currentBalance < sum {
		return ErrInsufficientFunds
	}

	// Обновляем баланс
	_, err = tx.ExecContext(ctx,
		`INSERT INTO balance (user_id, current, withdrawn) 
         VALUES ($1, $2, $3)
         ON CONFLICT (user_id) 
         DO UPDATE SET 
            current = balance.current - EXCLUDED.withdrawn,
            withdrawn = balance.withdrawn + EXCLUDED.withdrawn,
            updated_at = CURRENT_TIMESTAMP`,
		userID, 0, sum)
	if err != nil {
		return fmt.Errorf("update balance on withdraw: %w", err)
	}

	// Записываем списание
	_, err = tx.ExecContext(ctx,
		`INSERT INTO withdrawals (user_id, order_number, sum) 
         VALUES ($1, $2, $3)`,
		userID, orderNumber, sum)
	if err != nil {
		return fmt.Errorf("record withdrawal: %w", err)
	}

	return tx.Commit()
}

// GetWithdrawals возвращает историю списаний пользователя
func (r *BalanceRepository) GetWithdrawals(ctx context.Context, userID int) ([]*domain.Withdrawal, error) {
	query := `SELECT order_number, sum, processed_at 
              FROM withdrawals 
              WHERE user_id = $1 
              ORDER BY processed_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query withdrawals: %w", err)
	}
	defer rows.Close()

	var withdrawals []*domain.Withdrawal
	for rows.Next() {
		withdrawal := &domain.Withdrawal{UserID: userID}
		err := rows.Scan(&withdrawal.OrderNumber, &withdrawal.Sum, &withdrawal.ProcessedAt)
		if err != nil {
			return nil, fmt.Errorf("scan withdrawal: %w", err)
		}
		withdrawals = append(withdrawals, withdrawal)
	}

	return withdrawals, nil
}
