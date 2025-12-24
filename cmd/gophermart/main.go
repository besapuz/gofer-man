// cmd/server/main.go (полностью обновляем)
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/besapuz/gofer-man/internal/config"
	"github.com/besapuz/gofer-man/internal/database"
	"github.com/besapuz/gofer-man/internal/handler"
	"github.com/besapuz/gofer-man/internal/repository"
	"github.com/besapuz/gofer-man/internal/service"
	"github.com/besapuz/gofer-man/pkg/logger"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	// Инициализируем логер
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		logger.Fatal("Failed to initialize logger", zap.Error(err))
	}
	defer logger.Sync()

	logger.Info("Starting Gophermart server",
		zap.String("version", "1.0.0"),
		zap.String("log_level", cfg.LogLevel),
	)

	// Инициализируем базу данных
	logger.Info("Initializing database connection",
		zap.String("database_uri", maskPassword(cfg.DatabaseURI)),
	)

	db, err := database.InitDB(cfg.DatabaseURI)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection", zap.Error(err))
		}
	}()

	// Выполняем миграции
	if err := database.RunMigrations(db); err != nil {
		logger.Fatal("Failed to run migrations", zap.Error(err))
	}
	logger.Info("Database migrations completed successfully")

	// Инициализируем репозитории
	userRepo := repository.NewUserRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	balanceRepo := repository.NewBalanceRepository(db)

	// Инициализируем сервисы
	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	accrualService := service.NewAccrualService(cfg.AccrualSystemAddress)
	orderService := service.NewOrderService(orderRepo, balanceRepo, accrualService)
	balanceService := service.NewBalanceService(balanceRepo, orderRepo)

	// Инициализируем обработчики
	authHandler := handler.NewAuthHandler(authService)
	orderHandler := handler.NewOrderHandler(orderService)
	balanceHandler := handler.NewBalanceHandler(balanceService)

	// Настраиваем роутер с middleware логирования
	mux := http.NewServeMux()

	// Публичные маршруты
	mux.HandleFunc("/api/user/register", authHandler.Register)
	mux.HandleFunc("/api/user/login", authHandler.Login)

	// Защищенные маршруты
	protected := http.NewServeMux()
	protected.HandleFunc("/api/user/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			orderHandler.UploadOrder(w, r)
		case http.MethodGet:
			orderHandler.GetOrders(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	protected.HandleFunc("/api/user/balance", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			balanceHandler.GetBalance(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	protected.HandleFunc("/api/user/balance/withdraw", balanceHandler.Withdraw)
	protected.HandleFunc("/api/user/withdrawals", balanceHandler.GetWithdrawals)

	// Применяем middleware аутентификации к защищенным маршрутам
	mux.Handle("/", handler.AuthMiddleware(authService)(protected))

	// Оборачиваем весь роутер в middleware логирования
	handlerWithLogging := logger.RequestLogger(mux)

	// Запускаем обработчик заказов в фоновом режиме
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Запускаем обработчик заказов в фоновом режиме
	wg.Add(1)
	go func() {
		defer wg.Done()
		processOrders(ctx, orderService)
	}()

	// Настраиваем HTTP сервер
	server := &http.Server{
		Addr:         cfg.RunAddress,
		Handler:      handlerWithLogging,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdownComplete := make(chan struct{})

	// Настраиваем graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan

		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("Server shutdown error", zap.Error(err))
		}
		cancel()

		wg.Wait()

		logger.Info("All background goroutines stopped")
		close(shutdownComplete)
	}()

	// Запускаем сервер
	logger.Info("Server starting", zap.String("address", cfg.RunAddress))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
	<-shutdownComplete
	logger.Info("Server stopped gracefully")
}

// processOrders периодически обрабатывает необработанные заказы
func processOrders(ctx context.Context, orderService *service.OrderService) {
	logger.Info("Starting order processor")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Debug("Processing orders")
			if err := orderService.ProcessOrders(ctx); err != nil {
				logger.Error("Error processing orders", zap.Error(err))
			}
		case <-ctx.Done():
			logger.Info("Order processor stopped")
			return
		}
	}
}

// maskPassword маскирует пароль в строке подключения к БД для безопасного логирования
func maskPassword(connectionString string) string {
	if strings.Contains(connectionString, "://") {
		parsed, err := url.Parse(connectionString)
		if err == nil {
			// Если есть пароль, маскируем его
			if parsed.User != nil {
				if _, hasPassword := parsed.User.Password(); hasPassword {
					maskedUserInfo := fmt.Sprintf("%s:****", parsed.User.Username())
					parsed.User = url.UserPassword(maskedUserInfo, "")
					return parsed.String()
				}
			}
		}
	}
	if strings.Contains(connectionString, "password=") {
		start := strings.Index(connectionString, "password=")
		if start != -1 {
			start += 9 // длина "password="
			end := strings.Index(connectionString[start:], " ")
			if end == -1 {
				end = len(connectionString)
			} else {
				end += start
			}
			return connectionString[:start] + "****" + connectionString[end:]
		}
	}
	if len(connectionString) > 0 {
		return "[connection string with hidden password]"
	}

	return "[empty connection string]"
}
