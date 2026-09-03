package sign_in_two_factor_recovery_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in_two_factor_recovery"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestNew_WithPendingUser(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)

	// テスト用のクエリとリポジトリを作成
	q := testutil.QueriesWithTx(tx)
	userRepo := repository.NewUserRepository(q)
	userSessionRepo := repository.NewUserSessionRepository(q)
	userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// ユースケースとセッションマネージャーを作成
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	createValidator := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)
	createRecoveryCodeSessionUC := usecase.NewCreateRecoveryCodeSessionUsecase(db, createValidator, userTwoFactorAuthRepo, userSessionRepo)

	handler := sign_in_two_factor_recovery.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		createRecoveryCodeSessionUC,
	)

	// ペンディングユーザーIDを持つHTTPリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/sign_in/two_factor/recovery/new", nil)
	req.Header.Set("Accept-Language", "ja")

	// ペンディングユーザーIDのCookieを追加
	req.AddCookie(&http.Cookie{
		Name:  session.PendingUserCookieName,
		Value: "test-user-id",
	})

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// レスポンスボディを検証
	body := rr.Body.String()

	// リカバリーコードフォームが含まれているか確認
	if !strings.Contains(body, `action="/sign_in/two_factor/recovery"`) {
		t.Error("recovery form action not found in response")
	}

	// CSRFトークンが含まれているか確認
	if !strings.Contains(body, "test-csrf-token") {
		t.Error("CSRF token not found in response")
	}

	// リカバリーコード入力フィールドが含まれているか確認
	if !strings.Contains(body, `name="recovery_code"`) {
		t.Error("recovery_code input field not found in response")
	}

	for _, notWant := range []string{`<link rel="canonical"`, `property="og:url"`} {
		if strings.Contains(body, notWant) {
			t.Errorf("response unexpectedly contains %q", notWant)
		}
	}
}

func TestNew_WithoutPendingUser(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)

	// テスト用のクエリとリポジトリを作成
	q := testutil.QueriesWithTx(tx)
	userRepo := repository.NewUserRepository(q)
	userSessionRepo := repository.NewUserSessionRepository(q)
	userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// ユースケースとセッションマネージャーを作成
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	createValidator := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)
	createRecoveryCodeSessionUC := usecase.NewCreateRecoveryCodeSessionUsecase(db, createValidator, userTwoFactorAuthRepo, userSessionRepo)

	handler := sign_in_two_factor_recovery.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		createRecoveryCodeSessionUC,
	)

	// ペンディングユーザーIDなしのHTTPリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/sign_in/two_factor/recovery/new", nil)
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// ログインページにリダイレクトされるか確認
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/sign_in" {
		t.Errorf("wrong redirect location: got %v want /sign_in", location)
	}
}

// TestNew_BackParameter verifies that the back handed over by the two-factor screen is carried by
// the hidden field, so that authenticating with a recovery code reaches the same destination.
//
// [Ja] TestNew_BackParameter は、二要素認証画面から渡された back を隠しフィールドで引き継ぐことを
// 検証する。リカバリーコードで認証したときも同じ宛先へ戻せるようにするため。
func TestNew_BackParameter(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)

	q := testutil.QueriesWithTx(tx)
	userRepo := repository.NewUserRepository(q)
	userSessionRepo := repository.NewUserSessionRepository(q)
	userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)

	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	createValidator := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)
	createRecoveryCodeSessionUC := usecase.NewCreateRecoveryCodeSessionUsecase(db, createValidator, userTwoFactorAuthRepo, userSessionRepo)

	handler := sign_in_two_factor_recovery.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		createRecoveryCodeSessionUC,
	)

	backURL := "/s/example/topics/1/pages/new?title=%E3%83%A1%E3%83%A2"

	tests := []struct {
		name       string
		target     string
		wantInBody string
	}{
		{
			name:       "backパラメータあり",
			target:     "/sign_in/two_factor/recovery/new?back=" + url.QueryEscape(backURL),
			wantInBody: `name="back" value="` + backURL + `"`,
		},
		{
			name:       "backパラメータなし",
			target:     "/sign_in/two_factor/recovery/new",
			wantInBody: `name="back" value=""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			req.Header.Set("Accept-Language", "ja")
			req.AddCookie(&http.Cookie{
				Name:  session.PendingUserCookieName,
				Value: "test-user-id",
			})

			ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.New(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
			}
			if !strings.Contains(rr.Body.String(), tt.wantInBody) {
				t.Errorf("response doesn't contain %q", tt.wantInBody)
			}
		})
	}
}

// TestNew_WithoutPendingUserCarriesBackParameter verifies that the destination survives an expired
// pending user cookie. The cookie only lives for ten minutes, so a visitor who takes a while to
// find their recovery code would otherwise lose the page they asked for.
//
// [Ja] TestNew_WithoutPendingUserCarriesBackParameter は、pending user cookie が期限切れでも
// 遷移先が失われないことを検証する。cookie の寿命は 10 分しかないため、リカバリーコードを探すのに
// 手間取った訪問者が求めたページを失わないようにする。
func TestNew_WithoutPendingUserCarriesBackParameter(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)

	q := testutil.QueriesWithTx(tx)
	userRepo := repository.NewUserRepository(q)
	userSessionRepo := repository.NewUserSessionRepository(q)
	userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)

	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	createValidator := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)
	createRecoveryCodeSessionUC := usecase.NewCreateRecoveryCodeSessionUsecase(db, createValidator, userTwoFactorAuthRepo, userSessionRepo)

	handler := sign_in_two_factor_recovery.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		createRecoveryCodeSessionUC,
	)

	backURL := "/s/example/topics/1/pages/new?title=%E3%83%A1%E3%83%A2"

	req := httptest.NewRequest(http.MethodGet, "/sign_in/two_factor/recovery/new?back="+url.QueryEscape(backURL), nil)
	req.Header.Set("Accept-Language", "ja")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	wantLocation := "/sign_in?back=" + url.QueryEscape(backURL)
	if location := rr.Header().Get("Location"); location != wantLocation {
		t.Errorf("wrong redirect location: got %v want %v", location, wantLocation)
	}
}
