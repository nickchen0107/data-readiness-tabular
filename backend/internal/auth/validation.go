package auth

import (
	"errors"
	"strings"
)

// Validation errors
var (
	ErrInvalidEmail     = errors.New("帳號不可為空，且長度需至少 3 個字元")
	ErrPasswordTooShort = errors.New("密碼長度需至少 8 個字元")
	ErrPasswordTooLong  = errors.New("密碼長度不可超過 72 個字元")
)

// ValidateEmail 驗證帳號（允許 email 或簡單帳號名稱，長度 ≥ 3）
func ValidateEmail(email string) error {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" || len(trimmed) < 3 {
		return ErrInvalidEmail
	}
	return nil
}

// ValidatePassword 驗證密碼長度 (8-72 字元)
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	if len(password) > 72 {
		return ErrPasswordTooLong
	}
	return nil
}
