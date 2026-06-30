package cleaning

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository errors
var (
	ErrSessionNotFound = errors.New("梳理記錄不存在")
)

// Repository handles cleaning_sessions database operations
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new cleaning Repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create saves a new cleaning session to the database
func (r *Repository) Create(ctx context.Context, s *CleaningSession) error {
	rulesJSON, err := json.Marshal(s.RulesApplied)
	if err != nil {
		return err
	}
	logJSON, err := json.Marshal(s.CleaningLog)
	if err != nil {
		return err
	}

	// Use nil for empty OriginalFilename to store NULL in DB
	var originalFilename *string
	if s.OriginalFilename != "" {
		originalFilename = &s.OriginalFilename
	}

	err = r.pool.QueryRow(ctx,
		`INSERT INTO cleaning_sessions (id, assessment_id, user_id, rules_applied, rows_before, rows_after, score_before, score_after, cleaning_log, refined_file_path, original_filename)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING created_at`,
		s.ID, s.AssessmentID, s.UserID, rulesJSON,
		s.RowsBefore, s.RowsAfter, s.ScoreBefore, s.ScoreAfter,
		logJSON, s.RefinedFilePath, originalFilename,
	).Scan(&s.CreatedAt)
	return err
}

// GetByID retrieves a cleaning session by its ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*CleaningSession, error) {
	var s CleaningSession
	var rulesJSON, logJSON []byte
	var originalFilename *string

	err := r.pool.QueryRow(ctx,
		`SELECT id, assessment_id, user_id, rules_applied, rows_before, rows_after, score_before, score_after, cleaning_log, refined_file_path, original_filename, created_at
		 FROM cleaning_sessions WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.AssessmentID, &s.UserID, &rulesJSON,
		&s.RowsBefore, &s.RowsAfter, &s.ScoreBefore, &s.ScoreAfter,
		&logJSON, &s.RefinedFilePath, &originalFilename, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	if originalFilename != nil {
		s.OriginalFilename = *originalFilename
	}
	if rulesJSON != nil {
		if err := json.Unmarshal(rulesJSON, &s.RulesApplied); err != nil {
			return nil, err
		}
	}
	if logJSON != nil {
		if err := json.Unmarshal(logJSON, &s.CleaningLog); err != nil {
			return nil, err
		}
	}

	return &s, nil
}

// GetLatestByUser returns the most recent cleaning session for a user
func (r *Repository) GetLatestByUser(ctx context.Context, userID uuid.UUID) (*CleaningSession, error) {
	var s CleaningSession
	var rulesJSON, logJSON []byte
	var originalFilename *string

	err := r.pool.QueryRow(ctx,
		`SELECT id, assessment_id, user_id, rules_applied, rows_before, rows_after,
		        score_before, score_after, cleaning_log, refined_file_path, original_filename, created_at
		 FROM cleaning_sessions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`,
		userID,
	).Scan(&s.ID, &s.AssessmentID, &s.UserID, &rulesJSON,
		&s.RowsBefore, &s.RowsAfter, &s.ScoreBefore, &s.ScoreAfter,
		&logJSON, &s.RefinedFilePath, &originalFilename, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	if originalFilename != nil {
		s.OriginalFilename = *originalFilename
	}
	if rulesJSON != nil {
		if err := json.Unmarshal(rulesJSON, &s.RulesApplied); err != nil {
			return nil, err
		}
	}
	if logJSON != nil {
		if err := json.Unmarshal(logJSON, &s.CleaningLog); err != nil {
			return nil, err
		}
	}

	return &s, nil
}

// GetByIDAndUser retrieves a cleaning session by ID with ownership verification
func (r *Repository) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*CleaningSession, error) {
	var s CleaningSession
	var rulesJSON, logJSON []byte
	var originalFilename *string

	err := r.pool.QueryRow(ctx,
		`SELECT id, assessment_id, user_id, rules_applied, rows_before, rows_after, score_before, score_after, cleaning_log, refined_file_path, original_filename, created_at
		 FROM cleaning_sessions WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&s.ID, &s.AssessmentID, &s.UserID, &rulesJSON,
		&s.RowsBefore, &s.RowsAfter, &s.ScoreBefore, &s.ScoreAfter,
		&logJSON, &s.RefinedFilePath, &originalFilename, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	if originalFilename != nil {
		s.OriginalFilename = *originalFilename
	}
	if rulesJSON != nil {
		if err := json.Unmarshal(rulesJSON, &s.RulesApplied); err != nil {
			return nil, err
		}
	}
	if logJSON != nil {
		if err := json.Unmarshal(logJSON, &s.CleaningLog); err != nil {
			return nil, err
		}
	}

	return &s, nil
}
