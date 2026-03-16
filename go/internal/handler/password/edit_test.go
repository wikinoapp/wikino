package password_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/password"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/password_reset"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestEdit_Success(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).Build()

	// 有効なトークンを作成
	validToken := "valid-edit-test-token"
	validTokenDigest := password_reset.HashToken(validToken)
	testutil.NewPasswordResetTokenBuilder(t, tx).
		WithUserID(userID).
		WithTokenDigest(validTokenDigest).
		WithExpiresAt(time.Now().Add(1 * time.Hour)).
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
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ハンドラーを初期化
	getTokenDataUC := usecase.NewGetPasswordResetTokenDataUsecase(passwordResetTokenRepo)
	handler := password.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getTokenDataUC,
		nil, // updatePasswordUsecase
		nil, // updateValidator
	)

	req := httptest.NewRequest(http.MethodGet, "/password/edit?token="+validToken, nil)

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// フォームが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `action="/password"`) {
		t.Error("password form not found in response")
	}

	// CSRFトークンが含まれているか確認
	if !strings.Contains(body, "csrf_token") {
		t.Error("CSRF token not found in form")
	}

	// トークンが hidden フィールドに含まれているか確認
	if !strings.Contains(body, validToken) {
		t.Error("token not found in form")
	}

	// _method=PATCH が含まれているか確認
	if !strings.Contains(body, `value="PATCH"`) {
		t.Error("method override not found in form")
	}
}

func TestEdit_TokenNotFound(t *testing.T) {
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
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ハンドラーを初期化
	getTokenDataUC := usecase.NewGetPasswordResetTokenDataUsecase(passwordResetTokenRepo)
	handler := password.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getTokenDataUC,
		nil, // updatePasswordUsecase
		nil, // updateValidator
	)

	req := httptest.NewRequest(http.MethodGet, "/password/edit?token=invalid-token", nil)

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	// ステータスコードを検証（エラーページもOKで返す）
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "無効なリンク") {
		t.Error("invalid token error message not found in response")
	}
}

func TestEdit_TokenEmpty(t *testing.T) {
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
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ハンドラーを初期化
	getTokenDataUC := usecase.NewGetPasswordResetTokenDataUsecase(passwordResetTokenRepo)
	handler := password.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getTokenDataUC,
		nil, // updatePasswordUsecase
		nil, // updateValidator
	)

	req := httptest.NewRequest(http.MethodGet, "/password/edit", nil)

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "無効なリンク") {
		t.Error("invalid token error message not found in response")
	}
}

func TestEdit_TokenExpired(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).Build()

	// 期限切れトークンを作成
	expiredToken := "expired-edit-test-token"
	expiredTokenDigest := password_reset.HashToken(expiredToken)
	testutil.NewPasswordResetTokenBuilder(t, tx).
		WithUserID(userID).
		WithTokenDigest(expiredTokenDigest).
		WithExpiresAt(time.Now().Add(-1 * time.Hour)).
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
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ハンドラーを初期化
	getTokenDataUC := usecase.NewGetPasswordResetTokenDataUsecase(passwordResetTokenRepo)
	handler := password.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getTokenDataUC,
		nil, // updatePasswordUsecase
		nil, // updateValidator
	)

	req := httptest.NewRequest(http.MethodGet, "/password/edit?token="+expiredToken, nil)

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "有効期限が切れています") {
		t.Error("expired token error message not found in response")
	}
}

func TestEdit_TokenUsed(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).Build()

	// 使用済みトークンを作成
	usedToken := "used-edit-test-token"
	usedTokenDigest := password_reset.HashToken(usedToken)
	usedAt := time.Now().Add(-30 * time.Minute)
	testutil.NewPasswordResetTokenBuilder(t, tx).
		WithUserID(userID).
		WithTokenDigest(usedTokenDigest).
		WithExpiresAt(time.Now().Add(1 * time.Hour)).
		WithUsedAt(usedAt).
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
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ハンドラーを初期化
	getTokenDataUC := usecase.NewGetPasswordResetTokenDataUsecase(passwordResetTokenRepo)
	handler := password.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getTokenDataUC,
		nil, // updatePasswordUsecase
		nil, // updateValidator
	)

	req := httptest.NewRequest(http.MethodGet, "/password/edit?token="+usedToken, nil)

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "既に使用されています") {
		t.Error("used token error message not found in response")
	}
}

func TestEdit_I18n_English(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).Build()

	// 有効なトークンを作成
	validToken := "valid-edit-test-token-en"
	validTokenDigest := password_reset.HashToken(validToken)
	testutil.NewPasswordResetTokenBuilder(t, tx).
		WithUserID(userID).
		WithTokenDigest(validTokenDigest).
		WithExpiresAt(time.Now().Add(1 * time.Hour)).
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
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ハンドラーを初期化
	getTokenDataUC := usecase.NewGetPasswordResetTokenDataUsecase(passwordResetTokenRepo)
	handler := password.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getTokenDataUC,
		nil, // updatePasswordUsecase
		nil, // updateValidator
	)

	req := httptest.NewRequest(http.MethodGet, "/password/edit?token="+validToken, nil)

	// 英語ロケールを設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "en")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// 英語のテキストが含まれているか確認
	body := rr.Body.String()
	expectedTexts := []string{
		"Set New Password",
		"New Password",
		"Change password",
	}

	for _, expected := range expectedTexts {
		if !strings.Contains(body, expected) {
			t.Errorf("expected text not found: %s", expected)
		}
	}
}
