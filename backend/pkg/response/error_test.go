package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestNewValidationError(t *testing.T) {
	resp := NewValidationError("密碼長度需介於 8 至 72 字元之間")

	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
	assert.Equal(t, "密碼長度需介於 8 至 72 字元之間", resp.Error.Message)
	assert.Nil(t, resp.Error.Details)
}

func TestNewAuthError(t *testing.T) {
	resp := NewAuthError("帳號或密碼錯誤")

	assert.Equal(t, "AUTH_ERROR", resp.Error.Code)
	assert.Equal(t, "帳號或密碼錯誤", resp.Error.Message)
	assert.Nil(t, resp.Error.Details)
}

func TestNewNotFoundError(t *testing.T) {
	resp := NewNotFoundError("找不到指定的資源")

	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	assert.Equal(t, "找不到指定的資源", resp.Error.Message)
	assert.Nil(t, resp.Error.Details)
}

func TestNewInternalError(t *testing.T) {
	resp := NewInternalError()

	assert.Equal(t, "INTERNAL_ERROR", resp.Error.Code)
	assert.Equal(t, "系統內部錯誤，請稍後重試", resp.Error.Message)
	assert.Nil(t, resp.Error.Details)
}

func TestNewInternalErrorNeverExposesDetails(t *testing.T) {
	resp := NewInternalError()

	// JSON 序列化後不應包含 details 欄位
	data, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.NotContains(t, string(data), "details")
}

func TestSendValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SendValidationError(c, "欄位格式不正確")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
	assert.Equal(t, "欄位格式不正確", resp.Error.Message)
}

func TestSendAuthError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SendAuthError(c, "認證失敗")

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "AUTH_ERROR", resp.Error.Code)
}

func TestSendNotFoundError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SendNotFoundError(c, "找不到該檔案")

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
}

func TestSendInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SendInternalError(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "INTERNAL_ERROR", resp.Error.Code)
	assert.Equal(t, "系統內部錯誤，請稍後重試", resp.Error.Message)
}

func TestErrorResponseJSONFormat(t *testing.T) {
	resp := NewValidationError("測試訊息")

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	// 驗證 JSON 結構符合設計規格
	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	assert.NoError(t, err)

	errorObj, ok := raw["error"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "VALIDATION_ERROR", errorObj["code"])
	assert.Equal(t, "測試訊息", errorObj["message"])
	_, hasDetails := errorObj["details"]
	assert.False(t, hasDetails, "details 欄位在空值時不應出現")
}
