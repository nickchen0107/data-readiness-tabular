package config

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 應用程式配置結構（對應 config.yaml）
type Config struct {
	// 基礎設定（直接欄位，向後相容）
	DatabaseURL   string
	JWTSecret     string
	GeminiAPIKey  string
	BlockchainURL string
	ServerPort    string
	UploadDir     string

	// 完整結構化設定
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Auth       AuthConfig       `yaml:"auth"`
	Upload     UploadConfig     `yaml:"upload"`
	Assessment AssessmentConfig `yaml:"assessment"`
	LLM        LLMConfig        `yaml:"llm"`
	Blockchain BlockchainConfig `yaml:"blockchain"`
	Report     ReportConfig     `yaml:"report"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
	Mode string `yaml:"mode"`
}

type DatabaseConfig struct {
	URL             string `yaml:"url"`
	MaxConnections  int    `yaml:"max_connections"`
	MinConnections  int    `yaml:"min_connections"`
	MaxConnLifetime string `yaml:"max_conn_lifetime"`
	MaxConnIdleTime string `yaml:"max_conn_idle_time"`
}

type AuthConfig struct {
	JWTSecret string        `yaml:"jwt_secret"`
	JWTExpiry string        `yaml:"jwt_expiry"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

type RateLimitConfig struct {
	MaxAttempts     int    `yaml:"max_attempts"`
	Window          string `yaml:"window"`
	LockoutDuration string `yaml:"lockout_duration"`
}

type UploadConfig struct {
	Directory      string   `yaml:"directory"`
	MaxFileSizeMB  int      `yaml:"max_file_size_mb"`
	MaxRowCount    int      `yaml:"max_row_count"`
	AllowedFormats []string `yaml:"allowed_formats"`
}

type AssessmentConfig struct {
	DefaultWeights WeightsConfig      `yaml:"default_weights"`
	Levenshtein    LevenshteinConfig  `yaml:"levenshtein"`
	Grading        GradingConfig      `yaml:"grading"`
}

type WeightsConfig struct {
	RowCompleteness    float64 `yaml:"row_completeness"`
	ColumnCompleteness float64 `yaml:"column_completeness"`
	FormatConsistency  float64 `yaml:"format_consistency"`
	DuplicateSimilar   float64 `yaml:"duplicate_similar"`
	TableStructure     float64 `yaml:"table_structure"`
	AIQueryReadiness   float64 `yaml:"ai_query_readiness"`
}

type LevenshteinConfig struct {
	Threshold       int     `yaml:"threshold"`
	MaxColumns      int     `yaml:"max_columns"`
	MinCardinalPct  float64 `yaml:"min_cardinality_pct"`
	MaxCardinalPct  float64 `yaml:"max_cardinality_pct"`
}

type GradingConfig struct {
	ReadyThreshold       float64 `yaml:"ready_threshold"`
	ConditionalThreshold float64 `yaml:"conditional_threshold"`
}

type LLMConfig struct {
	Provider                    string       `yaml:"provider"`
	APIKey                      string       `yaml:"api_key"`
	Model                       string       `yaml:"model"`
	Timeout                     string       `yaml:"timeout"`
	MaxRetries                  int          `yaml:"max_retries"`
	DataInsufficiencyThreshold  float64      `yaml:"data_insufficiency_threshold"`
	Prompt                      PromptConfig `yaml:"prompt"`
}

type PromptConfig struct {
	SystemInstruction string `yaml:"system_instruction"`
	MaxDataRows       int    `yaml:"max_data_rows"`
}

type BlockchainConfig struct {
	APIURL       string `yaml:"api_url"`
	Timeout      string `yaml:"timeout"`
	FallbackMode string `yaml:"fallback_mode"`
	ToolVersion  string `yaml:"tool_version"`
	RuleVersion  string `yaml:"rule_version"`
}

type ReportConfig struct {
	Colors       ReportColors `yaml:"colors"`
	FontPath     string       `yaml:"font_path"`
	FontBoldPath string       `yaml:"font_bold_path"`
}

type ReportColors struct {
	Primary string `yaml:"primary"`
	Accent  string `yaml:"accent"`
	Green   string `yaml:"green"`
	Amber   string `yaml:"amber"`
	Rose    string `yaml:"rose"`
}

// Load 載入配置：先讀 config.yaml，再用環境變數覆蓋
func Load() *Config {
	cfg := &Config{}

	// 嘗試讀取 config.yaml
	configPaths := []string{"config.yaml", "/app/config.yaml", "./backend/config.yaml"}
	for _, path := range configPaths {
		if data, err := os.ReadFile(path); err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				log.Printf("警告：config.yaml 解析失敗: %v", err)
			} else {
				log.Printf("已載入配置檔：%s", path)
			}
			break
		}
	}

	// 環境變數覆蓋（優先權最高）
	cfg.DatabaseURL = getEnvOrDefault("DATABASE_URL", cfg.Database.URL, "")
	cfg.JWTSecret = getEnvOrDefault("JWT_SECRET", cfg.Auth.JWTSecret, "")
	cfg.GeminiAPIKey = getEnvOrDefault("GEMINI_API_KEY", cfg.LLM.APIKey, "")
	cfg.BlockchainURL = getEnvOrDefault("BLOCKCHAIN_API_URL", cfg.Blockchain.APIURL, "http://localhost:9000")
	cfg.ServerPort = getEnvOrDefault("SERVER_PORT", cfg.Server.Port, "8080")
	cfg.UploadDir = getEnvOrDefault("UPLOAD_DIR", cfg.Upload.Directory, "/app/uploads")

	// 設定預設值（若 yaml 和環境變數都沒給）
	setDefaults(cfg)

	return cfg
}

// GetJWTExpiry 取得 JWT 過期時間
func (c *Config) GetJWTExpiry() time.Duration {
	if c.Auth.JWTExpiry != "" {
		if d, err := time.ParseDuration(c.Auth.JWTExpiry); err == nil {
			return d
		}
	}
	return 24 * time.Hour
}

// GetLLMTimeout 取得 LLM 請求超時時間
func (c *Config) GetLLMTimeout() time.Duration {
	if c.LLM.Timeout != "" {
		if d, err := time.ParseDuration(c.LLM.Timeout); err == nil {
			return d
		}
	}
	return 30 * time.Second
}

// GetBlockchainTimeout 取得區塊鏈 API 超時時間
func (c *Config) GetBlockchainTimeout() time.Duration {
	if c.Blockchain.Timeout != "" {
		if d, err := time.ParseDuration(c.Blockchain.Timeout); err == nil {
			return d
		}
	}
	return 10 * time.Second
}

func getEnvOrDefault(envKey, yamlValue, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if yamlValue != "" {
		return yamlValue
	}
	return fallback
}

func setDefaults(cfg *Config) {
	if cfg.Upload.MaxFileSizeMB == 0 {
		cfg.Upload.MaxFileSizeMB = 50
	}
	if cfg.Upload.MaxRowCount == 0 {
		cfg.Upload.MaxRowCount = 100000
	}
	if cfg.Assessment.Levenshtein.Threshold == 0 {
		cfg.Assessment.Levenshtein.Threshold = 2
	}
	if cfg.Assessment.Levenshtein.MaxColumns == 0 {
		cfg.Assessment.Levenshtein.MaxColumns = 5
	}
	if cfg.Assessment.Grading.ReadyThreshold == 0 {
		cfg.Assessment.Grading.ReadyThreshold = 80.0
	}
	if cfg.Assessment.Grading.ConditionalThreshold == 0 {
		cfg.Assessment.Grading.ConditionalThreshold = 60.0
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = "gemini-2.0-flash"
	}
	if cfg.LLM.MaxRetries == 0 {
		cfg.LLM.MaxRetries = 1
	}
	if cfg.LLM.DataInsufficiencyThreshold == 0 {
		cfg.LLM.DataInsufficiencyThreshold = 0.50
	}
	if cfg.LLM.Prompt.MaxDataRows == 0 {
		cfg.LLM.Prompt.MaxDataRows = 500
	}
}

