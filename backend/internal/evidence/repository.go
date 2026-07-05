package evidence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository errors
var (
	ErrRecordNotFound = errors.New("存證記錄不存在")
	ErrT3UserExists   = errors.New("T3 使用者已存在")
	ErrT3MappingNotFound = errors.New("T3 帳號對應不存在")
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
		`INSERT INTO evidence_records (id, cleaning_session_id, dataset_hash, log_hash, report_hash, record_id, transaction_hash, signature_status, verification_url, t3_file_id, t3_cid, t3_token_id, t3_tx_id, t3_minted_at, t3_metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		 RETURNING created_at`,
		record.ID, record.CleaningSessionID, record.DatasetHash, record.LogHash, record.ReportHash,
		record.RecordID, record.TransactionHash, record.SignatureStatus, record.VerificationURL,
		nilIfEmpty(record.T3FileID), nilIfEmpty(record.T3CID), nilIfEmpty(record.T3TokenID), nilIfEmpty(record.T3TxID), record.T3MintedAt, nil,
	).Scan(&record.CreatedAt)
	return err
}

// GetByRecordID retrieves an evidence record by its blockchain record_id
func (r *Repository) GetByRecordID(ctx context.Context, recordID string) (*EvidenceRecord, error) {
	var record EvidenceRecord
	var t3FileID, t3CID, t3TokenID, t3TxID *string
	var t3MintedAt *time.Time

	err := r.pool.QueryRow(ctx,
		`SELECT id, cleaning_session_id, dataset_hash, log_hash, report_hash, record_id, transaction_hash, signature_status, verification_url, t3_file_id, t3_cid, t3_token_id, t3_tx_id, t3_minted_at, created_at
		 FROM evidence_records WHERE record_id = $1`,
		recordID,
	).Scan(&record.ID, &record.CleaningSessionID, &record.DatasetHash, &record.LogHash, &record.ReportHash,
		&record.RecordID, &record.TransactionHash, &record.SignatureStatus, &record.VerificationURL,
		&t3FileID, &t3CID, &t3TokenID, &t3TxID, &t3MintedAt, &record.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	if t3FileID != nil { record.T3FileID = *t3FileID }
	if t3CID != nil { record.T3CID = *t3CID }
	if t3TokenID != nil { record.T3TokenID = *t3TokenID }
	if t3TxID != nil { record.T3TxID = *t3TxID }
	record.T3MintedAt = t3MintedAt

	return &record, nil
}

// GetBySessionID retrieves an evidence record by its cleaning session ID
func (r *Repository) GetBySessionID(ctx context.Context, sessionID uuid.UUID) (*EvidenceRecord, error) {
	var record EvidenceRecord
	var t3FileID, t3CID, t3TokenID, t3TxID *string
	var t3MintedAt *time.Time

	err := r.pool.QueryRow(ctx,
		`SELECT id, cleaning_session_id, dataset_hash, log_hash, report_hash, record_id, transaction_hash, signature_status, verification_url, t3_file_id, t3_cid, t3_token_id, t3_tx_id, t3_minted_at, created_at
		 FROM evidence_records WHERE cleaning_session_id = $1`,
		sessionID,
	).Scan(&record.ID, &record.CleaningSessionID, &record.DatasetHash, &record.LogHash, &record.ReportHash,
		&record.RecordID, &record.TransactionHash, &record.SignatureStatus, &record.VerificationURL,
		&t3FileID, &t3CID, &t3TokenID, &t3TxID, &t3MintedAt, &record.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	if t3FileID != nil { record.T3FileID = *t3FileID }
	if t3CID != nil { record.T3CID = *t3CID }
	if t3TokenID != nil { record.T3TokenID = *t3TokenID }
	if t3TxID != nil { record.T3TxID = *t3TxID }
	record.T3MintedAt = t3MintedAt

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

// --- T3 User Mapping ---

// GetT3Mapping 取得本地使用者的 T3 帳號對應
func (r *Repository) GetT3Mapping(ctx context.Context, localUserID uuid.UUID) (*T3UserMapping, error) {
	var m T3UserMapping
	var tokenExpiresAt *time.Time
	var token *string

	err := r.pool.QueryRow(ctx,
		`SELECT id, local_user_id, t3_username, t3_password_encrypted, t3_token, t3_token_expires_at, created_at
		 FROM t3_user_mapping WHERE local_user_id = $1`,
		localUserID,
	).Scan(&m.ID, &m.LocalUserID, &m.T3Username, &m.T3PasswordEncrypted, &token, &tokenExpiresAt, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrT3MappingNotFound
		}
		return nil, err
	}

	if token != nil { m.T3Token = *token }
	m.T3TokenExpiresAt = tokenExpiresAt

	return &m, nil
}

// CreateT3Mapping 建立 T3 帳號對應
func (r *Repository) CreateT3Mapping(ctx context.Context, m *T3UserMapping) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO t3_user_mapping (id, local_user_id, t3_username, t3_password_encrypted, t3_token, t3_token_expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at`,
		m.ID, m.LocalUserID, m.T3Username, m.T3PasswordEncrypted, nilIfEmpty(m.T3Token), m.T3TokenExpiresAt,
	).Scan(&m.CreatedAt)
}

// UpdateT3Token 更新 T3 Token 快取
func (r *Repository) UpdateT3Token(ctx context.Context, localUserID uuid.UUID, token string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE t3_user_mapping SET t3_token = $1, t3_token_expires_at = $2 WHERE local_user_id = $3`,
		token, expiresAt, localUserID,
	)
	return err
}

// nilIfEmpty returns nil pointer for empty strings (for nullable DB columns)
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
