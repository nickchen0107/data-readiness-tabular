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

// ErrUsernameAlreadyExists 帳號已被註冊
var ErrUsernameAlreadyExists = errors.New("此帳號已被註冊")

// Repository 處理 auth 相關的資料庫操作
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository 建立新的 auth Repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateUser 建立新使用者，回傳建立的 User 或錯誤
func (r *Repository) CreateUser(ctx context.Context, username, email, passwordHash string) (*User, error) {
	var user User
	var emailVal *string
	if email != "" {
		emailVal = &email
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash, role) VALUES ($1, $2, $3, 'user')
		 RETURNING id, username, COALESCE(email, ''), password_hash, role, created_at, updated_at`,
		username, emailVal, passwordHash,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrUsernameAlreadyExists
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根據帳號查詢使用者
func (r *Repository) GetByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, COALESCE(email, ''), password_hash, role, created_at, updated_at FROM users WHERE username = $1`,
		username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail kept for backward compat — redirects to GetByUsername
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	return r.GetByUsername(ctx, email)
}

// GetByID 根據 ID 查詢使用者
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, COALESCE(email, ''), password_hash, role, created_at, updated_at FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)
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
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, username, COALESCE(email, ''), password_hash, role, created_at, updated_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// isDuplicateKeyError 檢查是否為 unique constraint violation
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "23505") || contains(err.Error(), "duplicate key")
}

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
