package quota

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository 處理配額相關的資料庫操作
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository 建立新的 quota Repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetSettings 取得全域配額設定（單筆）
func (r *Repository) GetSettings(ctx context.Context) (*Settings, error) {
	var s Settings
	err := r.pool.QueryRow(ctx,
		`SELECT id, max_assessments, reset_period, updated_at FROM quota_settings LIMIT 1`,
	).Scan(&s.ID, &s.MaxAssessments, &s.ResetPeriod, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// UpdateSettings 更新配額設定
func (r *Repository) UpdateSettings(ctx context.Context, maxAssessments int, resetPeriod string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE quota_settings SET max_assessments = $1, reset_period = $2, updated_at = NOW()`,
		maxAssessments, resetPeriod,
	)
	return err
}

// GetUsageCount 取得使用者在指定時間之後的評估次數
func (r *Repository) GetUsageCount(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM assessments a
		 JOIN uploads u ON a.upload_id = u.id
		 WHERE u.user_id = $1 AND a.created_at >= $2`,
		userID, since,
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetLastQuotaReset 取得使用者上次配額重置時間
func (r *Repository) GetLastQuotaReset(ctx context.Context, userID uuid.UUID) (time.Time, error) {
	var lastReset time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT last_quota_reset FROM users WHERE id = $1`,
		userID,
	).Scan(&lastReset)
	if err != nil {
		return time.Time{}, err
	}
	return lastReset, nil
}

// UpdateLastQuotaReset 更新使用者的配額重置時間
func (r *Repository) UpdateLastQuotaReset(ctx context.Context, userID uuid.UUID, resetTime time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET last_quota_reset = $1 WHERE id = $2`,
		resetTime, userID,
	)
	return err
}
