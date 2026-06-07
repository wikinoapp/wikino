// Package worker はバックグラウンドワーカー機能を提供します
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/email"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	wikinosentry "github.com/wikinoapp/wikino/go/internal/sentry"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Client は River クライアントのラッパー
type Client struct {
	riverClient *river.Client[pgx.Tx]
	pool        *pgxpool.Pool
}

// NewClient は新しい River クライアントを作成します
func NewClient(ctx context.Context, databaseURL string, cfg *config.Config, limiter *ratelimit.Limiter) (*Client, error) {
	// pgxpool の作成
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	// コネクションプール設定
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 5 * time.Minute
	poolConfig.MaxConnIdleTime = 2 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	// メール送信クライアントの作成
	var emailSender email.Sender
	if cfg.ResendAPIKey != "" {
		emailSender = email.NewResendSender(cfg.ResendAPIKey, cfg.ResendFromEmail, cfg.ResendFromName)
		slog.InfoContext(ctx, "Resend クライアントを初期化しました")
	} else {
		slog.WarnContext(ctx, "Resend API キーが設定されていません。メール送信機能は利用できません")
	}

	// River ワーカーの登録
	workers := river.NewWorkers()

	// メール送信ワーカーを登録
	if emailSender != nil {
		confirmationSender := email.NewConfirmationSender(emailSender)
		sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(confirmationSender)
		river.AddWorker(workers, NewSendEmailConfirmationWorker(sendEmailConfirmationUC))
		slog.InfoContext(ctx, "SendEmailConfirmationWorker を登録しました")

		passwordResetSender := email.NewPasswordResetSender(emailSender)
		sendPasswordResetUC := usecase.NewSendPasswordResetUsecase(passwordResetSender)
		river.AddWorker(workers, NewSendPasswordResetWorker(sendPasswordResetUC))
		slog.InfoContext(ctx, "SendPasswordResetWorker を登録しました")
	}

	// Rate Limit クリーンアップワーカーを登録
	cleanupRateLimitsUC := usecase.NewCleanupRateLimitsUsecase(limiter)
	river.AddWorker(workers, NewCleanupRateLimitsWorker(cleanupRateLimitsUC))
	slog.InfoContext(ctx, "CleanupRateLimitsWorker を登録しました")

	// River クライアントの作成
	// Wire the Sentry middleware via Config.Middleware. The deprecated
	// WorkerMiddleware field is avoided so future river upgrades that remove
	// it will not require revisiting this site.
	//
	// [Ja] Sentry ミドルウェアは Config.Middleware に登録する。
	// 将来 river のアップデートで削除される可能性のある WorkerMiddleware
	// フィールドは使わないことで、削除時の再対応を不要にする。
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
		Middleware: []rivertype.Middleware{
			wikinosentry.RiverWorkerMiddleware(),
		},
		Logger: slog.Default(),
	})
	if err != nil {
		pool.Close()
		return nil, err
	}

	return &Client{
		riverClient: riverClient,
		pool:        pool,
	}, nil
}

// Start は River クライアントを起動します
func (c *Client) Start(ctx context.Context) error {
	slog.InfoContext(ctx, "River クライアントを起動します")
	return c.riverClient.Start(ctx)
}

// Stop は River クライアントを停止します
func (c *Client) Stop(ctx context.Context) error {
	slog.InfoContext(ctx, "River クライアントを停止します")
	if err := c.riverClient.Stop(ctx); err != nil {
		return err
	}
	c.pool.Close()
	return nil
}

// Client は River クライアントへのアクセスを提供します
func (c *Client) Client() *river.Client[pgx.Tx] {
	return c.riverClient
}
