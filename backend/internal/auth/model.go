package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidCredentials 帳號或密碼錯誤（不揭露具體欄位）
var ErrInvalidCredentials = errors.New("帳號或密碼錯誤")

// User 使用者資料結構
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email,omitempty" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Role         string    `json:"role" db:"role"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// RegisterRequest 註冊請求結構
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// LoginRequest 登入請求結構
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse JWT token 回應結構
type TokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}
