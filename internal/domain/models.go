package domain

import (
	"time"
)

type User struct {
	ID           int       `json:"-"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"-"`
}

type Order struct {
	ID          int        `json:"-"`
	UserID      int        `json:"-"`
	Number      string     `json:"number"`
	Status      string     `json:"status"`
	Accrual     float64    `json:"accrual,omitempty"`
	UploadedAt  time.Time  `json:"uploaded_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

type Balance struct {
	ID        int       `json:"-"`
	UserID    int       `json:"-"`
	Current   float64   `json:"current"`
	Withdrawn float64   `json:"withdrawn"`
	UpdatedAt time.Time `json:"-"`
}

type Withdrawal struct {
	ID          int       `json:"-"`
	UserID      int       `json:"-"`
	OrderNumber string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

type AuthRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type AccrualResponse struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}
