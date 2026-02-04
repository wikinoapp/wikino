package sign_in_two_factor_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in_two_factor"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestNew_WithPendingUser(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)

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

	// バリデーターとセッションマネージャーを作成
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	createValidator := sign_in_two_factor.NewCreateValidator(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	handler := sign_in_two_factor.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		createValidator,
		createUserSessionUC,
	)

	// ペンディングユーザーIDを持つHTTPリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/sign_in/two_factor/new", nil)
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

	// 2FAフォームが含まれているか確認
	if !strings.Contains(body, `action="/sign_in/two_factor"`) {
		t.Error("2FA form action not found in response")
	}

	// CSRFトークンが含まれているか確認
	if !strings.Contains(body, "test-csrf-token") {
		t.Error("CSRF token not found in response")
	}

	// TOTPコード入力フィールドが含まれているか確認
	if !strings.Contains(body, `name="totp_code"`) {
		t.Error("totp_code input field not found in response")
	}

	// リカバリーコードへのリンクが含まれているか確認
	if !strings.Contains(body, `/sign_in/two_factor/recovery/new`) {
		t.Error("recovery code link not found in response")
	}
}

func TestNew_WithoutPendingUser(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)

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

	// バリデーターとセッションマネージャーを作成
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	createValidator := sign_in_two_factor.NewCreateValidator(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	handler := sign_in_two_factor.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		createValidator,
		createUserSessionUC,
	)

	// ペンディングユーザーIDなしのHTTPリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/sign_in/two_factor/new", nil)
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
