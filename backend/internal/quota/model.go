package quota

import (
	"time"

	"github.com/google/uuid"
)

// Settings 配額設定（全域單筆）
type Settings struct {
	ID             uuid.UUID `json:"id"`
	MaxAssessments int       `json:"max_assessments"`
	ResetPeriod    string    `json:"reset_period"` // "daily" | "weekly"
	UpdatedAt      time.Time `json:"updated_at"`
}

// QuotaInfo 使用者配額狀態資訊
type QuotaInfo struct {
	MaxAssessments int       `json:"max_assessments"`
	UsedCount      int       `json:"used_count"`
	Remaining      int       `json:"remaining"`
	ResetPeriod    string    `json:"reset_period"`
	NextReset      time.Time `json:"next_reset"`
}
