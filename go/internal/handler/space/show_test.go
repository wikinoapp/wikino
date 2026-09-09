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
	"github.com/wikinoapp/wikino/go/internal/i18n"
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

	// The breadcrumb header comes from the layout, so it renders outside <main> (the #main skip
	// link has to bypass it) and keeps this screen's max-w-3xl content width.
	//
	// [Ja] パンくずヘッダーはレイアウトが描画するため、<main> の外に出る (#main へのスキップ
	// リンクが飛ばせる必要があるため)。この画面の本文幅 max-w-3xl も維持する。
	if !strings.Contains(body, `<div class="max-w-3xl mx-auto flex w-full items-center justify-between gap-2 px-4">`) {
		t.Error("shared breadcrumb header should keep the max-w-3xl content width")
	}
	header, main := strings.Index(body, "<header"), strings.Index(body, `<main id="main" tabindex="-1">`)
	if header == -1 || main == -1 || header > main {
		t.Errorf("shared breadcrumb header (index %d) must precede <main> (index %d)", header, main)
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

// The space options dropdown trigger is icon-only, so its accessible name comes from the
// translated aria-label rather than from its content.
//
// [Ja] スペースオプションのドロップダウントリガーはアイコンのみのため、アクセシブルネームは
// 内容ではなく翻訳済みの aria-label が供給する。
func TestShow_スペースオプションのトリガーにアクセシブルネームがある(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		locale    string
		wantLabel string
	}{
		{
			name:      "日本語",
			locale:    i18n.LangJa,
			wantLabel: "スペースのオプション",
		},
		{
			name:      "英語",
			locale:    i18n.LangEn,
			wantLabel: "Space options",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)

			identifier := "ss-optlabel-" + tt.locale
			atname := "ssoptlabel" + tt.locale

			ownerID := testutil.NewUserBuilder(t, tx).
				WithEmail(identifier + "@example.com").
				WithAtname(atname).
				Build()
			spaceID := testutil.NewSpaceBuilder(t, tx).
				WithIdentifier(identifier).
				Build()
			testutil.NewSpaceMemberBuilder(t, tx).
				WithSpaceID(spaceID).
				WithUserID(ownerID).
				Build()

			handler := setupHandler(t, queries)

			req := newShowRequest(t, "/s/"+identifier, map[string]string{
				"space_identifier": identifier,
			})
			ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: atname})
			ctx = i18n.SetLocale(ctx, tt.locale)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
			}

			if !strings.Contains(rr.Body.String(), `aria-label="`+tt.wantLabel+`"`) {
				t.Errorf("スペースオプションのトリガーに aria-label %q が含まれていない", tt.wantLabel)
			}
		})
	}
}

// The stored identifier is what the canonical URL must point at. spaces.identifier is citext, so a
// request whose casing differs reaches the same screen and would otherwise declare a second
// canonical address for the same content.
//
// [Ja] 正規 URL が指すべきは保存済みの識別子である。spaces.identifier は citext のため大文字小文字が
// 違うリクエストでも同じ画面に到達し、そのままでは同じ内容に対して 2 つ目の正規アドレスを宣言して
// しまう。
func TestShow_CanonicalUsesStoredIdentifier(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-canonical").
		WithName("Canonical Space").
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/SS-CANONICAL", map[string]string{
		"space_identifier": "SS-CANONICAL",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, want := range []string{
		"<title>Canonical Space</title>",
		"<meta property=\"og:title\" content=\"Canonical Space\">",
		`<link rel="canonical" href="https://localhost/s/ss-canonical">`,
		`<meta property="og:url" content="https://localhost/s/ss-canonical">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q", want)
		}
	}
	if strings.Contains(body, "SS-CANONICAL") {
		t.Error("リクエストした表記がレスポンスに残っている")
	}
}

// Each page of the series carries different pages, so it declares itself rather than the first page
// as its canonical address.
//
// [Ja] 系列の各ページは載っているページが異なるため、1 ページ目ではなく自分自身を正規アドレスとして
// 宣言する。
func TestShow_PaginatedCanonicalPreservesPageParameter(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-canonical-page-owner@example.com").
		WithAtname("sscanonicalpageowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-canonical-page").
		WithName("Paginated Space").
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

	req := newShowRequest(t, "/s/ss-canonical-page?page=2", map[string]string{
		"space_identifier": "ss-canonical-page",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "sscanonicalpageowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	for _, want := range []string{
		"<title>Paginated Space (2 ページ目)</title>",
		"<meta property=\"og:title\" content=\"Paginated Space (2 ページ目)\">",
	} {
		if !strings.Contains(rr.Body.String(), want) {
			t.Errorf("response does not contain %q", want)
		}
	}

	if want := `<link rel="canonical" href="https://localhost/s/ss-canonical-page?page=2">`; !strings.Contains(rr.Body.String(), want) {
		t.Errorf("response does not contain %q", want)
	}
}

func TestShow_PageBeyondTotalReturnsNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-page-out-of-range").
		WithName("Out of Range Space").
		Build()

	handler := setupHandler(t, queries)

	// The last value is past int32: a positive page number too large for the offset names a page
	// that does not exist just as a smaller out-of-range one does, so it gets the same 404 instead of
	// falling back to the first page.
	//
	// [Ja] 最後の値は int32 を超える。offset に収まらない正のページ番号が指すのは、範囲内の範囲外値と
	// 同じく存在しないページのため、1 ページ目へフォールバックせず同じ 404 になる。
	for _, page := range []string{"2", "2147483647", "99999999999"} {
		t.Run("page="+page, func(t *testing.T) {
			req := newShowRequest(t, "/s/ss-page-out-of-range?page="+page, map[string]string{
				"space_identifier": "ss-page-out-of-range",
			})

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
			}

			body := rr.Body.String()
			if strings.Contains(body, "Out of Range Space") {
				t.Error("response should not contain the space name")
			}
			if strings.Contains(body, "/s/ss-page-out-of-range?page="+page) {
				t.Error("response should not contain a self-referencing canonical URL")
			}
		})
	}
}

// The space is the last item of its breadcrumb, so a signed-in viewer gets it marked as the current
// page under the /home parent used by the authenticated app. A signed-out viewer gets no /home
// crumb (matching Rails), which leaves the space itself as the only item: that trail has nothing to
// navigate to, so the breadcrumb is dropped rather than rendering a navigation landmark with no
// links. The signed-out variant publishes no BreadcrumbList structured data. The signed-in variant
// publishes the same home › space trail as its visible breadcrumb.
//
// [Ja] スペースはパンくずの末尾項目のため、ログイン済みの閲覧者には認証後アプリの親である /home の下で
// 現在ページとして伝える。未ログインの閲覧者には Rails 版と同じく /home 項目を出さないため、残るのは
// スペース自身だけになる。その経路にはたどれる項目が無いので、リンクの無いナビゲーションランドマークを
// 描画せずパンくずごと落とす。未ログイン時は BreadcrumbList 構造化データを出さない。ログイン時は
// 見た目のパンくずと同じホーム › スペースの経路を構造化データとして出す。
func TestShow_BreadcrumbMarksCurrentSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	viewerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-breadcrumb-viewer@example.com").
		WithAtname("ssbreadcrumbviewer").
		Build()
	testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-breadcrumb").
		WithName("Breadcrumb Space").
		Build()

	handler := setupHandler(t, queries)

	tests := []struct {
		name           string
		user           *model.User
		wantBreadcrumb bool
	}{
		{name: "signed-out viewer gets no breadcrumb at all", wantBreadcrumb: false},
		{name: "signed-in viewer gets the current space under home", user: &model.User{ID: viewerID, Atname: "ssbreadcrumbviewer"}, wantBreadcrumb: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newShowRequest(t, "/s/ss-breadcrumb", map[string]string{
				"space_identifier": "ss-breadcrumb",
			})
			if tt.user != nil {
				req = req.WithContext(middleware.SetUserToContext(req.Context(), tt.user))
			}

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()

			// Match the breadcrumb landmark by its own label: the global navigation bar is a <nav>
			// with an aria-label too.
			//
			// [Ja] パンくずのランドマークは専用のラベルで特定する。グローバルナビバーも aria-label 付きの
			// <nav> であるため。
			start := strings.Index(body, `<nav aria-label="パンくずリスト"`)
			if !tt.wantBreadcrumb {
				if start != -1 {
					t.Error("a trail holding only the current item should not render a breadcrumb")
				}
				if strings.Contains(body, `aria-current="page"`) {
					t.Error("no breadcrumb means no current item")
				}

				// The structured data mirrors the visible breadcrumb, so a trail that renders none
				// publishes none. This is the state a crawler sees.
				//
				// [Ja] 構造化データは見た目のパンくずを写したものなので、パンくずを描画しない経路では
				// 出さない。クローラーが見るのはこの状態である。
				if strings.Contains(body, "application/ld+json") {
					t.Error("a trail holding only the current item should not publish structured data")
				}
				return
			}

			if start == -1 {
				t.Fatal("response should contain the breadcrumb navigation")
			}
			end := strings.Index(body[start:], "</nav>")
			if end == -1 {
				t.Fatal("breadcrumb navigation should have a closing tag")
			}
			breadcrumb := body[start : start+end]
			if !strings.Contains(breadcrumb, `href="/home"`) {
				t.Error("signed-in breadcrumb should link back to /home")
			}
			if !strings.Contains(breadcrumb, `aria-current="page"`) {
				t.Error("current breadcrumb item must carry aria-current")
			}

			// The visible trail has something to navigate to here, so the machine-readable copy is
			// published from the same items: home as a link, the space as the non-linked current item.
			//
			// [Ja] ここでは見た目の経路にたどれる項目があるため、同じ項目列から機械可読な複製を出す。
			// ホームはリンク、スペースは非リンクの現在項目になる。
			for _, want := range []string{
				`"@type":"BreadcrumbList"`,
				`"position":1,"name":"ホーム","item":"https://localhost/home"`,
				`"position":2,"name":"Breadcrumb Space"}`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("response does not contain %q", want)
				}
			}
		})
	}
}
