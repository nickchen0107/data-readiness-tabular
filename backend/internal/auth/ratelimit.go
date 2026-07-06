package auth

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RateLimiter 登入速率限制器
// 使用 login_attempts table 追蹤登入嘗試次數
type RateLimiter struct {
	pool        *pgxpool.Pool
	maxAttempts int
	window      time.Duration
}

// NewRateLimiter 建立新的登入速率限制器
// maxAttempts: 時間窗口內允許的最大失敗次數（預設 5）
// window: 時間窗口（預設 15 分鐘）
func NewRateLimiter(pool *pgxpool.Pool, maxAttempts int, window time.Duration) *RateLimiter {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	return &RateLimiter{
		pool:        pool,
		maxAttempts: maxAttempts,
		window:      window,
	}
}

// IsBlocked 檢查 email 是否因過多失敗嘗試而被暫時鎖定
// 在指定時間窗口內，若失敗次數 >= maxAttempts，回傳 true（被鎖定）
func (rl *RateLimiter) IsBlocked(ctx context.Context, email string) (bool, error) {
	since := time.Now().Add(-rl.window)

	var count int
	err := rl.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM login_attempts 
		 WHERE username = $1 AND success = false AND attempted_at > $2`,
		email, since,
	).Scan(&count)
	if err != nil {
		return false, err
	}

	return count >= rl.maxAttempts, nil
}

// RecordAttempt 記錄登入嘗試（成功或失敗）
func (rl *RateLimiter) RecordAttempt(ctx context.Context, email string, success bool) error {
	_, err := rl.pool.Exec(ctx,
		`INSERT INTO login_attempts (username, success, attempted_at) VALUES ($1, $2, $3)`,
		email, success, time.Now(),
	)
	return err
}
