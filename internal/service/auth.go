package service

import (
	"context"
	"errors"

	"time"

	"github.com/besapuz/gofer-man/internal/domain"
	"github.com/besapuz/gofer-man/internal/repository"
	"github.com/besapuz/gofer-man/pkg/password"
	"github.com/golang-jwt/jwt/v4"
)

// AuthService предоставляет бизнес-логику для аутентификации и авторизации
type AuthService struct {
	userRepo *repository.UserRepository
	secret   []byte
}

// NewAuthService создает новый экземпляр AuthService
func NewAuthService(userRepo *repository.UserRepository, secret string) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		secret:   []byte(secret),
	}
}

// Register регистрирует нового пользователя в системе
// Возвращает созданного пользователя или ошибку если регистрация не удалась
func (s *AuthService) Register(ctx context.Context, req *domain.AuthRequest) (*domain.User, error) {
	// Валидируем входные данные
	if req.Login == "" || req.Password == "" {
		return nil, errors.New("login and password are required")
	}

	// Хешируем пароль
	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	// Создаем пользователя
	user := &domain.User{
		Login:        req.Login,
		PasswordHash: hashedPassword,
	}

	err = s.userRepo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Login выполняет аутентификацию пользователя
// Возвращает пользователя если аутентификация успешна, иначе возвращает ошибку
func (s *AuthService) Login(ctx context.Context, req *domain.AuthRequest) (*domain.User, error) {
	// Получаем пользователя
	user, err := s.userRepo.GetUserByLogin(ctx, req.Login)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Проверяем пароль
	if !password.CheckHash(req.Password, user.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}

// GenerateToken создает JWT токен для пользователя
// Возвращает токен в виде строки или ошибку если создание не удалось
func (s *AuthService) GenerateToken(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ValidateToken проверяет валидность JWT токена
// Возвращает идентификатор пользователя если токен валиден, иначе возвращает ошибку
func (s *AuthService) ValidateToken(tokenString string) (int, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID := int(claims["user_id"].(float64))
		return userID, nil
	}

	return 0, errors.New("invalid token")
}
