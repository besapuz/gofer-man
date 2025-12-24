package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/besapuz/gofer-man/internal/domain"
)

// UserRepository предоставляет методы доступа к данным для сущностей пользователей
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository создает новый экземпляр UserRepository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser создает нового пользователя в базе данных
func (r *UserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id`
	err := r.db.QueryRowContext(ctx, query, user.Login, user.PasswordHash).Scan(&user.ID)
	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"users_login_key\"" {
			return ErrUserExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// GetUserByLogin возвращает пользователя по его логину
func (r *UserRepository) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	query := `SELECT id, login, password_hash, created_at FROM users WHERE login = $1`
	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, login).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by login: %w", err)
	}
	return user, nil
}

// GetUserByID возвращает пользователя по его идентификатору
func (r *UserRepository) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	query := `SELECT id, login, password_hash, created_at FROM users WHERE id = $1`
	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}
