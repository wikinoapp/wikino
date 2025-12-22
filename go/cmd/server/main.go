package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/health"
	"github.com/wikinoapp/wikino/go/internal/handler/manifest"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in_two_factor"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in_two_factor_recovery"
	"github.com/wikinoapp/wikino/go/internal/handler/user_session"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/turnstile"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func main() {
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

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(queries)

	// ユースケースを初期化
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)
	verifyTwoFactorUC := usecase.NewVerifyTwoFactorUsecase(userTwoFactorAuthRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// Turnstileクライアントを初期化
	turnstileClient := turnstile.NewClient(cfg.TurnstileSecretKey)

	// ミドルウェアを初期化
	authMiddleware := middleware.NewAuth(sessionMgr)
	csrfMiddleware := middleware.NewCSRF(cfg)

	// ハンドラーを初期化
	healthHandler := health.NewHandler()
	manifestHandler := manifest.NewHandler(cfg)
	signInHandler := sign_in.NewHandler(cfg)
	userSessionHandler := user_session.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		userPasswordRepo,
		userSessionRepo,
		createUserSessionUC,
		turnstileClient,
	)
	signInTwoFactorHandler := sign_in_two_factor.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		verifyTwoFactorUC,
		createUserSessionUC,
	)
	signInTwoFactorRecoveryHandler := sign_in_two_factor_recovery.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		verifyTwoFactorUC,
		createUserSessionUC,
	)

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

	// 未認証ユーザー専用ルート
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireNoAuth)
		r.Get("/sign_in", signInHandler.New)
		r.Post("/user_session", userSessionHandler.Create)
		r.Get("/sign_in/two_factor/new", signInTwoFactorHandler.New)
		r.Post("/sign_in/two_factor", signInTwoFactorHandler.Create)
		r.Get("/sign_in/two_factor/recovery/new", signInTwoFactorRecoveryHandler.New)
		r.Post("/sign_in/two_factor/recovery", signInTwoFactorRecoveryHandler.Create)
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

	if err := srv.ListenAndServe(); err != nil {
		slog.Error("サーバーの起動に失敗しました", "error", err)
		os.Exit(1)
	}
}
