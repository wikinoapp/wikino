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
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestCreate_WithoutPendingUser(t *testing.T) {
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

			if rr.Code != http.StatusUnprocessableEntity {
				t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
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

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// リカバリーコード入力フィールドが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `name="recovery_code"`) {
		t.Error("recovery_code input field not found in response")
	}
}

func TestCreate_InvalidRecoveryCode(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)

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
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	createValidator := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)
	createRecoveryCodeSessionUC := usecase.NewCreateRecoveryCodeSessionUsecase(db, createValidator, userTwoFactorAuthRepo, userSessionRepo)

	handler := sign_in_two_factor_recovery.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		createRecoveryCodeSessionUC,
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
		Value: userID.String(),
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// リカバリーコード入力フィールドが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `name="recovery_code"`) {
		t.Error("recovery_code input field not found in response")
	}
}

func TestCreate_ValidRecoveryCode(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()

	// ユーザーと2FA設定を作成（リカバリーコード付き）
	secret := "JBSWY3DPEHPK3PXP"
	validCode := "code1234"
	recoveryCodes := []string{validCode, "code5678", "abcd1234"}
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("recovery-valid@example.com").
		WithAtname("recovery_valid_user").
		BuildWithTwoFactorAuthAndRecoveryCodes(secret, true, recoveryCodes)

	q := query.New(db)
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

	// 正しいリカバリーコードでリクエスト
	form := url.Values{}
	form.Add("recovery_code", validCode)
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor/recovery", strings.NewReader(form.Encode()))
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

	// Verify the sign-in success flash, so that a visitor sent on to a deep page still sees that
	// they signed in.
	//
	// [Ja] 深いページへ送られた訪問者にもサインインできたと伝わるよう、サインイン成功の
	// フラッシュを検証する。
	var flashCookieFound bool
	for _, c := range cookies {
		if c.Name == session.FlashCookieName {
			flashCookieFound = true
			break
		}
	}
	if !flashCookieFound {
		t.Error("flash cookie not found")
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

	db := testutil.GetTestDB()

	// 2FAが無効なユーザーを作成
	secret := "JBSWY3DPEHPK3PXP"
	recoveryCodes := []string{"code1234"}
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("2fa-disabled@example.com").
		WithAtname("2fa_disabled_user").
		BuildWithTwoFactorAuthAndRecoveryCodes(secret, false, recoveryCodes)

	q := query.New(db)
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

	backURL := "/s/example/pages/1/edit"

	form := url.Values{}
	form.Add("recovery_code", "code1234")
	form.Add("back", backURL)
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor/recovery", strings.NewReader(form.Encode()))
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

	// ログインページにリダイレクトされるか確認
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	// The back parameter is carried even when the visitor is sent back for having no two-factor
	// authentication enabled.
	//
	// [Ja] 二要素認証が無効で引き返すときも back を引き継ぐことを確認する。
	wantLocation := "/sign_in?back=" + url.QueryEscape(backURL)
	if location := rr.Header().Get("Location"); location != wantLocation {
		t.Errorf("wrong redirect location: got %v want %v", location, wantLocation)
	}
}

// TestCreate_BackParameter verifies where a visitor lands after authenticating with a recovery
// code. A safe back in the form returns them to their original destination, and anything else falls
// back to the home page.
//
// [Ja] TestCreate_BackParameter は、リカバリーコードで認証した後のリダイレクト先を検証する。
// フォームの back が安全なら元の宛先へ戻し、そうでなければホームへ落とす。
func TestCreate_BackParameter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		email        string
		atname       string
		recoveryCode string
		backURL      string
		wantLocation string
	}{
		{
			name:         "有効なbackパラメータのURLへ戻る",
			email:        "recovery-back-ok@example.com",
			atname:       "recovery_back_ok_user",
			recoveryCode: "backok12",
			backURL:      "/s/example/pages/1/edit",
			wantLocation: "/s/example/pages/1/edit",
		},
		{
			name:         "危険なbackパラメータはホームへ落とす",
			email:        "recovery-back-ng@example.com",
			atname:       "recovery_back_ng_user",
			recoveryCode: "backng12",
			backURL:      "https://evil.com",
			wantLocation: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testutil.GetTestDB()

			userID := testutil.NewUserBuilderDB(t, db).
				WithEmail(tt.email).
				WithAtname(tt.atname).
				BuildWithTwoFactorAuthAndRecoveryCodes("JBSWY3DPEHPK3PXP", true, []string{tt.recoveryCode})

			q := query.New(db)
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

			form := url.Values{}
			form.Add("recovery_code", tt.recoveryCode)
			form.Add("back", tt.backURL)
			req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor/recovery", strings.NewReader(form.Encode()))
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
				t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
			}
			if location := rr.Header().Get("Location"); location != tt.wantLocation {
				t.Errorf("wrong redirect location: got %v want %v", location, tt.wantLocation)
			}
		})
	}
}

// TestCreate_ValidationErrorPreservesBackParameter verifies that back survives the re-render that
// follows a wrong recovery code.
//
// [Ja] TestCreate_ValidationErrorPreservesBackParameter は、リカバリーコードを間違えてフォームが
// 再描画されるときも back を保持することを検証する。
func TestCreate_ValidationErrorPreservesBackParameter(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()

	q := query.New(db)
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

	form := url.Values{}
	// A code shorter than eight digits fails the format check.
	//
	// [Ja] 8 桁未満のコードは形式チェックに引っかかる。
	form.Add("recovery_code", "short")
	form.Add("back", "/s/example/pages/1/edit")
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor/recovery", strings.NewReader(form.Encode()))
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	wantInBody := `name="back" value="/s/example/pages/1/edit"`
	if !strings.Contains(rr.Body.String(), wantInBody) {
		t.Errorf("backパラメータがフォームに保持されていません\nwant: %s", wantInBody)
	}
}

// TestCreate_WithoutPendingUserCarriesBackParameter verifies that the destination in the form
// survives an expired pending user cookie, so that signing in again still reaches the page the
// visitor asked for.
//
// [Ja] TestCreate_WithoutPendingUserCarriesBackParameter は、pending user cookie が期限切れでも
// フォームの遷移先が失われないことを検証する。サインインし直したときも訪問者が求めたページへ
// 着けるようにするため。
func TestCreate_WithoutPendingUserCarriesBackParameter(t *testing.T) {
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

	backURL := "/s/example/pages/1/edit"

	form := url.Values{}
	form.Add("recovery_code", "abc12345")
	form.Add("back", backURL)
	req := httptest.NewRequest(http.MethodPost, "/sign_in/two_factor/recovery", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	wantLocation := "/sign_in?back=" + url.QueryEscape(backURL)
	if location := rr.Header().Get("Location"); location != wantLocation {
		t.Errorf("wrong redirect location: got %v want %v", location, wantLocation)
	}
}
