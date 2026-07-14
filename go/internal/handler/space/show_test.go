package space_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	spacehandler "github.com/wikinoapp/wikino/go/internal/handler/space"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
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

// setupHandler はテスト用のスペース詳細ハンドラーを作成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *spacehandler.Handler {
	t.Helper()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)

	getSpaceShowUC := usecase.NewGetSpaceShowUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo)

	return spacehandler.NewHandler(cfg, getSpaceShowUC)
}

func TestShow_存在しないスペースで404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/nonexistent", map[string]string{
		"space_identifier": "nonexistent",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_メンバーがピン留めと通常ページを閲覧できる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-pages-owner@example.com").
		WithAtname("sspagesowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-pages").
		WithName("Pages Space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("テストトピックラベル").
		WithVisibility(0).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("通常ページ").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("ピン留めページ").
		WithLinkedPageIDs([]model.PageID{}).
		WithPinnedAt(time.Now()).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ss-pages", map[string]string{
		"space_identifier": "ss-pages",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "sspagesowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Pages Space") {
		t.Error("response should contain the space name")
	}
	if !strings.Contains(body, "通常ページ") {
		t.Error("response should contain the regular page title")
	}
	if !strings.Contains(body, "ピン留めページ") {
		t.Error("response should contain the pinned page title")
	}
	// Cards show the topic label (pages span topics on the space detail).
	// [Ja] カードはトピックラベルを表示する (スペース詳細はページが複数トピックに跨る)。
	if !strings.Contains(body, "テストトピックラベル") {
		t.Error("response should contain the topic label on the cards")
	}
	// A space:admin member can edit, so the per-card edit links are rendered.
	// [Ja] space:admin メンバーは編集できるため、カードごとの編集リンクが描画される。
	if !strings.Contains(body, "/s/ss-pages/pages/1/edit") {
		t.Error("response should contain the edit link for the regular page")
	}
	if !strings.Contains(body, "/s/ss-pages/pages/2/edit") {
		t.Error("response should contain the edit link for the pinned page")
	}
}

func TestShow_ゲストは公開トピックのページのみ閲覧できる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-guest").
		WithName("Guest Space").
		Build()
	publicTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("公開トピック").
		WithVisibility(0). // public
		Build()
	privateTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("非公開トピック").
		WithVisibility(1). // private
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("公開ページ").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(2).
		WithTitle("非公開ページ").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ss-guest", map[string]string{
		"space_identifier": "ss-guest",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "公開ページ") {
		t.Error("response should contain the public-topic page title")
	}
	if strings.Contains(body, "非公開ページ") {
		t.Error("response should not contain the private-topic page title for a guest")
	}
	// The topic label is shown to guests too.
	// [Ja] トピックラベルはゲストにも表示される。
	if !strings.Contains(body, "公開トピック") {
		t.Error("response should contain the topic label for the public-topic card")
	}
	// A guest cannot edit, so no per-card edit link is rendered.
	// [Ja] ゲストは編集できないため、カードの編集リンクは描画されない。
	if strings.Contains(body, "/s/ss-guest/pages/1/edit") {
		t.Error("response should not contain an edit link for a guest")
	}
	// The topic section lists only public topics for a guest; the private topic must not appear.
	// [Ja] トピックセクションはゲストには公開トピックのみを並べ、非公開トピックは現れてはならない。
	if !strings.Contains(body, "/s/ss-guest/topics/1\"") {
		t.Error("response should contain the public topic detail link in the topic section")
	}
	if strings.Contains(body, "/s/ss-guest/topics/2") {
		t.Error("response should not contain the private topic detail link for a guest")
	}
	// A guest cannot create pages, so no per-topic new page action is rendered.
	// [Ja] ゲストはページを作成できないため、トピックごとの新規ページ作成アクションは描画されない。
	if strings.Contains(body, "/s/ss-guest/topics/1/pages/new") {
		t.Error("response should not contain a new page link for a guest")
	}
}

func TestShow_参加トピックが無いメンバーにトピック作成導線が出る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-notopic-owner@example.com").
		WithAtname("ssnotopicowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-notopic").
		Build()
	// space:admin scope (default) grants topic creation. [Ja] デフォルトの space:admin スコープでトピック作成が許可される。
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ss-notopic", map[string]string{
		"space_identifier": "ss-notopic",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "ssnotopicowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "トピックはありません") {
		t.Error("response should contain the no-topics empty state message")
	}
	if !strings.Contains(body, "新規トピック") {
		t.Error("response should contain the new topic button")
	}
	if !strings.Contains(body, "/s/ss-notopic/topics/new") {
		t.Error("response should contain the new topic form link")
	}
}

func TestShow_メンバーにトピックセクションと作成導線が表示され空状態のボタンは消える(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-nopage-owner@example.com").
		WithAtname("ssnopageowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-nopage").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(5).
		WithName("参加トピック").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ss-nopage", map[string]string{
		"space_identifier": "ss-nopage",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "ssnopageowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "ページはありません") {
		t.Error("response should contain the no-pages empty state message")
	}
	// The joined topic appears in the topic section, linking to its detail page.
	// [Ja] 参加トピックがトピックセクションに表示され、詳細ページへリンクする。
	if !strings.Contains(body, "参加トピック") {
		t.Error("response should contain the joined topic name in the topic section")
	}
	if !strings.Contains(body, "/s/ss-nopage/topics/5\"") {
		t.Error("response should contain the topic detail link in the topic section")
	}
	// The member may write, so the per-topic new page action appears in the section.
	// [Ja] メンバーは書き込めるため、トピックごとの新規ページ作成アクションがセクションに表示される。
	if !strings.Contains(body, "/s/ss-nopage/topics/5/pages/new") {
		t.Error("response should contain the per-topic new page link in the topic section")
	}
	// The space-level empty-state "new page" button is gone; its guidance text must not appear.
	// [Ja] スペースレベルの空状態「新規ページ」ボタンは消え、その説明文が表示されてはならない。
	if strings.Contains(body, "最初の1ページ目を作成しましょう") {
		t.Error("response should not contain the removed empty-state new page guidance")
	}
}

func TestShow_メンバーにスペースオプションメニューが全て表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-options-owner@example.com").
		WithAtname("ssoptionsowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-options").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ss-options", map[string]string{
		"space_identifier": "ss-options",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "ssoptionsowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	// The options dropdown shows the RSS feed plus the member-only topic / trash / settings links.
	// [Ja] オプションメニューには RSS フィードと、メンバー限定のトピック / ゴミ箱 / 設定リンクが表示される。
	for _, want := range []string{
		"/s/ss-options/atom",
		"/s/ss-options/topics/new",
		"/s/ss-options/trash",
		"/s/ss-options/settings",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response should contain the options menu link %q", want)
		}
	}
}

func TestShow_ゲストにはオプションメニューのRSSのみ表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-guest-options").
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ss-guest-options", map[string]string{
		"space_identifier": "ss-guest-options",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	// A guest sees only the RSS feed link; the member-only links must not appear.
	// [Ja] ゲストには RSS フィードリンクのみが見え、メンバー限定リンクは表示されない。
	if !strings.Contains(body, "/s/ss-guest-options/atom") {
		t.Error("response should contain the RSS feed link for a guest")
	}
	if strings.Contains(body, "/s/ss-guest-options/trash") {
		t.Error("response should not contain the trash link for a guest")
	}
	if strings.Contains(body, "/s/ss-guest-options/settings") {
		t.Error("response should not contain the settings link for a guest")
	}
}

func TestShow_ページネーションの次ページリンクが表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-paginate-owner@example.com").
		WithAtname("sspaginateowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-paginate").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()

	// Create 101 regular pages so the 100-per-page limit yields a second page.
	// [Ja] 1 ページ 100 件の上限を超える 101 件の通常ページを作成し、2 ページ目を発生させる。
	for i := int32(1); i <= 101; i++ {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(i)).
			WithTitle(fmt.Sprintf("ページ%d", i)).
			WithLinkedPageIDs([]model.PageID{}).
			Build()
	}

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ss-paginate", map[string]string{
		"space_identifier": "ss-paginate",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "sspaginateowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "/s/ss-paginate?page=2") {
		t.Error("response should contain the link to the next page")
	}
}
