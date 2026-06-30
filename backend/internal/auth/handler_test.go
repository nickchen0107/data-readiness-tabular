package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepository 模擬 Repository 用於 handler 測試
type mockRepository struct {
	users map[string]*User
}

func newMockRepo() *mockRepository {
	return &mockRepository{users: make(map[string]*User)}
}

func (m *mockRepository) createUser(ctx context.Context, email, passwordHash string) (*User, error) {
	if _, exists := m.users[email]; exists {
		return nil, ErrEmailAlreadyExists
	}
	user := &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.users[email] = user
	return user, nil
}

func (m *mockRepository) getByEmail(ctx context.Context, email string) (*User, error) {
	user, exists := m.users[email]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// testService 建立使用 mock repo 的 service (for handler unit tests)
type testService struct {
	mockRepo *mockRepository
	secret   string
}

func setupTestRouter(svc *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(svc)
	r.POST("/api/auth/register", h.Register)
	r.POST("/api/auth/login", h.Login)
	return r
}

// newTestServiceWithUser creates a service with a pre-registered user
func newTestServiceWithUser(email, password string) (*Service, *mockRepository) {
	mock := newMockRepo()
	hash, _ := HashPassword(password)
	mock.users[email] = &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	// We can't easily use the mock with the real service since it takes *Repository
	// So these tests exercise the handler error handling paths using integration-style tests
	return nil, mock
}

func TestHandler_Register_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Create a service with nil repo - we won't reach the repo since validation fails first
	svc := &Service{jwtSecret: "secret", jwtExpiry: 24 * time.Hour}
	r := setupTestRouter(svc)

	body := bytes.NewBufferString(`{"invalid json`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp, "error")
}

func TestHandler_Register_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &Service{jwtSecret: "secret", jwtExpiry: 24 * time.Hour}
	r := setupTestRouter(svc)

	body := bytes.NewBufferString(`{"email": ""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Login_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &Service{jwtSecret: "secret", jwtExpiry: 24 * time.Hour}
	r := setupTestRouter(svc)

	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Login_MissingPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &Service{jwtSecret: "secret", jwtExpiry: 24 * time.Hour}
	r := setupTestRouter(svc)

	body := bytes.NewBufferString(`{"email": "test@test.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
