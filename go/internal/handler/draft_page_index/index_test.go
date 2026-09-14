package draft_page_index_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/draft_page_index"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestIndex_Empty(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpi-empty@example.com").
		WithAtname("dpiempty").
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	draftPageRepo := repository.NewDraftPageRepository(queries)
	getDraftPagesUC := usecase.NewGetDraftPagesUsecase(draftPageRepo)

	handler := draft_page_index.NewHandler(cfg, getDraftPagesUC)

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "dpiempty"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "下書き") {
		t.Error("heading not found in response")
	}

	if !strings.Contains(body, "下書きはありません") {
		t.Error("empty message not found in response")
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

func TestIndex_WithDrafts(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpi-drafts@example.com").
		WithAtname("dpidrafts").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dpi-drafts-space").
		WithName("テストスペース").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("テストトピック").
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
		WithTitle("テストページ").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("下書きタイトル").
		WithBody("下書き本文").
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	draftPageRepo := repository.NewDraftPageRepository(queries)
	getDraftPagesUC := usecase.NewGetDraftPagesUsecase(draftPageRepo)

	handler := draft_page_index.NewHandler(cfg, getDraftPagesUC)

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "dpidrafts"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "テストスペース") {
		t.Error("space name not found in response")
	}

	if !strings.Contains(body, "テストトピック") {
		t.Error("topic name not found in response")
	}

	if !strings.Contains(body, "下書きタイトル") {
		t.Error("draft page title not found in response")
	}

	if !strings.Contains(body, "/s/dpi-drafts-space/pages/1/edit") {
		t.Error("edit page link not found in response")
	}
}

// Regression test verifying that a suggestion button is shown per topic group
// on the draft list screen and links to the correct creation screen path.
//
// [Ja] 下書き一覧画面でトピックグループごとに編集提案ボタンが表示され、
// 正しい作成画面のパスにリンクすることを検証する回帰テスト。
func TestIndex_編集提案ボタンがトピックグループに表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpi-sugg-btn@example.com").
		WithAtname("dpisuggbtn").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dpi-sugg-btn").
		WithName("提案ボタンスペース").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("提案ボタントピック").
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
		WithTitle("対象ページ").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("下書き").
		WithBody("本文").
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	draftPageRepo := repository.NewDraftPageRepository(queries)
	getDraftPagesUC := usecase.NewGetDraftPagesUsecase(draftPageRepo)

	handler := draft_page_index.NewHandler(cfg, getDraftPagesUC)

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "dpisuggbtn"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "新規編集提案") {
		t.Error("suggestion button not found in response")
	}
	if !strings.Contains(body, "/s/dpi-sugg-btn/topics/2/suggestions/new") {
		t.Error("suggestion new path not found in response")
	}
}

// The screen already drew its own name as the trail's last item, but without aria-current a screen
// reader had no way to tell which crumb the viewer was on. Scope the assertion to the breadcrumb
// because the same label also appears in the heading and the page title.
//
// [Ja] この画面はすでに自身の名前を経路の末尾に描いていたが、aria-current が無いと閲覧者がどの項目に
// いるかをスクリーンリーダーへ伝えられない。同じラベルは見出しとページタイトルにも出るため、
// パンくず内に絞って検証する。
func TestIndex_パンくずが現在地の項目で終わる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpi-crumb@example.com").
		WithAtname("dpicrumb").
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	handler := draft_page_index.NewHandler(cfg, usecase.NewGetDraftPagesUsecase(repository.NewDraftPageRepository(queries)))

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "dpicrumb"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	breadcrumb := draftPageIndexBreadcrumb(t, rr.Body.String())
	for _, want := range []string{
		`href="/home"`,
		`aria-current="page"`,
		"下書きのページ",
	} {
		if !strings.Contains(breadcrumb, want) {
			t.Errorf("breadcrumb does not contain %q", want)
		}
	}
	if strings.Contains(breadcrumb, `href="/drafts"`) {
		t.Error("current draft list breadcrumb item must not be a link")
	}
}

// draftPageIndexBreadcrumb returns the markup of the breadcrumb navigation alone, so that an
// assertion about the trail is not satisfied by the same text appearing elsewhere on the screen.
//
// [Ja] draftPageIndexBreadcrumb はパンくずのナビゲーション部分だけのマークアップを返す。経路に
// ついての検証が、画面の他の場所に出た同じ文字列で満たされてしまうのを防ぐ。
func draftPageIndexBreadcrumb(t *testing.T, body string) string {
	t.Helper()

	start := strings.Index(body, `<nav aria-label="パンくずリスト"`)
	if start == -1 {
		t.Fatal("response does not contain the breadcrumb navigation")
	}
	endOffset := strings.Index(body[start:], "</nav>")
	if endOffset == -1 {
		t.Fatal("breadcrumb navigation does not have a closing tag")
	}

	return body[start : start+endOffset]
}
