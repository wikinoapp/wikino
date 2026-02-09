// Package ratelimit はRate Limiting機能を提供します
package ratelimit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ErrRateLimitExceeded はRate Limitを超えた場合のエラー
var ErrRateLimitExceeded = errors.New("rate limit exceeded")

// Limiter はPostgreSQLベースのRate Limiter
type Limiter struct {
	repo *repository.RateLimitRepository
}

// NewLimiter は新しいLimiterを作成する
func NewLimiter(repo *repository.RateLimitRepository) *Limiter {
	return &Limiter{repo: repo}
}

// WithTx はトランザクションを使用する新しいLimiterを返す
func (l *Limiter) WithTx(tx *sql.Tx) *Limiter {
	return &Limiter{repo: l.repo.WithTx(tx)}
}

// CheckInput はRate Limitチェックの入力パラメータ
type CheckInput struct {
	// Key は識別子（例: "ip:192.168.1.1", "email:user@example.com"）
	Key string
	// Limit は許可される最大リクエスト数
	Limit int
	// Window はRate Limitの時間枠
	Window time.Duration
}

// CheckResult はRate Limitチェックの結果
type CheckResult struct {
	// Allowed はリクエストが許可されたかどうか
	Allowed bool
	// Count は現在のウィンドウでのリクエスト数
	Count int
	// Remaining は残りの許可リクエスト数
	Remaining int
	// ResetAt はウィンドウがリセットされる時刻
	ResetAt time.Time
}

// Check はRate Limitをチェックし、カウンターをインクリメントする
// リクエストが許可された場合はtrueを、Rate Limitを超えた場合はfalseを返す
func (l *Limiter) Check(ctx context.Context, input CheckInput) (*CheckResult, error) {
	if input.Key == "" {
		return nil, fmt.Errorf("key is required")
	}
	if input.Limit <= 0 {
		return nil, fmt.Errorf("limit must be positive")
	}
	if input.Window <= 0 {
		return nil, fmt.Errorf("window must be positive")
	}

	// 現在のウィンドウ開始時刻を計算（時間枠の切り捨て）
	now := time.Now().UTC()
	windowStart := now.Truncate(input.Window)
	resetAt := windowStart.Add(input.Window)

	// カウンターをインクリメント（Repository経由）
	result, err := l.repo.Increment(ctx, repository.IncrementInput{
		Key:         input.Key,
		WindowStart: windowStart,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to increment rate limit: %w", err)
	}

	count := int(result.Count)
	remaining := input.Limit - count
	if remaining < 0 {
		remaining = 0
	}

	return &CheckResult{
		Allowed:   count <= input.Limit,
		Count:     count,
		Remaining: remaining,
		ResetAt:   resetAt,
	}, nil
}

// Allow はRate Limitをチェックし、許可されない場合はエラーを返す
// 簡易的なAPIで、詳細な結果が不要な場合に使用
func (l *Limiter) Allow(ctx context.Context, input CheckInput) error {
	result, err := l.Check(ctx, input)
	if err != nil {
		return err
	}
	if !result.Allowed {
		return ErrRateLimitExceeded
	}
	return nil
}

// CleanupOldRecords は古いRate Limitレコードを削除する
// retentionは保持期間（この期間より古いレコードが削除される）
func (l *Limiter) CleanupOldRecords(ctx context.Context, retention time.Duration) error {
	cutoff := time.Now().UTC().Add(-retention)
	return l.repo.DeleteOldRecords(ctx, cutoff)
}

// IPKey はIPアドレス用のキーを生成する
func IPKey(ip string) string {
	return fmt.Sprintf("ip:%s", ip)
}

// EmailKey はメールアドレス用のキーを生成する
func EmailKey(email string) string {
	return fmt.Sprintf("email:%s", email)
}
