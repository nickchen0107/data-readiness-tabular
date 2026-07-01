package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateToken_ReturnsValidToken(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret-key"
	expiry := 24 * time.Hour

	token, expiresAt, err := GenerateToken(userID, "user", secret, expiry)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(time.Now()))
	assert.True(t, expiresAt.Before(time.Now().Add(25*time.Hour)))
}

func TestValidateToken_ValidToken(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret-key"
	expiry := 1 * time.Hour

	token, _, err := GenerateToken(userID, "user", secret, expiry)
	require.NoError(t, err)

	parsedID, role, err := ValidateToken(token, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, parsedID)
	assert.Equal(t, "user", role)
}

func TestValidateToken_AdminRole(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret-key"
	expiry := 1 * time.Hour

	token, _, err := GenerateToken(userID, "admin", secret, expiry)
	require.NoError(t, err)

	parsedID, role, err := ValidateToken(token, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, parsedID)
	assert.Equal(t, "admin", role)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	userID := uuid.New()
	token, _, err := GenerateToken(userID, "user", "secret-1", 1*time.Hour)
	require.NoError(t, err)

	_, _, err = ValidateToken(token, "secret-2")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret-key"
	// Generate token that already expired
	token, _, err := GenerateToken(userID, "user", secret, -1*time.Hour)
	require.NoError(t, err)

	_, _, err = ValidateToken(token, secret)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestValidateToken_InvalidTokenString(t *testing.T) {
	_, _, err := ValidateToken("not-a-valid-jwt", "secret")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestGenerateToken_ContainsCorrectUserID(t *testing.T) {
	userID := uuid.New()
	secret := "my-secret"

	token, _, err := GenerateToken(userID, "user", secret, 1*time.Hour)
	require.NoError(t, err)

	recovered, role, err := ValidateToken(token, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, recovered)
	assert.Equal(t, "user", role)
}
