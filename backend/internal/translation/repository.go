package translation

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrTranslationNotFound 翻譯項目不存在
var ErrTranslationNotFound = errors.New("翻譯項目不存在")

// Repository 處理翻譯相關的資料庫操作
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository 建立新的 translation Repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// FindByLocale 取得指定語系的所有翻譯，按 key 排序
func (r *Repository) FindByLocale(ctx context.Context, locale string) ([]Translation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, locale, key, value, updated_at FROM translations WHERE locale = $1 ORDER BY key`,
		locale,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var translations []Translation
	for rows.Next() {
		var t Translation
		if err := rows.Scan(&t.ID, &t.Locale, &t.Key, &t.Value, &t.UpdatedAt); err != nil {
			return nil, err
		}
		translations = append(translations, t)
	}
	return translations, rows.Err()
}

// FindByID 根據 ID 查詢翻譯項目
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Translation, error) {
	var t Translation
	err := r.pool.QueryRow(ctx,
		`SELECT id, locale, key, value, updated_at FROM translations WHERE id = $1`,
		id,
	).Scan(&t.ID, &t.Locale, &t.Key, &t.Value, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTranslationNotFound
		}
		return nil, err
	}
	return &t, nil
}

// Update 更新翻譯值
func (r *Repository) Update(ctx context.Context, id uuid.UUID, value string) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE translations SET value = $1, updated_at = NOW() WHERE id = $2`,
		value, id,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrTranslationNotFound
	}
	return nil
}

// Search 搜尋翻譯（模糊匹配 key 或 value），回傳結果及總筆數
func (r *Repository) Search(ctx context.Context, locale, query string, offset, limit int) ([]Translation, int, error) {
	// 計算符合條件的總筆數
	var total int
	pattern := "%" + query + "%"
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM translations WHERE locale = $1 AND (key ILIKE $2 OR value ILIKE $2)`,
		locale, pattern,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 查詢分頁結果
	rows, err := r.pool.Query(ctx,
		`SELECT id, locale, key, value, updated_at FROM translations
		 WHERE locale = $1 AND (key ILIKE $2 OR value ILIKE $2)
		 ORDER BY key
		 LIMIT $3 OFFSET $4`,
		locale, pattern, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var translations []Translation
	for rows.Next() {
		var t Translation
		if err := rows.Scan(&t.ID, &t.Locale, &t.Key, &t.Value, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		translations = append(translations, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return translations, total, nil
}
