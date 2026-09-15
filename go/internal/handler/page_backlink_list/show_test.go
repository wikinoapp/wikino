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

// setupHandlerはテスト用のハンドラーを生成するヘルパーです
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

// newRequestWithChiParamsはchiのURLパラメータ付きリクエストを作成するヘルパーです
func newRequestWithChiParams(t *testing.T, method, path string, params map[string]string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// TestShow_GuestCanOnlyFetchPagesInPublicTopicsは、公開のページ表示画面にある
// リンク先ページのバックリンクの「もっと見る」ボタンが叩くエンドポイントでゲストが得るものを固定する。
// 双方が公開トピックならバックリンクは返り、リンク先ページが非公開トピックなら404になり、
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
			// 一覧したページを編集できない閲覧者に編集リンクを出してはならない。
			wantNotContains: []string{"/pages/3/edit"},
		},
		{
			name:             "非公開トピックのリンク先ページのバックリンク一覧は404",
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
				t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, tt.wantStatus)
			}

			body := rr.Body.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(body, notWant) {
					t.Errorf("レスポンスに想定外の%qが含まれている", notWant)
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
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

	// リンク先ページ (バックリンクの対象)
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	// ページネーションコンテナが含まれること
	body := rr.Body.String()
	if !strings.Contains(body, "page-backlink-list-") {
		t.Error("レスポンスにページネーションのコンテナが含まれていない")
	}
}

// TestShow_MemberSeesLinkedPageBacklinksWithStoredIdentifierは、編集中のメンバーが1つの
// リンク先ページについて受け取る一覧を固定する。バックリンク元のページ・その編集リンク、および
// リクエストの表記ではなく保存済みスペース識別子から組み立てたURLが対象。
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

	// リンク先ページ (バックリンクの対象)
	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Target Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	// 編集中のページ (リンク先ページへのリンクを持つ)
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Editing Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	// バックリンク元ページ (リンク先ページへのリンクを持つ別ページ)
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// バックリンク元ページのタイトルが含まれること
	if !strings.Contains(body, "Backlink Source") {
		t.Error("レスポンスにバックリンク元ページのタイトル'Backlink Source'が含まれていない")
	}

	// 編集中のページはバックリンクから除外されること
	if strings.Contains(body, "Editing Page") {
		t.Error("レスポンスに編集中のページタイトル'Editing Page'が含まれている")
	}

	// ページを編集できるメンバーには各カードの編集リンクが出る。
	if !strings.Contains(body, "/pages/3/edit") {
		t.Error("レスポンスにバックリンク元ページの編集リンクが含まれていない")
	}
	if !strings.Contains(body, "/s/backlink-has/") || strings.Contains(body, "/s/BACKLINK-HAS/") {
		t.Error("レスポンスのURLが保存済みのスペース識別子を使っていない")
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Backlink Source") {
		t.Error("page=1に'Backlink Source'が含まれていない")
	}

	// 最終ページを超えたhtmxリクエストは、空の成功レスポンスではなく局所的な404フラグメントを返す。
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr2.Code, http.StatusNotFound)
	}

	body2 := rr2.Body.String()
	if !strings.Contains(body2, `role="alert"`) {
		t.Error("範囲外のページに局所的なエラーの断片が含まれていない")
	}
	if strings.Contains(strings.ToLower(body2), "<!doctype html") {
		t.Error("htmxの404レスポンスにページ全体のドキュメントが含まれている")
	}
	if strings.Contains(body2, "Backlink Source") {
		t.Error("page=999に'Backlink Source'が含まれている")
	}
}

// RepositoryはSQL offsetをint32演算の (page-1)*limitで求めるため、十分に大きいページでは
// 負のoffsetに回り込みPostgreSQLがクエリを拒否する。そのようなページは500ではなく404にする。
// 境界は本Handlerが渡す上限に依存するため、共有ヘルパーの単体テストではなくここでしか固定できない。
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

	// RelatedPageFollowingLimitは15なので、143165578ページ目がint32の上限を最初に超えるoffsetに
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
	if strings.Contains(rr.Body.String(), "Backlink Source") {
		t.Error("レスポンスに'Backlink Source'が含まれている")
	}
}

// TestShow_HTMXNotFoundReturnsLocalFragmentはページネーション領域の404で、htmxに完全な
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
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if strings.Contains(strings.ToLower(body), "<!doctype html") {
		t.Error("htmxの404に完全なHTMLドキュメントが含まれている")
	}
	if !strings.Contains(body, `role="alert"`) {
		t.Error("htmxの404に局所的なアラートの断片が含まれていない")
	}
}

// TestShow_ShowContextKeepsScreenStateは、ネストした一覧のフラグメントが画面全体の状態を指す
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

	// 1ページに収まる件数より1件多くバックリンクを作り、ネストした一覧に「もっと見る」リンクを
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
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	wantContains := []string{
		`hx-get="/s/pbl-context/pages/1/links/2/backlink_list?backlinks_page=3&amp;context=show&amp;links_page=2&amp;page=2&amp;parent_page=2"`,
		// リクエストは親ページを指定していない。これは同パラメータが存在しなかった頃のリンクの形で
		// ある。当時のリンクはカードを、一緒に運んでいるリンク一覧ページと組にしていたため、フォール
		// バックは1ページ目ではなくそのページを描画する。
		`href="/s/pbl-context/pages/1?backlinks_page=3&amp;linked_backlinks_page=2&amp;linked_page_number=2&amp;links_page=2#page-link-list-item-2"`,
		// リンクのアクセシブルネームでリンク先ページが一覧を言い表す。
		`aria-label="Linked Pageのバックリンクをもっと見る"`,
	}
	for _, want := range wantContains {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	if strings.Contains(body, "/pages/1/edit?") {
		t.Error("ページ詳細の文脈がエディタにフォールバックしている")
	}
}

// TestShow_CumulativeFetchLimitは、本フラグメントが受け付ける各一覧のページを編集画面の累積
// 取得上限が縛る一方、ページ表示画面には上限が無いことを固定する。編集画面は1回の下書き再取得で
// 3一覧の読み込み済み範囲すべてを描画し直すため、上限を超えた状態を本フラグメントが返しても、
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
				t.Errorf("ステータス = %d、期待値 = %d", rr.Code, tt.wantStatus)
			}
		})
	}
}
