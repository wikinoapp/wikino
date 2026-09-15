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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "下書き") {
		t.Error("レスポンスに見出しが見つからない")
	}

	if !strings.Contains(body, "下書きはありません") {
		t.Error("レスポンスに空状態のメッセージが見つからない")
	}

	// パンくずヘッダーはレイアウトが描画するため、<main> の外に出る (#mainへのスキップ
	// リンクが飛ばせる必要があるため)。この画面の本文幅max-w-3xlも維持する。
	if !strings.Contains(body, `<div class="max-w-3xl mx-auto flex w-full items-center justify-between gap-2 px-4">`) {
		t.Error("共通のパンくずヘッダーがmax-w-3xlのコンテンツ幅を保っていない")
	}
	header, main := strings.Index(body, "<header"), strings.Index(body, `<main id="main" tabindex="-1">`)
	if header == -1 || main == -1 || header > main {
		t.Errorf("共通のパンくずヘッダー (位置%d) が <main> (位置%d) より前にない", header, main)
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "テストスペース") {
		t.Error("レスポンスにスペース名が見つからない")
	}

	if !strings.Contains(body, "テストトピック") {
		t.Error("レスポンスにトピック名が見つからない")
	}

	if !strings.Contains(body, "下書きタイトル") {
		t.Error("レスポンスに下書きのタイトルが見つからない")
	}

	if !strings.Contains(body, "/s/dpi-drafts-space/pages/1/edit") {
		t.Error("レスポンスにページ編集のリンクが見つからない")
	}
}

// 下書き一覧画面でトピックグループごとに編集提案ボタンが表示され、
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
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "新規編集提案") {
		t.Error("レスポンスに編集提案のボタンが見つからない")
	}
	if !strings.Contains(body, "/s/dpi-sugg-btn/topics/2/suggestions/new") {
		t.Error("レスポンスに編集提案の作成パスが見つからない")
	}
}

// 行の操作メニューが横スクロールの祖先に切り取られることの回帰テスト。overflow-xを
// 持つ要素はoverflow-yもautoに計算されるため、テーブルを包む横スクロールが下端に近い行の
// メニューを切り落としていた。テーブルはタイトルのセルを折り返すことでどの幅にも収まり、
// 切り抜きを持つ祖先が1つも無くなる。
//
// タイトルのセルのclass属性は部分文字列ではなく値を丸ごと照合する。これにより、後から同じ
// 語で終わる別のクラスが増えても折り返しが落ちたことを捕まえられる。
func TestIndex_操作メニューを切り取る横スクロールがない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpi-no-scroller@example.com").
		WithAtname("dpinoscroller").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dpi-no-scroller").
		WithName("横スクロールスペース").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("横スクロールトピック").
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
	handler := draft_page_index.NewHandler(cfg, usecase.NewGetDraftPagesUsecase(repository.NewDraftPageRepository(queries)))

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "dpinoscroller"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if strings.Contains(body, "overflow-x-auto") {
		t.Error("下書き一覧に横スクロールの要素があり、行の操作メニューが切り取られる")
	}

	if !strings.Contains(body, `<td class="whitespace-normal [overflow-wrap:anywhere]">`) {
		t.Error("タイトルのセルが折り返さないため、テーブルが横スクロールを必要とする")
	}
}

// この画面はすでに自身の名前を経路の末尾に描いていたが、aria-currentが無いと閲覧者がどの項目に
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
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	breadcrumb := draftPageIndexBreadcrumb(t, rr.Body.String())
	for _, want := range []string{
		`href="/home"`,
		`aria-current="page"`,
		"下書きのページ",
	} {
		if !strings.Contains(breadcrumb, want) {
			t.Errorf("パンくずに%qが含まれていない", want)
		}
	}
	if strings.Contains(breadcrumb, `href="/drafts"`) {
		t.Error("現在の下書き一覧のパンくずの項目がリンクになっている")
	}
}

// draftPageIndexBreadcrumbはパンくずのナビゲーション部分だけのマークアップを返す。経路に
// ついての検証が、画面の他の場所に出た同じ文字列で満たされてしまうのを防ぐ。
func draftPageIndexBreadcrumb(t *testing.T, body string) string {
	t.Helper()

	start := strings.Index(body, `<nav aria-label="パンくずリスト"`)
	if start == -1 {
		t.Fatal("レスポンスにパンくずのナビゲーションが含まれていない")
	}
	endOffset := strings.Index(body[start:], "</nav>")
	if endOffset == -1 {
		t.Fatal("パンくずのナビゲーションに閉じタグが無い")
	}

	return body[start : start+endOffset]
}

// テーブルの見出しの関係とキャプションの回帰テスト。scopeの無い <th> は見出しの向きを
// ブラウザの推測に委ねることになり、キャプションの無いテーブルは、そこへ直接飛んだスクリーン
// リーダーへ、どのトピックの下書きなのかを伝えられない。
//
// キャプションを視覚的に隠すのは、カードの上のグループ見出しが画面上ではすでにトピック名を
// 示しているため。
func TestIndex_テーブルに見出しの関係とキャプションがある(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpi-caption@example.com").
		WithAtname("dpicaption").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dpi-caption").
		WithName("キャプションスペース").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("キャプショントピック").
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
	handler := draft_page_index.NewHandler(cfg, usecase.NewGetDraftPagesUsecase(repository.NewDraftPageRepository(queries)))

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "dpicaption"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, `<caption class="sr-only">キャプショントピック (キャプションスペース) の下書き一覧</caption>`) {
		t.Error("テーブルに、どのトピックの下書きかを伝える視覚的に隠したキャプションがない")
	}

	if got, want := strings.Count(body, `<th scope="col"`), 3; got != want {
		t.Errorf("scope=\"col\" を持つ見出しセルの数 = %d、期待値 = %d", got, want)
	}
}
