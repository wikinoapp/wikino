package email_confirmation_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/email_confirmation"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// mockTurnstileVerifierForEdit はテスト用のTurnstile検証モック
type mockTurnstileVerifierForEdit struct {
	valid bool
	err   error
}

func (m *mockTurnstileVerifierForEdit) Verify(_ context.Context, _ string) (bool, error) {
	return m.valid, m.err
}

// mockInserterForEdit はテスト用のモック inserter
type mockInserterForEdit struct{}

func (m *mockInserterForEdit) Insert(_ context.Context, _ river.JobArgs) (*rivertype.JobInsertResult, error) {
	return &rivertype.JobInsertResult{}, nil
}

func TestEdit(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTestDB(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:                "test",
		Port:               "8080",
		Domain:             "localhost",
		CookieDomain:       "",
		SessionSecure:      false,
		SessionHTTPOnly:    true,
		TurnstileSiteKey:   "test-site-key",
		TurnstileSecretKey: "",
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ユースケースを初期化
	inserter := &mockInserterForEdit{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)

	// Turnstile検証器（モック）
	turnstileVerifier := &mockTurnstileVerifierForEdit{valid: true}

	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// Rate Limiterを初期化
	limiter := ratelimit.NewLimiter(queries)

	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		emailConfirmationRepo,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		turnstileVerifier,
		limiter,
	)

	// HTTPリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/email_confirmation/edit", nil)
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	// email_confirmation_id を Cookie に設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: "test-email-confirmation-id",
	})

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// レスポンスボディを検証
	body := rr.Body.String()

	// 確認コード入力フォームが含まれているか確認
	if !strings.Contains(body, `action="/email_confirmation"`) {
		t.Error("confirmation form action not found in response")
	}

	// _method hidden フィールドが含まれているか確認
	if !strings.Contains(body, `name="_method" value="PATCH"`) {
		t.Error("_method hidden field not found in response")
	}

	// CSRFトークンが含まれているか確認
	if !strings.Contains(body, "test-csrf-token") {
		t.Error("CSRF token not found in response")
	}

	// コード入力フィールドが含まれているか確認
	if !strings.Contains(body, `name="code"`) {
		t.Error("code input field not found in response")
	}

	// 日本語の見出しが含まれているか確認
	if !strings.Contains(body, "確認コードを入力") {
		t.Error("Japanese heading not found in response")
	}

	// 新規登録に戻るリンクが含まれているか確認
	if !strings.Contains(body, `href="/sign_up"`) {
		t.Error("back to sign up link not found in response")
	}
}

func TestEdit_NoEmailConfirmationID(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTestDB(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:                "test",
		Port:               "8080",
		Domain:             "localhost",
		CookieDomain:       "",
		SessionSecure:      false,
		SessionHTTPOnly:    true,
		TurnstileSiteKey:   "test-site-key",
		TurnstileSecretKey: "",
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ユースケースを初期化
	inserter := &mockInserterForEdit{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)

	// Turnstile検証器（モック）
	turnstileVerifier := &mockTurnstileVerifierForEdit{valid: true}

	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// Rate Limiterを初期化
	limiter := ratelimit.NewLimiter(queries)

	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		emailConfirmationRepo,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		turnstileVerifier,
		limiter,
	)

	// HTTPリクエストを作成（email_confirmation_id のCookieなし）
	req := httptest.NewRequest(http.MethodGet, "/email_confirmation/edit", nil)
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

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

func TestEdit_EnglishLocale(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTestDB(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:                "test",
		Port:               "8080",
		Domain:             "localhost",
		CookieDomain:       "",
		SessionSecure:      false,
		SessionHTTPOnly:    true,
		TurnstileSiteKey:   "test-site-key",
		TurnstileSecretKey: "",
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ユースケースを初期化
	inserter := &mockInserterForEdit{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)

	// Turnstile検証器（モック）
	turnstileVerifier := &mockTurnstileVerifierForEdit{valid: true}

	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// Rate Limiterを初期化
	limiter := ratelimit.NewLimiter(queries)

	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		emailConfirmationRepo,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		turnstileVerifier,
		limiter,
	)

	// HTTPリクエストを作成（英語ロケール）
	req := httptest.NewRequest(http.MethodGet, "/email_confirmation/edit", nil)
	req.Header.Set("Accept-Language", "en")

	// CSRFトークンと言語設定をコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, i18n.LangEn)
	req = req.WithContext(ctx)

	// email_confirmation_id を Cookie に設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: "test-email-confirmation-id",
	})

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// 英語の見出しが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "Enter confirmation code") {
		t.Error("English heading not found in response")
	}

	// 英語のボタンテキストが含まれているか確認
	if !strings.Contains(body, "Verify") {
		t.Error("English submit button text not found in response")
	}

	// 英語の戻るリンクが含まれているか確認
	if !strings.Contains(body, "Back to sign up") {
		t.Error("English back link not found in response")
	}
}
