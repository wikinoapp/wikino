package sign_in_two_factor_recovery_test

import (
	"context"
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
	createValidator := sign_in_two_factor_recovery.NewCreateValidator(userTwoFactorAuthRepo)
	consumeRecoveryCodeUC := usecase.NewConsumeRecoveryCodeUsecase(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	handler := sign_in_two_factor_recovery.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		createValidator,
		consumeRecoveryCodeUC,
		createUserSessionUC,
	)

	// ペンディングユーザーIDなしのリクエスト
	form := url.Values{}
	form.Add("recovery_code", "abc12345")
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor/recovery", strings.NewReader(form.Encode()))
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

func TestCreate_InvalidRecoveryCodeFormat(t *testing.T) {
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
	createValidator := sign_in_two_factor_recovery.NewCreateValidator(userTwoFactorAuthRepo)
	consumeRecoveryCodeUC := usecase.NewConsumeRecoveryCodeUsecase(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	handler := sign_in_two_factor_recovery.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		createValidator,
		consumeRecoveryCodeUC,
		createUserSessionUC,
	)

	testCases := []struct {
		name string
		code string
	}{
		{"7文字", "abc1234"},
		{"9文字", "abc123456"},
		{"大文字を含む", "ABC12345"},
		{"記号を含む", "abc1234!"},
		{"空白を含む", "abc1 345"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			form := url.Values{}
			form.Add("recovery_code", tc.code)
			req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor/recovery", strings.NewReader(form.Encode()))
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

			// リカバリーコード入力フィールドが含まれているか確認
			body := rr.Body.String()
			if !strings.Contains(body, `name="recovery_code"`) {
				t.Error("recovery_code input field not found in response")
			}
		})
	}
}

func TestCreate_EmptyRecoveryCode(t *testing.T) {
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
	createValidator := sign_in_two_factor_recovery.NewCreateValidator(userTwoFactorAuthRepo)
	consumeRecoveryCodeUC := usecase.NewConsumeRecoveryCodeUsecase(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	handler := sign_in_two_factor_recovery.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		createValidator,
		consumeRecoveryCodeUC,
		createUserSessionUC,
	)

	// 空のリカバリーコードでリクエスト
	form := url.Values{}
	form.Add("recovery_code", "")
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor/recovery", strings.NewReader(form.Encode()))
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

	// リカバリーコード入力フィールドが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `name="recovery_code"`) {
		t.Error("recovery_code input field not found in response")
	}
}

func TestCreate_InvalidRecoveryCode(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)

	// ユーザーと2FA設定を作成（リカバリーコード付き）
	secret := "JBSWY3DPEHPK3PXP"
	recoveryCodes := []string{"code1234", "code5678", "abcd1234"}
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("recovery-test@example.com").
		WithAtname("recovery_test_user").
		BuildWithTwoFactorAuthAndRecoveryCodes(secret, true, recoveryCodes)

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
	createValidator := sign_in_two_factor_recovery.NewCreateValidator(userTwoFactorAuthRepo)
	consumeRecoveryCodeUC := usecase.NewConsumeRecoveryCodeUsecase(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	handler := sign_in_two_factor_recovery.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		createValidator,
		consumeRecoveryCodeUC,
		createUserSessionUC,
	)

	// 間違ったリカバリーコードでリクエスト
	form := url.Values{}
	form.Add("recovery_code", "wrongcod") // 存在しないコード
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor/recovery", strings.NewReader(form.Encode()))
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

	// リカバリーコード入力フィールドが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `name="recovery_code"`) {
		t.Error("recovery_code input field not found in response")
	}
}

func TestCreate_ValidRecoveryCode(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)

	// ユーザーと2FA設定を作成（リカバリーコード付き）
	secret := "JBSWY3DPEHPK3PXP"
	validCode := "code1234"
	recoveryCodes := []string{validCode, "code5678", "abcd1234"}
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("recovery-valid@example.com").
		WithAtname("recovery_valid_user").
		BuildWithTwoFactorAuthAndRecoveryCodes(secret, true, recoveryCodes)

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
	createValidator := sign_in_two_factor_recovery.NewCreateValidator(userTwoFactorAuthRepo)
	consumeRecoveryCodeUC := usecase.NewConsumeRecoveryCodeUsecase(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	handler := sign_in_two_factor_recovery.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		createValidator,
		consumeRecoveryCodeUC,
		createUserSessionUC,
	)

	// 正しいリカバリーコードでリクエスト
	form := url.Values{}
	form.Add("recovery_code", validCode)
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor/recovery", strings.NewReader(form.Encode()))
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

	// リカバリーコードが消費されたか確認
	twoFactorAuth, err := userTwoFactorAuthRepo.FindEnabledByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("2FA設定の取得に失敗: %v", err)
	}
	if twoFactorAuth == nil {
		t.Fatal("2FA設定が見つかりません")
	}

	// 使用したコードがリストから削除されているか確認
	for _, code := range twoFactorAuth.RecoveryCodes {
		if code == validCode {
			t.Errorf("使用済みのリカバリーコード %s がまだリストに存在します", validCode)
		}
	}

	// 残りのコードは保持されているか確認
	if len(twoFactorAuth.RecoveryCodes) != 2 {
		t.Errorf("リカバリーコードの数が不正: got %d want 2", len(twoFactorAuth.RecoveryCodes))
	}
}

func TestCreate_TwoFactorNotEnabled(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)

	// 2FAが無効なユーザーを作成
	secret := "JBSWY3DPEHPK3PXP"
	recoveryCodes := []string{"code1234"}
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("2fa-disabled@example.com").
		WithAtname("2fa_disabled_user").
		BuildWithTwoFactorAuthAndRecoveryCodes(secret, false, recoveryCodes)

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
	createValidator := sign_in_two_factor_recovery.NewCreateValidator(userTwoFactorAuthRepo)
	consumeRecoveryCodeUC := usecase.NewConsumeRecoveryCodeUsecase(userTwoFactorAuthRepo)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	handler := sign_in_two_factor_recovery.NewHandler(
		cfg,
		sessionMgr,
		userRepo,
		createValidator,
		consumeRecoveryCodeUC,
		createUserSessionUC,
	)

	form := url.Values{}
	form.Add("recovery_code", "code1234")
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor/recovery", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")

	req.AddCookie(&http.Cookie{
		Name:  session.PendingUserCookieName,
		Value: userID,
	})

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
