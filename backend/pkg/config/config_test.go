package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults(t *testing.T) {
	// 清除可能影響的環境變數
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("GEMINI_API_KEY")
	os.Unsetenv("BLOCKCHAIN_API_URL")
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("UPLOAD_DIR")

	cfg := Load()

	assert.Equal(t, "", cfg.DatabaseURL)
	assert.Equal(t, "", cfg.JWTSecret)
	assert.Equal(t, "", cfg.GeminiAPIKey)
	assert.Equal(t, "http://localhost:9000", cfg.BlockchainURL)
	assert.Equal(t, "8080", cfg.ServerPort)
	assert.Equal(t, "/app/uploads", cfg.UploadDir)
}

func TestLoadFromEnvironment(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	os.Setenv("JWT_SECRET", "my-secret")
	os.Setenv("GEMINI_API_KEY", "gemini-key-123")
	os.Setenv("BLOCKCHAIN_API_URL", "http://blockchain:3000")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("UPLOAD_DIR", "/tmp/uploads")

	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("BLOCKCHAIN_API_URL")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("UPLOAD_DIR")
	}()

	cfg := Load()

	assert.Equal(t, "postgres://user:pass@localhost/db", cfg.DatabaseURL)
	assert.Equal(t, "my-secret", cfg.JWTSecret)
	assert.Equal(t, "gemini-key-123", cfg.GeminiAPIKey)
	assert.Equal(t, "http://blockchain:3000", cfg.BlockchainURL)
	assert.Equal(t, "9090", cfg.ServerPort)
	assert.Equal(t, "/tmp/uploads", cfg.UploadDir)
}

func TestGetEnvWithDefault(t *testing.T) {
	os.Unsetenv("NON_EXISTENT_VAR")
	assert.Equal(t, "default_val", getEnv("NON_EXISTENT_VAR", "default_val"))

	os.Setenv("EXISTING_VAR", "real_val")
	defer os.Unsetenv("EXISTING_VAR")
	assert.Equal(t, "real_val", getEnv("EXISTING_VAR", "default_val"))
}
