package upload

import (
	"testing"

	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestService_IsAllowedFormat(t *testing.T) {
	cfg := &config.Config{
		Upload: config.UploadConfig{
			AllowedFormats: []string{"xlsx", "csv"},
		},
	}
	svc := &Service{cfg: cfg}

	assert.True(t, svc.isAllowedFormat("xlsx"))
	assert.True(t, svc.isAllowedFormat("csv"))
	assert.True(t, svc.isAllowedFormat("CSV"))
	assert.True(t, svc.isAllowedFormat("XLSX"))
	assert.False(t, svc.isAllowedFormat("pdf"))
	assert.False(t, svc.isAllowedFormat("xls"))
	assert.False(t, svc.isAllowedFormat(""))
}

func TestService_GenerateStoragePath(t *testing.T) {
	cfg := &config.Config{
		UploadDir: "/app/uploads",
	}
	svc := &Service{cfg: cfg}

	id := uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	path := svc.generateStoragePath(id, "xlsx")

	assert.Contains(t, path, "/app/uploads/a1/a1b2c3d4-e5f6-7890-abcd-ef1234567890.xlsx")
}

func TestService_IsAllowedFormat_DefaultFormats(t *testing.T) {
	cfg := &config.Config{
		Upload: config.UploadConfig{
			AllowedFormats: nil, // no explicit formats
		},
	}
	svc := &Service{cfg: cfg}

	// Should default to xlsx and csv
	assert.True(t, svc.isAllowedFormat("xlsx"))
	assert.True(t, svc.isAllowedFormat("csv"))
	assert.False(t, svc.isAllowedFormat("pdf"))
}
