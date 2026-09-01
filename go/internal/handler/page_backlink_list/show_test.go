package page_backlink_list_test

import (
	"context"
	"fmt"
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
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
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

// TestShow_GuestCanOnlyFetchPagesInPublicTopics pins what a guest gets from the endpoint
// behind the "load more" button of a linked page's backlinks on the public page detail screen: the
// backlinks when both pages sit in a public topic, a 404 when the linked page sits in a private
// one, and no edit link on the cards.
//
// [Ja] TestShow_GuestCanOnlyFetchPagesInPublicTopics は、公開のページ表示画面にある
// リンク先ページのバックリンクの「もっと見る」ボタンが叩くエンドポイントでゲストが得るものを固定する。
// 双方が公開トピックならバックリンクは返り、リンク先ページが非公開トピックなら 404 になり、
// カードには編集リンクが出ない。
func TestShow_GuestCanOnlyFetchPagesInPublicTopics(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("backlink-guest").
		Build()
	publicTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Public").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()
	privateTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Private").
		WithVisibility(int32(model.TopicVisibilityPrivate)).
		Build()

	publicLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(2).
		WithTitle("Guest Target Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	privateLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(4).
		WithTitle("Private Target Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Guest Source Page").
		WithLinkedPageIDs([]model.PageID{publicLinkedPageID, privateLinkedPageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(3).
		WithTitle("Guest Backlink Source").
		WithLinkedPageIDs([]model.PageID{publicLinkedPageID, privateLinkedPageID}).
		Build()

	handler := setupHandler(t, queries)

	tests := []struct {
		name             string
		linkedPageNumber string
		wantStatus       int
		wantContains     []string
		wantNotContains  []string
	}{
		{
			name:             "公開トピックのリンク先ページのバックリンク一覧は返る",
			linkedPageNumber: "2",
			wantStatus:       http.StatusOK,
			wantContains:     []string{"Guest Backlink Source"},
			// The edit link must not be offered to a viewer who cannot edit the listed page.
			//
			// [Ja] 一覧したページを編集できない閲覧者に編集リンクを出してはならない。
			wantNotContains: []string{"/pages/3/edit"},
		},
		{
			name:             "非公開トピックのリンク先ページのバックリンク一覧は 404",
			linkedPageNumber: "4",
			wantStatus:       http.StatusNotFound,
			wantNotContains:  []string{"Guest Backlink Source"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequestWithChiParams(t, http.MethodGet, "/s/backlink-guest/pages/1/links/"+tt.linkedPageNumber+"/backlink_list", map[string]string{
				"space_identifier":   "backlink-guest",
				"page_number":        "1",
				"linked_page_number": tt.linkedPageNumber,
			})

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("wrong status code: got %v want %v", rr.Code, tt.wantStatus)
			}

			body := rr.Body.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Errorf("response does not contain %q", want)
				}
			}
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(body, notWant) {
					t.Errorf("response unexpectedly contains %q", notWant)
				}
			}
		})
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

// TestShow_MemberSeesLinkedPageBacklinksWithStoredIdentifier pins the listing an editing member
// receives for one listed page: its backlinking pages, their edit links, and URLs built from the
// stored space identifier rather than from the one spelled in the request.
//
// [Ja] TestShow_MemberSeesLinkedPageBacklinksWithStoredIdentifier は、編集中のメンバーが 1 つの
// リンク先ページについて受け取る一覧を固定する。バックリンク元のページ・その編集リンク、および
// リクエストの表記ではなく保存済みスペース識別子から組み立てた URL が対象。
func TestShow_MemberSeesLinkedPageBacklinksWithStoredIdentifier(t *testing.T) {
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

	req := newRequestWithChiParams(t, http.MethodGet, "/s/BACKLINK-HAS/pages/1/links/2/backlink_list", map[string]string{
		"space_identifier":   "BACKLINK-HAS",
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

	// A member who can edit the page keeps the per-card edit link.
	//
	// [Ja] ページを編集できるメンバーには各カードの編集リンクが出る。
	if !strings.Contains(body, "/pages/3/edit") {
		t.Error("response should contain the edit link of the backlink source page")
	}
	if !strings.Contains(body, "/s/backlink-has/") || strings.Contains(body, "/s/BACKLINK-HAS/") {
		t.Error("response URLs should use the stored space identifier")
	}
}

func TestShow_PaginationAndOutOfRangePage(t *testing.T) {
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

	// An htmx request past the final page gets a local 404 fragment instead of an empty success.
	//
	// [Ja] 最終ページを超えた htmx リクエストは、空の成功レスポンスではなく局所的な 404 フラグメントを返す。
	req2 := newRequestWithChiParams(t, http.MethodGet, "/s/backlink-page/pages/1/links/2/backlink_list?page=999", map[string]string{
		"space_identifier":   "backlink-page",
		"page_number":        "1",
		"linked_page_number": "2",
	})
	req2.URL.RawQuery = "page=999"
	req2.Header.Set("HX-Request", "true")
	ctx2 := middleware.SetUserToContext(req2.Context(), &model.User{ID: userID})
	req2 = req2.WithContext(ctx2)

	rr2 := httptest.NewRecorder()
	handler.Show(rr2, req2)

	if rr2.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr2.Code, http.StatusNotFound)
	}

	body2 := rr2.Body.String()
	if !strings.Contains(body2, `role="alert"`) {
		t.Error("out-of-range page should contain a local error fragment")
	}
	if strings.Contains(strings.ToLower(body2), "<!doctype html") {
		t.Error("htmx 404 response should not contain a full-page document")
	}
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

	// RelatedPageFollowingLimit is 15, so page 143165578 has the first upper-bound offset past the int32 ceiling
	// (143165577 * 15 = 2147483655).
	//
	// [Ja] RelatedPageFollowingLimit は 15 なので、143165578 ページ目が int32 の上限を最初に超える offset に
	// なる (143165577 * 15 = 2147483655)。
	req := newRequestWithChiParams(t, http.MethodGet, "/s/backlink-offset/pages/1/links/2/backlink_list?page=143165578", map[string]string{
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

// TestShow_HTMXNotFoundReturnsLocalFragment verifies that htmx never receives a complete document
// for a pagination-target 404.
//
// [Ja] TestShow_HTMXNotFoundReturnsLocalFragment はページネーション領域の 404 で、htmx に完全な
// 文書を返さないことを確認する。
func TestShow_HTMXNotFoundReturnsLocalFragment(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	handler := setupHandler(t, testutil.QueriesWithTx(tx))
	req := newRequestWithChiParams(t, http.MethodGet, "/s/example/pages/invalid/links/2/backlink_list", map[string]string{
		"space_identifier":   "example",
		"page_number":        "invalid",
		"linked_page_number": "2",
	})
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if strings.Contains(strings.ToLower(body), "<!doctype html") {
		t.Error("htmx 404 must not contain a complete HTML document")
	}
	if !strings.Contains(body, `role="alert"`) {
		t.Error("htmx 404 should contain the local alert fragment")
	}
}

// TestShow_ShowContextKeepsScreenState pins that the nested listing's fragment renders links
// pointing at the state the whole screen is in: the page detail fallback rather than the editor,
// with the link list and the page's own backlinks left on the pages they are already showing.
//
// [Ja] TestShow_ShowContextKeepsScreenState は、ネストした一覧のフラグメントが画面全体の状態を指す
// リンクを描画することを固定する。フォールバック先は編集画面ではなくページ表示画面で、リンク一覧と
// ページ自身のバックリンクは現在表示中のページのままになる。
func TestShow_ShowContextKeepsScreenState(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pbl-context").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Public").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()

	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Linked Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	// One more backlink than fits on a page, so the nested listing renders a "load more" link.
	//
	// [Ja] 1 ページに収まる件数より 1 件多くバックリンクを作り、ネストした一覧に「もっと見る」リンクを
	// 描画させる。
	for index := int32(0); index <= viewmodel.BacklinkLimit; index++ {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(10 + index)).
			WithTitle(fmt.Sprintf("Nested Backlink Source %d", index)).
			WithLinkedPageIDs([]model.PageID{linkedPageID}).
			Build()
	}

	handler := setupHandler(t, queries)

	rawQuery := "context=show&page=1&links_page=2&backlinks_page=3"
	req := newRequestWithChiParams(t, http.MethodGet, "/s/pbl-context/pages/1/links/2/backlink_list?"+rawQuery, map[string]string{
		"space_identifier":   "pbl-context",
		"page_number":        "1",
		"linked_page_number": "2",
	})
	req.URL.RawQuery = rawQuery

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	wantContains := []string{
		`hx-get="/s/pbl-context/pages/1/links/2/backlink_list?backlinks_page=3&amp;context=show&amp;links_page=2&amp;page=2&amp;parent_page=2"`,
		// The request named no parent page, which is what a link built before that parameter existed
		// looked like. Such a link paired the card with the link-list page it also carried, so the
		// fallback renders that page rather than the first one.
		//
		// [Ja] リクエストは親ページを指定していない。これは同パラメータが存在しなかった頃のリンクの形で
		// ある。当時のリンクはカードを、一緒に運んでいるリンク一覧ページと組にしていたため、フォール
		// バックは 1 ページ目ではなくそのページを描画する。
		`href="/s/pbl-context/pages/1?backlinks_page=3&amp;linked_backlinks_page=2&amp;linked_page_number=2&amp;links_page=2#page-link-list-item-2"`,
		// The listed page names the listing in the link's accessible name.
		//
		// [Ja] リンクのアクセシブルネームでリンク先ページが一覧を言い表す。
		`aria-label="Linked Pageのバックリンクをもっと見る"`,
	}
	for _, want := range wantContains {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q", want)
		}
	}
	if strings.Contains(body, "/pages/1/edit?") {
		t.Error("the page detail context must not fall back to the editor")
	}
}

// TestShow_CumulativeFetchLimit pins that the editor's cumulative-fetch limit bounds every listing
// page this fragment accepts, while the page detail screen stays unbounded. The editor re-renders
// the whole loaded range of all three listings in one draft refresh, so a state this fragment
// handed back past the limit would only produce a request the draft endpoint has to reject.
//
// [Ja] TestShow_CumulativeFetchLimit は、本フラグメントが受け付ける各一覧のページを編集画面の累積
// 取得上限が縛る一方、ページ表示画面には上限が無いことを固定する。編集画面は 1 回の下書き再取得で
// 3 一覧の読み込み済み範囲すべてを描画し直すため、上限を超えた状態を本フラグメントが返しても、
// 下書きエンドポイントが拒否するほかないリクエストにしかならない。
func TestShow_CumulativeFetchLimit(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pbl-limit").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Public").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()

	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Limit Linked").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Limit Source").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	handler := setupHandler(t, queries)

	overLimit := usecase.MaxCumulativeRelatedPagePages + 1
	tests := []struct {
		name       string
		rawQuery   string
		wantStatus int
	}{
		{
			name:       "編集画面で自身のページが上限を超えると404",
			rawQuery:   fmt.Sprintf("context=edit&page=%d", overLimit),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "編集画面で引き継いだ一覧が上限を超えると404",
			rawQuery:   fmt.Sprintf("context=edit&page=1&%s=%d", viewmodel.LinkPageQueryParam, overLimit),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "1ページ単位の編集画面は上限を超えた引き継ぎ状態でも200",
			rawQuery:   fmt.Sprintf("context=edit_paginated&page=1&%s=%d", viewmodel.LinkPageQueryParam, overLimit),
			wantStatus: http.StatusOK,
		},
		{
			name:       "ページ表示画面は上限を超えた引き継ぎ状態でも200",
			rawQuery:   fmt.Sprintf("context=show&page=1&%s=%d", viewmodel.LinkPageQueryParam, overLimit),
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequestWithChiParams(t, http.MethodGet, "/s/pbl-limit/pages/1/links/2/backlink_list?"+tt.rawQuery, map[string]string{
				"space_identifier":   "pbl-limit",
				"page_number":        "1",
				"linked_page_number": "2",
			})
			req.URL.RawQuery = tt.rawQuery

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}
