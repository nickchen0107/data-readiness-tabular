package evidence

import (
	"time"

	"github.com/google/uuid"
)

// EvidenceRecord represents a blockchain evidence submission record
type EvidenceRecord struct {
	ID                uuid.UUID `json:"id"`
	CleaningSessionID uuid.UUID `json:"cleaning_session_id"`
	DatasetHash       string    `json:"dataset_hash"`
	LogHash           string    `json:"log_hash"`
	ReportHash        string    `json:"report_hash"`
	RecordID          string    `json:"record_id"`
	TransactionHash   string    `json:"transaction_hash,omitempty"`
	SignatureStatus   string    `json:"signature_status"` // "confirmed", "pending", "demo"
	VerificationURL   string    `json:"verification_url,omitempty"`
	// T3 TrustChain 欄位
	T3FileID  string     `json:"t3_file_id,omitempty"`  // deprecated, kept for compat
	T3CID     string     `json:"t3_cid,omitempty"`      // raw_dataset IPFS CID
	T3TokenID string     `json:"t3_token_id,omitempty"` // processed_dataset IPFS CID
	T3TxID    string     `json:"t3_tx_id,omitempty"`    // transaction ID on chain
	T3MintedAt *time.Time `json:"t3_minted_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	Timestamp time.Time  `json:"timestamp"` // 前端顯示用，等同 created_at
}

// EvidenceSubmitRequest is the request body for evidence submission
type EvidenceSubmitRequest struct {
	SessionID uuid.UUID `json:"session_id" binding:"required"`
}

// T3UserMapping 儲存本地使用者與 T3 Evidence API 的對應關係
type T3UserMapping struct {
	ID               uuid.UUID  `json:"id"`
	LocalUserID      uuid.UUID  `json:"local_user_id"`
	T3Username       string     `json:"t3_username"`       // externalUserId sent to T3
	T3PasswordEncrypted string  `json:"-"`                 // identityRef from T3
	T3Token          string     `json:"-"`                 // apiToken from T3
	T3TokenExpiresAt *time.Time `json:"-"`
	CreatedAt        time.Time  `json:"created_at"`
}

// --- T3 Evidence API Request/Response 結構 ---

// T3EvidenceRegisterRequest T3 Evidence 註冊請求
type T3EvidenceRegisterRequest struct {
	ExternalUserID string `json:"externalUserId"`
	DisplayName    string `json:"displayName"`
}

// T3EvidenceRegisterResponse T3 Evidence 註冊回應
type T3EvidenceRegisterResponse struct {
	APIToken    string `json:"apiToken"`
	UserID      string `json:"userId"`
	IdentityRef string `json:"identityRef"`
	ExpiresIn   int    `json:"expiresIn"` // seconds
}

// T3Artifact 單一存證產物
type T3Artifact struct {
	Type          string `json:"type"`                    // raw_dataset, processed_dataset, cleaning_log
	Hash          string `json:"hash"`                    // SHA-256 hex
	StorageOption string `json:"storageOption"`           // ipfs-upload, hash-only
	Data          string `json:"data,omitempty"`          // base64 encoded file content
	Description   string `json:"description,omitempty"`
}

// T3EvidenceRecordRequest T3 存證上傳請求
type T3EvidenceRecordRequest struct {
	Artifacts      []T3Artifact `json:"artifacts"`
	ToolVersion    string       `json:"toolVersion"`
	RuleVersion    string       `json:"ruleVersion,omitempty"`
	ParentRecordID string       `json:"parentRecordId,omitempty"`
}

// T3ArtifactResponse 存證回應中的產物資訊
type T3ArtifactResponse struct {
	Type    string  `json:"type"`
	Hash    string  `json:"hash"`
	IPFSCid *string `json:"ipfsCid"` // null for hash-only
}

// T3EvidenceRecordResponse T3 存證回應
type T3EvidenceRecordResponse struct {
	RecordID        string               `json:"recordId"`
	TransactionID   string               `json:"transactionId"`
	SignatureStatus string               `json:"signatureStatus"`
	Timestamp       string               `json:"timestamp"`
	Artifacts       []T3ArtifactResponse `json:"artifacts"`
}

// T3EvidenceQueryResponse T3 存證查詢回應
type T3EvidenceQueryResponse struct {
	RecordID            string  `json:"recordId"`
	UserID              string  `json:"userId"`
	IdentityRef         string  `json:"identityRef"`
	PlatformID          string  `json:"platformId"`
	RawDatasetHash      string  `json:"rawDatasetHash"`
	ProcessedDatasetHash string `json:"processedDatasetHash"`
	CleaningLogHash     *string `json:"cleaningLogHash"`
	RawDatasetCid       *string `json:"rawDatasetCid"`
	ProcessedDatasetCid *string `json:"processedDatasetCid"`
	CleaningLogCid      *string `json:"cleaningLogCid"`
	ToolVersion         string  `json:"toolVersion"`
	RuleVersion         string  `json:"ruleVersion"`
	SignatureStatus     string  `json:"signatureStatus"`
	TransactionID       string  `json:"transactionId"`
	Timestamp           string  `json:"timestamp"`
}

// EvidenceMetadata contains additional metadata for the evidence record
type EvidenceMetadata struct {
	OriginalFilename string  `json:"original_filename"`
	OriginalRows     int     `json:"original_rows"`
	RefinedRows      int     `json:"refined_rows"`
	ReadinessBefore  float64 `json:"readiness_before"`
	ReadinessAfter   float64 `json:"readiness_after"`
}
