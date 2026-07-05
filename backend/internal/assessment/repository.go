package assessment

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository 錯誤
var (
	ErrAssessmentNotFound = errors.New("評估記錄不存在")
	ErrWeightsNotFound    = errors.New("權重設定不存在")
)

// Repository 處理 assessment 相關的資料庫操作
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository 建立新的 assessment Repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create 建立評估記錄
func (r *Repository) Create(ctx context.Context, a *Assessment) error {
	weightsJSON, err := json.Marshal(a.WeightsSnapshot)
	if err != nil {
		return err
	}
	issuesJSON, err := json.Marshal(a.Issues)
	if err != nil {
		return err
	}
	// Store column_details and row_distribution together
	type combinedDetails struct {
		Columns         []ColumnDetail  `json:"columns"`
		RowDistribution RowDistribution `json:"row_distribution"`
		TotalRows       int             `json:"total_rows"`
		Filename        string          `json:"filename"`
	}
	combined := combinedDetails{
		Columns:         a.ColumnDetails,
		RowDistribution: a.RowDistribution,
		TotalRows:       a.TotalRows,
		Filename:        a.Filename,
	}
	columnDetailsJSON, err := json.Marshal(combined)
	if err != nil {
		return err
	}

	err = r.pool.QueryRow(ctx,
		`INSERT INTO assessments (id, upload_id, total_score, row_completeness, column_completeness, format_consistency, duplicate_similar, table_structure, ai_query_readiness, weights_snapshot, status, issues, column_details)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING created_at`,
		a.ID, a.UploadID, a.TotalScore, a.RowCompleteness, a.ColumnCompleteness,
		a.FormatConsistency, a.DuplicateSimilar, a.TableStructure, a.AIQueryReadiness,
		weightsJSON, a.Status, issuesJSON, columnDetailsJSON,
	).Scan(&a.CreatedAt)
	return err
}

// GetLatest 取得最新的評估記錄
func (r *Repository) GetLatest(ctx context.Context) (*Assessment, error) {
	var a Assessment
	var weightsJSON, issuesJSON, columnDetailsJSON []byte

	err := r.pool.QueryRow(ctx,
		`SELECT id, upload_id, total_score, row_completeness, column_completeness,
		        format_consistency, duplicate_similar, table_structure, ai_query_readiness,
		        weights_snapshot, status, issues, column_details, created_at
		 FROM assessments ORDER BY created_at DESC LIMIT 1`,
	).Scan(&a.ID, &a.UploadID, &a.TotalScore,
		&a.RowCompleteness, &a.ColumnCompleteness,
		&a.FormatConsistency, &a.DuplicateSimilar,
		&a.TableStructure, &a.AIQueryReadiness,
		&weightsJSON, &a.Status, &issuesJSON, &columnDetailsJSON, &a.CreatedAt)
	if err != nil {
		return nil, ErrAssessmentNotFound
	}

	// Parse JSON fields
	if weightsJSON != nil {
		if err := json.Unmarshal(weightsJSON, &a.WeightsSnapshot); err != nil {
			return nil, err
		}
	}
	if issuesJSON != nil {
		if err := json.Unmarshal(issuesJSON, &a.Issues); err != nil {
			return nil, err
		}
	}
	if columnDetailsJSON != nil {
		type combinedDetails struct {
			Columns         []ColumnDetail  `json:"columns"`
			RowDistribution RowDistribution `json:"row_distribution"`
			TotalRows       int             `json:"total_rows"`
			Filename        string          `json:"filename"`
		}
		var combined combinedDetails
		if err := json.Unmarshal(columnDetailsJSON, &combined); err != nil {
			// Try parsing as simple column details array
			_ = json.Unmarshal(columnDetailsJSON, &a.ColumnDetails)
		} else {
			a.ColumnDetails = combined.Columns
			a.RowDistribution = combined.RowDistribution
			a.TotalRows = combined.TotalRows
			a.Filename = combined.Filename
		}
	}

	return &a, nil
}

// GetByID 根據 ID 取得評估記錄
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Assessment, error) {
	var a Assessment
	var weightsJSON, issuesJSON, columnDetailsJSON []byte

	err := r.pool.QueryRow(ctx,
		`SELECT id, upload_id, total_score, row_completeness, column_completeness, format_consistency, duplicate_similar, table_structure, ai_query_readiness, weights_snapshot, status, issues, column_details, created_at
		 FROM assessments WHERE id = $1`,
		id,
	).Scan(&a.ID, &a.UploadID, &a.TotalScore, &a.RowCompleteness, &a.ColumnCompleteness,
		&a.FormatConsistency, &a.DuplicateSimilar, &a.TableStructure, &a.AIQueryReadiness,
		&weightsJSON, &a.Status, &issuesJSON, &columnDetailsJSON, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssessmentNotFound
		}
		return nil, err
	}

	if weightsJSON != nil {
		if err := json.Unmarshal(weightsJSON, &a.WeightsSnapshot); err != nil {
			return nil, err
		}
	}
	if issuesJSON != nil {
		if err := json.Unmarshal(issuesJSON, &a.Issues); err != nil {
			return nil, err
		}
	}
	if columnDetailsJSON != nil {
		// Parse combined details (columns + row_distribution)
		type combinedDetails struct {
			Columns         []ColumnDetail  `json:"columns"`
			RowDistribution RowDistribution `json:"row_distribution"`
			TotalRows       int             `json:"total_rows"`
			Filename        string          `json:"filename"`
		}
		var combined combinedDetails
		if err := json.Unmarshal(columnDetailsJSON, &combined); err != nil {
			// Fallback: try to parse as plain []ColumnDetail for backward compat
			if err2 := json.Unmarshal(columnDetailsJSON, &a.ColumnDetails); err2 != nil {
				return nil, err
			}
		} else {
			a.ColumnDetails = combined.Columns
			a.RowDistribution = combined.RowDistribution
			a.TotalRows = combined.TotalRows
			a.Filename = combined.Filename
		}
	}

	return &a, nil
}

// ListByUserID 分頁取得指定使用者的評估記錄（依 created_at 降序），回傳記錄及總筆數
func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]Assessment, int, error) {
	// 計算總筆數 (join through uploads to find user's assessments)
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM assessments a JOIN uploads u ON a.upload_id = u.id WHERE u.user_id = $1`,
		userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 查詢分頁結果
	rows, err := r.pool.Query(ctx,
		`SELECT a.id, a.upload_id, a.total_score, a.status, a.column_details, a.created_at
		 FROM assessments a JOIN uploads u ON a.upload_id = u.id
		 WHERE u.user_id = $1
		 ORDER BY a.created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var assessments []Assessment
	for rows.Next() {
		var a Assessment
		var columnDetailsJSON []byte
		if err := rows.Scan(&a.ID, &a.UploadID, &a.TotalScore, &a.Status, &columnDetailsJSON, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		// 解析 column_details 中的 filename
		if columnDetailsJSON != nil {
			type combinedDetails struct {
				Filename string `json:"filename"`
			}
			var combined combinedDetails
			if err := json.Unmarshal(columnDetailsJSON, &combined); err == nil {
				a.Filename = combined.Filename
			}
		}
		assessments = append(assessments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return assessments, total, nil
}

// ListAll 分頁取得所有使用者的評估記錄（依 created_at 降序），回傳記錄及總筆數
func (r *Repository) ListAll(ctx context.Context, offset, limit int) ([]Assessment, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM assessments`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT a.id, a.upload_id, a.total_score, a.status, a.column_details, a.created_at
		 FROM assessments a
		 ORDER BY a.created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var assessments []Assessment
	for rows.Next() {
		var a Assessment
		var columnDetailsJSON []byte
		if err := rows.Scan(&a.ID, &a.UploadID, &a.TotalScore, &a.Status, &columnDetailsJSON, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		if columnDetailsJSON != nil {
			type combinedDetails struct {
				Filename string `json:"filename"`
			}
			var combined combinedDetails
			if err := json.Unmarshal(columnDetailsJSON, &combined); err == nil {
				a.Filename = combined.Filename
			}
		}
		assessments = append(assessments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return assessments, total, nil
}

// SettingsRepository 處理系統設定的資料庫操作
type SettingsRepository struct {
	pool *pgxpool.Pool
}

// NewSettingsRepository 建立新的 SettingsRepository
func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

// GetWeights 取得當前的評估權重設定
// 如果 system_settings 中沒有設定，回傳預設權重
func (sr *SettingsRepository) GetWeights(ctx context.Context) (Weights, error) {
	var valueJSON []byte

	err := sr.pool.QueryRow(ctx,
		`SELECT value FROM system_settings WHERE key = 'assessment_weights'`,
	).Scan(&valueJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DefaultWeights(), nil
		}
		return Weights{}, err
	}

	var w Weights
	if err := json.Unmarshal(valueJSON, &w); err != nil {
		return DefaultWeights(), nil
	}

	// 驗證權重是否有效，無效則回傳預設值
	if !w.IsValid() {
		return DefaultWeights(), nil
	}

	return w, nil
}
