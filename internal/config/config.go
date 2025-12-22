package config

import (
	"flag"
	"log"
	"os"
)

// Config содержит все настройки приложения
type Config struct {
	RunAddress           string // Адрес и порт для запуска сервера
	DatabaseURI          string // URI подключения к базе данных
	AccrualSystemAddress string // Адрес системы начислений баллов
	JWTSecret            string // Секретный ключ для JWT токенов
	LogLevel             string // Уровень логирования
}

// Load загружает конфигурацию из флагов командной строки и переменных окружения
// Переменные окружения имеют приоритет над флагами
func Load() *Config {
	cfg := &Config{}

	// Определяем флаги командной строки
	flag.StringVar(&cfg.RunAddress, "a", "localhost:8080", "Address and port to run server")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "Database connection URI")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", "localhost:8081", "Accrual system address")
	flag.StringVar(&cfg.JWTSecret, "s", "", "JWT secret key")
	flag.StringVar(&cfg.LogLevel, "l", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Проверяем переменные окружения (имеют приоритет над флагами)
	if envRunAddr := os.Getenv("RUN_ADDRESS"); envRunAddr != "" {
		cfg.RunAddress = envRunAddr
	}

	if envDBURI := os.Getenv("DATABASE_URI"); envDBURI != "" {
		cfg.DatabaseURI = envDBURI
	}

	if envAccrualAddr := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrualAddr != "" {
		cfg.AccrualSystemAddress = envAccrualAddr
	}

	if envJWTSecret := os.Getenv("JWT_SECRET"); envJWTSecret != "" {
		cfg.JWTSecret = envJWTSecret
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	}

	// Проверяем обязательные параметры
	if cfg.JWTSecret == "" {
		log.Fatal("JWT secret key is required. Set JWT_SECRET environment variable or use -s flag")
	}

	if cfg.DatabaseURI == "" {
		log.Fatal("Database URI is required. Set DATABASE_URI environment variable or use -d flag")
	}

	return cfg
}
