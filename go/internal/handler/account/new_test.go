package account_test

import (
	"crypto/rand"
	"database/sql"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/account"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// generateCode はユニークな確認コードを生成します
func generateCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 6)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		code[i] = charset[n.Int64()]
	}
	return string(code)
}

func TestNew(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	createAccountUC := usecase.NewCreateAccountUsecase(
		&sql.DB{},
		emailConfirmationRepo,
		userRepo,
		userPasswordRepo,
	)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)
	_ = db

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// 確認済みのメール確認情報を作成
	now := time.Now()
	emailConfirmation, err := emailConfirmationRepo.Create(t.Context(), repository.CreateEmailConfirmationInput{
		Email:     "test@example.com",
		Event:     model.EmailConfirmationEventSignUp,
		Code:      generateCode(),
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create email confirmation: %v", err)
	}

	// メール確認を完了状態に更新
	err = emailConfirmationRepo.Succeed(t.Context(), emailConfirmation.ID)
	if err != nil {
		t.Fatalf("failed to succeed email confirmation: %v", err)
	}

	accountCreateValidator := validator.NewAccountCreateValidator(emailConfirmationRepo, userRepo)

	getAccountNewDataUC := usecase.NewGetAccountNewDataUsecase(emailConfirmationRepo)

	handler := account.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getAccountNewDataUC,
		accountCreateValidator,
		createAccountUC,
		createUserSessionUC,
	)

	// HTTPリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/accounts/new", nil)
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	// email_confirmation_id を Cookie に設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmation.ID,
	})

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// レスポンスボディを検証
	body := rr.Body.String()

	// フォームアクションが含まれているか確認
	if !strings.Contains(body, `action="/accounts"`) {
		t.Error("account creation form action not found in response")
	}

	// CSRFトークンが含まれているか確認
	if !strings.Contains(body, "test-csrf-token") {
		t.Error("CSRF token not found in response")
	}

	// メールアドレスが表示されているか確認
	if !strings.Contains(body, "test@example.com") {
		t.Error("email address not found in response")
	}

	// アットネーム入力フィールドが含まれているか確認
	if !strings.Contains(body, `name="atname"`) {
		t.Error("atname input field not found in response")
	}

	// パスワード入力フィールドが含まれているか確認
	if !strings.Contains(body, `name="password"`) {
		t.Error("password input field not found in response")
	}

	// 日本語の見出しが含まれているか確認
	if !strings.Contains(body, "アカウントを作成") {
		t.Error("Japanese heading not found in response")
	}

	// 新規登録に戻るリンクが含まれているか確認
	if !strings.Contains(body, `href="/sign_up"`) {
		t.Error("back to sign up link not found in response")
	}
}

func TestNew_NoEmailConfirmationID(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	createAccountUC := usecase.NewCreateAccountUsecase(
		&sql.DB{},
		emailConfirmationRepo,
		userRepo,
		userPasswordRepo,
	)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	accountCreateValidator := validator.NewAccountCreateValidator(emailConfirmationRepo, userRepo)

	getAccountNewDataUC := usecase.NewGetAccountNewDataUsecase(emailConfirmationRepo)

	handler := account.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getAccountNewDataUC,
		accountCreateValidator,
		createAccountUC,
		createUserSessionUC,
	)

	// HTTPリクエストを作成（email_confirmation_id のCookieなし）
	req := httptest.NewRequest(http.MethodGet, "/accounts/new", nil)
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// リダイレクトのステータスコードを検証
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/sign_up" {
		t.Errorf("wrong redirect location: got %v want %v", location, "/sign_up")
	}
}

func TestNew_EmailConfirmationNotFound(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	createAccountUC := usecase.NewCreateAccountUsecase(
		&sql.DB{},
		emailConfirmationRepo,
		userRepo,
		userPasswordRepo,
	)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	accountCreateValidator := validator.NewAccountCreateValidator(emailConfirmationRepo, userRepo)

	getAccountNewDataUC := usecase.NewGetAccountNewDataUsecase(emailConfirmationRepo)

	handler := account.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getAccountNewDataUC,
		accountCreateValidator,
		createAccountUC,
		createUserSessionUC,
	)

	// HTTPリクエストを作成（存在しない email_confirmation_id）
	req := httptest.NewRequest(http.MethodGet, "/accounts/new", nil)
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	// 存在しないIDをCookieに設定（有効なUUID形式だが存在しない）
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: "00000000-0000-0000-0000-000000000000",
	})

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// リダイレクトのステータスコードを検証
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/sign_up" {
		t.Errorf("wrong redirect location: got %v want %v", location, "/sign_up")
	}
}

func TestNew_EmailNotVerified(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	createAccountUC := usecase.NewCreateAccountUsecase(
		&sql.DB{},
		emailConfirmationRepo,
		userRepo,
		userPasswordRepo,
	)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// 未確認のメール確認情報を作成（succeeded_at = NULL）
	now := time.Now()
	emailConfirmation, err := emailConfirmationRepo.Create(t.Context(), repository.CreateEmailConfirmationInput{
		Email:     "test@example.com",
		Event:     model.EmailConfirmationEventSignUp,
		Code:      generateCode(),
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create email confirmation: %v", err)
	}

	accountCreateValidator := validator.NewAccountCreateValidator(emailConfirmationRepo, userRepo)

	getAccountNewDataUC := usecase.NewGetAccountNewDataUsecase(emailConfirmationRepo)

	handler := account.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getAccountNewDataUC,
		accountCreateValidator,
		createAccountUC,
		createUserSessionUC,
	)

	// HTTPリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/accounts/new", nil)
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	// email_confirmation_id を Cookie に設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmation.ID,
	})

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// リダイレクトのステータスコードを検証
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証（確認コード入力ページへ）
	location := rr.Header().Get("Location")
	if location != "/email_confirmation/edit" {
		t.Errorf("wrong redirect location: got %v want %v", location, "/email_confirmation/edit")
	}
}

func TestNew_EnglishLocale(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	createAccountUC := usecase.NewCreateAccountUsecase(
		&sql.DB{},
		emailConfirmationRepo,
		userRepo,
		userPasswordRepo,
	)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// 確認済みのメール確認情報を作成
	now := time.Now()
	emailConfirmation, err := emailConfirmationRepo.Create(t.Context(), repository.CreateEmailConfirmationInput{
		Email:     "test@example.com",
		Event:     model.EmailConfirmationEventSignUp,
		Code:      generateCode(),
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create email confirmation: %v", err)
	}

	// メール確認を完了状態に更新
	err = emailConfirmationRepo.Succeed(t.Context(), emailConfirmation.ID)
	if err != nil {
		t.Fatalf("failed to succeed email confirmation: %v", err)
	}

	accountCreateValidator := validator.NewAccountCreateValidator(emailConfirmationRepo, userRepo)

	getAccountNewDataUC := usecase.NewGetAccountNewDataUsecase(emailConfirmationRepo)

	handler := account.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getAccountNewDataUC,
		accountCreateValidator,
		createAccountUC,
		createUserSessionUC,
	)

	// HTTPリクエストを作成（英語ロケール）
	req := httptest.NewRequest(http.MethodGet, "/accounts/new", nil)
	req.Header.Set("Accept-Language", "en")

	// CSRFトークンと言語設定をコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, i18n.LangEn)
	req = req.WithContext(ctx)

	// email_confirmation_id を Cookie に設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmation.ID,
	})

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// 英語の見出しが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "Create your account") {
		t.Error("English heading not found in response")
	}

	// 英語のボタンテキストが含まれているか確認
	if !strings.Contains(body, "Create account") {
		t.Error("English submit button text not found in response")
	}

	// 英語のパスワードヒントが含まれているか確認
	if !strings.Contains(body, "Must be at least 8 characters") {
		t.Error("English password hint not found in response")
	}

	// 英語の戻るリンクが含まれているか確認
	if !strings.Contains(body, "Back to sign up") {
		t.Error("English back link not found in response")
	}
}
