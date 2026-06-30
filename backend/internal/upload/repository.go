package upload

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUploadNotFound 上傳記錄不存在
var ErrUploadNotFound = errors.New("上傳記錄不存在")

// Repository 處理 upload 相關的資料庫操作
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository 建立新的 upload Repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create 建立上傳記錄
func (r *Repository) Create(ctx context.Context, u *Upload) error {
	sheetNamesJSON, err := json.Marshal(u.SheetNames)
	if err != nil {
		return err
	}
	mergedCellsJSON, err := json.Marshal(u.MergedCells)
	if err != nil {
		return err
	}

	err = r.pool.QueryRow(ctx,
		`INSERT INTO uploads (id, user_id, filename, file_path, file_size, row_count, col_count, selected_sheet, sheet_names, merged_cells)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING created_at`,
		u.ID, u.UserID, u.Filename, u.FilePath, u.FileSize,
		u.RowCount, u.ColCount, u.SelectedSheet, sheetNamesJSON, mergedCellsJSON,
	).Scan(&u.CreatedAt)
	return err
}

// GetByID 根據 ID 取得上傳記錄
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Upload, error) {
	var u Upload
	var sheetNamesJSON, mergedCellsJSON []byte

	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, filename, file_path, file_size, row_count, col_count, selected_sheet, sheet_names, merged_cells, created_at
		 FROM uploads WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.UserID, &u.Filename, &u.FilePath, &u.FileSize,
		&u.RowCount, &u.ColCount, &u.SelectedSheet, &sheetNamesJSON, &mergedCellsJSON, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUploadNotFound
		}
		return nil, err
	}

	if sheetNamesJSON != nil {
		if err := json.Unmarshal(sheetNamesJSON, &u.SheetNames); err != nil {
			return nil, err
		}
	}
	if mergedCellsJSON != nil {
		if err := json.Unmarshal(mergedCellsJSON, &u.MergedCells); err != nil {
			return nil, err
		}
	}

	return &u, nil
}

// UpdateSelectedSheet 更新選取的工作表
func (r *Repository) UpdateSelectedSheet(ctx context.Context, id uuid.UUID, sheetName string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE uploads SET selected_sheet = $1 WHERE id = $2`,
		sheetName, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUploadNotFound
	}
	return nil
}

// GetByIDAndUser 根據 ID 和 UserID 取得上傳記錄（確保所有權）
func (r *Repository) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*Upload, error) {
	var u Upload
	var sheetNamesJSON, mergedCellsJSON []byte

	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, filename, file_path, file_size, row_count, col_count, selected_sheet, sheet_names, merged_cells, created_at
		 FROM uploads WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&u.ID, &u.UserID, &u.Filename, &u.FilePath, &u.FileSize,
		&u.RowCount, &u.ColCount, &u.SelectedSheet, &sheetNamesJSON, &mergedCellsJSON, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUploadNotFound
		}
		return nil, err
	}

	if sheetNamesJSON != nil {
		if err := json.Unmarshal(sheetNamesJSON, &u.SheetNames); err != nil {
			return nil, err
		}
	}
	if mergedCellsJSON != nil {
		if err := json.Unmarshal(mergedCellsJSON, &u.MergedCells); err != nil {
			return nil, err
		}
	}

	return &u, nil
}
