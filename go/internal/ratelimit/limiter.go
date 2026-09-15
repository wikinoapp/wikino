// Package ratelimitはRate Limiting機能を提供します
package ratelimit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ErrRateLimitExceededはRate Limitを超えた場合のエラー
var ErrRateLimitExceeded = errors.New("レート制限を超過しました")

// LimiterはPostgreSQLベースのRate Limiter
type Limiter struct {
	repo *repository.RateLimitRepository
}

// NewLimiterは新しいLimiterを作成する
func NewLimiter(repo *repository.RateLimitRepository) *Limiter {
	return &Limiter{repo: repo}
}

// WithTxはトランザクションを使用する新しいLimiterを返す
func (l *Limiter) WithTx(tx *sql.Tx) *Limiter {
	return &Limiter{repo: l.repo.WithTx(tx)}
}

// CheckInputはRate Limitチェックの入力パラメータ
type CheckInput struct {
	// Keyは識別子 (例: "ip:192.168.1.1", "email:user@example.com")
	Key string
	// Limitは許可される最大リクエスト数
	Limit int
	// WindowはRate Limitの時間枠
	Window time.Duration
}

// CheckResultはRate Limitチェックの結果
type CheckResult struct {
	// Allowedはリクエストが許可されたかどうか
	Allowed bool
	// Countは現在のウィンドウでのリクエスト数
	Count int
	// Remainingは残りの許可リクエスト数
	Remaining int
	// ResetAtはウィンドウがリセットされる時刻
	ResetAt time.Time
}

// CheckはRate Limitをチェックし、カウンターをインクリメントする
// リクエストが許可された場合はtrueを、Rate Limitを超えた場合はfalseを返す
func (l *Limiter) Check(ctx context.Context, input CheckInput) (*CheckResult, error) {
	if input.Key == "" {
		return nil, fmt.Errorf("keyが指定されていません")
	}
	if input.Limit <= 0 {
		return nil, fmt.Errorf("limitは正の値である必要があります")
	}
	if input.Window <= 0 {
		return nil, fmt.Errorf("windowは正の値である必要があります")
	}

	// 現在のウィンドウ開始時刻を計算 (時間枠の切り捨て)
	now := time.Now().UTC()
	windowStart := now.Truncate(input.Window)
	resetAt := windowStart.Add(input.Window)

	// カウンターをインクリメント (Repository経由)
	result, err := l.repo.Increment(ctx, repository.IncrementInput{
		Key:         input.Key,
		WindowStart: windowStart,
	})
	if err != nil {
		return nil, fmt.Errorf("レート制限のカウンターのインクリメントに失敗: %w", err)
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

// AllowはRate Limitをチェックし、許可されない場合はエラーを返す
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

// CleanupOldRecordsは古いRate Limitレコードを削除する
// retentionは保持期間 (この期間より古いレコードが削除される)
func (l *Limiter) CleanupOldRecords(ctx context.Context, retention time.Duration) error {
	cutoff := time.Now().UTC().Add(-retention)
	return l.repo.DeleteOldRecords(ctx, cutoff)
}

// IPKeyはIPアドレス用のキーを生成する
func IPKey(ip string) string {
	return fmt.Sprintf("ip:%s", ip)
}

// EmailKeyはメールアドレス用のキーを生成する
func EmailKey(email string) string {
	return fmt.Sprintf("email:%s", email)
}
