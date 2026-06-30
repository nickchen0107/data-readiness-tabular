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
	CreatedAt         time.Time `json:"created_at"`
}

// EvidenceSubmitRequest is the request body for evidence submission
type EvidenceSubmitRequest struct {
	SessionID uuid.UUID `json:"session_id" binding:"required"`
}

// BlockchainSubmitPayload is sent to the external blockchain API
type BlockchainSubmitPayload struct {
	DatasetHash    string           `json:"dataset_hash"`
	CleaningLogHash string          `json:"cleaning_log_hash"`
	ReportHash     string           `json:"report_hash"`
	Timestamp      time.Time        `json:"timestamp"`
	ToolVersion    string           `json:"tool_version"`
	RuleVersion    string           `json:"rule_version"`
	OperatorID     string           `json:"operator_id"`
	Metadata       EvidenceMetadata `json:"metadata"`
}

// EvidenceMetadata contains additional metadata for the evidence record
type EvidenceMetadata struct {
	OriginalFilename string  `json:"original_filename"`
	OriginalRows     int     `json:"original_rows"`
	RefinedRows      int     `json:"refined_rows"`
	ReadinessBefore  float64 `json:"readiness_before"`
	ReadinessAfter   float64 `json:"readiness_after"`
}

// BlockchainResponse is the response from the external blockchain API
type BlockchainResponse struct {
	RecordID        string `json:"record_id"`
	TransactionHash string `json:"transaction_hash"`
	Status          string `json:"status"`
	VerificationURL string `json:"verification_url"`
}
