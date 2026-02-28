package page_link_list_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/page_link_list"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// setupHandler はテスト用のハンドラーを生成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *page_link_list.Handler {
	t.Helper()

	return page_link_list.NewHandler(
		repository.NewSpaceRepository(queries),
		repository.NewSpaceMemberRepository(queries),
		repository.NewPageRepository(queries),
		repository.NewDraftPageRepository(queries),
		repository.NewTopicMemberRepository(queries),
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

func TestShow_未ログイン時に401が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/test-space/pages/1/link_list", map[string]string{
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
		WithEmail("linklist-notfound@example.com").
		WithAtname("linklistnotfound").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/nonexistent/pages/1/link_list", map[string]string{
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
		WithEmail("linklist-owner@example.com").
		WithAtname("linklistowner").
		Build()
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("linklist-outsider@example.com").
		WithAtname("linklistoutsider").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("linklist-private").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/linklist-private/pages/1/link_list", map[string]string{
		"space_identifier": "linklist-private",
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
		WithEmail("linklist-invalid@example.com").
		WithAtname("linklistinvalid").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/test-space/pages/abc/link_list", map[string]string{
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
		WithEmail("linklist-ok@example.com").
		WithAtname("linklistok").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("linklist-ok").
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

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/linklist-ok/pages/1/link_list", map[string]string{
		"space_identifier": "linklist-ok",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// SSEレスポンスのContent-Typeを確認
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
		WithEmail("linklist-links@example.com").
		WithAtname("linklistlinks").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("linklist-links").
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

	// リンク先ページを作成
	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Linked Page").
		Build()

	// リンク元ページを作成（リンク先を含む）
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/linklist-links/pages/1/link_list", map[string]string{
		"space_identifier": "linklist-links",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// SSEレスポンスにリンク先ページのタイトルが含まれることを確認
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
		WithEmail("linklist-draft@example.com").
		WithAtname("linklistdraft").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("linklist-draft").
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

	// リンク先ページ（公開版のリンク先）
	originalLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Original Link").
		Build()

	// リンク先ページ（下書き版のリンク先）
	draftLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Draft Link").
		Build()

	// リンク元ページ（公開版にはoriginalLinkedPageIDがリンクされている）
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source Page").
		WithLinkedPageIDs([]model.PageID{originalLinkedPageID}).
		Build()

	// 下書きを作成（draftLinkedPageIDがリンクされている）
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Source Page Draft").
		WithLinkedPageIDs([]model.PageID{draftLinkedPageID}).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/linklist-draft/pages/1/link_list", map[string]string{
		"space_identifier": "linklist-draft",
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

	// 下書きのリンク先が含まれること
	if !strings.Contains(body, "Draft Link") {
		t.Error("response should contain draft linked page title 'Draft Link'")
	}

	// 公開版のリンク先が含まれないこと
	if strings.Contains(body, "Original Link") {
		t.Error("response should not contain original linked page title 'Original Link'")
	}
}

func TestShow_正常系_ページネーションパラメータが反映される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("linklist-page@example.com").
		WithAtname("linklistpage").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("linklist-page").
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

	// リンク先ページを2件作成
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

	// リンク元ページを作成（2件のリンク先）
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source Page").
		WithLinkedPageIDs([]model.PageID{linkedPage1ID, linkedPage2ID}).
		Build()

	handler := setupHandler(t, queries)

	// page=1で全リンクが含まれる
	req := newRequestWithChiParams(t, http.MethodGet, "/go/s/linklist-page/pages/1/link_list?page=1", map[string]string{
		"space_identifier": "linklist-page",
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
	req2 := newRequestWithChiParams(t, http.MethodGet, "/go/s/linklist-page/pages/1/link_list?page=999", map[string]string{
		"space_identifier": "linklist-page",
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
		WithEmail("linklist-add@example.com").
		WithAtname("linklistadd").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("linklist-add").
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

	// リンク先ページを作成（ユーザーが [[Linked Target]] と入力する対象）
	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Linked Target").
		Build()

	// リンク元ページ（リンクなしの状態）
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	handler := setupHandler(t, queries)

	// 1. リンクなしの状態でSSEエンドポイントを呼び出す
	req1 := newRequestWithChiParams(t, http.MethodGet, "/go/s/linklist-add/pages/1/link_list", map[string]string{
		"space_identifier": "linklist-add",
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

	// 2. ユーザーが [[Linked Target]] と入力し、下書きが保存された状態をシミュレート
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Source Page").
		WithBody("Some text with [[Linked Target]] link").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	// 3. 下書き保存後にSSEエンドポイントを呼び出す（draft-autosavedイベント後の@get()に相当）
	req2 := newRequestWithChiParams(t, http.MethodGet, "/go/s/linklist-add/pages/1/link_list", map[string]string{
		"space_identifier": "linklist-add",
		"page_number":      "1",
	})
	ctx2 := middleware.SetUserToContext(req2.Context(), &model.User{ID: userID})
	req2 = req2.WithContext(ctx2)
	rr2 := httptest.NewRecorder()
	handler.Show(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("second request: wrong status code: got %v want %v", rr2.Code, http.StatusOK)
	}

	// SSEレスポンスのContent-Typeを確認
	contentType := rr2.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		t.Errorf("second request: wrong content type: got %v, want text/event-stream", contentType)
	}

	// 下書き保存後にリンク先が含まれること
	body2 := rr2.Body.String()
	if !strings.Contains(body2, "Linked Target") {
		t.Error("second request: should contain 'Linked Target' after draft with wikilink is saved")
	}
}
