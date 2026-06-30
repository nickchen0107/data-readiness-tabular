package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenBlacklist_Add_And_IsBlacklisted(t *testing.T) {
	bl := NewTokenBlacklist()

	// Token 未加入前不在黑名單
	assert.False(t, bl.IsBlacklisted("token123"))

	// 加入黑名單（過期時間為 1 小時後）
	bl.Add("token123", time.Now().Add(1*time.Hour))
	assert.True(t, bl.IsBlacklisted("token123"))
}

func TestTokenBlacklist_ExpiredTokenNotBlacklisted(t *testing.T) {
	bl := NewTokenBlacklist()

	// 加入已過期的 token
	bl.Add("expired-token", time.Now().Add(-1*time.Hour))
	assert.False(t, bl.IsBlacklisted("expired-token"))
}

func TestTokenBlacklist_MultipleTokens(t *testing.T) {
	bl := NewTokenBlacklist()

	bl.Add("token-a", time.Now().Add(1*time.Hour))
	bl.Add("token-b", time.Now().Add(1*time.Hour))

	assert.True(t, bl.IsBlacklisted("token-a"))
	assert.True(t, bl.IsBlacklisted("token-b"))
	assert.False(t, bl.IsBlacklisted("token-c"))
}

func TestTokenBlacklist_Cleanup(t *testing.T) {
	bl := NewTokenBlacklist()

	bl.Add("active", time.Now().Add(1*time.Hour))
	bl.Add("expired", time.Now().Add(-1*time.Hour))

	bl.Cleanup()

	// active 仍在黑名單中
	assert.True(t, bl.IsBlacklisted("active"))
	// expired 已被清除（IsBlacklisted 也會自動清除，但 Cleanup 確保批次清除）
}
