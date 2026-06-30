package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
	"github.com/safe-ai/excel-brushing-tool/internal/export"
	"github.com/safe-ai/excel-brushing-tool/pkg/config"
)

// Service handles evidence submission and retrieval
type Service struct {
	repo             *Repository
	cleanRepo        *cleaning.Repository
	exportSvc        *export.Service
	blockchainClient *BlockchainClient
	cfg              *config.Config
}

// NewService creates a new evidence Service
func NewService(
	repo *Repository,
	cleanRepo *cleaning.Repository,
	exportSvc *export.Service,
	blockchainClient *BlockchainClient,
	cfg *config.Config,
) *Service {
	return &Service{
		repo:             repo,
		cleanRepo:        cleanRepo,
		exportSvc:        exportSvc,
		blockchainClient: blockchainClient,
		cfg:              cfg,
	}
}

// Submit computes file hashes and submits evidence to blockchain
func (s *Service) Submit(ctx context.Context, sessionID, userID uuid.UUID) (*EvidenceRecord, error) {
	// Get the cleaning session with ownership verification
	session, err := s.cleanRepo.GetByIDAndUser(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("取得梳理記錄失敗: %w", err)
	}

	// Generate export files to get paths for hashing
	excelPath, err := s.exportSvc.GenerateExcelFile(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("產生 Excel 檔案失敗: %w", err)
	}

	logPath, err := s.exportSvc.GenerateLogFile(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("產生清理日誌失敗: %w", err)
	}

	pdfPath, err := s.exportSvc.GeneratePDFFile(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("產生 PDF 報告失敗: %w", err)
	}

	// Compute SHA-256 hashes
	datasetHash, err := computeFileHash(excelPath)
	if err != nil {
		return nil, fmt.Errorf("計算資料集雜湊失敗: %w", err)
	}

	logHash, err := computeFileHash(logPath)
	if err != nil {
		return nil, fmt.Errorf("計算日誌雜湊失敗: %w", err)
	}

	reportHash, err := computeFileHash(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("計算報告雜湊失敗: %w", err)
	}

	// Prepare blockchain payload
	payload := BlockchainSubmitPayload{
		DatasetHash:     datasetHash,
		CleaningLogHash: logHash,
		ReportHash:      reportHash,
		Timestamp:       time.Now(),
		ToolVersion:     s.cfg.Blockchain.ToolVersion,
		RuleVersion:     s.cfg.Blockchain.RuleVersion,
		OperatorID:      userID.String(),
		Metadata: EvidenceMetadata{
			OriginalRows:    session.RowsBefore,
			RefinedRows:     session.RowsAfter,
			ReadinessBefore: session.ScoreBefore,
			ReadinessAfter:  session.ScoreAfter,
		},
	}

	// Try to submit to blockchain
	var recordID string
	var txHash string
	var status string
	var verificationURL string

	blockchainResp, err := s.blockchainClient.Submit(ctx, payload)
	if err != nil {
		// Blockchain unavailable - handle gracefully
		recordID = fmt.Sprintf("demo-%s", uuid.New().String()[:8])
		status = "demo"
	} else {
		recordID = blockchainResp.RecordID
		txHash = blockchainResp.TransactionHash
		status = blockchainResp.Status
		verificationURL = blockchainResp.VerificationURL
	}

	// Save evidence record locally
	record := &EvidenceRecord{
		ID:                uuid.New(),
		CleaningSessionID: sessionID,
		DatasetHash:       datasetHash,
		LogHash:           logHash,
		ReportHash:        reportHash,
		RecordID:          recordID,
		TransactionHash:   txHash,
		SignatureStatus:   status,
		VerificationURL:   verificationURL,
	}

	if err := s.repo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("儲存存證記錄失敗: %w", err)
	}

	return record, nil
}

// GetRecord retrieves an evidence record by record_id, optionally refreshing from blockchain
func (s *Service) GetRecord(ctx context.Context, recordID string) (*EvidenceRecord, error) {
	record, err := s.repo.GetByRecordID(ctx, recordID)
	if err != nil {
		return nil, err
	}

	// Optionally query blockchain for updated status (non-blocking)
	if record.SignatureStatus == "pending" {
		blockchainResp, err := s.blockchainClient.Get(ctx, recordID)
		if err == nil && blockchainResp.Status != "" {
			// Update local status
			_ = s.repo.UpdateStatus(ctx, recordID, blockchainResp.Status, blockchainResp.TransactionHash)
			record.SignatureStatus = blockchainResp.Status
			record.TransactionHash = blockchainResp.TransactionHash
		}
		// If blockchain unavailable, just return local data
	}

	return record, nil
}

// computeFileHash computes SHA-256 hash of a file
func computeFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
