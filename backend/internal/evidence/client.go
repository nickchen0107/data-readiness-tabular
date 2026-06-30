package evidence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// BlockchainClient handles HTTP communication with the external blockchain API
type BlockchainClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewBlockchainClient creates a new BlockchainClient
func NewBlockchainClient(baseURL string, httpClient *http.Client) *BlockchainClient {
	return &BlockchainClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// Submit sends an evidence record to the blockchain API
func (c *BlockchainClient) Submit(ctx context.Context, req BlockchainSubmitPayload) (*BlockchainResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化區塊鏈請求失敗: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/evidence/submit", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("建立區塊鏈請求失敗: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("區塊鏈服務暫時無法連線: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("區塊鏈 API 回傳錯誤 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result BlockchainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析區塊鏈回應失敗: %w", err)
	}

	return &result, nil
}

// Get queries the blockchain API for an existing record
func (c *BlockchainClient) Get(ctx context.Context, recordID string) (*BlockchainResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/evidence/"+recordID, nil)
	if err != nil {
		return nil, fmt.Errorf("建立區塊鏈查詢請求失敗: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("區塊鏈服務暫時無法連線: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("區塊鏈記錄不存在")
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("區塊鏈 API 回傳錯誤 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result BlockchainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析區塊鏈回應失敗: %w", err)
	}

	return &result, nil
}
