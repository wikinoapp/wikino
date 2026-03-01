package page_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/page"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// setupHandler はテスト用のハンドラーを生成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *page.Handler {
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

	return page.NewHandler(
		cfg,
		flashMgr,
		repository.NewSpaceRepository(queries),
		repository.NewSpaceMemberRepository(queries),
		repository.NewPageRepository(queries),
		repository.NewDraftPageRepository(queries),
		repository.NewTopicRepository(queries),
		repository.NewTopicMemberRepository(queries),
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

func TestEdit(t *testing.T) {
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
		WithRole(0). // owner
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
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
		WithTitle("Test Page Title").
		WithBody("Test page body content").
		Build()

	handler := setupHandler(t, queries)

	// リクエストを作成
	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/my-space/pages/1/edit", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	})
	req.Header.Set("Accept-Language", "ja")

	// コンテキストを設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// フォームアクションが含まれているか確認
	if !strings.Contains(body, `/go/s/my-space/pages/1`) {
		t.Error("form action not found in response")
	}

	// CSRFトークンが含まれているか確認
	if !strings.Contains(body, "test-csrf-token") {
		t.Error("CSRF token not found in response")
	}

	// タイトルが表示されているか確認
	if !strings.Contains(body, "Test Page Title") {
		t.Error("page title not found in response")
	}

	// 本文が表示されているか確認
	if !strings.Contains(body, "Test page body content") {
		t.Error("page body not found in response")
	}

	// _methodがPATCHであることを確認
	if !strings.Contains(body, `value="PATCH"`) {
		t.Error("method override PATCH not found in response")
	}

	// 日本語のラベルが含まれているか確認
	if !strings.Contains(body, "タイトル") {
		t.Error("Japanese title label not found in response")
	}

	// 公開ボタンが含まれているか確認
	if !strings.Contains(body, "トピックに公開") {
		t.Error("Japanese publish button not found in response")
	}

	// キャンセルリンクが含まれているか確認
	if !strings.Contains(body, "/s/my-space/pages/1") {
		t.Error("cancel link not found in response")
	}

	// パンくずリストにトピック名が含まれているか確認
	if !strings.Contains(body, "General") {
		t.Error("topic name not found in breadcrumb")
	}

	// パンくずリストにスペースへのリンクが含まれているか確認
	if !strings.Contains(body, "/s/my-space") {
		t.Error("space link not found in breadcrumb")
	}

	// 下書きがない場合、下書きアラートが表示されないことを確認
	if strings.Contains(body, "現在下書きを表示しています") {
		t.Error("draft alert should not be shown when no draft exists")
	}

	// パンくずリストにトピックへのリンクが含まれているか確認
	if !strings.Contains(body, "/s/my-space/topics/1") {
		t.Error("topic link not found in breadcrumb")
	}

	// トピックのアイコン（公開トピックのためglobe-regular）が表示されているか確認
	// globe-regularのSVGパスデータに含まれる固有の文字列で検証
	if !strings.Contains(body, "a87.61,87.61") {
		t.Error("topic visibility icon (globe) not found in breadcrumb")
	}
}

func TestEdit_WithDraftPage(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draft@example.com").
		WithAtname("draftuser").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithTitle("Original Title").
		WithBody("Original body").
		Build()

	// DraftPageを作成
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Title").
		WithBody("Draft body content").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/draft-space/pages/1/edit", map[string]string{
		"space_identifier": "draft-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// DraftPageの内容が表示されていることを確認
	if !strings.Contains(body, "Draft Title") {
		t.Error("draft title not found in response")
	}
	if !strings.Contains(body, "Draft body content") {
		t.Error("draft body not found in response")
	}

	// 元のページの内容が表示されていないことを確認
	if strings.Contains(body, "Original Title") {
		t.Error("original title should not be shown when draft exists")
	}
	if strings.Contains(body, "Original body") {
		t.Error("original body should not be shown when draft exists")
	}

	// 下書きアラートが表示されていることを確認
	if !strings.Contains(body, "現在下書きを表示しています") {
		t.Error("draft alert message not found in response")
	}
}

func TestEdit_AutofocusTitle(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// タイトルなしのページを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("autofocus@example.com").
		WithAtname("autofocususer").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("autofocus-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNilTitle().
		WithBody("Body without title").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/autofocus-space/pages/1/edit", map[string]string{
		"space_identifier": "autofocus-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// タイトルが空のとき、タイトル入力欄にautofocusが設定されていることを確認
	if !strings.Contains(body, `id="page_title"`) {
		t.Error("page title input not found")
	}
	if !strings.Contains(body, "autofocus") {
		t.Error("autofocus attribute not found in page_title input")
	}
}

func TestEdit_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/my-space/pages/1/edit", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	// ログインページへリダイレクトされることを確認
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
	location := rr.Header().Get("Location")
	if location != "/sign_in" {
		t.Errorf("wrong redirect location: got %v want /sign_in", location)
	}
}

func TestEdit_SpaceNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("space-not-found@example.com").
		WithAtname("spacenotfound").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/nonexistent/pages/1/edit", map[string]string{
		"space_identifier": "nonexistent",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestEdit_NotSpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// スペースを作成するが、別のユーザーでアクセス
	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("owner@example.com").
		WithAtname("owner").
		Build()
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("outsider@example.com").
		WithAtname("outsider").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("private-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/private-space/pages/1/edit", map[string]string{
		"space_identifier": "private-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: outsiderID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestEdit_PageNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("page-not-found@example.com").
		WithAtname("pagenotfound").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-missing-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/page-missing-space/pages/999/edit", map[string]string{
		"space_identifier": "page-missing-space",
		"page_number":      "999",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestEdit_InvalidPageNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("invalid-num@example.com").
		WithAtname("invalidnum").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/my-space/pages/abc/edit", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "abc",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestEdit_LinkListAutoReload(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("linklist-reload@example.com").
		WithAtname("linklistreload").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("linklist-reload-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
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
		WithTitle("Link Reload Test").
		WithBody("Some content").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/linklist-reload-space/pages/1/edit", map[string]string{
		"space_identifier": "linklist-reload-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// リンク一覧セクションのコンテナが存在すること
	if !strings.Contains(body, `id="page-link-list"`) {
		t.Error("page-link-list container not found in response")
	}

	// Datastarのdata-on:draft-autosaved__window属性が含まれていること
	// この属性により、下書き保存後にリンク一覧がSSEで自動再読み込みされる
	if !strings.Contains(body, "data-on:draft-autosaved__window") {
		t.Error("data-on:draft-autosaved__window attribute not found - link list auto-reload will not work")
	}

	// SSEエンドポイントのURLが正しいこと
	if !strings.Contains(body, "/go/s/linklist-reload-space/pages/1/draft_page") {
		t.Error("draft_page SSE endpoint URL not found in response")
	}

	// @get()アクションでSSEエンドポイントが呼び出されること
	if !strings.Contains(body, "@get(") {
		t.Error("@get() action not found - SSE request will not be triggered")
	}
}

func TestEdit_EnglishLocale(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("english@example.com").
		WithAtname("englishuser").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("en-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithTitle("English Test Page").
		WithBody("English body").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/en-space/pages/1/edit", map[string]string{
		"space_identifier": "en-space",
		"page_number":      "1",
	})
	req.Header.Set("Accept-Language", "en")
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangEn)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// 英語のラベルが含まれているか確認
	if !strings.Contains(body, "Title") {
		t.Error("English title label not found in response")
	}
	if !strings.Contains(body, "Publish") {
		t.Error("English publish button not found in response")
	}
	if !strings.Contains(body, "Cancel") {
		t.Error("English cancel link not found in response")
	}
}
