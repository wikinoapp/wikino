package draft_page_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// newShowRequest はchiのURLパラメータ付きGETリクエストを作成するヘルパーです
func newShowRequest(t *testing.T, path string, params map[string]string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestShow_未ログイン時に401が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/go/s/test-space/pages/1/draft_page", map[string]string{
		"space_identifier": "test-space",
		"page_number":      "1",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnauthorized)
	}
}

func TestShow_存在しないスペースで404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("show-notfound@example.com").
		WithAtname("shownotfound").
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/go/s/nonexistent/pages/1/draft_page", map[string]string{
		"space_identifier": "nonexistent",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_スペースメンバーでない場合に404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("show-owner@example.com").
		WithAtname("showowner").
		Build()
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("show-outsider@example.com").
		WithAtname("showoutsider").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("show-private").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/go/s/show-private/pages/1/draft_page", map[string]string{
		"space_identifier": "show-private",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: outsiderID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_不正なページ番号で404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("show-invalid@example.com").
		WithAtname("showinvalid").
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/go/s/test-space/pages/abc/draft_page", map[string]string{
		"space_identifier": "test-space",
		"page_number":      "abc",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_正常系_リンクなしでSSEレスポンスが返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("show-ok@example.com").
		WithAtname("showok").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("show-ok").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
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
		WithRole(0).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/go/s/show-ok/pages/1/draft_page", map[string]string{
		"space_identifier": "show-ok",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		t.Errorf("wrong content type: got %v, want text/event-stream", contentType)
	}
}

func TestShow_正常系_リンクありでSSEレスポンスにリンク先が含まれる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("show-links@example.com").
		WithAtname("showlinks").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("show-links").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
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
		WithRole(0).
		Build()

	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Linked Page").
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/go/s/show-links/pages/1/draft_page", map[string]string{
		"space_identifier": "show-links",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Linked Page") {
		t.Error("response should contain linked page title 'Linked Page'")
	}
}

func TestShow_正常系_下書きのリンクが優先される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("show-draft@example.com").
		WithAtname("showdraft").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("show-draft").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
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
		WithRole(0).
		Build()

	originalLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Original Link").
		Build()

	draftLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Draft Link").
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source Page").
		WithLinkedPageIDs([]model.PageID{originalLinkedPageID}).
		Build()

	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Source Page Draft").
		WithLinkedPageIDs([]model.PageID{draftLinkedPageID}).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/go/s/show-draft/pages/1/draft_page", map[string]string{
		"space_identifier": "show-draft",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "Draft Link") {
		t.Error("response should contain draft linked page title 'Draft Link'")
	}

	if strings.Contains(body, "Original Link") {
		t.Error("response should not contain original linked page title 'Original Link'")
	}
}

func TestShow_正常系_ページネーションパラメータが反映される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("show-page@example.com").
		WithAtname("showpage").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("show-page").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
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
		WithRole(0).
		Build()

	linkedPage1ID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Link Page 1").
		Build()

	linkedPage2ID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Link Page 2").
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source Page").
		WithLinkedPageIDs([]model.PageID{linkedPage1ID, linkedPage2ID}).
		Build()

	handler := setupHandler(t, queries)

	// page=1で全リンクが含まれる
	req := newShowRequest(t, "/go/s/show-page/pages/1/draft_page?page=1", map[string]string{
		"space_identifier": "show-page",
		"page_number":      "1",
	})
	req.URL.RawQuery = "page=1"
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Link Page 1") {
		t.Error("page=1 should contain 'Link Page 1'")
	}
	if !strings.Contains(body, "Link Page 2") {
		t.Error("page=1 should contain 'Link Page 2'")
	}

	// page=999（存在しないページ）ではリンク先が含まれない
	req2 := newShowRequest(t, "/go/s/show-page/pages/1/draft_page?page=999", map[string]string{
		"space_identifier": "show-page",
		"page_number":      "1",
	})
	req2.URL.RawQuery = "page=999"
	ctx2 := middleware.SetUserToContext(req2.Context(), &model.User{ID: userID})
	req2 = req2.WithContext(ctx2)

	rr2 := httptest.NewRecorder()
	handler.Show(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr2.Code, http.StatusOK)
	}

	body2 := rr2.Body.String()
	if strings.Contains(body2, "Link Page 1") {
		t.Error("page=999 should not contain 'Link Page 1'")
	}
	if strings.Contains(body2, "Link Page 2") {
		t.Error("page=999 should not contain 'Link Page 2'")
	}
}

func TestShow_正常系_下書きにリンクを追加するとSSEレスポンスに反映される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("show-add@example.com").
		WithAtname("showadd").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("show-add").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
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
		WithRole(0).
		Build()

	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Linked Target").
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	handler := setupHandler(t, queries)

	// 1. リンクなしの状態でSSEエンドポイントを呼び出す
	req1 := newShowRequest(t, "/go/s/show-add/pages/1/draft_page", map[string]string{
		"space_identifier": "show-add",
		"page_number":      "1",
	})
	ctx1 := middleware.SetUserToContext(req1.Context(), &model.User{ID: userID})
	req1 = req1.WithContext(ctx1)
	rr1 := httptest.NewRecorder()
	handler.Show(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("first request: wrong status code: got %v want %v", rr1.Code, http.StatusOK)
	}

	body1 := rr1.Body.String()
	if strings.Contains(body1, "Linked Target") {
		t.Error("first request: should not contain 'Linked Target' before draft is saved")
	}

	// 2. 下書きが保存された状態をシミュレート
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Source Page").
		WithBody("Some text with [[Linked Target]] link").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	// 3. 下書き保存後にSSEエンドポイントを呼び出す
	req2 := newShowRequest(t, "/go/s/show-add/pages/1/draft_page", map[string]string{
		"space_identifier": "show-add",
		"page_number":      "1",
	})
	ctx2 := middleware.SetUserToContext(req2.Context(), &model.User{ID: userID})
	req2 = req2.WithContext(ctx2)
	rr2 := httptest.NewRecorder()
	handler.Show(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("second request: wrong status code: got %v want %v", rr2.Code, http.StatusOK)
	}

	contentType := rr2.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		t.Errorf("second request: wrong content type: got %v, want text/event-stream", contentType)
	}

	body2 := rr2.Body.String()
	if !strings.Contains(body2, "Linked Target") {
		t.Error("second request: should contain 'Linked Target' after draft with wikilink is saved")
	}
}

func TestShow_正常系_下書きが存在する場合に保存時刻フラグメントが含まれる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("show-time@example.com").
		WithAtname("showtime").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("show-time").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
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
		WithRole(0).
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	// 下書きを作成
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Title").
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/go/s/show-time/pages/1/draft_page", map[string]string{
		"space_identifier": "show-time",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// 保存時刻フラグメントが含まれること（page-draft-saved-atセレクタ）
	if !strings.Contains(body, "page-draft-saved-at") {
		t.Error("response should contain saved time fragment with 'page-draft-saved-at' selector")
	}
}

func TestShow_正常系_下書きが存在しない場合に保存時刻フラグメントが含まれない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("show-notime@example.com").
		WithAtname("shownotime").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("show-notime").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
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
		WithRole(0).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/go/s/show-notime/pages/1/draft_page", map[string]string{
		"space_identifier": "show-notime",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// リンク一覧フラグメントは含まれるが、保存時刻のdiv要素は含まれないこと
	// SSEレスポンスのpage-link-listセレクタは含まれる
	if !strings.Contains(body, "page-link-list") {
		t.Error("response should contain link list fragment with 'page-link-list' selector")
	}

	// 保存時刻のdiv要素（テンプレートがレンダリングするid属性付きdiv）は含まれないこと
	if strings.Contains(body, `id="page-draft-saved-at"`) {
		t.Error("response should not contain saved time div when no draft exists")
	}
}
