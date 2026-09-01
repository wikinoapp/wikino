package page_link_list_test

import (
	"context"
	"fmt"
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
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// setupHandler はテスト用のハンドラーを生成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *page_link_list.Handler {
	t.Helper()

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	getLinkListUC := usecase.NewGetLinkListUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo, draftPageRepo)

	return page_link_list.NewHandler(
		getLinkListUC,
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
// behind the "load more" button of the public page detail screen: the listing for a page in a
// public topic, a 404 for a page in a private one, and no edit link on the cards.
//
// [Ja] TestShow_GuestCanOnlyFetchPagesInPublicTopics は、公開のページ表示画面の
// 「もっと見る」ボタンが叩くエンドポイントでゲストが得るものを固定する。公開トピックのページの
// 一覧は返り、非公開トピックのページは 404 になり、カードには編集リンクが出ない。
func TestShow_GuestCanOnlyFetchPagesInPublicTopics(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pll-guest").
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

	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(2).
		WithTitle("Guest Linked Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Guest Source Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(3).
		WithTitle("Private Source Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	handler := setupHandler(t, queries)

	tests := []struct {
		name            string
		pageNumber      string
		wantStatus      int
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:         "公開トピックのページのリンク一覧は返る",
			pageNumber:   "1",
			wantStatus:   http.StatusOK,
			wantContains: []string{"Guest Linked Page"},
			// The edit link must not be offered to a viewer who cannot edit the linked page.
			//
			// [Ja] リンク先ページを編集できない閲覧者に編集リンクを出してはならない。
			wantNotContains: []string{"/pages/2/edit"},
		},
		{
			name:            "非公開トピックのページのリンク一覧は 404",
			pageNumber:      "3",
			wantStatus:      http.StatusNotFound,
			wantNotContains: []string{"Guest Linked Page"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequestWithChiParams(t, http.MethodGet, "/s/pll-guest/pages/"+tt.pageNumber+"/link_list", map[string]string{
				"space_identifier": "pll-guest",
				"page_number":      tt.pageNumber,
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
		WithEmail("pll-notfound@example.com").
		WithAtname("pllnotfound").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/nonexistent/pages/1/link_list", map[string]string{
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
		WithEmail("pll-owner@example.com").
		WithAtname("pllowner").
		Build()
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("pll-outsider@example.com").
		WithAtname("plloutsider").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pll-private").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/pll-private/pages/1/link_list", map[string]string{
		"space_identifier": "pll-private",
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
		WithEmail("pll-invalid@example.com").
		WithAtname("pllinvalid").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/test-space/pages/abc/link_list", map[string]string{
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

func TestShow_正常系_リンクなしで空レスポンスが返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pll-empty@example.com").
		WithAtname("pllempty").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pll-empty").
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

	// リンクなしのページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("No Links Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/pll-empty/pages/1/link_list", map[string]string{
		"space_identifier": "pll-empty",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// リンクなしの場合、ページネーションコンテナのみ返る
	body := rr.Body.String()
	if !strings.Contains(body, "page-link-list-pagination") {
		t.Error("response should contain pagination container")
	}
}

// TestShow_MemberSeesLinkedPagesWithStoredIdentifier pins the listing an editing member receives:
// the linked pages, their edit links, and URLs built from the stored space identifier rather than
// from the one spelled in the request.
//
// [Ja] TestShow_MemberSeesLinkedPagesWithStoredIdentifier は編集中のメンバーが受け取る一覧を固定する。
// リンク先ページ・その編集リンク、およびリクエストの表記ではなく保存済みスペース識別子から
// 組み立てた URL が対象。
func TestShow_MemberSeesLinkedPagesWithStoredIdentifier(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pll-has@example.com").
		WithAtname("pllhas").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pll-has").
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
		WithTitle("Linked Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	// 対象ページ（リンク先を持つ）
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/PLL-HAS/pages/1/link_list", map[string]string{
		"space_identifier": "PLL-HAS",
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

	// リンク先ページのタイトルが含まれること
	if !strings.Contains(body, "Linked Page") {
		t.Error("response should contain linked page title 'Linked Page'")
	}

	// ページネーションコンテナが含まれること
	if !strings.Contains(body, "page-link-list-pagination") {
		t.Error("response should contain pagination container")
	}

	// A member who can edit the page keeps the per-card edit link.
	//
	// [Ja] ページを編集できるメンバーには各カードの編集リンクが出る。
	if !strings.Contains(body, "/pages/2/edit") {
		t.Error("response should contain the edit link of the linked page")
	}
	if !strings.Contains(body, "/s/pll-has/") || strings.Contains(body, "/s/PLL-HAS/") {
		t.Error("response URLs should use the stored space identifier")
	}
}

func TestShow_PaginationAndOutOfRangePage(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pll-page@example.com").
		WithAtname("pllpage").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pll-page").
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
		WithTitle("Link Target").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	// 対象ページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	handler := setupHandler(t, queries)

	// page=1でリンクが含まれる
	req := newRequestWithChiParams(t, http.MethodGet, "/s/pll-page/pages/1/link_list?page=1", map[string]string{
		"space_identifier": "pll-page",
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
	if !strings.Contains(body, "Link Target") {
		t.Error("page=1 should contain 'Link Target'")
	}

	// An htmx request past the final page gets a local 404 fragment instead of an empty success.
	//
	// [Ja] 最終ページを超えた htmx リクエストは、空の成功レスポンスではなく局所的な 404 フラグメントを返す。
	req2 := newRequestWithChiParams(t, http.MethodGet, "/s/pll-page/pages/1/link_list?page=999", map[string]string{
		"space_identifier": "pll-page",
		"page_number":      "1",
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
	if strings.Contains(body2, "Link Target") {
		t.Error("page=999 should not contain 'Link Target'")
	}
}

// The repository computes the SQL offset as (page-1)*limit in int32 arithmetic, so a large enough
// page wraps to a negative offset and PostgreSQL rejects the query. Such a page has to be a 404
// rather than a 500.
//
// [Ja] Repository は SQL offset を int32 演算の (page-1)*limit で求めるため、十分に大きいページでは
// 負の offset に回り込み PostgreSQL がクエリを拒否する。そのようなページは 500 ではなく 404 にする。
func TestShow_OffsetBeyondInt32ReturnsNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pll-offset@example.com").
		WithAtname("plloffset").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pll-offset").
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
		WithTitle("Link Target").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	handler := setupHandler(t, queries)

	// RelatedPageFollowingLimit is 15, so page 143165578 has the first upper-bound offset past the int32 ceiling
	// (143165577 * 15 = 2147483655).
	//
	// [Ja] RelatedPageFollowingLimit は 15 なので、143165578 ページ目が int32 の上限を最初に超える offset になる
	// (143165577 * 15 = 2147483655)。
	req := newRequestWithChiParams(t, http.MethodGet, "/s/pll-offset/pages/1/link_list?page=143165578", map[string]string{
		"space_identifier": "pll-offset",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
	if strings.Contains(rr.Body.String(), "Link Target") {
		t.Error("response should not contain 'Link Target'")
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
	req := newRequestWithChiParams(t, http.MethodGet, "/s/example/pages/invalid/link_list", map[string]string{
		"space_identifier": "example",
		"page_number":      "invalid",
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

// TestShow_ShowContextKeepsSavedLinksAndScreenState pins the two things the context parameter
// decides: which link set the fragment continues, and where the links it renders point. The page
// detail screen must keep listing the saved links and must not have the fragment reset the other
// listings or send the fallback to the editor.
//
// [Ja] TestShow_ShowContextKeepsSavedLinksAndScreenState は context パラメータが決める 2 つを固定する。
// フラグメントがどのリンク集合の続きを返すかと、描画するリンクがどこを指すかである。ページ表示画面では
// 保存済みリンクを一覧し続け、他の一覧を戻したり編集画面へフォールバックしたりしてはならない。
func TestShow_ShowContextKeepsSavedLinksAndScreenState(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pll-context@example.com").
		WithAtname("pllcontext").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pll-context").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()

	// One more saved link than fits on a page, so the listing renders a "load more" link whose URLs
	// can be inspected.
	//
	// [Ja] 1 ページに収まる件数より 1 件多く保存済みリンクを作り、URL を検査できる「もっと見る」リンクを
	// 一覧に描画させる。
	savedPageIDs := make([]model.PageID, 0, int(viewmodel.LinkLimit)+1)
	for index := int32(0); index <= viewmodel.LinkLimit; index++ {
		savedPageIDs = append(savedPageIDs, testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(10+index)).
			WithTitle(fmt.Sprintf("Saved Link %d", index)).
			WithLinkedPageIDs([]model.PageID{}).
			Build())
	}
	draftLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(90).
		WithTitle("Draft Link").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	sourcePageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source Page").
		WithLinkedPageIDs(savedPageIDs).
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(sourcePageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithLinkedPageIDs([]model.PageID{draftLinkedPageID}).
		Build()

	handler := setupHandler(t, queries)

	tests := []struct {
		name            string
		query           string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:  "表示画面の文脈では保存済みリンクを返し、他の一覧の状態を引き継ぐ",
			query: "context=show&page=1&backlinks_page=3",
			wantContains: []string{
				"Saved Link",
				`hx-get="/s/pll-context/pages/1/link_list?backlinks_page=3&amp;context=show&amp;page=2"`,
				`href="/s/pll-context/pages/1?backlinks_page=3&amp;links_page=2#page-link-list-content"`,
			},
			wantNotContains: []string{"Draft Link"},
		},
		{
			name:         "編集画面の文脈では下書きのリンクを返す",
			query:        "context=edit&page=1",
			wantContains: []string{"Draft Link"},
			// The editor's fallback goes to the editor, not to the page detail screen.
			//
			// [Ja] 編集画面のフォールバック先は、ページ表示画面ではなく編集画面である。
			wantNotContains: []string{"Saved Link"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequestWithChiParams(t, http.MethodGet, "/s/pll-context/pages/1/link_list?"+tt.query, map[string]string{
				"space_identifier": "pll-context",
				"page_number":      "1",
			})
			req.URL.RawQuery = tt.query
			req = req.WithContext(middleware.SetUserToContext(req.Context(), &model.User{ID: userID}))

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
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
		WithIdentifier("pll-limit").
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
			rawQuery:   fmt.Sprintf("context=edit&page=1&%s=%d", viewmodel.PageBacklinkPageQueryParam, overLimit),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "1ページ単位の編集画面は上限を超えた引き継ぎ状態でも200",
			rawQuery:   fmt.Sprintf("context=edit_paginated&page=1&%s=%d", viewmodel.PageBacklinkPageQueryParam, overLimit),
			wantStatus: http.StatusOK,
		},
		{
			name:       "ページ表示画面は上限を超えた引き継ぎ状態でも200",
			rawQuery:   fmt.Sprintf("context=show&page=1&%s=%d", viewmodel.PageBacklinkPageQueryParam, overLimit),
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequestWithChiParams(t, http.MethodGet, "/s/pll-limit/pages/1/link_list?"+tt.rawQuery, map[string]string{
				"space_identifier": "pll-limit",
				"page_number":      "1",
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
