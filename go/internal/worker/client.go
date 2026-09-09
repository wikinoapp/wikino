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
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	wikinosentry "github.com/wikinoapp/wikino/go/internal/sentry"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// rescueStuckJobsAfter is how long River waits before it takes a job back from a worker that stopped
// reporting. River requires it to exceed the timeout any worker gives one attempt, and the longest
// of those is the export worker's, so it is derived from the same constant. Leaving the two to be
// chosen separately gives a configuration River refuses to start with.
//
// [Ja] rescueStuckJobsAfter は、報告が途絶えたワーカーからジョブを River が取り戻すまでの時間。
// River はこの値が、どのワーカーが 1 回の試行に与えるタイムアウトよりも大きいことを要求する。その
// うち最も長いのがエクスポートのワーカーのものなので、同じ定数から決める。両者を別々に選べるように
// すると、River が起動を拒否する設定ができてしまう。
const rescueStuckJobsAfter = model.ExportAttemptTimeout + time.Hour

// ExportDeps carries what the export worker needs but cannot make for itself. Repositories and the
// object storage are shared with the HTTP side, so they are handed in rather than built a second
// time here; the mail sender and the UseCase are the worker's alone and stay inside NewClient.
//
// A zero ExportDeps turns the export worker off, which is what a deployment without object storage
// configured gets: an export it cannot upload is better left unstarted.
//
// [Ja] ExportDeps は、エクスポートのワーカーが必要としながら自分では用意できないものを保持する。
// リポジトリとオブジェクトストレージは HTTP 側と共有するため、ここで作り直さず受け取る。メール
// 送信と UseCase はワーカーだけのものなので NewClient の中に留める。
//
// ゼロ値の ExportDeps はエクスポートのワーカーを無効にする。オブジェクトストレージが設定されて
// いないデプロイがこれにあたる。アップロードできないエクスポートは、開始できないほうがよい。
type ExportDeps struct {
	ExportRepo      *repository.ExportRepository
	SpaceRepo       *repository.SpaceRepository
	SpaceMemberRepo *repository.SpaceMemberRepository
	UserRepo        *repository.UserRepository
	TopicRepo       *repository.TopicRepository
	PageRepo        *repository.PageRepository
	AttachmentRepo  *repository.AttachmentRepository
	ObjectStorage   storage.ObjectStorage
}

// Client は River クライアントのラッパー
type Client struct {
	riverClient *river.Client[pgx.Tx]
	pool        *pgxpool.Pool
}

// NewClient は新しい River クライアントを作成します
func NewClient(ctx context.Context, databaseURL string, cfg *config.Config, limiter *ratelimit.Limiter, exportDeps ExportDeps) (*Client, error) {
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

	// The export worker is registered on the object storage alone. Its mails tell the member how a
	// generation ended, but an export that produces its archive without announcing it is still worth
	// more than no export at all, which is what a deployment with no mail configured would get.
	//
	// [Ja] エクスポートのワーカーはオブジェクトストレージの有無だけで登録する。メールは生成の結果を
	// メンバーへ伝えるものだが、アーカイブを作れて知らせられないエクスポートのほうが、エクスポート
	// 自体が無いよりは価値がある。メールの設定が無いデプロイでは後者になってしまう。
	if exportDeps.ObjectStorage != nil {
		var exportSender usecase.ExportSender
		if emailSender != nil {
			exportSender = email.NewExportSender(emailSender)
		}
		river.AddWorker(workers, NewGenerateExportFilesWorker(newGenerateExportFilesUsecase(cfg, exportDeps, exportSender)))
		slog.InfoContext(ctx, "GenerateExportFilesWorker を登録しました")
	} else {
		slog.WarnContext(ctx, "オブジェクトストレージが設定されていません。スペースのエクスポート機能は利用できません")
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

		RescueStuckJobsAfter: rescueStuckJobsAfter,
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

// newGenerateExportFilesUsecase assembles the export UseCase from the parts handed to the worker
// client and the mail sender it builds for itself.
//
// [Ja] newGenerateExportFilesUsecase は、ワーカークライアントが受け取った部品と、自身で用意した
// メール送信からエクスポートの UseCase を組み立てる。
func newGenerateExportFilesUsecase(cfg *config.Config, deps ExportDeps, sender usecase.ExportSender) *usecase.GenerateExportFilesUsecase {
	return usecase.NewGenerateExportFilesUsecase(
		cfg,
		deps.ExportRepo,
		deps.SpaceRepo,
		deps.SpaceMemberRepo,
		deps.UserRepo,
		deps.TopicRepo,
		deps.PageRepo,
		deps.AttachmentRepo,
		deps.ObjectStorage,
		sender,
	)
}
