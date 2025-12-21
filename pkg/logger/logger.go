package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log будет доступен всему коду как синглтон.
var Log *zap.Logger = zap.NewNop()

// Initialize инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize(level string) error {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	// создаём новую конфигурацию логера
	cfg := zap.NewProductionConfig()

	// устанавливаем уровень
	cfg.Level = lvl

	// настраиваем вывод (можно добавить файловый вывод при необходимости)
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	// настраиваем формат времени
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// создаём логер на основе конфигурации
	zl, err := cfg.Build()
	if err != nil {
		return err
	}

	// устанавливаем синглтон
	Log = zl
	return nil
}

// LoggingResponseWriter - оборачивает оригинальный ResponseWriter для отслеживания статуса и размера ответа.
type LoggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	bodyLength int
}

// NewLoggingResponseWriter создает новый LoggingResponseWriter.
func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader перехватывает вызов WriteHeader для сохранения кода статуса.
func (l *LoggingResponseWriter) WriteHeader(code int) {
	l.statusCode = code
	l.ResponseWriter.WriteHeader(code)
}

// Write перехватывает вызов Write для отслеживания размера тела ответа.
func (l *LoggingResponseWriter) Write(b []byte) (int, error) {
	n, err := l.ResponseWriter.Write(b)
	l.bodyLength += n
	return n, err
}

// GetStatusCode возвращает код статуса ответа.
func (l *LoggingResponseWriter) GetStatusCode() int {
	return l.statusCode
}

// GetBodyLength возвращает размер тела ответа.
func (l *LoggingResponseWriter) GetBodyLength() int {
	return l.bodyLength
}

// RequestLogger — middleware для логирования всех HTTP-запросов и ответов.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Создаем обертку для ResponseWriter
		lrw := NewLoggingResponseWriter(w)

		// Вызываем следующий обработчик
		next.ServeHTTP(lrw, r)

		// Вычисляем продолжительность запроса
		duration := time.Since(start)

		// Логируем информацию о запросе
		Log.Info("HTTP request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("query", r.URL.RawQuery),
			zap.String("ip", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
			zap.Int("status", lrw.GetStatusCode()),
			zap.Int("response_size", lrw.GetBodyLength()),
			zap.Duration("duration", duration),
		)

		// Для ошибок 4xx и 5xx добавляем дополнительное логирование
		if lrw.GetStatusCode() >= 400 {
			Log.Warn("HTTP error response",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", lrw.GetStatusCode()),
				zap.String("ip", r.RemoteAddr),
			)
		}
	})
}

// Sync синхронизирует логер (вызывать при завершении приложения).
func Sync() {
	_ = Log.Sync()
}

// Debug логирует сообщение на уровне отладки.
func Debug(msg string, fields ...zap.Field) {
	Log.Debug(msg, fields...)
}

// Info логирует информационное сообщение.
func Info(msg string, fields ...zap.Field) {
	Log.Info(msg, fields...)
}

// Warn логирует предупреждение.
func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}

// Error логирует сообщение об ошибке.
func Error(msg string, fields ...zap.Field) {
	Log.Error(msg, fields...)
}

// Fatal логирует критическое сообщение и завершает программу.
func Fatal(msg string, fields ...zap.Field) {
	Log.Fatal(msg, fields...)
}

// WithRequest создает логер с полями из HTTP запроса.
func WithRequest(r *http.Request) *zap.Logger {
	return Log.With(
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("ip", r.RemoteAddr),
	)
}
