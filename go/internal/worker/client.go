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

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/email"
	"github.com/wikinoapp/wikino/go/internal/i18n"
)

// Client は River クライアントのラッパー
type Client struct {
	riverClient *river.Client[pgx.Tx]
	pool        *pgxpool.Pool
}

// NewClient は新しい River クライアントを作成します
func NewClient(ctx context.Context, databaseURL string, cfg *config.Config) (*Client, error) {
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

	// メール確認送信ワーカーを登録
	if emailSender != nil {
		river.AddWorker(workers, NewSendEmailConfirmationWorker(emailSender))
		slog.InfoContext(ctx, "SendEmailConfirmationWorker を登録しました")
	}

	// River クライアントの作成
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
		Logger:  slog.Default(),
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

// EnqueueEmailConfirmationInput はメール確認送信ジョブのエンキュー入力
type EnqueueEmailConfirmationInput struct {
	Email  string
	Code   string
	AppURL string
	Locale string
}

// EnqueueEmailConfirmation はメール確認送信ジョブをエンキューします
func (c *Client) EnqueueEmailConfirmation(ctx context.Context, input EnqueueEmailConfirmationInput) error {
	// メール件名を取得
	subject := i18n.T(ctx, "email_confirmation_subject")

	_, err := c.riverClient.Insert(ctx, SendEmailConfirmationArgs{
		Email:   input.Email,
		Code:    input.Code,
		AppURL:  input.AppURL,
		Subject: subject,
		Locale:  input.Locale,
	}, &river.InsertOpts{
		Queue: river.QueueDefault,
	})
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "メール確認送信ジョブをエンキューしました",
		"email", input.Email,
	)

	return nil
}
