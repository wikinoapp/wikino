package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/ratelimit"
)

// CleanupRateLimitsArgs は古いRate Limitレコード削除ジョブの引数です
type CleanupRateLimitsArgs struct {
	// Retention は保持期間（この期間より古いレコードが削除される）
	RetentionHours int `json:"retention_hours"`
}

// Kind はジョブの種類を返します
func (CleanupRateLimitsArgs) Kind() string {
	return "cleanup_rate_limits"
}

// InsertOpts はジョブのInsertオプションを返します
func (CleanupRateLimitsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       river.QueueDefault,
		MaxAttempts: 3,
	}
}

// CleanupRateLimitsWorker は古いRate Limitレコードを削除するワーカーです
type CleanupRateLimitsWorker struct {
	river.WorkerDefaults[CleanupRateLimitsArgs]
	limiter *ratelimit.Limiter
}

// NewCleanupRateLimitsWorker は新しいCleanupRateLimitsWorkerを作成します
func NewCleanupRateLimitsWorker(limiter *ratelimit.Limiter) *CleanupRateLimitsWorker {
	return &CleanupRateLimitsWorker{
		limiter: limiter,
	}
}

// Work は古いRate Limitレコードを削除します
func (w *CleanupRateLimitsWorker) Work(ctx context.Context, job *river.Job[CleanupRateLimitsArgs]) error {
	retention := time.Duration(job.Args.RetentionHours) * time.Hour
	if retention <= 0 {
		// デフォルトは24時間
		retention = 24 * time.Hour
	}

	slog.InfoContext(ctx, "Rate Limitレコードのクリーンアップを開始します",
		"retention_hours", job.Args.RetentionHours,
	)

	err := w.limiter.CleanupOldRecords(ctx, retention)
	if err != nil {
		slog.ErrorContext(ctx, "Rate Limitレコードのクリーンアップに失敗しました",
			"error", err,
		)
		return err
	}

	slog.InfoContext(ctx, "Rate Limitレコードのクリーンアップが完了しました")
	return nil
}
