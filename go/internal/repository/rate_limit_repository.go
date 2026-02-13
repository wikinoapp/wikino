// Package repository はデータアクセス層を提供します
package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// RateLimitRepository はRate Limitのリポジトリ
type RateLimitRepository struct {
	q *query.Queries
}

// NewRateLimitRepository は RateLimitRepository を生成する
func NewRateLimitRepository(q *query.Queries) *RateLimitRepository {
	return &RateLimitRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *RateLimitRepository) WithTx(tx *sql.Tx) *RateLimitRepository {
	return &RateLimitRepository{q: r.q.WithTx(tx)}
}

// IncrementInput はRate Limitカウンターインクリメントの入力パラメータ
type IncrementInput struct {
	Key         string
	WindowStart time.Time
}

// IncrementResult はRate Limitカウンターインクリメントの結果
type IncrementResult struct {
	Count int32
}

// Increment はRate Limitカウンターをインクリメントする
func (r *RateLimitRepository) Increment(ctx context.Context, input IncrementInput) (*IncrementResult, error) {
	row, err := r.q.IncrementRateLimit(ctx, query.IncrementRateLimitParams{
		Key:         input.Key,
		WindowStart: input.WindowStart,
	})
	if err != nil {
		return nil, err
	}

	return &IncrementResult{
		Count: row.Count,
	}, nil
}

// DeleteOldRecords は指定された時刻より古いRate Limitレコードを削除する
func (r *RateLimitRepository) DeleteOldRecords(ctx context.Context, cutoff time.Time) error {
	return r.q.DeleteOldRateLimits(ctx, cutoff)
}
