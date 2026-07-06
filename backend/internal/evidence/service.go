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

	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
	"github.com/safe-ai/excel-brushing-tool/internal/export"
	"github.com/safe-ai/excel-brushing-tool/pkg/config"
)

// Service handles evidence submission and retrieval
type Service struct {
	repo      *Repository
	cleanRepo *cleaning.Repository
	exportSvc *export.Service
	t3Client  *T3Client
	cfg       *config.Config
}

// NewService creates a new evidence Service
func NewService(
	repo *Repository,
	cleanRepo *cleaning.Repository,
	exportSvc *export.Service,
	t3Client *T3Client,
	cfg *config.Config,
) *Service {
	return &Service{
		repo:      repo,
		cleanRepo: cleanRepo,
		exportSvc: exportSvc,
		t3Client:  t3Client,
		cfg:       cfg,
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

	pdfPath, err := s.exportSvc.GeneratePDFFile(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("產生 PDF 報告失敗: %w", err)
	}

	// Compute SHA-256 hashes
	datasetHash, _, err := computeFileHashAndBase64(excelPath)
	if err != nil {
		return nil, fmt.Errorf("計算資料集雜湊失敗: %w", err)
	}

	logHash, _, err := computeFileHashAndBase64(logPath)
	if err != nil {
		return nil, fmt.Errorf("計算日誌雜湊失敗: %w", err)
	}

	reportHash, _, err := computeFileHashAndBase64(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("計算報告雜湊失敗: %w", err)
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
		// 準備 artifacts — raw_dataset = 原始上傳的檔案, processed_dataset = 梳理後資料
		// 大檔案使用 hash-only 避免 IPFS upload timeout
		artifacts := []T3Artifact{
			{
				Type:          "raw_dataset",
				Hash:          datasetHash,
				StorageOption: "hash-only",
				Description:   fmt.Sprintf("梳理後資料集 (%d→%d 列)", session.RowsBefore, session.RowsAfter),
			},
			{
				Type:          "processed_dataset",
				Hash:          reportHash,
				StorageOption: "hash-only",
				Description:   "資料品質評估報告 (PDF)",
			},
			{
				Type:          "cleaning_log",
				Hash:          logHash,
				StorageOption: "hash-only",
				Description:   "清洗過程日誌（僅記錄 hash）",
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
		DatasetHash:       datasetHash,
		LogHash:           logHash,
		ReportHash:        reportHash,
		RecordID:          recordID,
		TransactionHash:   txHash,
		SignatureStatus:   status,
		T3CID:            rawCID,
		T3TokenID:        processedCID, // reuse field for processed CID
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
	// T3 文件說：Token 過期後需重新呼叫 register 取得新 token
	regResp, err := s.t3Client.Register(ctx, externalUserID, "SAFE-AI User")
	if err != nil {
		if errors.Is(err, ErrT3UserExists) {
			// 使用者已存在但 token 過期了 — 目前 T3 的 register 對已存在用戶回 409
			// 這表示需要另一種機制取得 token，暫時回傳錯誤
			return "", fmt.Errorf("T3 token 已過期且無法刷新（使用者已存在）")
		}
		return "", fmt.Errorf("T3 註冊失敗: %w", err)
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
		if errors.Is(err, ErrT3UserExists) {
			// 使用者在 T3 已存在（可能是之前的資料遺失），無法取得 token
			return "", fmt.Errorf("T3 使用者已存在但本地無記錄")
		}
		return "", fmt.Errorf("T3 註冊失敗: %w", err)
	}

	// 儲存對應關係
	expiresAt := time.Now().Add(time.Duration(regResp.ExpiresIn) * time.Second)
	mapping := &T3UserMapping{
		ID:                  uuid.New(),
		LocalUserID:         userID,
		T3Username:          externalUserID,
		T3PasswordEncrypted: regResp.IdentityRef, // 儲存 identityRef
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
