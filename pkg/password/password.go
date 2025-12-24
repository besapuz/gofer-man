package password

import (
	"golang.org/x/crypto/bcrypt"
)

// Hash создает хеш пароля с использованием bcrypt
// Возвращает хешированную строку и ошибку, если хеширование не удалось
func Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckHash проверяет соответствие пароля и хеша
// Возвращает true если пароль соответствует хешу, false в противном случае
func CheckHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
