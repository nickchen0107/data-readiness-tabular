package auth

import (
	"context"
	"time"
)

// Service 處理 auth 相關的業務邏輯
type Service struct {
	repo      *Repository
	jwtSecret string
	jwtExpiry time.Duration
}

// NewService 建立新的 auth Service
func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		jwtExpiry: 24 * time.Hour, // 預設 24 小時
	}
}

// Register 註冊新使用者
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*User, error) {
	// 1. 驗證 email 格式
	if err := ValidateEmail(req.Email); err != nil {
		return nil, err
	}

	// 2. 驗證密碼長度
	if err := ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// 3. 將密碼進行 bcrypt 雜湊
	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 4. 建立使用者
	user, err := s.repo.CreateUser(ctx, req.Email, hash)
	if err != nil {
		return nil, err
	}

	// 5. 回傳建立的使用者
	return user, nil
}

// getTokenExpiry 回傳 token 的預設過期時間（從現在起算）
func (s *Service) getTokenExpiry() time.Time {
	return time.Now().Add(s.jwtExpiry)
}

// Login 使用者登入，驗證憑證並回傳 JWT
func (s *Service) Login(ctx context.Context, req LoginRequest) (*TokenResponse, error) {
	// 1. 根據 email 查詢使用者
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}

	// 2. 驗證密碼
	if err := CheckPassword(user.PasswordHash, req.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 3. 產生 JWT
	token, expiresAt, err := GenerateToken(user.ID, user.Role, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, err
	}

	// 4. 回傳 token
	return &TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}
