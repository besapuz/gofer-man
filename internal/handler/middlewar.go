package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/besapuz/gofer-man/internal/service"
)

// contextKey тип для ключей контекста
type contextKey string

// userIDKey ключ для хранения идентификатора пользователя в контексте
const userIDKey contextKey = "userID"

// AuthMiddleware создает middleware для аутентификации пользователей
// Проверяет JWT токен из заголовка Authorization или куки
func AuthMiddleware(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем токен из заголовка Authorization или куки
			var token string

			// Проверяем заголовок Authorization (имеет приоритет)
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}

			// Проверяем куки
			if token == "" {
				cookie, err := r.Cookie("token")
				if err == nil {
					token = cookie.Value
				}
			}

			if token == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Валидируем токен
			userID, err := authService.ValidateToken(token)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Добавляем userID в контекст
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext извлекает идентификатор пользователя из контекста
// Возвращает 0 если идентификатор не найден
func GetUserIDFromContext(ctx context.Context) int {
	userID, _ := ctx.Value(userIDKey).(int)
	return userID
}
