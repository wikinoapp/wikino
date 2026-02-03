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
)

func TestCreate_WithoutPendingUser(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)

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
	createValidator := sign_in_two_factor.NewCreateValidator(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	handler := sign_in_two_factor.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		createValidator,
		createUserSessionUC,
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/sign_in" {
		t.Errorf("wrong redirect location: got %v want /sign_in", location)
	}
}

func TestCreate_InvalidTOTPCodeFormat(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)

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
	createValidator := sign_in_two_factor.NewCreateValidator(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	handler := sign_in_two_factor.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		createValidator,
		createUserSessionUC,
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

	// 200 OK（フォーム再表示）が返されるか確認
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// エラーメッセージが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `name="totp_code"`) {
		t.Error("totp_code input field not found in response")
	}
}

func TestCreate_InvalidTOTPCode(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)

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
	createValidator := sign_in_two_factor.NewCreateValidator(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	handler := sign_in_two_factor.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		createValidator,
		createUserSessionUC,
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
		Value: userID,
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// 200 OK（フォーム再表示）が返されるか確認
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// エラーメッセージが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `name="totp_code"`) {
		t.Error("totp_code input field not found in response")
	}
}

func TestCreate_ValidTOTPCode(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)

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
	createValidator := sign_in_two_factor.NewCreateValidator(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	handler := sign_in_two_factor.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		createValidator,
		createUserSessionUC,
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
		Value: userID,
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// ホームページにリダイレクトされるか確認
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("wrong redirect location: got %v want /", location)
	}

	// セッションCookieが設定されているか確認
	cookies := rr.Result().Cookies()
	var sessionCookieFound bool
	for _, c := range cookies {
		if c.Name == session.CookieName {
			sessionCookieFound = true
			if c.Value == "" {
				t.Error("session cookie value is empty")
			}
			break
		}
	}
	if !sessionCookieFound {
		t.Error("session cookie not found")
	}

	// ペンディングユーザーIDのCookieが削除されているか確認
	for _, c := range cookies {
		if c.Name == session.PendingUserCookieName {
			if c.MaxAge >= 0 && c.Value != "" {
				t.Error("pending user cookie should be deleted")
			}
			break
		}
	}
}
