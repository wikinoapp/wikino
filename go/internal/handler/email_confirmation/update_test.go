package email_confirmation_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/email_confirmation"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// mockInserterForUpdate はテスト用のモック inserter
type mockInserterForUpdate struct{}

func (m *mockInserterForUpdate) Insert(_ context.Context, _ river.JobArgs) (*rivertype.JobInsertResult, error) {
	return &rivertype.JobInsertResult{}, nil
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// メール確認情報を作成
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("ABC123").
		WithEvent(model.EmailConfirmationEventSignUp).
		Build()

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
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserterForUpdate{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// Rate Limiterを初期化
	rateLimitRepo := repository.NewRateLimitRepository(queries)
	limiter := ratelimit.NewLimiter(rateLimitRepo)

	// バリデーターを初期化
	emailConfirmationCreateValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	emailConfirmationUpdateValidator := validator.NewEmailConfirmationUpdateValidator(emailConfirmationRepo)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		emailConfirmationCreateValidator,
		emailConfirmationUpdateValidator,
		mockTurnstile,
		limiter,
	)

	// フォームデータを作成
	form := url.Values{}
	form.Set("code", "ABC123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("_method", "PATCH")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// email_confirmation_id Cookieを設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmationID,
	})

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// リダイレクトを検証
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/accounts/new" {
		t.Errorf("wrong redirect location: got %v want /accounts/new", location)
	}

	// メール確認が成功状態になっているか確認
	confirmation, err := emailConfirmationRepo.FindByID(ctx, emailConfirmationID)
	if err != nil {
		t.Fatalf("failed to find email confirmation: %v", err)
	}
	if !confirmation.IsSucceeded() {
		t.Error("email confirmation should be succeeded")
	}
}

func TestUpdate_NoSession(t *testing.T) {
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
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserterForUpdate{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// Rate Limiterを初期化
	rateLimitRepo := repository.NewRateLimitRepository(queries)
	limiter := ratelimit.NewLimiter(rateLimitRepo)

	// バリデーターを初期化
	emailConfirmationCreateValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	emailConfirmationUpdateValidator := validator.NewEmailConfirmationUpdateValidator(emailConfirmationRepo)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		emailConfirmationCreateValidator,
		emailConfirmationUpdateValidator,
		mockTurnstile,
		limiter,
	)

	// フォームデータを作成（Cookieなし）
	form := url.Values{}
	form.Set("code", "ABC123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// /sign_up にリダイレクトされることを検証
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/sign_up" {
		t.Errorf("wrong redirect location: got %v want /sign_up", location)
	}
}

func TestUpdate_EmptyCode(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// メール確認情報を作成
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("ABC123").
		WithEvent(model.EmailConfirmationEventSignUp).
		Build()

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
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserterForUpdate{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// Rate Limiterを初期化
	rateLimitRepo := repository.NewRateLimitRepository(queries)
	limiter := ratelimit.NewLimiter(rateLimitRepo)

	// バリデーターを初期化
	emailConfirmationCreateValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	emailConfirmationUpdateValidator := validator.NewEmailConfirmationUpdateValidator(emailConfirmationRepo)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		emailConfirmationCreateValidator,
		emailConfirmationUpdateValidator,
		mockTurnstile,
		limiter,
	)

	// 空のコードでリクエスト
	form := url.Values{}
	form.Set("code", "")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// email_confirmation_id Cookieを設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmationID,
	})

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "入力してください") {
		t.Error("validation error message not found in response")
	}
}

func TestUpdate_InvalidCodeLength(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// メール確認情報を作成
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("ABC123").
		WithEvent(model.EmailConfirmationEventSignUp).
		Build()

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
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserterForUpdate{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// Rate Limiterを初期化
	rateLimitRepo := repository.NewRateLimitRepository(queries)
	limiter := ratelimit.NewLimiter(rateLimitRepo)

	// バリデーターを初期化
	emailConfirmationCreateValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	emailConfirmationUpdateValidator := validator.NewEmailConfirmationUpdateValidator(emailConfirmationRepo)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		emailConfirmationCreateValidator,
		emailConfirmationUpdateValidator,
		mockTurnstile,
		limiter,
	)

	// 6文字未満のコードでリクエスト
	form := url.Values{}
	form.Set("code", "ABC")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// email_confirmation_id Cookieを設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmationID,
	})

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "6文字") {
		t.Error("invalid code length error message not found in response")
	}
}

func TestUpdate_CodeMismatch(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// メール確認情報を作成
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("ABC123").
		WithEvent(model.EmailConfirmationEventSignUp).
		Build()

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
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserterForUpdate{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// Rate Limiterを初期化
	rateLimitRepo := repository.NewRateLimitRepository(queries)
	limiter := ratelimit.NewLimiter(rateLimitRepo)

	// バリデーターを初期化
	emailConfirmationCreateValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	emailConfirmationUpdateValidator := validator.NewEmailConfirmationUpdateValidator(emailConfirmationRepo)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		emailConfirmationCreateValidator,
		emailConfirmationUpdateValidator,
		mockTurnstile,
		limiter,
	)

	// 間違ったコードでリクエスト
	form := url.Values{}
	form.Set("code", "WRONG1")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// email_confirmation_id Cookieを設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmationID,
	})

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "正しくありません") {
		t.Error("code mismatch error message not found in response")
	}
}

func TestUpdate_Expired(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// 有効期限切れのメール確認情報を作成
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("ABC123").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithStartedAt(time.Now().Add(-16 * time.Minute)).
		Build()

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
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserterForUpdate{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// Rate Limiterを初期化
	rateLimitRepo := repository.NewRateLimitRepository(queries)
	limiter := ratelimit.NewLimiter(rateLimitRepo)

	// バリデーターを初期化
	emailConfirmationCreateValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	emailConfirmationUpdateValidator := validator.NewEmailConfirmationUpdateValidator(emailConfirmationRepo)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		emailConfirmationCreateValidator,
		emailConfirmationUpdateValidator,
		mockTurnstile,
		limiter,
	)

	// リクエスト
	form := url.Values{}
	form.Set("code", "ABC123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// email_confirmation_id Cookieを設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmationID,
	})

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "有効期限") {
		t.Error("expired error message not found in response")
	}
}

func TestUpdate_CaseInsensitiveCode(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// メール確認情報を作成（大文字）
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("ABC123").
		WithEvent(model.EmailConfirmationEventSignUp).
		Build()

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
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserterForUpdate{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// Rate Limiterを初期化
	rateLimitRepo := repository.NewRateLimitRepository(queries)
	limiter := ratelimit.NewLimiter(rateLimitRepo)

	// バリデーターを初期化
	emailConfirmationCreateValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	emailConfirmationUpdateValidator := validator.NewEmailConfirmationUpdateValidator(emailConfirmationRepo)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		emailConfirmationCreateValidator,
		emailConfirmationUpdateValidator,
		mockTurnstile,
		limiter,
	)

	// 小文字でリクエスト
	form := url.Values{}
	form.Set("code", "abc123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// email_confirmation_id Cookieを設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmationID,
	})

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// 大文字小文字を区別しないため、成功してリダイレクトされる
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/accounts/new" {
		t.Errorf("wrong redirect location: got %v want /accounts/new", location)
	}
}

func TestUpdate_AlreadySucceeded(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// 既に確認済みのメール確認情報を作成
	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("ABC123").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithSucceededAt(time.Now()).
		Build()

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
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserterForUpdate{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// Rate Limiterを初期化
	rateLimitRepo := repository.NewRateLimitRepository(queries)
	limiter := ratelimit.NewLimiter(rateLimitRepo)

	// バリデーターを初期化
	emailConfirmationCreateValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	emailConfirmationUpdateValidator := validator.NewEmailConfirmationUpdateValidator(emailConfirmationRepo)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		emailConfirmationCreateValidator,
		emailConfirmationUpdateValidator,
		mockTurnstile,
		limiter,
	)

	// リクエスト
	form := url.Values{}
	form.Set("code", "ABC123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// email_confirmation_id Cookieを設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmationID,
	})

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// 既に確認済みの場合、/accounts/new にリダイレクトされる
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/accounts/new" {
		t.Errorf("wrong redirect location: got %v want /accounts/new", location)
	}
}
