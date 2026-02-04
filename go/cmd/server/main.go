package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/account"
	"github.com/wikinoapp/wikino/go/internal/handler/email_confirmation"
	"github.com/wikinoapp/wikino/go/internal/handler/health"
	"github.com/wikinoapp/wikino/go/internal/handler/manifest"
	"github.com/wikinoapp/wikino/go/internal/handler/password"
	"github.com/wikinoapp/wikino/go/internal/handler/password_reset"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in_two_factor"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in_two_factor_recovery"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_up"
	"github.com/wikinoapp/wikino/go/internal/handler/user_session"
	"github.com/wikinoapp/wikino/go/internal/handler/welcome"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/turnstile"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/worker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 設定を読み込む
	cfg, err := config.Load()
	if err != nil {
		slog.Error("設定の読み込みに失敗しました", "error", err)
		os.Exit(1)
	}

	// データベース接続
	db, err := sql.Open("postgres", cfg.DatabaseDSN())
	if err != nil {
		slog.Error("データベース接続に失敗しました", "error", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	// 接続確認
	if err := db.Ping(); err != nil {
		slog.Error("データベースへのpingに失敗しました", "error", err)
		os.Exit(1)
	}

	// クエリを初期化
	queries := query.New(db)

	// River クライアントを初期化（バックグラウンドジョブ用）
	riverClient, err := worker.NewClient(ctx, cfg.DatabaseURL, cfg)
	if err != nil {
		slog.Error("River クライアントの初期化に失敗しました", "error", err)
		os.Exit(1)
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		if err := riverClient.Stop(stopCtx); err != nil {
			slog.Error("River クライアントの停止に失敗しました", "error", err)
		}
	}()

	// River クライアントを起動
	if err := riverClient.Start(ctx); err != nil {
		slog.Error("River クライアントの起動に失敗しました", "error", err)
		os.Exit(1)
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(queries)

	// ユースケースを初期化
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)
	consumeRecoveryCodeUC := usecase.NewConsumeRecoveryCodeUsecase(userTwoFactorAuthRepo)
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, riverClient)
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)
	createAccountUC := usecase.NewCreateAccountUsecase(db, emailConfirmationRepo, userRepo, userPasswordRepo)
	createPasswordResetTokenUC := usecase.NewCreatePasswordResetTokenUsecase(cfg, db, passwordResetTokenRepo, riverClient)
	updatePasswordResetUC := usecase.NewUpdatePasswordResetUsecase(db, passwordResetTokenRepo, userPasswordRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// Turnstileクライアントを初期化
	turnstileClient := turnstile.NewClient(cfg.TurnstileSecretKey)

	// Rate Limiterを初期化
	rateLimitRepo := repository.NewRateLimitRepository(queries)
	rateLimiter := ratelimit.NewLimiter(rateLimitRepo)

	// ミドルウェアを初期化
	authMiddleware := middleware.NewAuth(sessionMgr)
	csrfMiddleware := middleware.NewCSRF(cfg)

	// ハンドラーを初期化
	healthHandler := health.NewHandler()
	manifestHandler := manifest.NewHandler(cfg)
	signInHandler := sign_in.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		userPasswordRepo,
		userSessionRepo,
		createUserSessionUC,
		turnstileClient,
	)
	userSessionHandler := user_session.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userSessionRepo,
	)
	signInTwoFactorValidator := sign_in_two_factor.NewCreateValidator(userTwoFactorAuthRepo)
	signInTwoFactorHandler := sign_in_two_factor.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		signInTwoFactorValidator,
		createUserSessionUC,
	)
	signInTwoFactorRecoveryValidator := sign_in_two_factor_recovery.NewCreateValidator(userTwoFactorAuthRepo)
	signInTwoFactorRecoveryHandler := sign_in_two_factor_recovery.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		signInTwoFactorRecoveryValidator,
		consumeRecoveryCodeUC,
		createUserSessionUC,
	)
	signUpHandler := sign_up.NewHandler(
		cfg,
		sessionMgr,
	)
	emailConfirmationHandler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		emailConfirmationRepo,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		turnstileClient,
		rateLimiter,
	)
	accountHandler := account.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		emailConfirmationRepo,
		userRepo,
		createAccountUC,
		createUserSessionUC,
	)
	passwordResetHandler := password_reset.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		rateLimiter,
		turnstileClient,
		createPasswordResetTokenUC,
	)
	passwordHandler := password.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		passwordResetTokenRepo,
		updatePasswordResetUC,
	)
	welcomeHandler := welcome.NewHandler(cfg, flashMgr)

	r := chi.NewRouter()

	// Method Overrideミドルウェア（HTMLフォームからDELETE/PATCH/PUTを使用可能にする）
	r.Use(middleware.MethodOverride)

	// リバースプロキシミドルウェアを初期化（Rails版へのプロキシ）
	// 注: RailsAppURLが設定されている場合のみ有効化
	if cfg.RailsAppURL != "" {
		reverseProxyMiddleware, err := middleware.NewReverseProxyMiddleware(cfg.RailsAppURL, cfg)
		if err != nil {
			slog.Error("リバースプロキシミドルウェアの初期化に失敗しました", "error", err)
			os.Exit(1)
		}
		r.Use(reverseProxyMiddleware.Middleware)
	}

	// 共通ミドルウェア
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(i18n.Middleware)
	r.Use(csrfMiddleware.Middleware)

	// 静的ファイルの配信 (Tailwind CLI + esbuild のビルド結果)
	fileServer := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	// ヘルスチェック（認証不要）
	r.Get("/health", healthHandler.Show)

	// Web App Manifest（認証不要）
	r.Get("/manifest.json", manifestHandler.Show)

	// トップページ（ログイン状態に応じてハンドラー内でリダイレクト）
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.SetUser)
		r.Get("/", welcomeHandler.Show)
	})

	// 未認証ユーザー専用ルート
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireNoAuth)
		r.Get("/sign_in", signInHandler.New)
		r.Post("/sign_in", signInHandler.Create)
		r.Get("/sign_in/two_factor/new", signInTwoFactorHandler.New)
		r.Post("/sign_in/two_factor", signInTwoFactorHandler.Create)
		r.Get("/sign_in/two_factor/recovery/new", signInTwoFactorRecoveryHandler.New)
		r.Post("/sign_in/two_factor/recovery", signInTwoFactorRecoveryHandler.Create)
		r.Get("/sign_up", signUpHandler.New)
		r.Post("/email_confirmation", emailConfirmationHandler.Create)
		r.Get("/email_confirmation/edit", emailConfirmationHandler.Edit)
		r.Patch("/email_confirmation", emailConfirmationHandler.Update)
		r.Get("/accounts/new", accountHandler.New)
		r.Post("/accounts", accountHandler.Create)
		r.Get("/password/reset", passwordResetHandler.New)
		r.Post("/password/reset", passwordResetHandler.Create)
		r.Get("/password/edit", passwordHandler.Edit)
		r.Patch("/password", passwordHandler.Update)
	})

	// 認証済みユーザー専用ルート
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)
		r.Delete("/user_session", userSessionHandler.Delete)
	})

	addr := fmt.Sprintf("0.0.0.0:%s", cfg.Port)
	slog.Info("HTTPサーバーを起動します", "addr", addr, "env", cfg.Env)

	srv := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// グレースフルシャットダウン
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		slog.Info("シャットダウンシグナルを受信しました")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("サーバーのシャットダウンに失敗しました", "error", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("サーバーの起動に失敗しました", "error", err)
		os.Exit(1)
	}

	slog.Info("サーバーを停止しました")
}
