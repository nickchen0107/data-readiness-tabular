package auth

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

// HashPassword 使用 bcrypt 將明文密碼雜湊（cost = 12）
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword 比對雜湊密碼與明文密碼
func CheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
