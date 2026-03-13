package password_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
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
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestUpdate_Success(t *testing.T) {
	// このテストはUsecaseの内部動作（トランザクション）とテストのトランザクションが
	// 競合するため、スキップします。Usecaseの動作は usecase/update_password_reset_test.go
	// でテストされています。
	t.Skip("Usecase uses separate transaction, tested in usecase package")
}

func TestUpdate_ValidationError_EmptyPassword(t *testing.T) {
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
	passwordUpdateValidator := validator.NewPasswordUpdateValidator(passwordResetTokenRepo)
	getTokenDataUC := usecase.NewGetPasswordResetTokenDataUsecase(passwordResetTokenRepo)
	handler := password.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getTokenDataUC,
		nil, // updatePasswordUsecase
		passwordUpdateValidator,
	)

	// フォームデータを作成（パスワードが空）
	form := url.Values{}
	form.Set("token", "some-token")
	form.Set("password", "")
	form.Set("password_confirmation", "newpassword123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// フォームが再表示されることを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "パスワードを入力してください") {
		t.Error("password required error message not found in response")
	}
}

func TestUpdate_ValidationError_PasswordMismatch(t *testing.T) {
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
	passwordUpdateValidator := validator.NewPasswordUpdateValidator(passwordResetTokenRepo)
	getTokenDataUC := usecase.NewGetPasswordResetTokenDataUsecase(passwordResetTokenRepo)
	handler := password.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getTokenDataUC,
		nil, // updatePasswordUsecase
		passwordUpdateValidator,
	)

	// フォームデータを作成（パスワード不一致）
	form := url.Values{}
	form.Set("token", "some-token")
	form.Set("password", "newpassword123")
	form.Set("password_confirmation", "different456")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// フォームが再表示されることを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "パスワードが一致しません") {
		t.Error("password mismatch error message not found in response")
	}
}

func TestUpdate_ValidationError_InvalidToken(t *testing.T) {
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
	passwordUpdateValidator := validator.NewPasswordUpdateValidator(passwordResetTokenRepo)
	getTokenDataUC := usecase.NewGetPasswordResetTokenDataUsecase(passwordResetTokenRepo)
	handler := password.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getTokenDataUC,
		nil, // updatePasswordUsecase
		passwordUpdateValidator,
	)

	// フォームデータを作成（無効なトークン）
	form := url.Values{}
	form.Set("token", "invalid-token")
	form.Set("password", "newpassword123")
	form.Set("password_confirmation", "newpassword123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// フォームが再表示されることを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "無効なリンク") {
		t.Error("invalid token error message not found in response")
	}
}

func TestUpdate_ValidationError_ExpiredToken(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).Build()

	// 期限切れトークンを作成
	expiredToken := "expired-update-test-token"
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
	passwordUpdateValidator := validator.NewPasswordUpdateValidator(passwordResetTokenRepo)
	getTokenDataUC := usecase.NewGetPasswordResetTokenDataUsecase(passwordResetTokenRepo)
	handler := password.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getTokenDataUC,
		nil, // updatePasswordUsecase
		passwordUpdateValidator,
	)

	// フォームデータを作成
	form := url.Values{}
	form.Set("token", expiredToken)
	form.Set("password", "newpassword123")
	form.Set("password_confirmation", "newpassword123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// フォームが再表示されることを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "有効期限が切れています") {
		t.Error("expired token error message not found in response")
	}
}

func TestUpdate_ValidationError_UsedToken(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).Build()

	// 使用済みトークンを作成
	usedToken := "used-update-test-token"
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
	passwordUpdateValidator := validator.NewPasswordUpdateValidator(passwordResetTokenRepo)
	getTokenDataUC := usecase.NewGetPasswordResetTokenDataUsecase(passwordResetTokenRepo)
	handler := password.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getTokenDataUC,
		nil, // updatePasswordUsecase
		passwordUpdateValidator,
	)

	// フォームデータを作成
	form := url.Values{}
	form.Set("token", usedToken)
	form.Set("password", "newpassword123")
	form.Set("password_confirmation", "newpassword123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// フォームが再表示されることを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "既に使用されています") {
		t.Error("used token error message not found in response")
	}
}
