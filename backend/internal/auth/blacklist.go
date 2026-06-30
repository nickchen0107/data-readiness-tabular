package auth

import (
	"sync"
	"time"
)

// TokenBlacklist 管理已失效的 JWT token（MVP 使用 in-memory sync.Map）
type TokenBlacklist struct {
	store sync.Map // key: token string, value: expiry time
}

// NewTokenBlacklist 建立新的 token blacklist
func NewTokenBlacklist() *TokenBlacklist {
	return &TokenBlacklist{}
}

// Add 將 token 加入黑名單，附帶過期時間
func (bl *TokenBlacklist) Add(token string, expiry time.Time) {
	bl.store.Store(token, expiry)
}

// IsBlacklisted 檢查 token 是否在黑名單中（未過期）
func (bl *TokenBlacklist) IsBlacklisted(token string) bool {
	val, ok := bl.store.Load(token)
	if !ok {
		return false
	}
	expiry, ok := val.(time.Time)
	if !ok {
		return false
	}
	// 若已過期，從 blacklist 中移除並回傳 false
	if time.Now().After(expiry) {
		bl.store.Delete(token)
		return false
	}
	return true
}

// Cleanup 清除已過期的 token（可選擇定期呼叫）
func (bl *TokenBlacklist) Cleanup() {
	now := time.Now()
	bl.store.Range(func(key, value interface{}) bool {
		expiry, ok := value.(time.Time)
		if ok && now.After(expiry) {
			bl.store.Delete(key)
		}
		return true
	})
}
