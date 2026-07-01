package quota

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// taipeiLocation 台北時區 (UTC+8)
var taipeiLocation *time.Location

func init() {
	var err error
	taipeiLocation, err = time.LoadLocation("Asia/Taipei")
	if err != nil {
		// 若時區資料不可用，使用固定偏移量
		taipeiLocation = time.FixedZone("Asia/Taipei", 8*60*60)
	}
}

// Service 處理配額邏輯
type Service struct {
	repo *Repository
}

// NewService 建立新的 quota Service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CheckAndConsume 檢查使用者是否仍有配額（含惰性重置）
// 回傳 (allowed, remaining, error)
// 注意：實際「消耗」由 assessment service 建立記錄時完成，此處僅做檢查
func (s *Service) CheckAndConsume(ctx context.Context, userID uuid.UUID) (bool, int, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return false, 0, err
	}

	lastReset, err := s.repo.GetLastQuotaReset(ctx, userID)
	if err != nil {
		return false, 0, err
	}

	// 計算本期邊界
	boundary := s.getPeriodBoundary(settings.ResetPeriod)

	// 惰性重置：若上次重置時間早於本期邊界，則更新重置時間
	if lastReset.Before(boundary) {
		now := time.Now()
		if err := s.repo.UpdateLastQuotaReset(ctx, userID, now); err != nil {
			return false, 0, err
		}
		lastReset = now
	}

	// 計算已使用次數（自上次重置以來）
	usedCount, err := s.repo.GetUsageCount(ctx, userID, lastReset)
	if err != nil {
		return false, 0, err
	}

	// 若已達上限，拒絕
	if usedCount >= settings.MaxAssessments {
		return false, 0, nil
	}

	// 允許，回傳本次消耗後剩餘數量
	remaining := settings.MaxAssessments - usedCount - 1
	return true, remaining, nil
}

// GetUserQuotaInfo 取得使用者配額狀態（含惰性重置檢查，但不消耗）
func (s *Service) GetUserQuotaInfo(ctx context.Context, userID uuid.UUID) (*QuotaInfo, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	lastReset, err := s.repo.GetLastQuotaReset(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 計算本期邊界
	boundary := s.getPeriodBoundary(settings.ResetPeriod)

	// 惰性重置
	if lastReset.Before(boundary) {
		now := time.Now()
		if err := s.repo.UpdateLastQuotaReset(ctx, userID, now); err != nil {
			return nil, err
		}
		lastReset = now
	}

	// 計算已使用次數
	usedCount, err := s.repo.GetUsageCount(ctx, userID, lastReset)
	if err != nil {
		return nil, err
	}

	remaining := settings.MaxAssessments - usedCount
	if remaining < 0 {
		remaining = 0
	}

	// 計算下次重置時間
	nextReset := s.getNextReset(settings.ResetPeriod)

	return &QuotaInfo{
		MaxAssessments: settings.MaxAssessments,
		UsedCount:      usedCount,
		Remaining:      remaining,
		ResetPeriod:    settings.ResetPeriod,
		NextReset:      nextReset,
	}, nil
}

// getPeriodBoundary 取得本期的起始邊界時間
func (s *Service) getPeriodBoundary(resetPeriod string) time.Time {
	now := time.Now().In(taipeiLocation)
	switch resetPeriod {
	case "weekly":
		// 本週一 00:00 Asia/Taipei
		weekday := now.Weekday()
		daysFromMonday := int(weekday) - int(time.Monday)
		if daysFromMonday < 0 {
			daysFromMonday += 7
		}
		monday := now.AddDate(0, 0, -daysFromMonday)
		return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, taipeiLocation)
	default: // "daily"
		// 今日 00:00 Asia/Taipei
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, taipeiLocation)
	}
}

// getNextReset 取得下次重置時間
func (s *Service) getNextReset(resetPeriod string) time.Time {
	now := time.Now().In(taipeiLocation)
	switch resetPeriod {
	case "weekly":
		// 下週一 00:00 Asia/Taipei
		weekday := now.Weekday()
		daysUntilMonday := (int(time.Monday) - int(weekday) + 7) % 7
		if daysUntilMonday == 0 {
			daysUntilMonday = 7
		}
		nextMonday := now.AddDate(0, 0, daysUntilMonday)
		return time.Date(nextMonday.Year(), nextMonday.Month(), nextMonday.Day(), 0, 0, 0, 0, taipeiLocation)
	default: // "daily"
		// 明日 00:00 Asia/Taipei
		tomorrow := now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, taipeiLocation)
	}
}
