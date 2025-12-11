package user_session_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/user_session"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// mockTurnstileVerifier はテスト用のTurnstile検証モック
type mockTurnstileVerifier struct {
	valid bool
	err   error
}

func (m *mockTurnstileVerifier) Verify(ctx context.Context, token string) (bool, error) {
	return m.valid, m.err
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTestDB(t)
	queries := testutil.QueriesWithTx(tx)

	// テスト用パスワードをハッシュ化
	password := "testpassword123"
	passwordDigest, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("test@example.com").
		WithAtname("testuser").
		BuildWithPassword(passwordDigest)

	// 設定を作成
	cfg := &config.Config{
		Env:                "test",
		Port:               "8080",
		Domain:             "localhost",
		CookieDomain:       "",
		SessionSecure:      false,
		SessionHTTPOnly:    true,
		TurnstileSiteKey:   "",
		TurnstileSecretKey: "",
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)

	// ユースケースを初期化
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ハンドラーを初期化
	handler := user_session.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		userPasswordRepo,
		userSessionRepo,
		createUserSessionUC,
		mockTurnstile,
	)

	// フォームデータを作成
	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("password", password)
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/user_session", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// リダイレクトを検証
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("wrong redirect location: got %v want /", location)
	}

	// セッションCookieが設定されているか確認
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.CookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Error("session cookie not set")
	}

	// セッションがDBに保存されているか確認
	if sessionCookie != nil {
		savedSession, err := userSessionRepo.FindByToken(context.Background(), sessionCookie.Value)
		if err != nil {
			t.Fatalf("セッション取得でエラー: %v", err)
		}
		if savedSession == nil {
			t.Error("session not saved to database")
		}
		if savedSession != nil && savedSession.UserID != userID {
			t.Errorf("wrong user ID in session: got %v want %v", savedSession.UserID, userID)
		}
	}
}

func TestCreate_InvalidEmail(t *testing.T) {
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
		TurnstileSiteKey:   "",
		TurnstileSecretKey: "",
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)

	// ユースケースを初期化
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ハンドラーを初期化
	handler := user_session.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		userPasswordRepo,
		userSessionRepo,
		createUserSessionUC,
		mockTurnstile,
	)

	// 無効なメールアドレスでリクエスト
	form := url.Values{}
	form.Set("email", "invalid-email")
	form.Set("password", "password123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/user_session", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// フォームが再表示されることを検証（リダイレクトしない）
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// エラーメッセージが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `action="/user_session"`) {
		t.Error("login form not found in response")
	}
}

func TestCreate_WrongPassword(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTestDB(t)
	queries := testutil.QueriesWithTx(tx)

	// テスト用パスワードをハッシュ化
	password := "testpassword123"
	passwordDigest, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}

	// テストユーザーを作成
	_ = testutil.NewUserBuilder(t, tx).
		WithEmail("test@example.com").
		WithAtname("testuser").
		BuildWithPassword(passwordDigest)

	// 設定を作成
	cfg := &config.Config{
		Env:                "test",
		Port:               "8080",
		Domain:             "localhost",
		CookieDomain:       "",
		SessionSecure:      false,
		SessionHTTPOnly:    true,
		TurnstileSiteKey:   "",
		TurnstileSecretKey: "",
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)

	// ユースケースを初期化
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ハンドラーを初期化
	handler := user_session.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		userPasswordRepo,
		userSessionRepo,
		createUserSessionUC,
		mockTurnstile,
	)

	// 間違ったパスワードでリクエスト
	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("password", "wrongpassword")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/user_session", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// フォームが再表示されることを検証（リダイレクトしない）
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// セッションCookieが設定されていないことを確認
	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == session.CookieName {
			t.Error("session cookie should not be set for wrong password")
		}
	}
}

func TestCreate_UserNotFound(t *testing.T) {
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
		TurnstileSiteKey:   "",
		TurnstileSecretKey: "",
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)

	// ユースケースを初期化
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ハンドラーを初期化
	handler := user_session.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		userPasswordRepo,
		userSessionRepo,
		createUserSessionUC,
		mockTurnstile,
	)

	// 存在しないユーザーでリクエスト
	form := url.Values{}
	form.Set("email", "nonexistent@example.com")
	form.Set("password", "password123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/user_session", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// フォームが再表示されることを検証（リダイレクトしない）
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// セッションCookieが設定されていないことを確認
	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == session.CookieName {
			t.Error("session cookie should not be set for nonexistent user")
		}
	}
}

func TestCreate_TurnstileFailure(t *testing.T) {
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
		TurnstileSiteKey:   "",
		TurnstileSecretKey: "",
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)

	// ユースケースを初期化
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に失敗）
	mockTurnstile := &mockTurnstileVerifier{valid: false, err: nil}

	// ハンドラーを初期化
	handler := user_session.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		userPasswordRepo,
		userSessionRepo,
		createUserSessionUC,
		mockTurnstile,
	)

	// リクエスト
	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("password", "password123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "invalid-token")

	req := httptest.NewRequest(http.MethodPost, "/user_session", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// フォームが再表示されることを検証（リダイレクトしない）
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// セッションCookieが設定されていないことを確認
	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == session.CookieName {
			t.Error("session cookie should not be set for Turnstile failure")
		}
	}
}
