package repository

import (
	"errors"
)

// Общие ошибки уровня репозитория
var (
	// ErrUserExists указывает, что пользователь с данным логином уже существует
	ErrUserExists = errors.New("user already exists")

	// ErrUserNotFound указывает, что пользователь с указанными данными не найден
	ErrUserNotFound = errors.New("user not found")

	// ErrOrderExists указывает, что заказ с данным номером уже существует
	ErrOrderExists = errors.New("order already exists")

	// ErrOrderNotFound указывает, что заказ не найден по заданным критериям
	ErrOrderNotFound = errors.New("order not found")

	// ErrInsufficientFunds указывает, что у пользователя недостаточно средств для списания
	ErrInsufficientFunds = errors.New("insufficient funds")
)
