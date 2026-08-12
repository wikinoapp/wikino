package page_backlink_list_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/page_backlink_list"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// setupHandler はテスト用のハンドラーを生成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *page_backlink_list.Handler {
	t.Helper()

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	getBacklinkListUC := usecase.NewGetBacklinkListUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo)

	return page_backlink_list.NewHandler(
		getBacklinkListUC,
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

	req := newRequestWithChiParams(t, http.MethodGet, "/s/test-space/pages/1/links/2/backlink_list", map[string]string{
		"space_identifier":   "test-space",
		"page_number":        "1",
		"linked_page_number": "2",
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
		WithEmail("backlink-notfound@example.com").
		WithAtname("backlinknotfound").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/nonexistent/pages/1/links/2/backlink_list", map[string]string{
		"space_identifier":   "nonexistent",
		"page_number":        "1",
		"linked_page_number": "2",
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
		WithEmail("backlink-owner@example.com").
		WithAtname("backlinkowner").
		Build()
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("backlink-outsider@example.com").
		WithAtname("backlinkoutsider").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("backlink-private").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/backlink-private/pages/1/links/2/backlink_list", map[string]string{
		"space_identifier":   "backlink-private",
		"page_number":        "1",
		"linked_page_number": "2",
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
		WithEmail("backlink-invalid@example.com").
		WithAtname("backlinkinvalid").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/test-space/pages/abc/links/2/backlink_list", map[string]string{
		"space_identifier":   "test-space",
		"page_number":        "abc",
		"linked_page_number": "2",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_不正なリンク先ページ番号で404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("backlink-invalidlink@example.com").
		WithAtname("backlinkinvalidlink").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/test-space/pages/1/links/abc/backlink_list", map[string]string{
		"space_identifier":   "test-space",
		"page_number":        "1",
		"linked_page_number": "abc",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_正常系_バックリンクなしでHTMLレスポンスが返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("backlink-ok@example.com").
		WithAtname("backlinkok").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("backlink-ok").
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
		Build()

	// 編集中のページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Editing Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	// リンク先ページ（バックリンクの対象）
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Linked Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/backlink-ok/pages/1/links/2/backlink_list", map[string]string{
		"space_identifier":   "backlink-ok",
		"page_number":        "1",
		"linked_page_number": "2",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// ページネーションコンテナが含まれること
	body := rr.Body.String()
	if !strings.Contains(body, "page-backlink-list-") {
		t.Error("response should contain pagination container")
	}
}

func TestShow_正常系_バックリンクありでHTMLレスポンスにバックリンクが含まれる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("backlink-has@example.com").
		WithAtname("backlinkhas").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("backlink-has").
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
		Build()

	// リンク先ページ（バックリンクの対象）
	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Target Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	// 編集中のページ（リンク先ページへのリンクを持つ）
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Editing Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	// バックリンク元ページ（リンク先ページへのリンクを持つ別ページ）
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Backlink Source").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/backlink-has/pages/1/links/2/backlink_list", map[string]string{
		"space_identifier":   "backlink-has",
		"page_number":        "1",
		"linked_page_number": "2",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// バックリンク元ページのタイトルが含まれること
	if !strings.Contains(body, "Backlink Source") {
		t.Error("response should contain backlink source page title 'Backlink Source'")
	}

	// 編集中のページはバックリンクから除外されること
	if strings.Contains(body, "Editing Page") {
		t.Error("response should not contain editing page title 'Editing Page'")
	}
}

func TestShow_正常系_ページネーションパラメータが反映される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("backlink-page@example.com").
		WithAtname("backlinkpage").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("backlink-page").
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
		Build()

	// リンク先ページ
	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Target Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	// 編集中のページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Editing Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	// バックリンク元ページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Backlink Source").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	handler := setupHandler(t, queries)

	// page=1でバックリンクが含まれる
	req := newRequestWithChiParams(t, http.MethodGet, "/s/backlink-page/pages/1/links/2/backlink_list?page=1", map[string]string{
		"space_identifier":   "backlink-page",
		"page_number":        "1",
		"linked_page_number": "2",
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
	if !strings.Contains(body, "Backlink Source") {
		t.Error("page=1 should contain 'Backlink Source'")
	}

	// page=999（存在しないページ）ではバックリンクが含まれない
	req2 := newRequestWithChiParams(t, http.MethodGet, "/s/backlink-page/pages/1/links/2/backlink_list?page=999", map[string]string{
		"space_identifier":   "backlink-page",
		"page_number":        "1",
		"linked_page_number": "2",
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
	if strings.Contains(body2, "Backlink Source") {
		t.Error("page=999 should not contain 'Backlink Source'")
	}
}

// The repository computes the SQL offset as (page-1)*limit in int32 arithmetic, so a large enough
// page wraps to a negative offset and PostgreSQL rejects the query. Such a page has to be a 404
// rather than a 500. The boundary depends on the limit this handler passes, so it can only be
// pinned here and not in the shared helper's unit test.
//
// [Ja] Repository は SQL offset を int32 演算の (page-1)*limit で求めるため、十分に大きいページでは
// 負の offset に回り込み PostgreSQL がクエリを拒否する。そのようなページは 500 ではなく 404 にする。
// 境界は本 Handler が渡す上限に依存するため、共有ヘルパーの単体テストではなくここでしか固定できない。
func TestShow_OffsetBeyondInt32ReturnsNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("backlink-offset@example.com").
		WithAtname("backlinkoffset").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("backlink-offset").
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
		Build()

	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Target Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Editing Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Backlink Source").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	handler := setupHandler(t, queries)

	// BacklinkLimit is 13, so page 165191051 has the first offset past the int32 ceiling
	// (165191050 * 13 = 2147483650).
	//
	// [Ja] BacklinkLimit は 13 なので、165191051 ページ目が int32 の上限を最初に超える offset に
	// なる (165191050 * 13 = 2147483650)。
	req := newRequestWithChiParams(t, http.MethodGet, "/s/backlink-offset/pages/1/links/2/backlink_list?page=165191051", map[string]string{
		"space_identifier":   "backlink-offset",
		"page_number":        "1",
		"linked_page_number": "2",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
	if strings.Contains(rr.Body.String(), "Backlink Source") {
		t.Error("response should not contain 'Backlink Source'")
	}
}
