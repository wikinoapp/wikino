// Package workerはバックグラウンドワーカー機能を提供します
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
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/email"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	wikinosentry "github.com/wikinoapp/wikino/go/internal/sentry"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// rescueStuckJobsAfterは、報告が途絶えたワーカーからジョブをRiverが取り戻すまでの時間。
// Riverはこの値が、どのワーカーが1回の試行に与えるタイムアウトよりも大きいことを要求する。その
// うち最も長いのがエクスポートのワーカーのものなので、同じ定数から決める。両者を別々に選べるように
// すると、Riverが起動を拒否する設定ができてしまう。
const rescueStuckJobsAfter = model.ExportAttemptTimeout + time.Hour

// rateLimitRetentionHoursは、レート制限の記録を残す時間。プライバシーポリシーに書く保持期間 (1日)
// に合わせる。レート制限の窓がこれより長いと判定中の記録を消してしまうため、窓はこれより短く保つ。
const rateLimitRetentionHours = 24

// cleanupRateLimitsIntervalは、古いレート制限の記録を消すジョブを投入する間隔。
const cleanupRateLimitsInterval = time.Hour

// emailConfirmationRetentionHoursは、メール確認の記録を開始から残す時間。プライバシーポリシーに書く
// 保持期間 (1日) に合わせる。確認コードの有効期限 (15分) より長く保ち、確認の途中の記録を消さない。
const emailConfirmationRetentionHours = 24

// cleanupEmailConfirmationsIntervalは、古いメール確認の記録を消すジョブを投入する間隔。
const cleanupEmailConfirmationsInterval = time.Hour

// ExportDepsは、エクスポートのワーカーが必要としながら自分では用意できないものを保持する。
// リポジトリとオブジェクトストレージはHTTP側と共有するため、ここで作り直さず受け取る。メール
// 送信とUseCaseはワーカーだけのものなのでNewClientの中に留める。
//
// ゼロ値のExportDepsはエクスポートのワーカーを無効にする。オブジェクトストレージが設定されて
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

// ClientはRiverクライアントのラッパー
type Client struct {
	riverClient *river.Client[pgx.Tx]
	pool        *pgxpool.Pool
}

// NewClientは新しいRiverクライアントを作成します
func NewClient(ctx context.Context, databaseURL string, cfg *config.Config, limiter *ratelimit.Limiter, emailConfirmationRepo *repository.EmailConfirmationRepository, exportDeps ExportDeps) (*Client, error) {
	// pgxpoolの作成
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
		slog.InfoContext(ctx, "Resendクライアントを初期化しました")
	} else {
		slog.WarnContext(ctx, "Resend APIキーが設定されていません。メール送信機能は利用できません")
	}

	// Riverワーカーの登録
	workers := river.NewWorkers()

	// メール送信ワーカーを登録
	if emailSender != nil {
		confirmationSender := email.NewConfirmationSender(emailSender)
		sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(confirmationSender)
		river.AddWorker(workers, NewSendEmailConfirmationWorker(sendEmailConfirmationUC))
		slog.InfoContext(ctx, "SendEmailConfirmationWorkerを登録しました")

		passwordResetSender := email.NewPasswordResetSender(emailSender)
		sendPasswordResetUC := usecase.NewSendPasswordResetUsecase(passwordResetSender)
		river.AddWorker(workers, NewSendPasswordResetWorker(sendPasswordResetUC))
		slog.InfoContext(ctx, "SendPasswordResetWorkerを登録しました")
	}

	// エクスポートのワーカーはオブジェクトストレージの有無だけで登録する。メールは生成の結果を
	// メンバーへ伝えるものだが、アーカイブを作れて知らせられないエクスポートのほうが、エクスポート
	// 自体が無いよりは価値がある。メールの設定が無いデプロイでは後者になってしまう。
	if exportDeps.ObjectStorage != nil {
		var exportSender usecase.ExportSender
		if emailSender != nil {
			exportSender = email.NewExportSender(emailSender)
		}
		river.AddWorker(workers, NewGenerateExportFilesWorker(newGenerateExportFilesUsecase(cfg, exportDeps, exportSender)))
		slog.InfoContext(ctx, "GenerateExportFilesWorkerを登録しました")
	} else {
		slog.WarnContext(ctx, "オブジェクトストレージが設定されていません。スペースのエクスポート機能は利用できません")
	}

	// Rate Limitクリーンアップワーカーを登録
	cleanupRateLimitsUC := usecase.NewCleanupRateLimitsUsecase(limiter)
	river.AddWorker(workers, NewCleanupRateLimitsWorker(cleanupRateLimitsUC))
	slog.InfoContext(ctx, "CleanupRateLimitsWorkerを登録しました")

	cleanupEmailConfirmationsUC := usecase.NewCleanupEmailConfirmationsUsecase(emailConfirmationRepo)
	river.AddWorker(workers, NewCleanupEmailConfirmationsWorker(cleanupEmailConfirmationsUC))
	slog.InfoContext(ctx, "CleanupEmailConfirmationsWorkerを登録しました")

	// Riverクライアントの作成
	// SentryミドルウェアはConfig.Middlewareに登録する。
	// 将来riverのアップデートで削除される可能性のあるWorkerMiddleware
	// フィールドは使わないことで、削除時の再対応を不要にする。
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers:      workers,
		PeriodicJobs: toPeriodicJobs(periodicJobSpecs()),

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

// StartはRiverクライアントを起動します
func (c *Client) Start(ctx context.Context) error {
	slog.InfoContext(ctx, "Riverクライアントを起動します")
	return c.riverClient.Start(ctx)
}

// StopはRiverクライアントを停止します
func (c *Client) Stop(ctx context.Context) error {
	slog.InfoContext(ctx, "Riverクライアントを停止します")
	if err := c.riverClient.Stop(ctx); err != nil {
		return err
	}
	c.pool.Close()
	return nil
}

// ClientはRiverクライアントへのアクセスを提供します
func (c *Client) Client() *river.Client[pgx.Tx] {
	return c.riverClient
}

// newGenerateExportFilesUsecaseは、ワーカークライアントが受け取った部品と、自身で用意した
// メール送信からエクスポートのUseCaseを組み立てる。
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

// periodicJobSpecは、定期的に投入するジョブ1件分の設定。river.PeriodicJobは投入する引数と間隔を
// 非公開のフィールドに持つため、先にこの形で宣言してからriverの型へ変換する。何をどの間隔で投入
// するかを、テストから確かめられるようにするためである。
type periodicJobSpec struct {
	// argsは、投入のたびに引数の型が宣言するInsertOptsを渡すため、InsertOptsを持つ型に限る。
	args interface {
		river.JobArgs
		river.JobArgsWithInsertOpts
	}
	interval time.Duration
}

// periodicJobSpecsは、ワーカーのクライアントが定期的に投入するジョブを返す。ここに足すジョブの
// ワーカーは、設定によらず登録されるものに限る。登録されていないワーカーのジョブを投入すると、
// 誰も処理できないジョブが溜まり続けるためである。
func periodicJobSpecs() []periodicJobSpec {
	return []periodicJobSpec{
		{
			args:     dispatcher.CleanupRateLimitsArgs{RetentionHours: rateLimitRetentionHours},
			interval: cleanupRateLimitsInterval,
		},
		{
			args:     dispatcher.CleanupEmailConfirmationsArgs{RetentionHours: emailConfirmationRetentionHours},
			interval: cleanupEmailConfirmationsInterval,
		},
	}
}

// toPeriodicJobsは、定期的に投入するジョブの設定をriverの定期ジョブへ変換する。
//
// どのジョブも、リーダーに選ばれたときに1件投入する (RunOnStart)。定期実行の予定はリーダーの
// メモリにしか無いため、これが無いと、間隔より短い周期でデプロイが続く間は投入が先送りされ続ける。
func toPeriodicJobs(specs []periodicJobSpec) []*river.PeriodicJob {
	jobs := make([]*river.PeriodicJob, 0, len(specs))
	for _, spec := range specs {
		jobs = append(jobs, river.NewPeriodicJob(
			river.PeriodicInterval(spec.interval),
			func() (river.JobArgs, *river.InsertOpts) {
				opts := spec.args.InsertOpts()
				return spec.args, &opts
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		))
	}
	return jobs
}
