package database

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/besapuz/gofer-man/pkg/logger"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// InitDB инициализирует подключение к PostgreSQL базе данных
// Возвращает *sql.DB или ошибку если подключение не удалось
func InitDB(dsn string) (*sql.DB, error) {
	logger.Info("Connecting to database")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Error("Failed to open database connection", zap.Error(err))
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		logger.Error("Failed to ping database", zap.Error(err))
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

// RunMigrations выполняет SQL миграции из файла
// Создает необходимые таблицы и индексы в базе данных
func RunMigrations(db *sql.DB) error {
	// Читаем файл миграции
	logger.Info("Running database migrations")
	migration, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		logger.Error("Failed to read migration file", zap.Error(err))
		return fmt.Errorf("read migration file: %w", err)
	}

	// Выполняем миграцию
	_, err = db.Exec(string(migration))
	if err != nil {
		logger.Error("Failed to execute migration", zap.Error(err))
		return fmt.Errorf("execute migration: %w", err)
	}

	logger.Info("Database migrations completed successfully")
	return nil
}
