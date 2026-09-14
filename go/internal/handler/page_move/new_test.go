package page_move_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/page_move"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// setupHandler はテスト用のハンドラーを生成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *page_move.Handler {
	t.Helper()

	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)

	getPageMoveDataUC := usecase.NewGetPageMoveDataUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo)

	return page_move.NewHandler(
		cfg,
		flashMgr,
		getPageMoveDataUC,
		nil,
	)
}

// newRequestWithChiParams はchiのURLパラメータ付きリクエストを作成するヘルパーです
func newRequestWithChiParams(t *testing.T, method, path string, params map[string]string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// addChiParams は既存のリクエストにchiのURLパラメータを追加するヘルパーです
func addChiParams(t *testing.T, req *http.Request, params map[string]string) *http.Request {
	t.Helper()

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestNew(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("my-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID1 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Topic 1").
		Build()
	topicID2 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Topic 2").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID1).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID2).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID1).
		WithNumber(1).
		WithTitle("Test Page Title").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/my-space/pages/1/move", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "testuser"})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// ページタイトルが表示されていること
	if !strings.Contains(body, "Test Page Title") {
		t.Error("page title not found in response")
	}

	// 現在のトピック名が表示されていること
	if !strings.Contains(body, "Topic 1") {
		t.Error("current topic name not found in response")
	}

	// 移動先トピック（Topic 2）がセレクトボックスに含まれていること
	if !strings.Contains(body, "Topic 2") {
		t.Error("destination topic not found in response")
	}

	// CSRFトークンが含まれていること
	if !strings.Contains(body, "test-csrf-token") {
		t.Error("CSRF token not found in response")
	}

	// フォームアクションが含まれていること
	if !strings.Contains(body, "/s/my-space/pages/1/move") {
		t.Error("form action not found in response")
	}

	for _, want := range []string{`<span class="label">ページ</span>`, `<span class="label">現在のトピック</span>`} {
		if !strings.Contains(body, want) {
			t.Errorf("static page detail should be headed by a span, want %s", want)
		}
	}
	if !strings.Contains(body, `<label class="label" for="dest_topic">`) {
		t.Error("destination topic label should point at the select")
	}

	// The breadcrumb header comes from the layout, so it renders outside <main> (the #main skip
	// link has to bypass it) and keeps this screen's narrower max-w-2xl content width.
	//
	// [Ja] パンくずヘッダーはレイアウトが描画するため、<main> の外に出る (#main へのスキップ
	// リンクが飛ばせる必要があるため)。この画面の狭い本文幅 max-w-2xl も維持する。
	if !strings.Contains(body, `<div class="max-w-2xl mx-auto flex w-full items-center justify-between gap-2 px-4">`) {
		t.Error("shared breadcrumb header should keep the max-w-2xl content width")
	}
	header, main := strings.Index(body, "<header"), strings.Index(body, `<main id="main" tabindex="-1">`)
	if header == -1 || main == -1 || header > main {
		t.Errorf("shared breadcrumb header (index %d) must precede <main> (index %d)", header, main)
	}
}

func TestNew_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/my-space/pages/1/move", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("wrong redirect location: got %v want /sign_in", location)
	}
}

func TestNew_SpaceNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/nonexistent/pages/1/move", map[string]string{
		"space_identifier": "nonexistent",
		"page_number":      "1",
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestNew_NoAvailableTopics(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストデータを作成（トピックが1つだけ）
	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("my-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Only Topic").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/my-space/pages/1/move", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "testuser"})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// 送信ボタンが無効化されていること
	if !strings.Contains(body, "disabled") {
		t.Error("submit button should be disabled when no topics available")
	}
	if !strings.Contains(body, `<span class="label">移動先トピック</span>`) {
		t.Error("destination topic heading should be a span when there is no select to label")
	}
	if strings.Contains(body, `<label class="label"`) {
		t.Error("a label should not be rendered when there is no destination topic control")
	}
}

// The trail ends with the screen itself, so the last item must be a plain label carrying
// aria-current rather than a link back to the topic. Scope the assertions to the breadcrumb because
// the same label also appears in the heading and the page title.
//
// [Ja] 経路はこの画面自身で終わるため、末尾の項目はトピックへのリンクではなく aria-current を持つ
// ラベルになる。同じラベルは見出しとページタイトルにも出るため、パンくず内に絞って検証する。
func TestNew_パンくずが現在地の項目で終わる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("move-crumb").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Topic 1").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Move Crumb Page").
		Build()

	req := newRequestWithChiParams(t, http.MethodGet, "/s/move-crumb/pages/1/move", map[string]string{
		"space_identifier": "move-crumb",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "testuser"})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	setupHandler(t, queries).New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	breadcrumb := pageMoveBreadcrumb(t, rr.Body.String())
	for _, want := range []string{
		`href="/s/move-crumb"`,
		`href="/s/move-crumb/topics/1"`,
		`aria-current="page"`,
		"ページの移動",
	} {
		if !strings.Contains(breadcrumb, want) {
			t.Errorf("breadcrumb does not contain %q", want)
		}
	}
	if strings.Contains(breadcrumb, `href="/s/move-crumb/pages/1/move"`) {
		t.Error("current page move breadcrumb item must not be a link")
	}
}

// pageMoveBreadcrumb returns the markup of the breadcrumb navigation alone, so that an assertion
// about the trail is not satisfied by the same text appearing elsewhere on the screen.
//
// [Ja] pageMoveBreadcrumb はパンくずのナビゲーション部分だけのマークアップを返す。経路についての
// 検証が、画面の他の場所に出た同じ文字列で満たされてしまうのを防ぐ。
func pageMoveBreadcrumb(t *testing.T, body string) string {
	t.Helper()

	start := strings.Index(body, `<nav aria-label="パンくずリスト"`)
	if start == -1 {
		t.Fatal("response does not contain the breadcrumb navigation")
	}
	endOffset := strings.Index(body[start:], "</nav>")
	if endOffset == -1 {
		t.Fatal("breadcrumb navigation does not have a closing tag")
	}

	return body[start : start+endOffset]
}
