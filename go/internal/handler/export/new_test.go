package export_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	exporthandler "github.com/wikinoapp/wikino/go/internal/handler/export"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// setupHandler builds an export handler over the given transaction. The start of an export is left
// out: it manages its own transaction, and none of the screens tested here reach it.
//
// [Ja] setupHandler は与えられたトランザクションの上にエクスポートのハンドラーを組み立てる。
// エクスポートの開始は含めない。開始は自身でトランザクションを管理し、ここでテストするどの画面も
// そこへ到達しないためである。
func setupHandler(t *testing.T, queries *query.Queries) *exporthandler.Handler {
	t.Helper()

	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	exportRepo := repository.NewExportRepository(queries)

	return exporthandler.NewHandler(
		cfg,
		session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly),
		usecase.NewGetExportNewUsecase(spaceRepo, spaceMemberRepo, exportRepo),
		usecase.NewGetExportShowUsecase(spaceRepo, spaceMemberRepo, exportRepo),
		nil,
	)
}

// newRequest builds a request carrying the chi URL parameters, the signed-in user and the locale
// the screens are rendered in.
//
// [Ja] newRequest は chi の URL パラメータ・ログイン中のユーザー・画面を描画するロケールを載せた
// リクエストを組み立てる。
func newRequest(t *testing.T, method, path string, params map[string]string, userID model.UserID) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)

	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "export-user"})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// exportSpace seeds a space with one member and returns what the tests address them by.
//
// [Ja] exportSpace はメンバーが 1 人いるスペースを用意し、テストがそれらを指すための値を返す。
func exportSpace(t *testing.T, tx *sql.Tx, identifier string, scopes []model.Scope) (model.UserID, model.SpaceID, model.SpaceMemberID) {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(identifier).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(identifier).
		Build()
	memberBuilder := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID)
	if scopes != nil {
		memberBuilder = memberBuilder.WithScopes(scopes)
	}

	return userID, spaceID, memberBuilder.Build()
}

func TestNew_ShowsStartButton(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, _, _ := exportSpace(t, tx, "exp-new-ok", nil)

	req := newRequest(t, http.MethodGet, "/s/exp-new-ok/settings/exports/new", map[string]string{
		"space_identifier": "exp-new-ok",
	}, userID)

	rr := httptest.NewRecorder()
	setupHandler(t, queries).New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "エクスポートを開始する") {
		t.Error("開始ボタンが表示されていない")
	}
	if !strings.Contains(body, `action="/s/exp-new-ok/settings/exports"`) {
		t.Error("開始フォームの送信先が表示されていない")
	}
}

func TestNew_HidesStartButtonWhileExporting(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, spaceID, spaceMemberID := exportSpace(t, tx, "exp-new-running", nil)
	exportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(spaceID).
		WithQueuedByID(spaceMemberID).
		WithStatus(model.ExportStatusStarted).
		WithHeartbeatAt(time.Now()).
		Build()

	req := newRequest(t, http.MethodGet, "/s/exp-new-running/settings/exports/new", map[string]string{
		"space_identifier": "exp-new-running",
	}, userID)

	rr := httptest.NewRecorder()
	setupHandler(t, queries).New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if strings.Contains(body, "エクスポートを開始する") {
		t.Error("実行中なのに開始ボタンが表示されている")
	}
	if !strings.Contains(body, "/s/exp-new-running/settings/exports/"+exportID.String()) {
		t.Error("実行中のエクスポートへの導線が表示されていない")
	}
}

func TestNew_NotFoundWithoutExportPermission(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, _, _ := exportSpace(t, tx, "exp-new-forbidden", []model.Scope{model.ScopeSpaceRead})

	req := newRequest(t, http.MethodGet, "/s/exp-new-forbidden/settings/exports/new", map[string]string{
		"space_identifier": "exp-new-forbidden",
	}, userID)

	rr := httptest.NewRecorder()
	setupHandler(t, queries).New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestNew_NotFoundForNonMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	exportSpace(t, tx, "exp-new-outsider-space", nil)
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("exp-new-outsider@example.com").
		WithAtname("exp-new-outsider").
		Build()

	req := newRequest(t, http.MethodGet, "/s/exp-new-outsider-space/settings/exports/new", map[string]string{
		"space_identifier": "exp-new-outsider-space",
	}, outsiderID)

	rr := httptest.NewRecorder()
	setupHandler(t, queries).New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusNotFound)
	}
}
