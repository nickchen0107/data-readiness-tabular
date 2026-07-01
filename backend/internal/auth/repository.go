package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUserNotFound 使用者不存在
var ErrUserNotFound = errors.New("使用者不存在")

// ErrEmailAlreadyExists email 已被註冊
var ErrEmailAlreadyExists = errors.New("此 email 已被註冊")

// Repository 處理 auth 相關的資料庫操作
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository 建立新的 auth Repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateUser 建立新使用者，回傳建立的 User 或錯誤
func (r *Repository) CreateUser(ctx context.Context, email, passwordHash string) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1, $2, 'user')
		 RETURNING id, email, password_hash, role, created_at, updated_at`,
		email, passwordHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		// Check for unique constraint violation on email
		if isDuplicateKeyError(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail 根據 email 查詢使用者
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, role, created_at, updated_at FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByID 根據 ID 查詢使用者
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, role, created_at, updated_at FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// ListAll 分頁取得所有使用者，回傳使用者列表及總筆數
func (r *Repository) ListAll(ctx context.Context, offset, limit int) ([]User, int, error) {
	// 計算總筆數
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 查詢分頁結果
	rows, err := r.pool.Query(ctx,
		`SELECT id, email, password_hash, role, created_at, updated_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// isDuplicateKeyError 檢查是否為 unique constraint violation (PostgreSQL error code 23505)
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// pgx wraps PostgreSQL errors; check the error message for unique violation
	return contains(err.Error(), "23505") || contains(err.Error(), "duplicate key")
}

// contains 簡單字串包含檢查
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
