package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wikinoapp/wikino/go/internal/ratelimit"
)

// CleanupRateLimitsUsecase は古いRate Limitレコードを削除するユースケース
type CleanupRateLimitsUsecase struct {
	limiter *ratelimit.Limiter
}

// NewCleanupRateLimitsUsecase は CleanupRateLimitsUsecase を生成する
func NewCleanupRateLimitsUsecase(limiter *ratelimit.Limiter) *CleanupRateLimitsUsecase {
	return &CleanupRateLimitsUsecase{
		limiter: limiter,
	}
}

// CleanupRateLimitsInput は古いRate Limitレコード削除の入力パラメータ
type CleanupRateLimitsInput struct {
	RetentionHours int
}

// Execute は古いRate Limitレコードを削除する
func (uc *CleanupRateLimitsUsecase) Execute(ctx context.Context, input CleanupRateLimitsInput) error {
	retention := time.Duration(input.RetentionHours) * time.Hour
	if retention <= 0 {
		retention = 24 * time.Hour
	}

	slog.InfoContext(ctx, "Rate Limitレコードのクリーンアップを開始します",
		"retention_hours", input.RetentionHours,
	)

	err := uc.limiter.CleanupOldRecords(ctx, retention)
	if err != nil {
		slog.ErrorContext(ctx, "Rate Limitレコードのクリーンアップに失敗しました",
			"error", err,
		)
		return fmt.Errorf("rate limitレコードのクリーンアップに失敗: %w", err)
	}

	slog.InfoContext(ctx, "Rate Limitレコードのクリーンアップが完了しました")
	return nil
}
