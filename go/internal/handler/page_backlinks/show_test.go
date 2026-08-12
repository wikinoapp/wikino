package page_backlinks_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/page_backlinks"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// setupHandler はテスト用のハンドラーを生成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *page_backlinks.Handler {
	t.Helper()

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	getPageBacklinksUC := usecase.NewGetPageBacklinksUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo)

	return page_backlinks.NewHandler(
		getPageBacklinksUC,
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

	req := newRequestWithChiParams(t, http.MethodGet, "/s/test-space/pages/1/backlinks", map[string]string{
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
		WithEmail("pb-notfound@example.com").
		WithAtname("pbnotfound").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/nonexistent/pages/1/backlinks", map[string]string{
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
		WithEmail("pb-owner@example.com").
		WithAtname("pbowner").
		Build()
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("pb-outsider@example.com").
		WithAtname("pboutsider").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pb-private").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/pb-private/pages/1/backlinks", map[string]string{
		"space_identifier": "pb-private",
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
		WithEmail("pb-invalid@example.com").
		WithAtname("pbinvalid").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/test-space/pages/abc/backlinks", map[string]string{
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

func TestShow_正常系_バックリンクなしでHTMLレスポンスが返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pb-ok@example.com").
		WithAtname("pbok").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pb-ok").
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

	// 対象ページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Target Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/pb-ok/pages/1/backlinks", map[string]string{
		"space_identifier": "pb-ok",
		"page_number":      "1",
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
	if !strings.Contains(body, "page-backlink-list-pagination") {
		t.Error("response should contain pagination container")
	}
}

func TestShow_正常系_バックリンクありでHTMLレスポンスにバックリンクが含まれる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pb-has@example.com").
		WithAtname("pbhas").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pb-has").
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

	// 対象ページ
	targetPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Target Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	// バックリンク元ページ（対象ページへのリンクを持つ）
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Backlink Source").
		WithLinkedPageIDs([]model.PageID{targetPageID}).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/pb-has/pages/1/backlinks", map[string]string{
		"space_identifier": "pb-has",
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

	// バックリンク元ページのタイトルが含まれること
	if !strings.Contains(body, "Backlink Source") {
		t.Error("response should contain backlink source page title 'Backlink Source'")
	}
}

func TestShow_正常系_ページネーションパラメータが反映される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pb-page@example.com").
		WithAtname("pbpage").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pb-page").
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

	// 対象ページ
	targetPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Target Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	// バックリンク元ページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Backlink Source").
		WithLinkedPageIDs([]model.PageID{targetPageID}).
		Build()

	handler := setupHandler(t, queries)

	// page=1でバックリンクが含まれる
	req := newRequestWithChiParams(t, http.MethodGet, "/s/pb-page/pages/1/backlinks?page=1", map[string]string{
		"space_identifier": "pb-page",
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
	if !strings.Contains(body, "Backlink Source") {
		t.Error("page=1 should contain 'Backlink Source'")
	}

	// page=999（存在しないページ）ではバックリンクが含まれない
	req2 := newRequestWithChiParams(t, http.MethodGet, "/s/pb-page/pages/1/backlinks?page=999", map[string]string{
		"space_identifier": "pb-page",
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
		WithEmail("pb-offset@example.com").
		WithAtname("pboffset").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pb-offset").
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

	targetPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Target Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Backlink Source").
		WithLinkedPageIDs([]model.PageID{targetPageID}).
		Build()

	handler := setupHandler(t, queries)

	// PageBacklinkLimit is 14, so page 153391691 has the first offset past the int32 ceiling
	// (153391690 * 14 = 2147483660).
	//
	// [Ja] PageBacklinkLimit は 14 なので、153391691 ページ目が int32 の上限を最初に超える offset に
	// なる (153391690 * 14 = 2147483660)。
	req := newRequestWithChiParams(t, http.MethodGet, "/s/pb-offset/pages/1/backlinks?page=153391691", map[string]string{
		"space_identifier": "pb-offset",
		"page_number":      "1",
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
