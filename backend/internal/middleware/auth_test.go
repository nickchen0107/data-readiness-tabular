package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/internal/auth"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter(secret string, blacklist *auth.TokenBlacklist) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(JWTAuth(secret, blacklist))
	r.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		role, _ := c.Get("user_role")
		c.JSON(http.StatusOK, gin.H{"user_id": userID, "role": role})
	})
	return r
}

func TestJWTAuth_NoAuthHeader(t *testing.T) {
	bl := auth.NewTokenBlacklist()
	r := setupTestRouter("test-secret", bl)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "未提供認證令牌")
}

func TestJWTAuth_InvalidFormat(t *testing.T) {
	bl := auth.NewTokenBlacklist()
	r := setupTestRouter("test-secret", bl)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "無效的認證令牌")
}

func TestJWTAuth_BlacklistedToken(t *testing.T) {
	secret := "test-secret"
	bl := auth.NewTokenBlacklist()
	r := setupTestRouter(secret, bl)

	// 產生 valid token
	userID := uuid.New()
	token, _, _ := auth.GenerateToken(userID, "user", secret, 24*time.Hour)

	// 加入黑名單
	bl.Add(token, time.Now().Add(24*time.Hour))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "token 已失效")
}

func TestJWTAuth_ValidToken(t *testing.T) {
	secret := "test-secret"
	bl := auth.NewTokenBlacklist()
	r := setupTestRouter(secret, bl)

	userID := uuid.New()
	token, _, _ := auth.GenerateToken(userID, "user", secret, 24*time.Hour)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), userID.String())
}

func TestJWTAuth_ValidToken_SetsRole(t *testing.T) {
	secret := "test-secret"
	bl := auth.NewTokenBlacklist()
	r := setupTestRouter(secret, bl)

	userID := uuid.New()
	token, _, _ := auth.GenerateToken(userID, "admin", secret, 24*time.Hour)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "admin")
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	bl := auth.NewTokenBlacklist()
	r := setupTestRouter(secret, bl)

	userID := uuid.New()
	// 產生已過期的 token（使用負數 duration）
	token, _, _ := auth.GenerateToken(userID, "user", secret, -1*time.Hour)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "token 已過期")
}

func TestJWTAuth_WrongSecret(t *testing.T) {
	bl := auth.NewTokenBlacklist()
	r := setupTestRouter("correct-secret", bl)

	userID := uuid.New()
	token, _, _ := auth.GenerateToken(userID, "user", "wrong-secret", 24*time.Hour)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "無效的認證令牌")
}
