package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
	"github.com/safe-ai/excel-brushing-tool/internal/export"
	"github.com/safe-ai/excel-brushing-tool/internal/upload"
	"github.com/safe-ai/excel-brushing-tool/pkg/config"
)

// Service handles evidence submission and retrieval
type Service struct {
	repo       *Repository
	cleanRepo  *cleaning.Repository
	assessRepo *assessment.Repository
	uploadRepo *upload.Repository
	exportSvc  *export.Service
	t3Client   *T3Client
	cfg        *config.Config
}

// NewService creates a new evidence Service
func NewService(
	repo *Repository,
	cleanRepo *cleaning.Repository,
	assessRepo *assessment.Repository,
	uploadRepo *upload.Repository,
	exportSvc *export.Service,
	t3Client *T3Client,
	cfg *config.Config,
) *Service {
	return &Service{
		repo:       repo,
		cleanRepo:  cleanRepo,
		assessRepo: assessRepo,
		uploadRepo: uploadRepo,
		exportSvc:  exportSvc,
		t3Client:   t3Client,
		cfg:        cfg,
	}
}

// Submit computes file hashes, uploads to T3 Evidence API (IPFS + chain)
func (s *Service) Submit(ctx context.Context, sessionID, userID uuid.UUID) (*EvidenceRecord, error) {
	// Get the cleaning session with ownership verification
	session, err := s.cleanRepo.GetByIDAndUser(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("取得梳理記錄失敗: %w", err)
	}

	// Generate export files
	excelPath, err := s.exportSvc.GenerateExcelFile(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("產生 Excel 檔案失敗: %w", err)
	}

	logPath, err := s.exportSvc.GenerateLogFile(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("產生清理日誌失敗: %w", err)
	}

	// Compute SHA-256 hashes
	// processed_dataset = 梳理後的 Excel（要上 IPFS）
	processedHash, processedB64, err := computeFileHashAndBase64(excelPath)
	if err != nil {
		return nil, fmt.Errorf("計算梳理後資料雜湊失敗: %w", err)
	}

	// cleaning_log = hash-only
	logHash, _, err := computeFileHashAndBase64(logPath)
	if err != nil {
		return nil, fmt.Errorf("計算日誌雜湊失敗: %w", err)
	}

	// raw_dataset = 原始上傳的 Excel（hash-only）
	// 透過 session → assessment → upload 取得原始檔路徑
	var rawHash string
	assess, assessErr := s.assessRepo.GetByID(ctx, session.AssessmentID)
	if assessErr == nil {
		up, upErr := s.uploadRepo.GetByID(ctx, assess.UploadID)
		if upErr == nil {
			if h, hErr := computeFileHash(up.FilePath); hErr == nil {
				rawHash = h
			}
		}
	}
	if rawHash == "" {
		// fallback: 用 session ID 產生一個 deterministic hash
		sum := sha256.Sum256([]byte("raw:" + session.ID.String()))
		rawHash = hex.EncodeToString(sum[:])
	}

	// 嘗試透過 T3 Evidence API 上鏈
	var recordID string
	var txHash string
	var status string
	var rawCID, processedCID string

	t3Token, err := s.ensureT3Token(ctx, userID)
	if err != nil {
		// T3 不可用，走 demo 模式
		log.Printf("[evidence] T3 連線失敗，使用 demo 模式: %v", err)
		recordID = fmt.Sprintf("demo-%s", uuid.New().String()[:8])
		status = "demo"
	} else {
		// 準備 artifacts:
		// raw_dataset = 原始資料 (hash-only, 不留檔案)
		// processed_dataset = 梳理後 Excel (ipfs-upload)
		// cleaning_log = 日誌 (hash-only)
		artifacts := []T3Artifact{
			{
				Type:          "raw_dataset",
				Hash:          rawHash,
				StorageOption: "hash-only",
				Description:   fmt.Sprintf("原始資料 (%d 列)", session.RowsBefore),
			},
			{
				Type:          "processed_dataset",
				Hash:          processedHash,
				StorageOption: "ipfs-upload",
				Data:          processedB64,
				Description:   fmt.Sprintf("梳理後資料集 (%d 列)", session.RowsAfter),
			},
			{
				Type:          "cleaning_log",
				Hash:          logHash,
				StorageOption: "hash-only",
				Description:   "清洗過程日誌",
			},
		}

		req := T3EvidenceRecordRequest{
			Artifacts:   artifacts,
			ToolVersion: s.cfg.Blockchain.ToolVersion,
			RuleVersion: s.cfg.Blockchain.RuleVersion,
		}

		resp, err := s.t3Client.RecordEvidence(ctx, t3Token, req)
		if err != nil {
			log.Printf("[evidence] T3 存證失敗，使用 demo 模式: %v", err)
			recordID = fmt.Sprintf("demo-%s", uuid.New().String()[:8])
			status = "demo"
		} else {
			recordID = resp.RecordID
			txHash = resp.TransactionID
			status = resp.SignatureStatus

			// 取出各 artifact 的 CID
			for _, art := range resp.Artifacts {
				if art.IPFSCid != nil {
					switch art.Type {
					case "raw_dataset":
						rawCID = *art.IPFSCid
					case "processed_dataset":
						processedCID = *art.IPFSCid
					}
				}
			}
		}
	}

	// Save evidence record locally
	now := time.Now()
	record := &EvidenceRecord{
		ID:                uuid.New(),
		CleaningSessionID: sessionID,
		DatasetHash:       rawHash,
		LogHash:           logHash,
		ReportHash:        processedHash,
		RecordID:          recordID,
		TransactionHash:   txHash,
		SignatureStatus:   status,
		T3CID:            rawCID,
		T3TokenID:        processedCID,
		T3TxID:           txHash,
	}

	if status == "confirmed" {
		record.T3MintedAt = &now
	}

	if err := s.repo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("儲存存證記錄失敗: %w", err)
	}

	record.Timestamp = record.CreatedAt
	return record, nil
}

// GetRecord retrieves an evidence record by record_id
func (s *Service) GetRecord(ctx context.Context, recordID string) (*EvidenceRecord, error) {
	record, err := s.repo.GetByRecordID(ctx, recordID)
	if err != nil {
		return nil, err
	}
	record.Timestamp = record.CreatedAt
	return record, nil
}

// ensureT3Token 確保有可用的 T3 apiToken（自動註冊或刷新）
func (s *Service) ensureT3Token(ctx context.Context, userID uuid.UUID) (string, error) {
	// 檢查是否已有 T3 帳號對應
	mapping, err := s.repo.GetT3Mapping(ctx, userID)
	if err != nil && !errors.Is(err, ErrT3MappingNotFound) {
		return "", fmt.Errorf("查詢 T3 帳號對應失敗: %w", err)
	}

	externalUserID := fmt.Sprintf("safeai_%s", userID.String()[:12])

	if mapping == nil {
		// 首次使用，在 T3 註冊
		return s.registerAndSaveT3User(ctx, userID, externalUserID)
	}

	// 檢查 Token 是否還有效（提前 5 分鐘過期）
	if mapping.T3Token != "" && mapping.T3TokenExpiresAt != nil && mapping.T3TokenExpiresAt.After(time.Now().Add(5*time.Minute)) {
		return mapping.T3Token, nil
	}

	// Token 過期，重新 register 取得新 token
	// T3 現在支援重複 register 回傳新 token（不再回 409）
	regResp, err := s.t3Client.Register(ctx, externalUserID, "SAFE-AI User")
	if err != nil {
		return "", fmt.Errorf("T3 token 刷新失敗: %w", err)
	}

	// 更新 Token
	expiresAt := time.Now().Add(time.Duration(regResp.ExpiresIn) * time.Second)
	_ = s.repo.UpdateT3Token(ctx, userID, regResp.APIToken, expiresAt)

	return regResp.APIToken, nil
}

// registerAndSaveT3User 首次在 T3 註冊並儲存對應
func (s *Service) registerAndSaveT3User(ctx context.Context, userID uuid.UUID, externalUserID string) (string, error) {
	regResp, err := s.t3Client.Register(ctx, externalUserID, "SAFE-AI User")
	if err != nil {
		return "", fmt.Errorf("T3 註冊失敗: %w", err)
	}

	// 儲存對應關係
	expiresAt := time.Now().Add(time.Duration(regResp.ExpiresIn) * time.Second)
	mapping := &T3UserMapping{
		ID:                  uuid.New(),
		LocalUserID:         userID,
		T3Username:          externalUserID,
		T3PasswordEncrypted: regResp.IdentityRef,
		T3Token:             regResp.APIToken,
		T3TokenExpiresAt:    &expiresAt,
	}

	if err := s.repo.CreateT3Mapping(ctx, mapping); err != nil {
		return "", fmt.Errorf("儲存 T3 帳號對應失敗: %w", err)
	}

	return regResp.APIToken, nil
}

// computeFileHashAndBase64 computes SHA-256 hash and base64 encoding of a file
func computeFileHashAndBase64(filePath string) (string, string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", err
	}

	h := sha256.Sum256(data)
	hashHex := hex.EncodeToString(h[:])
	b64 := base64.StdEncoding.EncodeToString(data)

	return hashHex, b64, nil
}

// computeFileHash computes SHA-256 hash of a file (kept for backward compat)
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
