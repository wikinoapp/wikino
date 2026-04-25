package email_confirmation_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/handler/email_confirmation"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// mockJobInserterForUpdate はテスト用のモック inserter
type mockJobInserterForUpdate struct{}

func (m *mockJobInserterForUpdate) Insert(_ context.Context, _ river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return &rivertype.JobInsertResult{}, nil
}

func newTestHandlerForUpdate(t *testing.T, queries *query.Queries) *email_confirmation.Handler {
	t.Helper()

	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	mockInserter := &mockJobInserterForUpdate{}
	d := dispatcher.NewDispatcher(mockInserter)
	emailConfirmationCreateValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	emailConfirmationUpdateValidator := validator.NewEmailConfirmationUpdateValidator(emailConfirmationRepo)
	createEmailConfirmationUC := usecase.NewCreateEmailConfirmationUsecase(cfg, emailConfirmationRepo, d, emailConfirmationCreateValidator)
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo, emailConfirmationUpdateValidator)

	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	rateLimitRepo := repository.NewRateLimitRepository(queries)
	limiter := ratelimit.NewLimiter(rateLimitRepo)

	return email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		createEmailConfirmationUC,
		markEmailAsConfirmedUC,
		mockTurnstile,
		limiter,
	)
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("ABC123").
		WithEvent(model.EmailConfirmationEventSignUp).
		Build()

	handler := newTestHandlerForUpdate(t, queries)

	form := url.Values{}
	form.Set("code", "ABC123")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("_method", "PATCH")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmationID,
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/accounts/new" {
		t.Errorf("wrong redirect location: got %v want /accounts/new", location)
	}

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)
	confirmation, err := emailConfirmationRepo.FindByID(ctx, emailConfirmationID)
	if err != nil {
		t.Fatalf("failed to find email confirmation: %v", err)
	}
	if !confirmation.IsSucceeded() {
		t.Error("email confirmation should be succeeded")
	}
}

func TestUpdate_NoSession(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := newTestHandlerForUpdate(t, queries)

	form := url.Values{}
	form.Set("code", "ABC123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/sign_up" {
		t.Errorf("wrong redirect location: got %v want /sign_up", location)
	}
}

func TestUpdate_EmptyCode(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("ABC123").
		WithEvent(model.EmailConfirmationEventSignUp).
		Build()

	handler := newTestHandlerForUpdate(t, queries)

	form := url.Values{}
	form.Set("code", "")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmationID,
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "入力してください") {
		t.Error("validation error message not found in response")
	}
}

func TestUpdate_InvalidCodeLength(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("ABC123").
		WithEvent(model.EmailConfirmationEventSignUp).
		Build()

	handler := newTestHandlerForUpdate(t, queries)

	form := url.Values{}
	form.Set("code", "ABC")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmationID,
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "6文字") {
		t.Error("invalid code length error message not found in response")
	}
}

func TestUpdate_CodeMismatch(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("ABC123").
		WithEvent(model.EmailConfirmationEventSignUp).
		Build()

	handler := newTestHandlerForUpdate(t, queries)

	form := url.Values{}
	form.Set("code", "WRONG1")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmationID,
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "正しくありません") {
		t.Error("code mismatch error message not found in response")
	}
}

func TestUpdate_Expired(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("ABC123").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithStartedAt(time.Now().Add(-16 * time.Minute)).
		Build()

	handler := newTestHandlerForUpdate(t, queries)

	form := url.Values{}
	form.Set("code", "ABC123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmationID,
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "有効期限") {
		t.Error("expired error message not found in response")
	}
}

func TestUpdate_CaseInsensitiveCode(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("ABC123").
		WithEvent(model.EmailConfirmationEventSignUp).
		Build()

	handler := newTestHandlerForUpdate(t, queries)

	form := url.Values{}
	form.Set("code", "abc123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmationID,
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/accounts/new" {
		t.Errorf("wrong redirect location: got %v want /accounts/new", location)
	}
}

func TestUpdate_AlreadySucceeded(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("test@example.com").
		WithCode("ABC123").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithSucceededAt(time.Now()).
		Build()

	handler := newTestHandlerForUpdate(t, queries)

	form := url.Values{}
	form.Set("code", "ABC123")
	form.Set("csrf_token", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmationID,
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/accounts/new" {
		t.Errorf("wrong redirect location: got %v want /accounts/new", location)
	}
}
