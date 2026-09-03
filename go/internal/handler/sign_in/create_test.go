package sign_in_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// mockTurnstileVerifier はテスト用のTurnstile検証モック
type mockTurnstileVerifier struct {
	valid bool
	err   error
}

func (m *mockTurnstileVerifier) Verify(ctx context.Context, token string) (bool, error) {
	return m.valid, m.err
}

func setupHandler(t *testing.T, tx *sql.Tx, turnstileValid bool) *sign_in.Handler {
	t.Helper()

	queries := testutil.QueriesWithTx(tx)

	cfg := &config.Config{
		Env:                "test",
		Port:               "8080",
		Domain:             "localhost",
		CookieDomain:       "",
		SessionSecure:      false,
		SessionHTTPOnly:    true,
		TurnstileSiteKey:   "test-site-key",
		TurnstileSecretKey: "",
	}

	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(queries)

	signInValidator := validator.NewSignInCreateValidator(userRepo, userPasswordRepo, userTwoFactorAuthRepo)
	signInUC := usecase.NewCreateSignInUsecase(signInValidator, userSessionRepo)

	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	mockTurnstile := &mockTurnstileVerifier{valid: turnstileValid, err: nil}

	return sign_in.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		signInUC,
		mockTurnstile,
	)
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)

	password := "testpassword123"
	passwordDigest, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("test@example.com").
		WithAtname("testuser").
		BuildWithPassword(passwordDigest)

	handler := setupHandler(t, tx, true)

	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("password", password)
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("wrong redirect location: got %v want /", location)
	}

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

	if sessionCookie != nil {
		queries := testutil.QueriesWithTx(tx)
		userSessionRepo := repository.NewUserSessionRepository(queries)
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

	_, tx := testutil.SetupTx(t)
	handler := setupHandler(t, tx, true)

	form := url.Values{}
	form.Set("email", "invalid-email")
	form.Set("password", "password123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `action="/sign_in"`) {
		t.Error("login form not found in response")
	}
}

func TestCreate_WrongPassword(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)

	password := "testpassword123"
	passwordDigest, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}

	_ = testutil.NewUserBuilder(t, tx).
		WithEmail("wrong-pw@example.com").
		WithAtname("wrongpw").
		BuildWithPassword(passwordDigest)

	handler := setupHandler(t, tx, true)

	form := url.Values{}
	form.Set("email", "wrong-pw@example.com")
	form.Set("password", "wrongpassword")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == session.CookieName {
			t.Error("session cookie should not be set for wrong password")
		}
	}
}

func TestCreate_UserNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	handler := setupHandler(t, tx, true)

	form := url.Values{}
	form.Set("email", "nonexistent@example.com")
	form.Set("password", "password123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == session.CookieName {
			t.Error("session cookie should not be set for nonexistent user")
		}
	}
}

func TestCreate_TurnstileFailure(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	handler := setupHandler(t, tx, false)

	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("password", "password123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "invalid-token")

	req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == session.CookieName {
			t.Error("session cookie should not be set for Turnstile failure")
		}
	}
}

func TestCreate_WithBackParameter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		backURL          string
		wantRedirectPath string
	}{
		{
			name:             "有効なbackパラメータ",
			backURL:          "/dashboard",
			wantRedirectPath: "/dashboard",
		},
		{
			name:             "クエリパラメータ付きのbackパラメータ",
			backURL:          "/oauth/authorize?client_id=xxx",
			wantRedirectPath: "/oauth/authorize?client_id=xxx",
		},
		{
			name:             "backパラメータなし",
			backURL:          "",
			wantRedirectPath: "/",
		},
		{
			name:             "危険なbackパラメータ（絶対URL）は無視される",
			backURL:          "https://evil.com",
			wantRedirectPath: "/",
		},
		{
			name:             "危険なbackパラメータ（プロトコル相対URL）は無視される",
			backURL:          "//evil.com",
			wantRedirectPath: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)

			password := "testpassword123"
			passwordDigest, err := auth.HashPassword(password)
			if err != nil {
				t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
			}

			_ = testutil.NewUserBuilder(t, tx).
				WithEmail("back-test@example.com").
				WithAtname("backtest").
				BuildWithPassword(passwordDigest)

			handler := setupHandler(t, tx, true)

			form := url.Values{}
			form.Set("email", "back-test@example.com")
			form.Set("password", password)
			form.Set("csrf_token", "test-csrf-token")
			form.Set("cf-turnstile-response", "test-token")
			form.Set("back", tt.backURL)

			req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.Create(rr, req)

			if rr.Code != http.StatusFound {
				t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
			}

			location := rr.Header().Get("Location")
			if location != tt.wantRedirectPath {
				t.Errorf("wrong redirect location: got %v want %v", location, tt.wantRedirectPath)
			}
		})
	}
}

func TestCreate_ValidationErrorPreservesBackParameter(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	handler := setupHandler(t, tx, true)

	backURL := "/dashboard"
	form := url.Values{}
	form.Set("email", "invalid-email")
	form.Set("password", "password123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")
	form.Set("back", backURL)

	req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	wantInBody := `name="back" value="/dashboard"`
	if !strings.Contains(body, wantInBody) {
		t.Errorf("backパラメータがフォームに保持されていません\nwant: %s", wantInBody)
	}
}

// TestCreate_TwoFactorRequiredCarriesBackParameter verifies that back is carried over even when
// two-factor authentication comes in between. Without it, only the users who enabled two-factor
// authentication would lose their original destination after signing in.
//
// [Ja] TestCreate_TwoFactorRequiredCarriesBackParameter は、二要素認証を挟むときも back を
// 引き継ぐことを検証する。引き継がないと、二要素認証を有効にしているユーザーだけが
// サインイン後に元の宛先を失う。
func TestCreate_TwoFactorRequiredCarriesBackParameter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		email        string
		atname       string
		backURL      string
		wantLocation string
	}{
		{
			name:         "有効なbackパラメータを引き継ぐ",
			email:        "2fa-back-valid@example.com",
			atname:       "twofabackvalid",
			backURL:      "/s/example/topics/1/pages/new?title=%E3%83%A1%E3%83%A2",
			wantLocation: "/sign_in/two_factor/new?back=" + url.QueryEscape("/s/example/topics/1/pages/new?title=%E3%83%A1%E3%83%A2"),
		},
		{
			name:         "backパラメータなし",
			email:        "2fa-back-none@example.com",
			atname:       "twofabacknone",
			backURL:      "",
			wantLocation: "/sign_in/two_factor/new",
		},
		{
			name:         "危険なbackパラメータは引き継がない",
			email:        "2fa-back-evil@example.com",
			atname:       "twofabackevil",
			backURL:      "//evil.com",
			wantLocation: "/sign_in/two_factor/new",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, tx := testutil.SetupTx(t)

			password := "testpassword123"
			passwordDigest, err := auth.HashPassword(password)
			if err != nil {
				t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
			}

			testutil.NewUserBuilder(t, tx).
				WithEmail(tt.email).
				WithAtname(tt.atname).
				BuildWithPasswordAndTwoFactorAuth(passwordDigest, "JBSWY3DPEHPK3PXP", true)

			handler := setupHandler(t, tx, true)

			form := url.Values{}
			form.Set("email", tt.email)
			form.Set("password", password)
			form.Set("csrf_token", "test-csrf-token")
			form.Set("cf-turnstile-response", "test-token")
			form.Set("back", tt.backURL)

			req := httptest.NewRequest(http.MethodPost, "/sign_in", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.Create(rr, req)

			if rr.Code != http.StatusFound {
				t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
			}
			if location := rr.Header().Get("Location"); location != tt.wantLocation {
				t.Errorf("wrong redirect location: got %v want %v", location, tt.wantLocation)
			}
		})
	}
}
