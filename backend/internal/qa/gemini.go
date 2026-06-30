package qa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/safe-ai/excel-brushing-tool/pkg/config"
)

// GeminiClient handles communication with the Gemini API
type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	cfg        *config.Config
}

// NewGeminiClient creates a new GeminiClient
func NewGeminiClient(cfg *config.Config, httpClient *http.Client) *GeminiClient {
	return &GeminiClient{
		apiKey:     cfg.GeminiAPIKey,
		model:      cfg.LLM.Model,
		httpClient: httpClient,
		cfg:        cfg,
	}
}

// Ask sends a question with CSV data to Gemini and returns the answer
func (g *GeminiClient) Ask(ctx context.Context, csvData string, question string) (string, error) {
	systemInstruction := g.cfg.LLM.Prompt.SystemInstruction
	if systemInstruction == "" {
		systemInstruction = "你是一位資料分析師，根據提供的結構化資料回答問題。請使用繁體中文回答。回答時請具體引用資料數值。若資料不足以回答問題，請明確說明缺少哪些資料，不要猜測。"
	}

	// Build the prompt with data context
	userPrompt := fmt.Sprintf("資料 (CSV 格式):\n%s\n\n問題: %s", csvData, question)

	reqBody := GeminiRequest{
		SystemInstruction: &GeminiContent{
			Parts: []GeminiPart{{Text: systemInstruction}},
		},
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{{Text: userPrompt}},
				Role:  "user",
			},
		},
		GenerationConfig: &GeminiGenerationConfig{
			Temperature: 0.3,
			MaxTokens:   2048,
		},
	}

	answer, err := g.callAPI(ctx, reqBody)
	if err != nil {
		// Retry once on 5xx errors
		if is5xxError(err) {
			answer, err = g.callAPI(ctx, reqBody)
			if err != nil {
				return "", err
			}
			return answer, nil
		}
		return "", err
	}

	return answer, nil
}

// callAPI makes the actual HTTP call to Gemini
func (g *GeminiClient) callAPI(ctx context.Context, reqBody GeminiRequest) (string, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化 Gemini 請求失敗: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("建立 Gemini 請求失敗: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("呼叫 Gemini API 失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", &apiError{
			statusCode: resp.StatusCode,
			message:    fmt.Sprintf("Gemini API 伺服器錯誤 (HTTP %d): %s", resp.StatusCode, string(respBody)),
		}
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini API 回傳錯誤 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("解析 Gemini 回應失敗: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini 未返回有效回答")
	}

	// Concatenate all text parts
	var result strings.Builder
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		result.WriteString(part.Text)
	}

	return result.String(), nil
}

// apiError represents an API error with status code
type apiError struct {
	statusCode int
	message    string
}

func (e *apiError) Error() string {
	return e.message
}

// is5xxError checks if an error is a server-side error
func is5xxError(err error) bool {
	if apiErr, ok := err.(*apiError); ok {
		return apiErr.statusCode >= 500
	}
	return false
}
