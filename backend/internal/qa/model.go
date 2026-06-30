package qa

import "github.com/google/uuid"

// QARequest is the request body for the QA ask endpoint
type QARequest struct {
	SessionID uuid.UUID `json:"session_id" binding:"required"`
	Question  string    `json:"question" binding:"required"`
	Consent   bool      `json:"consent"`
}

// QAResponse contains both original and cleaned data answers
type QAResponse struct {
	OriginalAnswer string `json:"original_answer"`
	CleanedAnswer  string `json:"cleaned_answer"`
}

// SuggestionsResponse contains suggested questions
type SuggestionsResponse struct {
	Suggestions []string `json:"suggestions"`
}

// GeminiRequest is the structured request to the Gemini API
type GeminiRequest struct {
	Contents         []GeminiContent        `json:"contents"`
	SystemInstruction *GeminiContent        `json:"systemInstruction,omitempty"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

// GeminiContent represents a content block for the Gemini API
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

// GeminiPart represents a part within content
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiResponse is the response from the Gemini API
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

// GeminiCandidate represents a response candidate
type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

// GeminiGenerationConfig holds generation parameters
type GeminiGenerationConfig struct {
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"maxOutputTokens,omitempty"`
}

// GuardrailResult contains the result of data insufficiency check
type GuardrailResult struct {
	Blocked     bool   `json:"blocked"`
	Explanation string `json:"explanation,omitempty"`
}
