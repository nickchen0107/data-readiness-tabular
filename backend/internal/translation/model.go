package translation

import (
	"time"

	"github.com/google/uuid"
)

// Translation 翻譯項目
type Translation struct {
	ID        uuid.UUID `json:"id"`
	Locale    string    `json:"locale"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}
