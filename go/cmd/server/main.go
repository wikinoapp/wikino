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
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/handler/account"
	"github.com/wikinoapp/wikino/go/internal/handler/attachment_og_image"
	"github.com/wikinoapp/wikino/go/internal/handler/draft_page"
	"github.com/wikinoapp/wikino/go/internal/handler/draft_page_index"
	"github.com/wikinoapp/wikino/go/internal/handler/draft_page_revision"
	"github.com/wikinoapp/wikino/go/internal/handler/email_confirmation"
	"github.com/wikinoapp/wikino/go/internal/handler/health"
	"github.com/wikinoapp/wikino/go/internal/handler/home"
	"github.com/wikinoapp/wikino/go/internal/handler/manifest"
	"github.com/wikinoapp/wikino/go/internal/handler/page"
	"github.com/wikinoapp/wikino/go/internal/handler/page_backlink_list"
	"github.com/wikinoapp/wikino/go/internal/handler/page_backlinks"
	"github.com/wikinoapp/wikino/go/internal/handler/page_link_list"
	"github.com/wikinoapp/wikino/go/internal/handler/page_location"
	"github.com/wikinoapp/wikino/go/internal/handler/page_move"
	"github.com/wikinoapp/wikino/go/internal/handler/password"
	"github.com/wikinoapp/wikino/go/internal/handler/password_reset"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in_two_factor"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in_two_factor_recovery"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_up"
	suggestionhandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion"
	suggestionapplyhandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion_apply"
	suggestionchangehandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion_change"
	suggestionclosehandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion_close"
	suggestioncommenthandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion_comment"
	suggestioncommentedithandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion_comment_edit"
	suggestionpagehandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion_page"
	suggestionpageedithandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion_page_edit"
	topichandler "github.com/wikinoapp/wikino/go/internal/handler/topic"
	"github.com/wikinoapp/wikino/go/internal/handler/user_session"
	"github.com/wikinoapp/wikino/go/internal/handler/welcome"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/image"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/turnstile"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
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

	// Rate Limiterを初期化
	rateLimitRepo := repository.NewRateLimitRepository(queries)
	rateLimiter := ratelimit.NewLimiter(rateLimitRepo)

	// River クライアントを初期化（バックグラウンドジョブ用）
	riverClient, err := worker.NewClient(ctx, cfg.DatabaseURL, cfg, rateLimiter)
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
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	attachmentRepo := repository.NewAttachmentRepository(queries)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(queries)
	pageRevisionRepo := repository.NewPageRevisionRepository(queries)
	pageEditorRepo := repository.NewPageEditorRepository(queries)
	featureFlagRepo := repository.NewFeatureFlagRepository(queries)
	suggestionRepo := repository.NewSuggestionRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	suggestionPageRevisionRepo := repository.NewSuggestionPageRevisionRepository(queries)
	suggestionCommentRepo := repository.NewSuggestionCommentRepository(queries)

	// Dispatcher を初期化（ジョブキューへの投入を抽象化）
	jobDispatcher := dispatcher.NewDispatcher(riverClient.Client())

	// ユースケースを初期化
	signInCreateValidator := validator.NewSignInCreateValidator(userRepo, userPasswordRepo, userTwoFactorAuthRepo)
	signInUC := usecase.NewCreateSignInUsecase(signInCreateValidator, userSessionRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	emailConfirmationCreateValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	emailConfirmationUpdateValidator := validator.NewEmailConfirmationUpdateValidator(emailConfirmationRepo)
	createEmailConfirmationUC := usecase.NewCreateEmailConfirmationUsecase(cfg, emailConfirmationRepo, jobDispatcher, emailConfirmationCreateValidator)
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo, emailConfirmationUpdateValidator)
	accountCreateValidator := validator.NewAccountCreateValidator(userRepo)
	createAccountUC := usecase.NewCreateAccountUsecase(db, emailConfirmationRepo, userRepo, userPasswordRepo, accountCreateValidator)
	passwordResetCreateValidator := validator.NewPasswordResetCreateValidator()
	createPasswordResetTokenUC := usecase.NewCreatePasswordResetTokenUsecase(cfg, db, userRepo, passwordResetTokenRepo, jobDispatcher, passwordResetCreateValidator)
	passwordUpdateValidator := validator.NewPasswordUpdateValidator(passwordResetTokenRepo)
	updatePasswordResetUC := usecase.NewUpdatePasswordResetUsecase(db, passwordResetTokenRepo, userPasswordRepo, passwordUpdateValidator)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(queries)
	autoSaveDraftPageUC := usecase.NewAutoSaveDraftPageUsecase(db, spaceRepo, spaceMemberRepo, draftPageRepo, pageRepo, pageEditorRepo, topicRepo, topicMemberRepo, attachmentRepo)
	manualSaveDraftPageUC := usecase.NewManualSaveDraftPageUsecase(db, spaceRepo, spaceMemberRepo, draftPageRepo, draftPageRevisionRepo, pageRepo, pageEditorRepo, topicRepo, topicMemberRepo, attachmentRepo)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)
	publishPageUC := usecase.NewPublishPageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, pageRevisionRepo, pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, attachmentRepo, pageAttachmentRefRepo, pageUpdateValidator)
	pageMoveCreateValidator := validator.NewPageMoveCreateValidator(pageRepo, topicRepo, topicMemberRepo, suggestionPageRepo)
	movePageUC := usecase.NewMovePageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo, draftPageRepo, pageMoveCreateValidator)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// Turnstileクライアントを初期化
	turnstileClient := turnstile.NewClient(cfg.TurnstileEnabled, cfg.TurnstileSecretKey)

	// ミドルウェアを初期化
	authMiddleware := middleware.NewAuth(sessionMgr)
	csrfMiddleware := middleware.NewCSRF(cfg)

	// imgproxy ヘルパー / OgImageBuilder を初期化（環境変数が揃っている場合のみ有効）
	// 環境変数が未設定でもサーバー起動は継続するが、その場合 /attachments/:id/og_image に到達した
	// リクエストは 500 を受け取る。本番環境への適用ミスを早期に検知できるよう、起動時に WARN ログで
	// 状態を可視化する。
	var ogImageBuilder *image.OgImageBuilder
	switch {
	case cfg.ImgproxyURL == "":
		slog.Warn("WIKINO_IMGPROXY_URL が未設定のため og:image エンドポイントは無効。リクエストには 500 を返します")
	case cfg.R2BucketName == "":
		slog.Warn("WIKINO_R2_BUCKET_NAME が未設定のため og:image エンドポイントは無効。リクエストには 500 を返します")
	default:
		imgproxyHelper, err := image.NewHelper(cfg.ImgproxyURL, cfg.ImgproxyKey, cfg.ImgproxySalt)
		if err != nil {
			slog.Error("imgproxy ヘルパーの初期化に失敗しました", "error", err)
			os.Exit(1)
		}
		ogImageBuilder, err = image.NewOgImageBuilder(imgproxyHelper, cfg.R2BucketName)
		if err != nil {
			slog.Error("OgImageBuilder の初期化に失敗しました", "error", err)
			os.Exit(1)
		}
	}

	// ハンドラーを初期化
	healthHandler := health.NewHandler()
	manifestHandler := manifest.NewHandler(cfg)

	getAttachmentOgImageUC := usecase.NewGetAttachmentOgImageUsecase(attachmentRepo)
	attachmentOgImageHandler := attachment_og_image.NewHandler(ogImageBuilder, getAttachmentOgImageUC)
	signInHandler := sign_in.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		signInUC,
		turnstileClient,
	)
	deleteUserSessionUC := usecase.NewDeleteUserSessionUsecase(userSessionRepo)
	userSessionHandler := user_session.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		deleteUserSessionUC,
	)
	signInTwoFactorValidator := validator.NewSignInTwoFactorCreateValidator(userTwoFactorAuthRepo)
	createTwoFactorSessionUC := usecase.NewCreateTwoFactorSessionUsecase(signInTwoFactorValidator, createUserSessionUC)
	signInTwoFactorHandler := sign_in_two_factor.NewHandler(
		cfg,
		sessionMgr,
		createTwoFactorSessionUC,
	)
	signInTwoFactorRecoveryValidator := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)
	createRecoveryCodeSessionUC := usecase.NewCreateRecoveryCodeSessionUsecase(db, signInTwoFactorRecoveryValidator, userTwoFactorAuthRepo, userSessionRepo)
	signInTwoFactorRecoveryHandler := sign_in_two_factor_recovery.NewHandler(
		cfg,
		sessionMgr,
		createRecoveryCodeSessionUC,
	)
	signUpHandler := sign_up.NewHandler(
		cfg,
		sessionMgr,
	)
	emailConfirmationHandler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		createEmailConfirmationUC,
		markEmailAsConfirmedUC,
		turnstileClient,
		rateLimiter,
	)
	getAccountNewDataUC := usecase.NewGetAccountNewDataUsecase(emailConfirmationRepo)
	accountHandler := account.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getAccountNewDataUC,
		createAccountUC,
		createUserSessionUC,
	)
	passwordResetHandler := password_reset.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		rateLimiter,
		turnstileClient,
		createPasswordResetTokenUC,
	)
	getTokenDataUC := usecase.NewGetPasswordResetTokenDataUsecase(passwordResetTokenRepo)
	passwordHandler := password.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getTokenDataUC,
		updatePasswordResetUC,
	)
	welcomeHandler := welcome.NewHandler(cfg, flashMgr)
	sidebarHelper := sidebar.NewHelper(topicRepo, draftPageRepo)
	getPageDetailUC := usecase.NewGetPageDetailUsecase(
		spaceRepo,
		spaceMemberRepo,
		pageRepo,
		draftPageRepo,
		topicRepo,
		topicMemberRepo,
		suggestionPageRepo,
		suggestionRepo,
	)
	getEditLinkDataUC := usecase.NewGetEditLinkDataUsecase(pageRepo, topicRepo)
	getPageLocationsUC := usecase.NewGetPageLocationsUsecase(spaceRepo, spaceMemberRepo, pageRepo)
	getPageBacklinksUC := usecase.NewGetPageBacklinksUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo)
	getBacklinkListUC := usecase.NewGetBacklinkListUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo)
	getLinkListUC := usecase.NewGetLinkListUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo, draftPageRepo)
	pageHandler := page.NewHandler(
		cfg,
		flashMgr,
		getPageDetailUC,
		getEditLinkDataUC,
		publishPageUC,
		sidebarHelper,
	)
	pageLocationHandler := page_location.NewHandler(
		getPageLocationsUC,
	)
	getDraftPagesUC := usecase.NewGetDraftPagesUsecase(draftPageRepo)
	draftPageIndexHandler := draft_page_index.NewHandler(
		cfg,
		getDraftPagesUC,
		sidebarHelper,
	)
	getHomeShowUC := usecase.NewGetHomeShowUsecase(spaceRepo)
	homeHandler := home.NewHandler(
		cfg,
		getHomeShowUC,
		sidebarHelper,
	)
	deleteDraftPageUC := usecase.NewDeleteDraftPageUsecase(
		db,
		spaceRepo,
		spaceMemberRepo,
		draftPageRepo,
		draftPageRevisionRepo,
		pageRepo,
		topicRepo,
		topicMemberRepo,
	)
	draftPageHandler := draft_page.NewHandler(
		flashMgr,
		getPageDetailUC,
		autoSaveDraftPageUC,
		deleteDraftPageUC,
		getEditLinkDataUC,
	)
	draftPageRevisionHandler := draft_page_revision.NewHandler(
		flashMgr,
		manualSaveDraftPageUC,
	)
	pageBacklinkListHandler := page_backlink_list.NewHandler(
		getBacklinkListUC,
	)
	pageBacklinksHandler := page_backlinks.NewHandler(
		getPageBacklinksUC,
	)
	pageLinkListHandler := page_link_list.NewHandler(
		getLinkListUC,
	)
	getPageMoveDataUC := usecase.NewGetPageMoveDataUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo)
	pageMoveHandler := page_move.NewHandler(
		cfg,
		flashMgr,
		getPageMoveDataUC,
		movePageUC,
		sidebarHelper,
	)
	getTopicDetailUC := usecase.NewGetTopicDetailUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, pageRepo)
	topicHandler := topichandler.NewHandler(
		cfg,
		flashMgr,
		getTopicDetailUC,
		sidebarHelper,
	)
	getSuggestionListUC := usecase.NewGetSuggestionListUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, userRepo)
	getSuggestionDetailUC := usecase.NewGetSuggestionDetailUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, suggestionPageRepo, suggestionCommentRepo, pageRepo, userRepo)
	getSuggestionEditUC := usecase.NewGetSuggestionEditUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, userRepo)
	getSuggestionNewUC := usecase.NewGetSuggestionNewUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, draftPageRepo)
	suggestionCreateValidator := validator.NewSuggestionCreateValidator(draftPageRepo, pageRepo)
	createSuggestionUC := usecase.NewCreateSuggestionUsecase(db, spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, suggestionPageRepo, suggestionPageRevisionRepo, draftPageRepo, pageRevisionRepo, suggestionCreateValidator)
	getSuggestionDiffUC := usecase.NewGetSuggestionDiffUsecase(pageRevisionRepo)
	suggestionUpdateValidator := validator.NewSuggestionUpdateValidator()
	updateSuggestionUC := usecase.NewUpdateSuggestionUsecase(db, spaceRepo, spaceMemberRepo, topicMemberRepo, suggestionRepo, suggestionUpdateValidator)
	suggestionHandler := suggestionhandler.NewHandler(
		cfg,
		flashMgr,
		getSuggestionListUC,
		getSuggestionDetailUC,
		getSuggestionEditUC,
		getSuggestionNewUC,
		createSuggestionUC,
		updateSuggestionUC,
		sidebarHelper,
	)
	suggestionChangeHandler := suggestionchangehandler.NewHandler(
		cfg,
		getSuggestionDetailUC,
		getSuggestionDiffUC,
		sidebarHelper,
	)
	suggestionApplyValidator := validator.NewSuggestionApplyValidator(pageUpdateValidator)
	applySuggestionUC := usecase.NewApplySuggestionUsecase(
		db, spaceRepo, spaceMemberRepo, topicMemberRepo,
		suggestionRepo, suggestionPageRepo, pageRepo, pageRevisionRepo,
		pageEditorRepo, attachmentRepo, pageAttachmentRefRepo, draftPageRepo,
		suggestionApplyValidator,
	)
	suggestionApplyHandler := suggestionapplyhandler.NewHandler(
		cfg,
		flashMgr,
		applySuggestionUC,
		getSuggestionDetailUC,
		sidebarHelper,
	)
	closeSuggestionUC := usecase.NewCloseSuggestionUsecase(db, spaceRepo, spaceMemberRepo, topicMemberRepo, suggestionRepo, draftPageRepo)
	suggestionCloseHandler := suggestionclosehandler.NewHandler(
		flashMgr,
		closeSuggestionUC,
	)
	startSuggestionPageEditUC := usecase.NewStartSuggestionPageEditUsecase(db, spaceRepo, spaceMemberRepo, topicMemberRepo, suggestionRepo, suggestionPageRepo, draftPageRepo, pageRepo)
	suggestionPageEditHandler := suggestionpageedithandler.NewHandler(
		cfg,
		flashMgr,
		getSuggestionDetailUC,
		startSuggestionPageEditUC,
		sidebarHelper,
	)
	suggestionPageUpdateValidator := validator.NewSuggestionPageUpdateValidator(draftPageRepo)
	updateSuggestionPageUC := usecase.NewUpdateSuggestionPageUsecase(
		db, spaceRepo, spaceMemberRepo, topicMemberRepo,
		suggestionRepo, suggestionPageRepo, suggestionPageRevisionRepo, suggestionPageUpdateValidator,
	)
	getSuggestionPageNewUC := usecase.NewGetSuggestionPageNewUsecase(spaceRepo, spaceMemberRepo, topicMemberRepo, suggestionRepo, topicRepo, draftPageRepo)
	suggestionPageCreateValidator := validator.NewSuggestionPageCreateValidator(draftPageRepo, pageRepo, suggestionPageRepo)
	addSuggestionPageUC := usecase.NewAddSuggestionPageUsecase(
		db, spaceRepo, spaceMemberRepo, topicMemberRepo,
		suggestionRepo, suggestionPageRepo, suggestionPageRevisionRepo, draftPageRepo, pageRevisionRepo,
		suggestionPageCreateValidator,
	)
	removeSuggestionPageUC := usecase.NewRemoveSuggestionPageUsecase(
		db, spaceRepo, spaceMemberRepo, topicMemberRepo,
		suggestionRepo, suggestionPageRepo, suggestionPageRevisionRepo, draftPageRepo,
	)
	suggestionPageHandler := suggestionpagehandler.NewHandler(
		cfg,
		flashMgr,
		getSuggestionPageNewUC,
		addSuggestionPageUC,
		updateSuggestionPageUC,
		removeSuggestionPageUC,
		sidebarHelper,
	)
	suggestionCommentCreateValidator := validator.NewSuggestionCommentCreateValidator()
	createSuggestionCommentUC := usecase.NewCreateSuggestionCommentUsecase(
		db, spaceRepo, spaceMemberRepo, topicMemberRepo, suggestionRepo, suggestionCommentRepo, suggestionCommentCreateValidator,
	)
	suggestionCommentHandler := suggestioncommenthandler.NewHandler(
		flashMgr,
		createSuggestionCommentUC,
	)
	getSuggestionCommentUC := usecase.NewGetSuggestionCommentUsecase(suggestionCommentRepo)
	suggestionCommentUpdateValidator := validator.NewSuggestionCommentUpdateValidator()
	updateSuggestionCommentUC := usecase.NewUpdateSuggestionCommentUsecase(
		db, spaceRepo, spaceMemberRepo, topicMemberRepo,
		suggestionRepo, suggestionCommentRepo, suggestionCommentUpdateValidator,
	)
	suggestionCommentEditHandler := suggestioncommentedithandler.NewHandler(
		cfg,
		flashMgr,
		getSuggestionEditUC,
		getSuggestionCommentUC,
		updateSuggestionCommentUC,
		sidebarHelper,
	)
	r := chi.NewRouter()

	// ルーティングにマッチしなかった場合のNotFoundハンドラーを設定
	r.NotFound(handler.NotFound)

	// リバースプロキシミドルウェアを初期化（Rails版へのプロキシ）
	// 注: RailsAppURLが設定されている場合のみ有効化
	// リバースプロキシはMethod OverrideやCSRFミドルウェアより前に配置する。
	// これらのミドルウェアはr.ParseForm()やr.FormValue()でリクエストボディを
	// 消費するため、プロキシ前に実行するとRails版への転送時にボディが空になる。
	if cfg.RailsAppURL != "" {
		reverseProxyMiddleware, err := middleware.NewReverseProxyMiddleware(cfg.RailsAppURL, cfg, featureFlagRepo)
		if err != nil {
			slog.Error("リバースプロキシミドルウェアの初期化に失敗しました", "error", err)
			os.Exit(1)
		}
		r.Use(reverseProxyMiddleware.Middleware)
	}

	// メンテナンスモードミドルウェア
	maintenanceMW := middleware.NewMaintenanceMiddleware(cfg)
	r.Use(maintenanceMW.Middleware)

	// リクエストボディサイズ制限ミドルウェア
	// r.ParseForm()やr.FormValue()を呼ぶMethod Override・CSRFミドルウェアより前に配置する必要がある。
	// reverseProxyより後に配置することで、Rails版へプロキシするリクエストにはGo側の制限を適用しない。
	r.Use(middleware.BodyLimit)

	// Method Overrideミドルウェア（HTMLフォームからDELETE/PATCH/PUTを使用可能にする）
	r.Use(middleware.MethodOverride)

	// 共通ミドルウェア
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(i18n.Middleware)
	r.Use(csrfMiddleware.Middleware)
	r.Use(flashMgr.Middleware)

	// 静的ファイルの配信 (Tailwind CLI + esbuild のビルド結果)
	fileServer := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	// ヘルスチェック（認証不要）
	r.Get("/health", healthHandler.Show)

	// Web App Manifest（認証不要）
	r.Get("/manifest.json", manifestHandler.Show)

	// og:image 配信エンドポイント（認証不要、公開トピックの og:image を imgproxy 経由で配信する）
	r.Get("/attachments/{attachment_id}/og_image", attachmentOgImageHandler.Show)

	// トップページ（ログイン状態に応じてハンドラー内でリダイレクト）
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.SetUser)
		r.Use(middleware.TimeZone)
		r.Get("/", welcomeHandler.Show)

		// トピック詳細画面（公開トピックは未ログインでも閲覧可能）
		r.Get("/s/{space_identifier}/topics/{topic_number}", topicHandler.Show)

		// 編集提案（公開トピックは未ログインでも閲覧可能）
		r.Get("/s/{space_identifier}/topics/{topic_number}/suggestions", suggestionHandler.Index)
		r.Get("/s/{space_identifier}/suggestions/{suggestion_number}", suggestionHandler.Show)
		r.Get("/s/{space_identifier}/suggestions/{suggestion_number}/changes", suggestionChangeHandler.Index)
	})

	// 未認証ユーザー専用ルート
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireNoAuth)
		r.Use(middleware.TimeZone)
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
		r.Use(middleware.TimeZone)
		r.Delete("/user_session", userSessionHandler.Delete)

		// ホーム画面
		r.Get("/home", homeHandler.Show)

		// 下書き一覧
		r.Get("/drafts", draftPageIndexHandler.Index)

		// ページ編集・公開
		r.Get("/s/{space_identifier}/pages/{page_number}/edit", pageHandler.Edit)
		r.Patch("/s/{space_identifier}/pages/{page_number}", pageHandler.Update)

		// 下書きページSSE・自動保存API
		r.Get("/s/{space_identifier}/pages/{page_number}/draft_page", draftPageHandler.Show)
		r.Patch("/s/{space_identifier}/pages/{page_number}/draft_page", draftPageHandler.Update)
		r.Delete("/s/{space_identifier}/pages/{page_number}/draft_page", draftPageHandler.Delete)

		// 下書きリビジョン手動保存
		r.Patch("/s/{space_identifier}/pages/{page_number}/draft_page_revision", draftPageRevisionHandler.Update)

		// リンク一覧（htmx）
		r.Get("/s/{space_identifier}/pages/{page_number}/link_list", pageLinkListHandler.Show)

		// バックリンク一覧（htmx）
		r.Get("/s/{space_identifier}/pages/{page_number}/links/{linked_page_number}/backlink_list", pageBacklinkListHandler.Show)

		// ページレベルのバックリンク一覧（htmx）
		r.Get("/s/{space_identifier}/pages/{page_number}/backlinks", pageBacklinksHandler.Show)

		// ページ移動
		r.Get("/s/{space_identifier}/pages/{page_number}/move", pageMoveHandler.New)
		r.Post("/s/{space_identifier}/pages/{page_number}/move", pageMoveHandler.Create)

		// 編集提案作成・編集
		r.Get("/s/{space_identifier}/topics/{topic_number}/suggestions/new", suggestionHandler.New)
		r.Post("/s/{space_identifier}/topics/{topic_number}/suggestions", suggestionHandler.Create)
		r.Get("/s/{space_identifier}/suggestions/{suggestion_number}/edit", suggestionHandler.Edit)
		r.Patch("/s/{space_identifier}/suggestions/{suggestion_number}", suggestionHandler.Update)

		// 編集提案反映
		r.Post("/s/{space_identifier}/suggestions/{suggestion_number}/apply", suggestionApplyHandler.Create)

		// 編集提案クローズ
		r.Post("/s/{space_identifier}/suggestions/{suggestion_number}/close", suggestionCloseHandler.Create)

		// 編集提案コメント
		r.Post("/s/{space_identifier}/suggestions/{suggestion_number}/comments", suggestionCommentHandler.Create)
		r.Get("/s/{space_identifier}/suggestions/{suggestion_number}/comments/{comment_number}/edit", suggestionCommentEditHandler.Edit)
		r.Patch("/s/{space_identifier}/suggestions/{suggestion_number}/comments/{comment_number}", suggestionCommentEditHandler.Update)

		// 編集提案ページ編集開始
		r.Get("/s/{space_identifier}/suggestions/{suggestion_number}/page_edits/{suggestion_page_id}", suggestionPageEditHandler.Show)
		r.Post("/s/{space_identifier}/suggestions/{suggestion_number}/page_edits", suggestionPageEditHandler.Create)

		// 編集提案ページ追加・更新
		r.Get("/s/{space_identifier}/suggestions/{suggestion_number}/suggestion_pages/new", suggestionPageHandler.New)
		r.Post("/s/{space_identifier}/suggestions/{suggestion_number}/suggestion_pages", suggestionPageHandler.Create)
		r.Patch("/s/{space_identifier}/suggestions/{suggestion_number}/suggestion_pages/{suggestion_page_id}", suggestionPageHandler.Update)
		r.Delete("/s/{space_identifier}/suggestions/{suggestion_number}/suggestion_pages/{suggestion_page_id}", suggestionPageHandler.Delete)

		// ページロケーション検索API（Wikiリンク補完用）
		r.Get("/s/{space_identifier}/page_locations", pageLocationHandler.Index)
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
