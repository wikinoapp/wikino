package email_confirmation_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/handler/email_confirmation"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// mockTurnstileVerifierForEdit はテスト用のTurnstile検証モック
type mockTurnstileVerifierForEdit struct {
	valid bool
	err   error
}

func (m *mockTurnstileVerifierForEdit) Verify(_ context.Context, _ string) (bool, error) {
	return m.valid, m.err
}

// mockJobInserterForEdit はテスト用のモック inserter
type mockJobInserterForEdit struct{}

func (m *mockJobInserterForEdit) Insert(_ context.Context, _ river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return &rivertype.JobInsertResult{}, nil
}

func newTestHandlerForEdit(t *testing.T, queries *query.Queries) *email_confirmation.Handler {
	t.Helper()

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
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	mockInserter := &mockJobInserterForEdit{}
	d := dispatcher.NewDispatcher(mockInserter)
	emailConfirmationCreateValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	emailConfirmationUpdateValidator := validator.NewEmailConfirmationUpdateValidator(emailConfirmationRepo)
	createEmailConfirmationUC := usecase.NewCreateEmailConfirmationUsecase(cfg, emailConfirmationRepo, d, emailConfirmationCreateValidator)
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo, emailConfirmationUpdateValidator)

	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	turnstileVerifier := &mockTurnstileVerifierForEdit{valid: true}

	rateLimitRepo := repository.NewRateLimitRepository(queries)
	limiter := ratelimit.NewLimiter(rateLimitRepo)

	return email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		createEmailConfirmationUC,
		markEmailAsConfirmedUC,
		turnstileVerifier,
		limiter,
	)
}

func TestEdit(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := newTestHandlerForEdit(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/email_confirmation/edit", nil)
	req.Header.Set("Accept-Language", "ja")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: "test-email-confirmation-id",
	})

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, `action="/email_confirmation"`) {
		t.Error("confirmation form action not found in response")
	}

	if !strings.Contains(body, `name="_method" value="PATCH"`) {
		t.Error("_method hidden field not found in response")
	}

	if !strings.Contains(body, "test-csrf-token") {
		t.Error("CSRF token not found in response")
	}

	if !strings.Contains(body, `name="code"`) {
		t.Error("code input field not found in response")
	}

	if !strings.Contains(body, "確認コードを入力") {
		t.Error("Japanese heading not found in response")
	}

	if !strings.Contains(body, `href="/sign_up"`) {
		t.Error("back to sign up link not found in response")
	}

	for _, notWant := range []string{`<link rel="canonical"`, `property="og:url"`} {
		if strings.Contains(body, notWant) {
			t.Errorf("response unexpectedly contains %q", notWant)
		}
	}
}

func TestEdit_NoEmailConfirmationID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := newTestHandlerForEdit(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/email_confirmation/edit", nil)
	req.Header.Set("Accept-Language", "ja")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/sign_up" {
		t.Errorf("wrong redirect location: got %v want %v", location, "/sign_up")
	}
}

func TestEdit_EnglishLocale(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := newTestHandlerForEdit(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/email_confirmation/edit", nil)
	req.Header.Set("Accept-Language", "en")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, i18n.LangEn)
	req = req.WithContext(ctx)

	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: "test-email-confirmation-id",
	})

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Enter confirmation code") {
		t.Error("English heading not found in response")
	}

	if !strings.Contains(body, "Verify") {
		t.Error("English submit button text not found in response")
	}

	if !strings.Contains(body, "Back to sign up") {
		t.Error("English back link not found in response")
	}
}
