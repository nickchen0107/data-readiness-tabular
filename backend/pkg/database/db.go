package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxRetries     = 3
	baseBackoff    = 1 * time.Second
	backoffFactor  = 2
)

// Connect 建立資料庫連線池，支援重試機制（3 次，指數退避：1s, 2s, 4s）
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		pool, err = pgxpool.New(ctx, databaseURL)
		if err == nil {
			// 驗證連線是否可用
			if pingErr := pool.Ping(ctx); pingErr == nil {
				log.Printf("資料庫連線成功 (第 %d 次嘗試)", attempt+1)
				return pool, nil
			} else {
				err = pingErr
				pool.Close()
			}
		}

		if attempt < maxRetries-1 {
			backoff := baseBackoff * time.Duration(pow(backoffFactor, attempt))
			log.Printf("資料庫連線失敗 (第 %d 次嘗試): %v，將於 %v 後重試", attempt+1, err, backoff)
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("資料庫連線失敗，已重試 %d 次: %w", maxRetries, err)
}

// pow 計算整數次冪
func pow(base, exp int) int {
	result := 1
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}
