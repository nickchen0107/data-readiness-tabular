package evidence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository errors
var (
	ErrRecordNotFound = errors.New("存證記錄不存在")
)

// Repository handles evidence_records database operations
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new evidence Repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create saves a new evidence record to the database
func (r *Repository) Create(ctx context.Context, record *EvidenceRecord) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO evidence_records (id, cleaning_session_id, dataset_hash, log_hash, report_hash, record_id, transaction_hash, signature_status, verification_url)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING created_at`,
		record.ID, record.CleaningSessionID, record.DatasetHash, record.LogHash, record.ReportHash,
		record.RecordID, record.TransactionHash, record.SignatureStatus, record.VerificationURL,
	).Scan(&record.CreatedAt)
	return err
}

// GetByRecordID retrieves an evidence record by its blockchain record_id
func (r *Repository) GetByRecordID(ctx context.Context, recordID string) (*EvidenceRecord, error) {
	var record EvidenceRecord

	err := r.pool.QueryRow(ctx,
		`SELECT id, cleaning_session_id, dataset_hash, log_hash, report_hash, record_id, transaction_hash, signature_status, verification_url, created_at
		 FROM evidence_records WHERE record_id = $1`,
		recordID,
	).Scan(&record.ID, &record.CleaningSessionID, &record.DatasetHash, &record.LogHash, &record.ReportHash,
		&record.RecordID, &record.TransactionHash, &record.SignatureStatus, &record.VerificationURL, &record.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	return &record, nil
}

// GetBySessionID retrieves an evidence record by its cleaning session ID
func (r *Repository) GetBySessionID(ctx context.Context, sessionID uuid.UUID) (*EvidenceRecord, error) {
	var record EvidenceRecord

	err := r.pool.QueryRow(ctx,
		`SELECT id, cleaning_session_id, dataset_hash, log_hash, report_hash, record_id, transaction_hash, signature_status, verification_url, created_at
		 FROM evidence_records WHERE cleaning_session_id = $1`,
		sessionID,
	).Scan(&record.ID, &record.CleaningSessionID, &record.DatasetHash, &record.LogHash, &record.ReportHash,
		&record.RecordID, &record.TransactionHash, &record.SignatureStatus, &record.VerificationURL, &record.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	return &record, nil
}

// UpdateStatus updates the signature status of an evidence record
func (r *Repository) UpdateStatus(ctx context.Context, recordID string, status string, txHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE evidence_records SET signature_status = $1, transaction_hash = $2 WHERE record_id = $3`,
		status, txHash, recordID,
	)
	return err
}
