package draft_page_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// newShowRequestはchiのURLパラメータ付きGETリクエストを作成するヘルパーです
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

	req := newShowRequest(t, "/s/test-space/pages/1/draft_page", map[string]string{
		"space_identifier": "test-space",
		"page_number":      "1",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnauthorized)
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

	req := newShowRequest(t, "/s/nonexistent/pages/1/draft_page", map[string]string{
		"space_identifier": "nonexistent",
		"page_number":      "1",
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

	req := newShowRequest(t, "/s/show-private/pages/1/draft_page", map[string]string{
		"space_identifier": "show-private",
		"page_number":      "1",
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
		WithEmail("show-invalid@example.com").
		WithAtname("showinvalid").
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/test-space/pages/abc/draft_page", map[string]string{
		"space_identifier": "test-space",
		"page_number":      "abc",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_正常系_リンクなしでOOBスワップレスポンスが返る(t *testing.T) {
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
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/show-ok/pages/1/draft_page", map[string]string{
		"space_identifier": "show-ok",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// OOBスワップ用の要素が含まれること
	if !strings.Contains(body, `hx-swap-oob="innerHTML"`) {
		t.Error("レスポンスにOOBスワップの属性が含まれていない")
	}
}

func TestShow_正常系_リンクありでレスポンスにリンク先が含まれる(t *testing.T) {
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

	req := newShowRequest(t, "/s/show-links/pages/1/draft_page", map[string]string{
		"space_identifier": "show-links",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Linked Page") {
		t.Error("レスポンスにリンク先ページのタイトル'Linked Page'が含まれていない")
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

	req := newShowRequest(t, "/s/show-draft/pages/1/draft_page", map[string]string{
		"space_identifier": "show-draft",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "Draft Link") {
		t.Error("レスポンスに下書きのリンク先ページのタイトル'Draft Link'が含まれていない")
	}

	if strings.Contains(body, "Original Link") {
		t.Error("レスポンスに元のリンク先ページのタイトル'Original Link'が含まれている")
	}
}

// TestShow_OutOfRangePageStillReturnsTheRemainingLinksは、画面のページ番号が一覧より長生き
// したときに下書き再取得が何を返すかを固定する。返すのは空のコンテナではなく残っているリンク全部で
// ある。本応答はコンテナへ追記せず差し替えるため、何も返さないと次のフルページ読み込みまで編集画面の
// リンク一覧が空のままになってしまう。
func TestShow_OutOfRangePageStillReturnsTheRemainingLinks(t *testing.T) {
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
	req := newShowRequest(t, "/s/show-page/pages/1/draft_page?links_page=1", map[string]string{
		"space_identifier": "show-page",
		"page_number":      "1",
	})
	req.URL.RawQuery = "links_page=1"
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Link Page 1") {
		t.Error("links_page=1に'Link Page 1'が含まれていない")
	}
	if !strings.Contains(body, "Link Page 2") {
		t.Error("links_page=1に'Link Page 2'が含まれていない")
	}

	// 一覧がもう持たないページ番号でも、累積取得上限内なら残っているリンクをすべて返す。
	limitQuery := fmt.Sprintf("%s=%d", viewmodel.LinkPageQueryParam, usecase.MaxCumulativeRelatedPagePages)
	req2 := newShowRequest(t, "/s/show-page/pages/1/draft_page?"+limitQuery, map[string]string{
		"space_identifier": "show-page",
		"page_number":      "1",
	})
	req2.URL.RawQuery = limitQuery
	ctx2 := middleware.SetUserToContext(req2.Context(), &model.User{ID: userID})
	req2 = req2.WithContext(ctx2)

	rr2 := httptest.NewRecorder()
	handler.Show(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr2.Code, http.StatusOK)
	}

	body2 := rr2.Body.String()
	if !strings.Contains(body2, "Link Page 1") {
		t.Errorf("links_page=%dに'Link Page 1'が含まれなくなっている", usecase.MaxCumulativeRelatedPagePages)
	}
	if !strings.Contains(body2, "Link Page 2") {
		t.Errorf("links_page=%dに'Link Page 2'が含まれなくなっている", usecase.MaxCumulativeRelatedPagePages)
	}

	// 各関連ページ一覧の次ページは、累積クエリがサーバー側の予算を超える前に拒否する。
	for _, param := range []string{viewmodel.LinkPageQueryParam, viewmodel.LinkedBacklinkPageQueryParam, viewmodel.PageBacklinkPageQueryParam} {
		t.Run(param, func(t *testing.T) {
			overLimitQuery := fmt.Sprintf("%s=%d", param, usecase.MaxCumulativeRelatedPagePages+1)
			req3 := newShowRequest(t, "/s/show-page/pages/1/draft_page?"+overLimitQuery, map[string]string{
				"space_identifier": "show-page",
				"page_number":      "1",
			})
			req3.URL.RawQuery = overLimitQuery
			req3 = req3.WithContext(middleware.SetUserToContext(req3.Context(), &model.User{ID: userID}))

			rr3 := httptest.NewRecorder()
			handler.Show(rr3, req3)
			if rr3.Code != http.StatusNotFound {
				t.Errorf("%sの累積上限を超えたときのステータス = %d、期待値 = %d", param, rr3.Code, http.StatusNotFound)
			}
		})
	}
}

// TestShow_ReturnsEveryPageThroughTheRequestedOneは、下書き再取得が閲覧者の読み込み済みの
// 範囲すべてを描画し直すことを固定する。htmxは「もっと見る」の各ページをコンテナへ追記するため、
// 要求ページだけを返す応答ではそれ以前の範囲が消えてしまう。
func TestShow_ReturnsEveryPageThroughTheRequestedOne(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("show-cumulative@example.com").
		WithAtname("showcumulative").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("show-cumulative").
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

	// 一覧は1ページ目より1枚多いため、2ページ目には最後のカードだけが載る。
	baseTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	linkedCount := int(viewmodel.LinkLimit) + 1
	linkedPageIDs := make([]model.PageID, 0, linkedCount)
	for i := range linkedCount {
		linkedPageIDs = append(linkedPageIDs, testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(100+i)).
			WithTitle(fmt.Sprintf("Cumulative Link %02d", i)).
			WithModifiedAt(baseTime.Add(time.Duration(linkedCount-i)*time.Hour)).
			Build())
	}

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Cumulative Source Page").
		WithLinkedPageIDs(linkedPageIDs).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/show-cumulative/pages/1/draft_page?links_page=2", map[string]string{
		"space_identifier": "show-cumulative",
		"page_number":      "1",
	})
	req.URL.RawQuery = "links_page=2"
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	firstOfPage1 := "Cumulative Link 00"
	lastOfPage1 := fmt.Sprintf("Cumulative Link %02d", viewmodel.LinkLimit-1)
	onlyOfPage2 := fmt.Sprintf("Cumulative Link %02d", viewmodel.LinkLimit)

	for _, want := range []string{firstOfPage1, lastOfPage1, onlyOfPage2} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}

	// 一覧に残りは無いため、応答はこれ以上のページを提示してはならない。
	if strings.Contains(body, "/s/show-cumulative/pages/1/link_list?") {
		t.Error("最後のページを表示したのにレスポンスに「さらに読み込む」リンクが描画されている")
	}
}

func TestShow_正常系_下書きにリンクを追加するとレスポンスに反映される(t *testing.T) {
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

	// 1. リンクなしの状態でエンドポイントを呼び出す
	req1 := newShowRequest(t, "/s/show-add/pages/1/draft_page", map[string]string{
		"space_identifier": "show-add",
		"page_number":      "1",
	})
	ctx1 := middleware.SetUserToContext(req1.Context(), &model.User{ID: userID})
	req1 = req1.WithContext(ctx1)
	rr1 := httptest.NewRecorder()
	handler.Show(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("1回目のリクエスト: ステータスコード = %v、期待値 = %v", rr1.Code, http.StatusOK)
	}

	body1 := rr1.Body.String()
	if strings.Contains(body1, "Linked Target") {
		t.Error("1回目のリクエスト: 下書きの保存前に'Linked Target'が含まれている")
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

	// 3. 下書き保存後にエンドポイントを呼び出す
	req2 := newShowRequest(t, "/s/show-add/pages/1/draft_page", map[string]string{
		"space_identifier": "show-add",
		"page_number":      "1",
	})
	ctx2 := middleware.SetUserToContext(req2.Context(), &model.User{ID: userID})
	req2 = req2.WithContext(ctx2)
	rr2 := httptest.NewRecorder()
	handler.Show(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("2回目のリクエスト: ステータスコード = %v、期待値 = %v", rr2.Code, http.StatusOK)
	}

	body2 := rr2.Body.String()
	if !strings.Contains(body2, "Linked Target") {
		t.Error("2回目のリクエスト: Wikiリンクを含む下書きの保存後に'Linked Target'が含まれていない")
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

	req := newShowRequest(t, "/s/show-time/pages/1/draft_page", map[string]string{
		"space_identifier": "show-time",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// 保存時刻のOOBスワップ要素が含まれること
	if !strings.Contains(body, `id="page-draft-saved-at"`) {
		t.Error("レスポンスにidが'page-draft-saved-at'の保存日時の要素が含まれていない")
	}
	if !strings.Contains(body, `hx-swap-oob="outerHTML"`) {
		t.Error("レスポンスに保存日時のouterHTMLのOOBスワップが含まれていない")
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
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/show-notime/pages/1/draft_page", map[string]string{
		"space_identifier": "show-notime",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// リンク一覧のOOBスワップ要素は含まれる
	if !strings.Contains(body, `id="page-link-list"`) {
		t.Error("レスポンスにリンク一覧のOOBの要素が含まれていない")
	}

	// 保存時刻のOOBスワップ要素 (outerHTML) は含まれないこと
	if strings.Contains(body, `id="page-draft-saved-at"`) {
		t.Error("下書きが無いのにレスポンスに保存日時のOOBスワップが含まれている")
	}
}

// TestShow_CombinedStateFromSeveralListingsKeepsEveryRangeは、編集画面で共有する状態の終端を
// 固定する。別々の「もっと見る」で進めた一覧を含むリクエストには、そのすべての範囲が返る。各
// フラグメント応答は共有状態のうち自分の分だけを進めるため
// (TestRelatedPageListResponses_AdvanceOnlyTheirOwnState)、下書き再取得が発火する時点で状態は一覧
// ごとに1つのページを指しており、本応答はそれらを同時に満たす必要がある。
func TestShow_CombinedStateFromSeveralListingsKeepsEveryRange(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("show-combined@example.com").
		WithAtname("showcombined").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("show-combined").
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

	// どちらの一覧も1ページ目より1枚多くし、それぞれに進める2ページ目を作る。
	baseTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	linkedCount := int(viewmodel.LinkLimit) + 1
	linkedPageIDs := make([]model.PageID, 0, linkedCount)
	for i := range linkedCount {
		linkedPageIDs = append(linkedPageIDs, testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(100+i)).
			WithTitle(fmt.Sprintf("Combined Link %02d", i)).
			WithModifiedAt(baseTime.Add(time.Duration(linkedCount-i)*time.Hour)).
			Build())
	}

	sourcePageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Combined Source Page").
		WithLinkedPageIDs(linkedPageIDs).
		Build()

	backlinkCount := int(viewmodel.PageBacklinkLimit) + 1
	for i := range backlinkCount {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(200 + i)).
			WithTitle(fmt.Sprintf("Combined Backlink %02d", i)).
			WithLinkedPageIDs([]model.PageID{sourcePageID}).
			WithModifiedAt(baseTime.Add(time.Duration(backlinkCount-i) * time.Hour)).
			Build()
	}

	handler := setupHandler(t, queries)

	rawQuery := fmt.Sprintf("%s=2&%s=2", viewmodel.LinkPageQueryParam, viewmodel.PageBacklinkPageQueryParam)
	req := newShowRequest(t, "/s/show-combined/pages/1/draft_page?"+rawQuery, map[string]string{
		"space_identifier": "show-combined",
		"page_number":      "1",
	})
	req.URL.RawQuery = rawQuery
	req = req.WithContext(middleware.SetUserToContext(req.Context(), &model.User{ID: userID}))

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	wantContains := []string{
		"Combined Link 00",
		fmt.Sprintf("Combined Link %02d", viewmodel.LinkLimit),
		"Combined Backlink 00",
		fmt.Sprintf("Combined Backlink %02d", viewmodel.PageBacklinkLimit),
	}
	for _, want := range wantContains {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
}
