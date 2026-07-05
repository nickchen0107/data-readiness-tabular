package evidence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// T3Client handles HTTP communication with the T3 TrustChain Evidence API
type T3Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewT3Client creates a new T3Client
func NewT3Client(baseURL string, apiKey string, httpClient *http.Client) *T3Client {
	return &T3Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

// Register 在 T3 Evidence API 註冊使用者（也用於 token 過期後重新取得 token）
func (c *T3Client) Register(ctx context.Context, externalUserID string, displayName string) (*T3EvidenceRegisterResponse, error) {
	reqBody := T3EvidenceRegisterRequest{
		ExternalUserID: externalUserID,
		DisplayName:    displayName,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化 T3 註冊請求失敗: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/evidence/register", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("建立 T3 註冊請求失敗: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("T3 服務暫時無法連線: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		// 使用者已存在 — 需要重新 register 來取得新 token
		// T3 的 register 對已存在使用者回傳 409
		return nil, ErrT3UserExists
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("T3 註冊失敗 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result T3EvidenceRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 T3 註冊回應失敗: %w", err)
	}

	return &result, nil
}

// RecordEvidence 上傳存證資料（上傳 IPFS + 寫入區塊鏈，一步完成）
func (c *T3Client) RecordEvidence(ctx context.Context, apiToken string, req T3EvidenceRecordRequest) (*T3EvidenceRecordResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化 T3 存證請求失敗: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/evidence/records", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("建立 T3 存證請求失敗: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.apiKey)
	httpReq.Header.Set("Authorization", "Bearer "+apiToken)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("T3 服務暫時無法連線: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("T3 存證失敗 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result T3EvidenceRecordResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 T3 存證回應失敗: %w", err)
	}

	return &result, nil
}

// GetRecord 查詢存證記錄
func (c *T3Client) GetRecord(ctx context.Context, apiToken string, recordID string) (*T3EvidenceQueryResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/evidence/records/"+recordID, nil)
	if err != nil {
		return nil, fmt.Errorf("建立 T3 查詢請求失敗: %w", err)
	}
	httpReq.Header.Set("X-API-Key", c.apiKey)
	httpReq.Header.Set("Authorization", "Bearer "+apiToken)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("T3 服務暫時無法連線: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("T3 存證記錄不存在")
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("T3 查詢失敗 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result T3EvidenceQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 T3 查詢回應失敗: %w", err)
	}

	return &result, nil
}

// HealthCheck 檢查 T3 服務是否可用
func (c *T3Client) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("T3 服務暫時無法連線: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("T3 健康檢查失敗 (HTTP %d)", resp.StatusCode)
	}
	return nil
}
