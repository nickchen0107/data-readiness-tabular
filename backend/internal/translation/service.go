package translation

import (
	"context"

	"github.com/google/uuid"
)

// Service 處理翻譯業務邏輯
type Service struct {
	repo *Repository
}

// NewService 建立新的 translation Service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetByLocale 取得指定語系的翻譯，回傳 key→value map
func (s *Service) GetByLocale(ctx context.Context, locale string) (map[string]string, error) {
	translations, err := s.repo.FindByLocale(ctx, locale)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(translations))
	for _, t := range translations {
		result[t.Key] = t.Value
	}
	return result, nil
}

// Update 更新翻譯值
func (s *Service) Update(ctx context.Context, id uuid.UUID, value string) error {
	return s.repo.Update(ctx, id, value)
}

// Search 搜尋翻譯，支援分頁
func (s *Service) Search(ctx context.Context, locale, query string, page, pageSize int) ([]Translation, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.repo.Search(ctx, locale, query, offset, pageSize)
}
