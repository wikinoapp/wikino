package sign_in_two_factor_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in_two_factor"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestCreate_WithoutPendingUser(t *testing.T) {
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

	// ペンディングユーザーIDなしのリクエスト
	form := url.Values{}
	form.Add("totp_code", "123456")
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// ログインページにリダイレクトされるか確認
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/sign_in" {
		t.Errorf("リダイレクト先 = %v、期待値 = /sign_in", location)
	}
}

func TestCreate_InvalidTOTPCodeFormat(t *testing.T) {
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

	// 無効な形式のTOTPコードでリクエスト
	form := url.Values{}
	form.Add("totp_code", "12345") // 5桁
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")

	// ペンディングユーザーIDのCookieを追加
	req.AddCookie(&http.Cookie{
		Name:  session.PendingUserCookieName,
		Value: "test-user-id",
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `name="totp_code"`) {
		t.Error("レスポンスにtotp_codeの入力フィールドが見つからない")
	}
}

func TestCreate_InvalidTOTPCode(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)

	// ユーザーと2FA設定を作成
	secret := "JBSWY3DPEHPK3PXP" // テスト用の固定シークレット
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("2fa-test@example.com").
		WithAtname("2fa_test_user").
		BuildWithTwoFactorAuth(secret, true)

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

	// 間違ったTOTPコードでリクエスト
	form := url.Values{}
	form.Add("totp_code", "000000") // 間違ったコード
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")

	// ペンディングユーザーIDのCookieを追加
	req.AddCookie(&http.Cookie{
		Name:  session.PendingUserCookieName,
		Value: userID.String(),
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `name="totp_code"`) {
		t.Error("レスポンスにtotp_codeの入力フィールドが見つからない")
	}
}

func TestCreate_ValidTOTPCode(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)

	// ユーザーと2FA設定を作成
	secret := "JBSWY3DPEHPK3PXP" // テスト用の固定シークレット
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("2fa-valid@example.com").
		WithAtname("2fa_valid_user").
		BuildWithTwoFactorAuth(secret, true)

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

	// 正しいTOTPコードを生成
	validCode, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("TOTPコード生成に失敗: %v", err)
	}

	form := url.Values{}
	form.Add("totp_code", validCode)
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")

	// ペンディングユーザーIDのCookieを追加
	req.AddCookie(&http.Cookie{
		Name:  session.PendingUserCookieName,
		Value: userID.String(),
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// ホームページにリダイレクトされるか確認
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先 = %v、期待値 = /", location)
	}

	// セッションCookieが設定されているか確認
	cookies := rr.Result().Cookies()
	var sessionCookieFound bool
	for _, c := range cookies {
		if c.Name == session.CookieName {
			sessionCookieFound = true
			if c.Value == "" {
				t.Error("セッションCookieの値が空")
			}
			break
		}
	}
	if !sessionCookieFound {
		t.Error("セッションCookieが見つからない")
	}

	// 深いページへ送られた訪問者にもサインインできたと伝わるよう、サインイン成功の
	// フラッシュを検証する。
	var flashCookieFound bool
	for _, c := range cookies {
		if c.Name == session.FlashCookieName {
			flashCookieFound = true
			break
		}
	}
	if !flashCookieFound {
		t.Error("フラッシュのCookieが見つからない")
	}

	// ペンディングユーザーIDのCookieが削除されているか確認
	for _, c := range cookies {
		if c.Name == session.PendingUserCookieName {
			if c.MaxAge >= 0 && c.Value != "" {
				t.Error("保留中ユーザーのCookieが削除されていない")
			}
			break
		}
	}
}

func TestCreate_TwoFactorNotEnabled(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("2fa-disabled-back@example.com").
		WithAtname("twofa_not_enabled").
		BuildWithTwoFactorAuth("JBSWY3DPEHPK3PXP", false)

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

	backURL := "/s/example/pages/1/edit"

	form := url.Values{}
	form.Add("totp_code", "123456")
	form.Add("back", backURL)
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")
	req.AddCookie(&http.Cookie{
		Name:  session.PendingUserCookieName,
		Value: userID.String(),
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	wantLocation := "/sign_in?back=" + url.QueryEscape(backURL)
	if location := rr.Header().Get("Location"); location != wantLocation {
		t.Errorf("リダイレクト先 = %v、期待値 = %v", location, wantLocation)
	}

	var pendingUserCookieDeleted bool
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == session.PendingUserCookieName && cookie.MaxAge < 0 {
			pendingUserCookieDeleted = true
			break
		}
	}
	if !pendingUserCookieDeleted {
		t.Error("保留中ユーザーのCookieが削除されていない")
	}
}

// TestCreate_BackParameterは、二要素認証を通過した後のリダイレクト先を検証する。
// フォームのbackが安全なら元の宛先へ戻し、そうでなければホームへ落とす。
func TestCreate_BackParameter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		email        string
		atname       string
		backURL      string
		wantLocation string
	}{
		{
			name:         "有効なbackパラメータのURLへ戻る",
			email:        "2fa-back-ok@example.com",
			atname:       "2fa_back_ok_user",
			backURL:      "/s/example/pages/1/edit",
			wantLocation: "/s/example/pages/1/edit",
		},
		{
			name:         "危険なbackパラメータはホームへ落とす",
			email:        "2fa-back-ng@example.com",
			atname:       "2fa_back_ng_user",
			backURL:      "https://evil.com",
			wantLocation: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, tx := testutil.SetupTx(t)

			secret := "JBSWY3DPEHPK3PXP"
			userID := testutil.NewUserBuilder(t, tx).
				WithEmail(tt.email).
				WithAtname(tt.atname).
				BuildWithTwoFactorAuth(secret, true)

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

			validCode, err := totp.GenerateCode(secret, time.Now())
			if err != nil {
				t.Fatalf("TOTPコード生成に失敗: %v", err)
			}

			form := url.Values{}
			form.Add("totp_code", validCode)
			form.Add("back", tt.backURL)
			req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Accept-Language", "ja")
			req.AddCookie(&http.Cookie{
				Name:  session.PendingUserCookieName,
				Value: userID.String(),
			})

			ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.Create(rr, req)

			if rr.Code != http.StatusFound {
				t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
			}
			if location := rr.Header().Get("Location"); location != tt.wantLocation {
				t.Errorf("リダイレクト先 = %v、期待値 = %v", location, tt.wantLocation)
			}
		})
	}
}

// TestCreate_ValidationErrorPreservesBackParameterは、コードを間違えてフォームが再描画される
// ときもbackを保持することを検証する。落とすと、入力し直した時点で戻り先が失われる。
func TestCreate_ValidationErrorPreservesBackParameter(t *testing.T) {
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

	form := url.Values{}
	// 5桁のコードは形式チェックに引っかかる。
	form.Add("totp_code", "12345")
	form.Add("back", "/s/example/pages/1/edit")
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")
	req.AddCookie(&http.Cookie{
		Name:  session.PendingUserCookieName,
		Value: "test-user-id",
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	wantInBody := `name="back" value="/s/example/pages/1/edit"`
	if !strings.Contains(rr.Body.String(), wantInBody) {
		t.Errorf("backパラメータがフォームに保持されていません\n期待値: %s", wantInBody)
	}
}

// TestCreate_WithoutPendingUserCarriesBackParameterは、pending user cookieが期限切れでも
// フォームの遷移先が失われないことを検証する。サインインし直したときも訪問者が求めたページへ
// 着けるようにするため。
func TestCreate_WithoutPendingUserCarriesBackParameter(t *testing.T) {
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

	backURL := "/s/example/pages/1/edit"

	form := url.Values{}
	form.Add("totp_code", "123456")
	form.Add("back", backURL)
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	wantLocation := "/sign_in?back=" + url.QueryEscape(backURL)
	if location := rr.Header().Get("Location"); location != wantLocation {
		t.Errorf("リダイレクト先 = %v、期待値 = %v", location, wantLocation)
	}
}
