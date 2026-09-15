package sign_in_two_factor_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in_two_factor"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestNew_WithPendingUser(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)

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
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	createValidator := validator.NewSignInTwoFactorCreateValidator(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)
	createTwoFactorSessionUC := usecase.NewCreateTwoFactorSessionUsecase(createValidator, createUserSessionUC)

	handler := sign_in_two_factor.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		createTwoFactorSessionUC,
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	// レスポンスボディを検証
	body := rr.Body.String()

	// 2FAフォームが含まれているか確認
	if !strings.Contains(body, `action="/sign_in/two_factor"`) {
		t.Error("レスポンスに2FAフォームの送信先が見つからない")
	}

	// CSRFトークンが含まれているか確認
	if !strings.Contains(body, "test-csrf-token") {
		t.Error("レスポンスにCSRFトークンが見つからない")
	}

	// TOTPコード入力フィールドが含まれているか確認
	if !strings.Contains(body, `name="totp_code"`) {
		t.Error("レスポンスにtotp_codeの入力フィールドが見つからない")
	}

	// リカバリーコードへのリンクが含まれているか確認
	if !strings.Contains(body, `/sign_in/two_factor/recovery/new`) {
		t.Error("レスポンスにリカバリーコードのリンクが見つからない")
	}

	for _, notWant := range []string{`<link rel="canonical"`, `property="og:url"`} {
		if strings.Contains(body, notWant) {
			t.Errorf("レスポンスに想定外の%qが含まれている", notWant)
		}
	}
}

func TestNew_WithoutPendingUser(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)

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
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	createValidator := validator.NewSignInTwoFactorCreateValidator(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)
	createTwoFactorSessionUC := usecase.NewCreateTwoFactorSessionUsecase(createValidator, createUserSessionUC)

	handler := sign_in_two_factor.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		createTwoFactorSessionUC,
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/sign_in" {
		t.Errorf("リダイレクト先 = %v、期待値 = /sign_in", location)
	}
}

// TestNew_BackParameterは、サインイン画面から渡されたbackを隠しフィールドと
// リカバリーコード画面へのリンクの両方で引き継ぐことを検証する。
// リンクに載せないと、リカバリーコードに切り替えた時点で戻り先が失われる。
func TestNew_BackParameter(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)

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
	createValidator := validator.NewSignInTwoFactorCreateValidator(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)
	createTwoFactorSessionUC := usecase.NewCreateTwoFactorSessionUsecase(createValidator, createUserSessionUC)

	handler := sign_in_two_factor.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		createTwoFactorSessionUC,
	)

	backURL := "/s/example/topics/1/pages/new?title=%E3%83%A1%E3%83%A2"

	tests := []struct {
		name       string
		target     string
		wantInBody []string
	}{
		{
			name:   "backパラメータあり",
			target: "/sign_in/two_factor/new?back=" + url.QueryEscape(backURL),
			wantInBody: []string{
				`name="back" value="` + backURL + `"`,
				`href="/sign_in/two_factor/recovery/new?back=` + url.QueryEscape(backURL) + `"`,
			},
		},
		{
			name:   "backパラメータなし",
			target: "/sign_in/two_factor/new",
			wantInBody: []string{
				`name="back" value=""`,
				`href="/sign_in/two_factor/recovery/new"`,
			},
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
				t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()
			for _, want := range tt.wantInBody {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}
		})
	}
}

// TestNew_WithoutPendingUserCarriesBackParameterは、pending user cookieが期限切れでも
// 遷移先が失われないことを検証する。cookieの寿命は10分しかないため、コードの確認に手間取った
// 訪問者が求めたページを失わないようにする。
func TestNew_WithoutPendingUserCarriesBackParameter(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)

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
	createValidator := validator.NewSignInTwoFactorCreateValidator(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)
	createTwoFactorSessionUC := usecase.NewCreateTwoFactorSessionUsecase(createValidator, createUserSessionUC)

	handler := sign_in_two_factor.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		createTwoFactorSessionUC,
	)

	backURL := "/s/example/topics/1/pages/new?title=%E3%83%A1%E3%83%A2"

	req := httptest.NewRequest(http.MethodGet, "/sign_in/two_factor/new?back="+url.QueryEscape(backURL), nil)
	req.Header.Set("Accept-Language", "ja")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	wantLocation := "/sign_in?back=" + url.QueryEscape(backURL)
	if location := rr.Header().Get("Location"); location != wantLocation {
		t.Errorf("リダイレクト先 = %v、期待値 = %v", location, wantLocation)
	}
}
