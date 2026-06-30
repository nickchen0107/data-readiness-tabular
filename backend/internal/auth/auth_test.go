package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Password Tests ---

func TestHashPassword_ProducesBcryptHash(t *testing.T) {
	hash, err := HashPassword("validPass1")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$"))
}

func TestHashPassword_DifferentHashesForSameInput(t *testing.T) {
	h1, err := HashPassword("samePassword")
	require.NoError(t, err)
	h2, err := HashPassword("samePassword")
	require.NoError(t, err)
	assert.NotEqual(t, h1, h2)
}

func TestCheckPassword_CorrectPassword(t *testing.T) {
	hash, err := HashPassword("myPassword123")
	require.NoError(t, err)
	assert.NoError(t, CheckPassword(hash, "myPassword123"))
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, err := HashPassword("myPassword123")
	require.NoError(t, err)
	assert.Error(t, CheckPassword(hash, "wrongPassword"))
}

// --- Validation Tests ---

func TestValidateEmail_ValidEmails(t *testing.T) {
	validEmails := []string{
		"user@example.com",
		"test.user@domain.co.jp",
		"name+tag@gmail.com",
	}
	for _, email := range validEmails {
		assert.NoError(t, ValidateEmail(email), "expected valid: %s", email)
	}
}

func TestValidateEmail_InvalidEmails(t *testing.T) {
	invalidEmails := []string{
		"",
		"notanemail",
		"@domain.com",
		"user@",
		"user @example.com",
	}
	for _, email := range invalidEmails {
		assert.ErrorIs(t, ValidateEmail(email), ErrInvalidEmail, "expected invalid: %q", email)
	}
}

func TestValidatePassword_ValidLength(t *testing.T) {
	assert.NoError(t, ValidatePassword("12345678"))       // exactly 8
	assert.NoError(t, ValidatePassword(strings.Repeat("a", 72))) // exactly 72
	assert.NoError(t, ValidatePassword("normalPassword")) // 14 chars
}

func TestValidatePassword_TooShort(t *testing.T) {
	assert.ErrorIs(t, ValidatePassword(""), ErrPasswordTooShort)
	assert.ErrorIs(t, ValidatePassword("1234567"), ErrPasswordTooShort) // 7 chars
}

func TestValidatePassword_TooLong(t *testing.T) {
	assert.ErrorIs(t, ValidatePassword(strings.Repeat("a", 73)), ErrPasswordTooLong)
	assert.ErrorIs(t, ValidatePassword(strings.Repeat("b", 100)), ErrPasswordTooLong)
}
