package page_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/page"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// setupHandlerはテスト用のハンドラーを生成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *page.Handler {
	t.Helper()

	return setupHandlerWithDB(t, nil, queries)
}

// setupHandlerWithDBは、自前でトランザクションを管理するUseCaseが必要とする *sql.DBを
// 渡してハンドラーを生成する。トランザクションに紐づくQueriesで駆動するテストはそれらの
// UseCaseに到達しないためnilを渡す。
func setupHandlerWithDB(t *testing.T, db *sql.DB, queries *query.Queries) *page.Handler {
	t.Helper()

	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(queries)
	pageRepo := repository.NewPageRepository(queries)

	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	suggestionRepo := repository.NewSuggestionRepository(queries)

	getPageDetailUC := usecase.NewGetPageDetailUsecase(
		spaceRepo,
		spaceMemberRepo,
		pageRepo,
		draftPageRepo,
		draftPageRevisionRepo,
		topicRepo,
		topicMemberRepo,
		suggestionPageRepo,
		suggestionRepo,
	)
	getEditLinkDataUC := usecase.NewGetEditLinkDataUsecase(pageRepo, topicRepo)

	pageRevisionRepo := repository.NewPageRevisionRepository(queries)
	pageEditorRepo := repository.NewPageEditorRepository(queries)
	attachmentRepo := repository.NewAttachmentRepository(queries)

	getPageShowUC := usecase.NewGetPageShowUsecase(
		spaceRepo,
		spaceMemberRepo,
		pageRepo,
		topicRepo,
		topicMemberRepo,
		attachmentRepo,
	)

	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(queries)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)

	publishPageUC := usecase.NewPublishPageUsecase(
		db,
		spaceRepo,
		spaceMemberRepo,
		pageRepo,
		pageRevisionRepo,
		pageEditorRepo,
		draftPageRepo,
		draftPageRevisionRepo,
		topicRepo,
		topicMemberRepo,
		attachmentRepo,
		pageAttachmentRefRepo,
		pageUpdateValidator,
	)

	createPageUC := usecase.NewCreatePageUsecase(
		db,
		spaceRepo,
		spaceMemberRepo,
		topicRepo,
		topicMemberRepo,
		pageRepo,
		pageEditorRepo,
		draftPageRepo,
		attachmentRepo,
	)

	return page.NewHandler(
		cfg,
		flashMgr,
		getPageShowUC,
		getPageDetailUC,
		getEditLinkDataUC,
		publishPageUC,
		createPageUC,
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

func TestEdit(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("my-space").
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
		WithTitle("Test Page Title").
		WithBody("Test page body content").
		Build()

	handler := setupHandler(t, queries)

	// リクエストを作成
	req := newRequestWithChiParams(t, http.MethodGet, "/s/my-space/pages/1/edit", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	})
	req.Header.Set("Accept-Language", "ja")

	// コンテキストを設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// フォームアクションが含まれているか確認
	if !strings.Contains(body, `/s/my-space/pages/1`) {
		t.Error("レスポンスにフォームの送信先が見つからない")
	}

	// CSRFトークンが含まれているか確認
	if !strings.Contains(body, "test-csrf-token") {
		t.Error("レスポンスにCSRFトークンが見つからない")
	}

	// 文書のタイトルがトピック配下の他の画面と同じく、トピック名をスペース名の前に挟むこと
	if !strings.Contains(body, "<title>ページを編集 | General | Test Space</title>") {
		t.Error("レスポンスにトピック名を含む文書のタイトルが見つからない")
	}

	// タイトルが表示されているか確認
	if !strings.Contains(body, "Test Page Title") {
		t.Error("レスポンスにページタイトルが見つからない")
	}

	// 本文が表示されているか確認
	if !strings.Contains(body, "Test page body content") {
		t.Error("レスポンスにページ本文が見つからない")
	}

	// _methodがPATCHであることを確認
	if !strings.Contains(body, `value="PATCH"`) {
		t.Error("レスポンスにPATCHのメソッドオーバーライドが見つからない")
	}

	// defaultレイアウトのコンテンツラッパーが、md未満で固定ナビに最下部コンテンツが隠れない
	// よう下部ナビの高さ + 下端safe-area分の下部余白を確保していること (PWAスタンドアロン表示で
	// ナビをホームインジケータの上へ押し上げても足りるようにする)。下部バーが描画されなくなる幅と
	// 揃えてmdで余白を外す。ラッパーに固定するため開始タグ全体で照合する。
	if !strings.Contains(body, `<div class="flex-1 flex flex-col min-h-screen pb-[calc(var(--app-bottom-nav-max-height)+0.5rem+env(safe-area-inset-bottom))] md:pb-0">`) {
		t.Error("レスポンスにコンテンツのラッパーの余白クラスが見つからない")
	}

	// 下部ナビの固定ラッパーがpb-safeを持ち、PWAスタンドアロン表示でナビのピルをホーム
	// インジケータの上へ押し上げること。ラッパーに固定するため開始タグ全体で照合する。
	if !strings.Contains(body, `<div class="fixed bottom-2 left-1/2 z-sticky-bar flex w-full -translate-x-1/2 flex-col items-center px-2 pb-safe">`) {
		t.Error("レスポンスに下部ナビゲーションの固定ラッパーのpb-safeクラスが見つからない")
	}

	// 日本語のラベルが含まれているか確認
	if !strings.Contains(body, "タイトル") {
		t.Error("レスポンスに日本語のタイトルのラベルが見つからない")
	}

	// 公開ボタンが含まれているか確認
	if !strings.Contains(body, "トピックに公開") {
		t.Error("レスポンスに日本語の公開ボタンが見つからない")
	}

	// キャンセルリンクが含まれているか確認
	if !strings.Contains(body, "/s/my-space/pages/1") {
		t.Error("レスポンスにキャンセルのリンクが見つからない")
	}

	// パンくずリストにトピック名が含まれているか確認
	if !strings.Contains(body, "General") {
		t.Error("パンくずにトピック名が見つからない")
	}

	// パンくずリストにスペースへのリンクが含まれているか確認
	if !strings.Contains(body, "/s/my-space") {
		t.Error("パンくずにスペースのリンクが見つからない")
	}

	// 下書きがない場合、下書きアラートが表示されないことを確認
	if strings.Contains(body, "現在下書きを表示しています") {
		t.Error("下書きが無いのに下書きのアラートが表示されている")
	}

	// メンバーに下書きが1件もないとき、下書き一覧カラムに空状態テキストが表示されること
	if !strings.Contains(body, "下書きはありません") {
		t.Error("下書きが無いときの下書き一覧の空状態テキストが見つからない")
	}

	// 下書きが無い (= リビジョンも無い) とき、編集履歴カラムに空状態テキストが表示されること
	if !strings.Contains(body, "下書きを保存すると、ここに編集履歴が表示されます") {
		t.Error("下書きが無いときの編集履歴の空状態テキストが見つからない")
	}

	// パンくずリストにトピックへのリンクが含まれているか確認
	if !strings.Contains(body, "/s/my-space/topics/1") {
		t.Error("パンくずにトピックのリンクが見つからない")
	}

	// トピックのアイコン (公開トピックのためglobe-regular) が表示されているか確認
	// globe-regularのSVGパスデータに含まれる固有の文字列で検証
	if !strings.Contains(body, "a87.61,87.61") {
		t.Error("パンくずにトピックの公開範囲アイコン (globe) が見つからない")
	}

	// パンくずヘッダーはレイアウトが描画するため、<main> の外に出る (#mainへのスキップ
	// リンクが飛ばせる必要があるため)。この画面の広い本文幅max-w-6xlも維持する。
	if !strings.Contains(body, `<div class="max-w-6xl mx-auto flex w-full items-center justify-between gap-2 px-4">`) {
		t.Error("共通のパンくずヘッダーがmax-w-6xlのコンテンツ幅を保っていない")
	}
	header, main := strings.Index(body, "<header"), strings.Index(body, `<main id="main" tabindex="-1">`)
	if header == -1 || main == -1 || header > main {
		t.Errorf("共通のパンくずヘッダー (位置%d) が <main> (位置%d) より前にない", header, main)
	}
}

// spaces.identifierはcitextのため、表記の異なるリクエストでも同じスペースに解決される。
// 画面内のリンクはすべて保存済みの識別子から組み立てるので、リクエストの表記がマークアップへ漏れず、
// 1画面のリンクの表記が1つに揃う。編集画面はリンク組み立ての種類が最も多い (パンくず・グローバル
// ナビの検索パス・下書き保存のエンドポイント)。
func TestEdit_BuildsLinksFromStoredIdentifier(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("stored-case-space").
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
		WithTitle("Stored Case Page").
		WithBody("body").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/STORED-CASE-SPACE/pages/1/edit", map[string]string{
		"space_identifier": "STORED-CASE-SPACE",
		"page_number":      "1",
	})
	req.Header.Set("Accept-Language", "ja")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if strings.Contains(body, "/s/STORED-CASE-SPACE") {
		t.Error("リンクがリクエストされたスペース識別子の表記のままになっている")
	}
	for _, want := range []string{
		`href="/s/stored-case-space"`,
		`content="/search?q=space:stored-case-space"`,
		`/s/stored-case-space/pages/1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
}

func TestEdit_WithDraftPage(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draft@example.com").
		WithAtname("draftuser").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithTitle("Original Title").
		WithBody("Original body").
		Build()

	// DraftPageを作成
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Title").
		WithBody("Draft body content").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/draft-space/pages/1/edit", map[string]string{
		"space_identifier": "draft-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// DraftPageの内容が表示されていることを確認
	if !strings.Contains(body, "Draft Title") {
		t.Error("レスポンスに下書きのタイトルが見つからない")
	}
	if !strings.Contains(body, "Draft body content") {
		t.Error("レスポンスに下書きの本文が見つからない")
	}

	// 元のページの内容が表示されていないことを確認
	if strings.Contains(body, "Original Title") {
		t.Error("下書きがあるのに元のタイトルが表示されている")
	}
	if strings.Contains(body, "Original body") {
		t.Error("下書きがあるのに元の本文が表示されている")
	}

	// 下書きアラートが表示されていることを確認
	if !strings.Contains(body, "現在下書きを表示しています") {
		t.Error("レスポンスに下書きのアラートメッセージが見つからない")
	}
}

func TestEdit_DraftListColumnAndNoGlobalSidebar(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draftcol@example.com").
		WithAtname("draftcoluser").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draftcol-space").
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

	// 編集対象のページ (page 1)
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Edited Page").
		WithBody("body").
		Build()

	// 同一スペース内の別の下書き (左カラムに一覧表示される)
	otherPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Other Page").
		WithBody("other body").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(otherPageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Sidebar Column Draft").
		WithBody("draft body").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/draftcol-space/pages/1/edit", map[string]string{
		"space_identifier": "draftcol-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// 左カラムに別の下書きが表示され、その編集画面へのリンクを持つこと
	if !strings.Contains(body, "Sidebar Column Draft") {
		t.Error("下書き一覧のカラムにもう一方の下書きが含まれていない")
	}
	if !strings.Contains(body, "/s/draftcol-space/pages/2/edit") {
		t.Error("もう一方のページのエディタへの下書きカードのリンクが見つからない")
	}

	// 下書き一覧の「すべて表示」リンクが見出しに表示されること
	if !strings.Contains(body, "すべて表示") {
		t.Error("下書き一覧のすべて表示リンクが見つからない")
	}

	// 狭幅時のドロワー (本体と開閉ボタン) が結線されていること
	if !strings.Contains(body, `id="page-edit-draft-pages-drawer"`) {
		t.Error("下書き一覧のドロワーが見つからない")
	}
	if !strings.Contains(body, `data-drawer-open="page-edit-draft-pages-drawer"`) {
		t.Error("下書き一覧のドロワーを開くボタンが見つからない")
	}

	// 廃止したサイドバーの痕跡が残っていないこと。off-canvasのサイドバー要素・BreadcrumbHeaderの
	// 開閉ボタン・サイドバーを開くdispatchのいずれも無い。画面内の下書きナビゲーションは
	// サイドバーではなく左カラムが担う。
	if strings.Contains(body, `id="sidebar"`) {
		t.Error("ページエディタにグローバルサイドバーが描画されている")
	}
	if strings.Contains(body, "サイドバーの開閉") {
		t.Error("ページエディタにBreadcrumbHeaderのサイドバー切り替えボタンが描画されている")
	}
	if strings.Contains(body, "basecoat:sidebar") {
		t.Error("ページエディタにサイドバーを開くボタンが描画されている")
	}

	// 編集画面も他のページと同様にグローバルナビ (上部バー + 下部バー) へ結線されていること
	if !strings.Contains(body, `aria-label="グローバルナビゲーション"`) {
		t.Error("ページエディタにグローバルナビゲーションの上部バーが描画されていない")
	}

	// 編集画面も他の画面と同じくmdで切り替わる。上部バーはmd以上で現れ、下部バーはそれ未満で
	// 残る。各 <nav> のclass属性全体で照合し、ナビのラッパーに固定する。
	if !strings.Contains(body, `<nav class="shrink-0 hidden md:flex"`) {
		t.Error("ページエディタでグローバルナビゲーションの上部バーがmdで切り替わっていない")
	}
	if !strings.Contains(body, `<nav class="md:hidden"`) {
		t.Error("ページエディタでグローバルナビゲーションの下部バーがmdで切り替わっていない")
	}
}

func TestEdit_RevisionColumn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("revcol@example.com").
		WithAtname("revcoluser").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("revcol-space").
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
		WithTitle("Edited Page").
		WithBody("body").
		Build()
	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft With Revisions").
		WithBody("draft body").
		Build()

	// 保存済みリビジョン2件: カラムにv1とv2が表示されること
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(queries)
	for _, title := range []string{"Rev One", "Rev Two"} {
		_, err := draftPageRevisionRepo.Create(context.Background(), repository.CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         title,
			Body:          "body of " + title,
			BodyHTML:      "<p>body of " + title + "</p>",
		})
		if err != nil {
			t.Fatalf("Create() (revision) のエラー = %v", err)
		}
	}

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/revcol-space/pages/1/edit", map[string]string{
		"space_identifier": "revcol-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// 編集履歴カラムの見出しが表示されること
	if !strings.Contains(body, "編集履歴") {
		t.Error("編集履歴の見出しが見つからない")
	}

	// 総件数から算出したバージョン番号が表示されること (最古 = v1)
	if !strings.Contains(body, "v2") {
		t.Error("編集履歴のカラムにバージョンv2が見つからない")
	}
	if !strings.Contains(body, "v1") {
		t.Error("編集履歴のカラムにバージョンv1が見つからない")
	}

	// 最新リビジョンに「現在」バッジが付くこと
	if !strings.Contains(body, "現在") {
		t.Error("編集履歴のカラムに現在のバッジが見つからない")
	}

	// 狭幅時のドロワー (本体と開閉ボタン) が結線されていること
	if !strings.Contains(body, `id="page-edit-draft-revisions-drawer"`) {
		t.Error("編集履歴のドロワーが見つからない")
	}
	if !strings.Contains(body, `data-drawer-open="page-edit-draft-revisions-drawer"`) {
		t.Error("編集履歴のドロワーを開くボタンが見つからない")
	}

	// 手動保存レスポンスのOOBスワップターゲットが、カラム2箇所を別々のidで
	// 包んでいること
	if !strings.Contains(body, `id="page-revision-list"`) {
		t.Error("静的なリビジョン一覧のOOBの対象が見つからない")
	}
	if !strings.Contains(body, `id="page-revision-list-drawer"`) {
		t.Error("ドロワーのリビジョン一覧のOOBの対象が見つからない")
	}

	// 「下書き保存」ボタンがhtmxのPATCHで送信され、保存で画面遷移しないこと
	if !strings.Contains(body, `id="page-edit-save-draft-button"`) {
		t.Error("下書き保存ボタンが見つからない")
	}
	if !strings.Contains(body, `hx-patch="/s/revcol-space/pages/1/draft_page_revision"`) {
		t.Error("下書き保存ボタンのhx-patch属性が見つからない")
	}

	// リビジョンが存在するときは空状態テキストが表示されないこと
	if strings.Contains(body, "下書きを保存すると、ここに編集履歴が表示されます") {
		t.Error("リビジョンがあるのに編集履歴の空状態が表示されている")
	}
}

func TestEdit_AutofocusTitle(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// タイトルなしのページを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("autofocus@example.com").
		WithAtname("autofocususer").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("autofocus-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNilTitle().
		WithBody("Body without title").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/autofocus-space/pages/1/edit", map[string]string{
		"space_identifier": "autofocus-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// タイトルが空のとき、タイトル入力欄にautofocusが設定されていることを確認
	if !strings.Contains(body, `id="page_title"`) {
		t.Error("ページタイトルの入力欄が見つからない")
	}
	if !strings.Contains(body, "autofocus") {
		t.Error("page_titleの入力欄にautofocus属性が見つからない")
	}
}

func TestEdit_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/my-space/pages/1/edit", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	// ログインページへリダイレクトされることを確認
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}
	location := rr.Header().Get("Location")
	if location != "/sign_in" {
		t.Errorf("リダイレクト先 = %v、期待値 = /sign_in", location)
	}
}

func TestEdit_SpaceNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("space-not-found@example.com").
		WithAtname("spacenotfound").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/nonexistent/pages/1/edit", map[string]string{
		"space_identifier": "nonexistent",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestEdit_NotSpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// スペースを作成するが、別のユーザーでアクセス
	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("owner@example.com").
		WithAtname("owner").
		Build()
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("outsider@example.com").
		WithAtname("outsider").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("private-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/private-space/pages/1/edit", map[string]string{
		"space_identifier": "private-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: outsiderID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestEdit_PageNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("page-not-found@example.com").
		WithAtname("pagenotfound").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-missing-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/page-missing-space/pages/999/edit", map[string]string{
		"space_identifier": "page-missing-space",
		"page_number":      "999",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestEdit_InvalidPageNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("invalid-num@example.com").
		WithAtname("invalidnum").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/my-space/pages/abc/edit", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "abc",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

// TestEdit_RelatedPagePaginationは編集画面固有のフルページフォールバックを固定する。リンク一覧は
// メンバーの下書きを参照し、3一覧すべてが指定スライスへ到達し、不正・失効・別スライスを指す状態は
// HTTP境界で拒否される。
func TestEdit_RelatedPagePagination(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("edit-related-pagination@example.com").
		WithAtname("editrelatedpagination").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("edit-related-space").
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

	baseTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	linkedCount := int(viewmodel.LinkLimit + 10*viewmodel.RelatedPageFollowingLimit + 1)
	linkedPageIDs := make([]model.PageID, 0, linkedCount)
	for i := range linkedCount {
		linkedPageIDs = append(linkedPageIDs, testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(100+i)).
			WithTitle(fmt.Sprintf("Draft Linked Page %02d", i)).
			WithModifiedAt(baseTime.Add(time.Duration(i)*time.Hour)).
			WithLinkedPageIDs([]model.PageID{}).
			Build())
	}

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Stored Editor Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Editor Page").
		WithLinkedPageIDs(linkedPageIDs).
		Build()

	selectedLinkedPageIndex := linkedCount - 1 - int(viewmodel.LinkLimit)
	selectedLinkedPageID := linkedPageIDs[selectedLinkedPageIndex]
	selectedLinkedPageNumber := model.PageNumber(100 + selectedLinkedPageIndex)
	for i := range int(viewmodel.BacklinkLimit + viewmodel.RelatedPageFollowingLimit + 1) {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(1000 + i)).
			WithTitle(fmt.Sprintf("Nested Backlink %02d", i)).
			WithModifiedAt(baseTime.Add(time.Duration(100+i) * time.Hour)).
			WithLinkedPageIDs([]model.PageID{selectedLinkedPageID}).
			Build()
	}
	for i := range int(viewmodel.PageBacklinkLimit + viewmodel.RelatedPageFollowingLimit + 1) {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(2000 + i)).
			WithTitle(fmt.Sprintf("Page Backlink %02d", i)).
			WithModifiedAt(baseTime.Add(time.Duration(200+i) * time.Hour)).
			WithLinkedPageIDs([]model.PageID{pageID}).
			Build()
	}

	handler := setupHandler(t, queries)
	tests := []struct {
		name         string
		rawQuery     string
		wantStatus   int
		wantContains []string
	}{
		{
			name: "下書き由来の3一覧で2ページ目と次リンクを描画する",
			rawQuery: fmt.Sprintf(
				"links_page=2&linked_page_number=%d&linked_backlinks_page=2&backlinks_page=2",
				selectedLinkedPageNumber,
			),
			wantStatus: http.StatusOK,
			wantContains: []string{
				fmt.Sprintf("Draft Linked Page %02d", selectedLinkedPageIndex),
				fmt.Sprintf("Nested Backlink %02d", viewmodel.BacklinkLimit),
				fmt.Sprintf("Page Backlink %02d", viewmodel.PageBacklinkLimit),
				`href="/s/edit-related-space/pages/1/edit?backlinks_page=2&amp;links_page=3#page-link-list-content"`,
				fmt.Sprintf(`href="/s/edit-related-space/pages/1/edit?backlinks_page=2&amp;linked_backlinks_page=3&amp;linked_page_number=%[1]d&amp;links_page=2#page-link-list-item-%[1]d"`, selectedLinkedPageNumber),
				fmt.Sprintf(`href="/s/edit-related-space/pages/1/edit?backlinks_page=3&amp;linked_backlinks_page=2&amp;linked_page_number=%d&amp;links_page=2#page-backlink-list-content"`, selectedLinkedPageNumber),
			},
		},
		{
			name:       "1ページ単位の編集画面で下書き固有の11ページ目と次リンクを描画する",
			rawQuery:   "context=edit_paginated&links_page=11",
			wantStatus: http.StatusOK,
			wantContains: []string{
				fmt.Sprintf("Draft Linked Page %02d", linkedCount-1-int(viewmodel.LinkLimit+9*viewmodel.RelatedPageFollowingLimit)),
				`name="context" value="edit_paginated"`,
				`hx-get="/s/edit-related-space/pages/1/link_list?context=edit_paginated&amp;page=12"`,
				`hx-target="#page-link-list-content"`,
				`href="/s/edit-related-space/pages/1/edit?context=edit_paginated&amp;links_page=12#page-link-list-content"`,
			},
		},
		{
			name:       "数値でない対象カード番号は404",
			rawQuery:   "linked_page_number=abc",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "総ページ数を超えた一覧ページは404",
			rawQuery:   "links_page=999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "累積取得上限を超えた一覧ページは404",
			rawQuery:   fmt.Sprintf("links_page=%d", usecase.MaxCumulativeRelatedPagePages+1),
			wantStatus: http.StatusNotFound,
		},
		{
			name: "描画中の親ページにないリンク先カードは404",
			rawQuery: fmt.Sprintf(
				"links_page=2&linked_page_number=%d&linked_backlinks_page=2",
				100+linkedCount-1,
			),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequestWithChiParams(t, http.MethodGet, "/s/edit-related-space/pages/1/edit", map[string]string{
				"space_identifier": "edit-related-space",
				"page_number":      "1",
			})
			req.URL.RawQuery = tt.rawQuery
			ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
			ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
			ctx = i18n.SetLocale(ctx, i18n.LangJa)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.Edit(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, tt.wantStatus)
			}
			body := rr.Body.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}
		})
	}
}

func TestEdit_LinkListAutoReload(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("linklist-reload@example.com").
		WithAtname("linklistreload").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("linklist-reload-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
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
		WithTitle("Link Reload Test").
		WithBody("Some content").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/linklist-reload-space/pages/1/edit", map[string]string{
		"space_identifier": "linklist-reload-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// リンク一覧セクションのコンテナが存在すること
	if !strings.Contains(body, `id="page-link-list"`) {
		t.Error("レスポンスにpage-link-listのコンテナが見つからない")
	}

	// 専用の再取得トリガーは、2つの一覧コンテナのOOB内容スワップ後も残る。
	if !strings.Contains(body, `id="page-draft-refresh-trigger"`) {
		t.Error("下書きの再読み込み専用のトリガーが見つからない")
	}

	// hx-triggerは下書き保存後、OOB応答を通じて一覧を再取得する。
	if !strings.Contains(body, `hx-trigger="draft-autosaved from:window"`) {
		t.Error("hx-trigger属性が見つからない (リンク一覧の自動再読み込みが動かない)")
	}

	// hx-getは下書きフラグメントエンドポイントを指す。
	if !strings.Contains(body, "/s/linklist-reload-space/pages/1/draft_page") {
		t.Error("レスポンスにdraft_pageのエンドポイントのURLが見つからない")
	}

	// hx-swap=noneによりメインターゲットは変更せず、OOBスワップだけを適用する。
	if !strings.Contains(body, `hx-swap="none"`) {
		t.Error("hx-swap=noneが見つからない (OOBスワップが正しく動かない)")
	}

	// 共有の関連ページ状態は初回描画から、かつ2つの一覧コンテナの外に置かれる。最初の
	// 「もっと見る」の時点で他の一覧を読む先があり、内容スワップで巻き添えに消えることもない。
	if !strings.Contains(body, `id="page-related-page-state"`) {
		t.Error("共有の関連ページの状態が見つからない")
	}
	if !strings.Contains(body, `hx-include="#page-related-page-state"`) {
		t.Error("下書きの再読み込みが共有の関連ページの状態を読んでいない")
	}

	// 見出し (h2) は #page-link-listの外 (OOBスワップ対象外) にあり、常にレンダリングされる。
	// リンクが無いこのケースでもDOM上には存在し、CSSのnot-hasでセクションごと非表示になる。
	if !strings.Contains(body, `<h2 class="font-bold antialiased">`) {
		t.Error("リンク見出しのh2が見つからない (呼び出し側でのレンダリングが壊れている可能性)")
	}

	// セクションはnot-hasバリアントでリストが空のとき非表示になる。
	// このCSSフックにより、見出しを再描画せずに表示・非表示が内容に追従する。
	if !strings.Contains(body, `not-has-[#page-link-list>*]:hidden`) {
		t.Error("リンクセクションの表示制御クラスnot-has-[#page-link-list>*]:hiddenが見つからない")
	}
	if !strings.Contains(body, `not-has-[#page-backlink-list>*]:hidden`) {
		t.Error("バックリンクセクションの表示制御クラスnot-has-[#page-backlink-list>*]:hiddenが見つからない")
	}
}

func TestEdit_PreviewTab(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("preview-tab@example.com").
		WithAtname("previewtab").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("preview-tab-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
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
		WithTitle("Preview Tab Page").
		WithBody("body").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/preview-tab-space/pages/1/edit", map[string]string{
		"space_identifier": "preview-tab-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// タブのコンテナと編集/プレビューの両タブが描画されること
	if !strings.Contains(body, `id="page-edit-tabs"`) {
		t.Error("レスポンスにタブのコンテナが見つからない")
	}

	// タブコンテナがmin-w-0を持ち、プレビューの <pre> のoverflow-autoが長いコード行を
	// 収める (gridアイテムが中央カラムを超えて広がらない) こと。開始タグ全体で照合し、min-w-0が
	// このコンテナに付いていることを担保する (min-w-0は他の要素にも現れる)。
	if !strings.Contains(body, `<div class="tabs grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-2 gap-y-4 min-w-0" id="page-edit-tabs">`) {
		t.Error("タブのコンテナにmin-w-0が無い (プレビューのコードブロックのはみ出しを防ぐため)")
	}

	// Basecoat 1.0はタブリストを .tabsの直接の子としてのみスタイルするため、リストを
	// 別の要素で包んではいけない。
	if !strings.Contains(body, `id="page-edit-tabs"><nav`) {
		t.Error("タブの一覧がタブのコンテナの直接の子になっていない")
	}
	if !strings.Contains(body, `id="page-edit-tab-edit"`) {
		t.Error("レスポンスに編集タブが見つからない")
	}
	if !strings.Contains(body, `id="page-edit-tab-preview"`) {
		t.Error("レスポンスにプレビュータブが見つからない")
	}

	// プレビュータブがhtmxでフォームの現在値をプレビューエンドポイントにPOSTすること
	if !strings.Contains(body, `hx-post="/s/preview-tab-space/pages/1/preview"`) {
		t.Error("レスポンスにプレビューのエンドポイントへのプレビュータブのhx-postが見つからない")
	}

	// プレビュータブ発火時、htmx 4は編集フォーム全体 (hiddenの _method=PATCHを含む) を
	// 収集するため、ボタンはhx-valsで _methodを空にする。overrideミドルウェアは空の _methodを
	// 無視するので、プレビュー要求はPATCHに書き換えられずPOSTのままになる。レンダリングされる
	// 二重引用符は &#34; にHTMLエスケープされる (ブラウザがデコードして元に戻す)。
	if !strings.Contains(body, `hx-vals="{&#34;_method&#34;: &#34;&#34;}"`) {
		t.Error("レスポンスに_methodを空にするプレビュータブのhx-valsが見つからない")
	}
	// CSRFトークン入力はフォーム内に残るため、自動収集されるプレビューPOSTに含まれ、
	// CSRFミドルウェアを通過する。
	if !strings.Contains(body, `id="page-edit-csrf-token"`) {
		t.Error("レスポンスにCSRFトークンの入力欄が見つからない")
	}

	// 結果はプレビューパネルにスワップされ、ローディング表示が用意されていること
	if !strings.Contains(body, `hx-target="#page-edit-preview-content"`) {
		t.Error("レスポンスにプレビュータブのhx-targetが見つからない")
	}
	if !strings.Contains(body, `id="page-edit-preview-content"`) {
		t.Error("レスポンスにプレビューのコンテンツパネルが見つからない")
	}
	if !strings.Contains(body, `id="page-edit-preview-loading"`) {
		t.Error("レスポンスにプレビューの読み込みインジケーターが見つからない")
	}

	// タブのラベルが翻訳されていること。プレビューのアサーションがローディングテキスト
	// ("プレビューを生成中...") で成立しないよう、各ラベルを閉じ </button> タグ隣接で検証する。
	if !strings.Contains(body, "編集</button>") {
		t.Error("レスポンスに編集タブのラベルが見つからない")
	}
	if !strings.Contains(body, "プレビュー</button>") {
		t.Error("レスポンスにプレビュータブのラベルが見つからない")
	}
}

func TestEdit_KeyboardHint(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("keyboard-hint@example.com").
		WithAtname("keyboardhint").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("keyboard-hint-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
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
		WithTitle("Keyboard Hint Page").
		WithBody("body").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/keyboard-hint-space/pages/1/edit", map[string]string{
		"space_identifier": "keyboard-hint-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// ⌘ (Mac) とCtrl+ (それ以外) の両版が描画され、CSSがhtml[data-os] で切り替えられること。
	// Macの修飾キーは区切り無しでキーに詰めてチップを細く保ち、非Macの修飾キーは "Ctrl+" で
	// キーの前に "+" が入る。公開ボタンはMod-Enterを修飾キーの直後の折り返し矢印アイコンで、
	// 保存ボタンはMod-sをテキストで表記すること。
	if !strings.Contains(body, `>⌘<svg class="size-3.5"`) {
		t.Error("レスポンスに公開ボタンのMac向けショートカットヒント (⌘ + リターン矢印アイコン) が見つからない")
	}
	if !strings.Contains(body, `>Ctrl+<svg class="size-3.5"`) {
		t.Error("レスポンスに公開ボタンのMac以外向けショートカットヒント (Ctrl+とリターン矢印アイコン) が見つからない")
	}
	// 折り返しキーは小さな ↵ グリフではなくarrow-elbow-down-leftアイコン (パス先頭の
	// move/vertical-lineコマンド) で描画すること。
	if !strings.Contains(body, "M200,32V176") {
		t.Error("公開ボタンのショートカットヒントがarrow-elbow-down-leftアイコンを使っていない")
	}
	if !strings.Contains(body, ">⌘S</kbd>") {
		t.Error("レスポンスに保存ボタンのMac向けショートカットヒント (⌘S) が見つからない")
	}
	if !strings.Contains(body, ">Ctrl+S</kbd>") {
		t.Error("レスポンスに保存ボタンのMac以外向けショートカットヒント (Ctrl+S) が見つからない")
	}

	// 表記チップはタッチ以外の端末で、プラットフォーム属性バリアントによりOSで出し分けること。
	if !strings.Contains(body, "non-touch:in-[[data-os=mac]]:inline-flex") {
		t.Error("レスポンスにMac向けキーボードヒントの表示制御クラスが見つからない")
	}
	if !strings.Contains(body, "non-touch:in-[[data-os=other]]:inline-flex") {
		t.Error("レスポンスにMac以外向けキーボードヒントの表示制御クラスが見つからない")
	}

	// 各チップはホストボタンの前景色で淡く染める (前景色の文字 + 低不透明度の前景色背景)。
	// 公開は既定のprimaryバリアントの .btnを、保存はsecondaryバリアントを染めること。
	if !strings.Contains(body, "bg-primary-foreground/20 text-primary-foreground") {
		t.Error("レスポンスに公開のキーボードヒントの色付きクラス (bg-primary-foreground/20 text-primary-foreground) が見つからない")
	}
	if !strings.Contains(body, "bg-secondary-foreground/8 text-secondary-foreground") {
		t.Error("レスポンスに保存のキーボードヒントの色付きクラス (bg-secondary-foreground/8 text-secondary-foreground) が見つからない")
	}

	// チップは装飾的なため、スクリーンリーダーはボタンラベルのみを読み上げること。
	if !strings.Contains(body, `<kbd class="kbd hidden non-touch:in-[[data-os=mac]]:inline-flex bg-secondary-foreground/8 text-secondary-foreground" aria-hidden="true">`) {
		t.Error("レスポンスにaria-hidden付きのキーボードヒントのkbdチップが見つからない")
	}
}

func TestEdit_ActionRowLayout(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("action-row@example.com").
		WithAtname("actionrow").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("action-row-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
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
		WithTitle("Action Row Page").
		WithBody("body").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/action-row-space/pages/1/edit", map[string]string{
		"space_identifier": "action-row-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// 操作行は広い画面では1行、1152pxのコンテナ幅未満では2段に折り返すこと (この幅未満では
	// 中央カラムが狭く1行に収まらないため)。
	if !strings.Contains(body, `<div class="flex flex-col gap-1 min-[1152px]:flex-row min-[1152px]:items-center">`) {
		t.Error("レスポンスに操作行のレスポンシブレイアウトのコンテナが見つからない")
	}

	// 狭い非タッチ環境では、ボタンのラベルとキーボードヒントを合わせた幅がカードの
	// 内容幅を超えるため、ボタン行は折り返せる必要がある。class属性全体を外側の段と併せて
	// 照合し、この行からflex-wrapが外れた場合に検出できるようにする。
	wantWrappingButtonRow := `<div class="flex flex-col gap-1 min-[1152px]:flex-row min-[1152px]:items-center">` +
		`<div class="flex flex-wrap gap-x-2 gap-y-1">`
	if !strings.Contains(body, wantWrappingButtonRow) {
		t.Errorf("レスポンスに折り返す操作行%qが含まれていない", wantWrappingButtonRow)
	}

	// 保存時刻表示とキャンセルは1つのコンテナに反転させず (保存時刻→キャンセル順) 並べる。
	// 2段時は右揃えでまとめ (justify-end)、1行時は左右へ振り分ける (min-[1152px]:justify-between)。
	// 時刻は折り返さないこと。
	if !strings.Contains(body, `<div class="flex items-center justify-end gap-2 min-[1152px]:flex-1 min-[1152px]:justify-between">`) {
		t.Error("レスポンスに保存日時とキャンセルのコンテナ (反転なし、2段で右寄せ、レスポンシブ) が見つからない")
	}
	if !strings.Contains(body, `<div id="page-draft-saved-at" class="text-xs text-muted-foreground whitespace-nowrap">`) {
		t.Error("レスポンスにwhitespace-nowrap付きの保存日時の表示が見つからない")
	}
	if strings.Contains(body, "flex-row-reverse") {
		t.Error("操作行で保存日時とキャンセルの順序が反転している (flex-row-reverseがある)")
	}
}

func TestEdit_ZenMode(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("zen-mode@example.com").
		WithAtname("zenmode").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("zen-mode-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
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
		WithTitle("Zen Mode Page").
		WithBody("body").
		Build()

	handler := setupHandler(t, queries)

	// "page-edit-zen" の部分一致では常に存在するTailwindバリアントクラス
	// (in-[.page-edit-zen]:lg:hiddenなど) にもマッチしてしまうため、コンテナのclass属性全体で
	// 検証する。
	const containerClassOff = `class="max-w-6xl w-full mx-auto lg:px-4"`
	const containerClassOn = `class="max-w-6xl w-full mx-auto lg:px-4 page-edit-zen"`

	tests := []struct {
		name        string
		cookieValue string // 空はクッキーなし
		wantZen     bool
	}{
		{
			name:        "正常系: クッキーなしの場合は通常モードで表示する",
			cookieValue: "",
			wantZen:     false,
		},
		{
			name:        "正常系: クッキーが1の場合はZenモードで表示する",
			cookieValue: "1",
			wantZen:     true,
		},
		{
			name:        "正常系: クッキーが1以外の値の場合は通常モードで表示する",
			cookieValue: "0",
			wantZen:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequestWithChiParams(t, http.MethodGet, "/s/zen-mode-space/pages/1/edit", map[string]string{
				"space_identifier": "zen-mode-space",
				"page_number":      "1",
			})
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: "wikino_zen_mode", Value: tt.cookieValue})
			}
			ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
			ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
			ctx = i18n.SetLocale(ctx, i18n.LangJa)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.Edit(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()

			// トグルボタンがラベルとサーバー描画の押下状態付きで描画されること
			if !strings.Contains(body, "data-zen-mode-toggle") {
				t.Error("レスポンスにZenモードの切り替えボタンが見つからない")
			}
			if !strings.Contains(body, "Zenモード") {
				t.Error("レスポンスにZenモードのボタンのラベルが見つからない")
			}

			if tt.wantZen {
				if !strings.Contains(body, containerClassOn) {
					t.Error("エディタのコンテナにZenモードのクラスが見つからない")
				}
				if !strings.Contains(body, `aria-pressed="true"`) {
					t.Error(`Zenモードの切り替えにaria-pressed="true"が見つからない`)
				}
			} else {
				if !strings.Contains(body, containerClassOff) {
					t.Error("Zenモードのクラスが無いエディタのコンテナが見つからない")
				}
				if strings.Contains(body, containerClassOn) {
					t.Error("エディタのコンテナにZenモードのクラスが付いている")
				}
				if !strings.Contains(body, `aria-pressed="false"`) {
					t.Error(`Zenモードの切り替えにaria-pressed="false"が見つからない`)
				}
			}

			// Zenモードはデスクトップ (lg以上) のレイアウトだけを変えるため、Zenバリアントは
			// すべてlg限定になる (左右カラム・リンク / バックリンク一覧の非表示、グリッド解除、
			// 中央カラムの拡幅はいずれもlgでのみ効く)。モバイルには畳む対象のサイドカラムが無い。

			// 左右サイドカラムはZen非表示をlgでラップする。同じin-[.page-edit-zen]:lg:hidden
			// バリアントを持つようになったリンク / バックリンクセクションではなく、常にlg非表示の
			// サイドカラムに固定するため、class属性全体で検証する。
			if !strings.Contains(body, `class="hidden lg:block in-[.page-edit-zen]:lg:hidden"`) {
				t.Error("レスポンスにZenモードのサイドカラムを隠すクラスが見つからない")
			}
			// リンク / バックリンク一覧セクションはZen非表示をlgでラップし、Zenクッキーが
			// 設定されていてもモバイルでは一覧を表示したままにする (常にlg非表示のサイドカラムでは
			// なくリンクセクションに固定するため、class属性全体で検証する)。
			if !strings.Contains(body, `class="flex flex-col gap-4 px-4 in-[.page-edit-zen]:lg:hidden"`) {
				t.Error("レスポンスにZenモードのリンク一覧のlg限定の非表示クラスが見つからない")
			}
			if !strings.Contains(body, "in-[.page-edit-zen]:lg:block") {
				t.Error("レスポンスにZenモードのグリッドを畳むクラスが見つからない")
			}
			if !strings.Contains(body, "in-[.page-edit-zen]:lg:max-w-4xl") {
				t.Error("レスポンスにZenモードの中央カラムを広げるクラスが見つからない")
			}
			// Zenトグルはlg未満で非表示にし (モバイルには畳むサイドカラムが無く、Zen ONの
			// ままトグルが無いとモバイルのユーザーが戻せなくなる)、lgでinline-flexに戻す。
			// トグルボタンに固定するためclass属性全体で検証する。
			if !strings.Contains(body, `class="hidden lg:inline-flex btn rounded-full w-fit justify-self-end"`) {
				t.Error("レスポンスにZenモードの切り替えのlg限定の表示クラスが見つからない")
			}
			if !strings.Contains(body, `data-variant="outline" data-size="sm" data-zen-mode-toggle`) {
				t.Error("レスポンスにZenモードの切り替えのBasecoatのvariant属性とsize属性が見つからない")
			}
		})
	}
}

func TestEdit_EnglishLocale(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("english@example.com").
		WithAtname("englishuser").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("en-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithTitle("English Test Page").
		WithBody("English body").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/en-space/pages/1/edit", map[string]string{
		"space_identifier": "en-space",
		"page_number":      "1",
	})
	req.Header.Set("Accept-Language", "en")
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangEn)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// 英語のラベルが含まれているか確認
	if !strings.Contains(body, "Title") {
		t.Error("レスポンスに英語のタイトルのラベルが見つからない")
	}
	if !strings.Contains(body, "Publish") {
		t.Error("レスポンスに英語の公開ボタンが見つからない")
	}
	if !strings.Contains(body, "Cancel") {
		t.Error("レスポンスに英語のキャンセルリンクが見つからない")
	}
}

func TestEdit_SuggestionMode(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("suggestion-edit@example.com").
		WithAtname("suggestionedit").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("suggestion-edit-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
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
		WithTitle("Suggestion Page Title").
		WithBody("Original body").
		Build()

	// 編集提案を作成
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("テスト提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	// ページリビジョンを作成 (SuggestionPageのベース用)
	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithPageID(pageID).
		WithTitle("Suggestion Page Title").
		WithBody("Original body").
		Build()

	// 編集提案ページを作成
	suggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		WithTitle("Suggested Title").
		WithBody("Suggested body").
		Build()

	// DraftPageを編集提案にリンク
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft for suggestion").
		WithBody("Draft body for suggestion").
		WithSuggestionPageID(suggestionPageID).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/suggestion-edit-space/pages/1/edit", map[string]string{
		"space_identifier": "suggestion-edit-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// 編集提案モードのメッセージが表示されていることを確認
	if !strings.Contains(body, "編集提案 #") {
		t.Error("レスポンスに編集提案の編集中のメッセージが見つからない")
	}
	if !strings.Contains(body, "のページを編集中です") {
		t.Error("レスポンスに編集提案の編集中の接尾辞が見つからない")
	}

	// 編集提案へのリンクが含まれていることを確認
	if !strings.Contains(body, `/s/suggestion-edit-space/suggestions/`) {
		t.Error("レスポンスに編集提案の詳細リンクが見つからない")
	}

	// 「編集提案を更新」ボタンが表示されていることを確認
	if !strings.Contains(body, "編集提案を更新") {
		t.Error("レスポンスに編集提案の更新ボタンが見つからない")
	}

	// 「トピックに公開」ボタンが表示されていないことを確認
	if strings.Contains(body, "トピックに公開") {
		t.Error("編集提案モードでトピックへの公開ボタンが表示されている")
	}

	// フォームのアクションが編集提案ページのURLであることを確認
	if !strings.Contains(body, "/suggestions/") {
		t.Error("フォームの送信先に編集提案ページのURLが見つからない")
	}
	if !strings.Contains(body, "/suggestion_pages/") {
		t.Error("フォームの送信先にsuggestion_pagesのパスが見つからない")
	}

	// _method=PATCHが含まれていることを確認 (PATCHメソッドで送信)
	if !strings.Contains(body, `value="PATCH"`) {
		t.Error("編集提案モードで_method=PATCHが無い")
	}

	// 編集提案モードでも編集履歴カラムが描画されること (編集提案の編集も実体は下書きページと
	// そのリビジョンであるため)
	if !strings.Contains(body, "編集履歴") {
		t.Error("編集提案モードで編集履歴の見出しが見つからない")
	}
	if !strings.Contains(body, `id="page-edit-draft-revisions-drawer"`) {
		t.Error("編集提案モードで編集履歴のドロワーが見つからない")
	}

	// 下書き保存ボタンが表示されていることを確認
	if !strings.Contains(body, "下書き保存") {
		t.Error("編集提案モードで下書き保存ボタンが表示されていない")
	}

	// 編集提案モードの「下書き保存」ボタンもhtmxのPATCHで画面遷移なしに送信されること
	if !strings.Contains(body, `hx-patch="/s/suggestion-edit-space/pages/1/draft_page_revision"`) {
		t.Error("編集提案モードで下書き保存ボタンのhx-patch属性が見つからない")
	}

	// 通常の下書きアラートが表示されていないことを確認 (編集提案メッセージが代わりに表示される)
	if strings.Contains(body, "現在下書きを表示しています") {
		t.Error("編集提案モードで通常の下書きアラートが表示されている")
	}
}

// 下書き保存の分割ボタンのキャレット側はアイコンのみのため、アクセシブルネームは内容では
// なく翻訳済みのaria-labelが供給する。
func TestEdit_下書き保存オプションのトリガーにアクセシブルネームがある(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		locale    string
		wantLabel string
	}{
		{
			name:      "日本語",
			locale:    i18n.LangJa,
			wantLabel: "下書き保存のオプション",
		},
		{
			name:      "英語",
			locale:    i18n.LangEn,
			wantLabel: "Save draft options",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)

			identifier := "save-draft-optlabel-" + tt.locale
			atname := "savedraftoptlabel" + tt.locale

			userID := testutil.NewUserBuilder(t, tx).
				WithEmail(identifier + "@example.com").
				WithAtname(atname).
				Build()
			spaceID := testutil.NewSpaceBuilder(t, tx).
				WithIdentifier(identifier).
				Build()
			spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
				WithSpaceID(spaceID).
				WithUserID(userID).
				Build()
			topicID := testutil.NewTopicBuilder(t, tx).
				WithSpaceID(spaceID).
				WithNumber(1).
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
				WithTitle("Save Draft Options Page").
				WithBody("body").
				Build()

			handler := setupHandler(t, queries)

			req := newRequestWithChiParams(t, http.MethodGet, "/s/"+identifier+"/pages/1/edit", map[string]string{
				"space_identifier": identifier,
				"page_number":      "1",
			})
			ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
			ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
			ctx = i18n.SetLocale(ctx, tt.locale)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.Edit(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
			}

			if !strings.Contains(rr.Body.String(), `aria-label="`+tt.wantLabel+`"`) {
				t.Errorf("下書き保存オプションのトリガーにaria-label %qが含まれていない", tt.wantLabel)
			}
		})
	}
}

// TestEdit_RelatedPageSectionsは、編集画面の一覧がページ表示画面と同じ3セクションに並ぶことを
// 固定する。リンク先ページ、リンク先ページごとに束ねたそのバックリンク、そしてこのページ自身の
// バックリンクである。セクションのラッパーは、コンテナが空の間セクションを隠すnot-hasのフックを
// 保つ。下書き自動保存のスワップは見出しを描き直さずにコンテナだけを詰め直すためである。
func TestEdit_RelatedPageSections(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("edit-sections@example.com").
		WithAtname("editsections").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("edit-sections-space").
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

	// リンク先ページのうち片方にはバックリンクがあり、もう片方には無い。セクションにはちょうど
	// 1つのグループが残る。
	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Linked Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	lonelyLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Lonely Linked Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Edited Page").
		WithBody("body").
		WithLinkedPageIDs([]model.PageID{linkedPageID, lonelyLinkedPageID}).
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(4).
		WithTitle("Related Link Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(5).
		WithTitle("Backlink Page").
		WithLinkedPageIDs([]model.PageID{pageID}).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/edit-sections-space/pages/1/edit", map[string]string{
		"space_identifier": "edit-sections-space",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, want := range []string{
		"関連リンク",
		"このページが直接リンクしているページ",
		"リンク先のページからさらに辿れるページ",
		"このページにリンクしているページ",
		`id="page-link-list-item-2"`,
		`not-has-[#page-related-link-list>*]:hidden`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}

	// どこからもリンクされていないリンク先ページにはグループが付かない。
	if strings.Contains(body, `id="page-link-list-item-3"`) {
		t.Error("バックリンクの無いリンク先ページに関連リンクのグループが付いている")
	}

	// 3セクションはリンク、関連リンク、バックリンクの順に並ぶ。リンク先ページのバックリンクは
	// リンクセクションのカードの隣ではなく、関連リンクのセクションに置かれる。
	linksIndex := strings.Index(body, `id="page-link-list-content"`)
	relatedIndex := strings.Index(body, `id="page-related-link-list"`)
	backlinksIndex := strings.Index(body, `id="page-backlink-list-content"`)
	if linksIndex == -1 || relatedIndex == -1 || backlinksIndex == -1 {
		t.Fatalf(
			"セクションの位置 = links:%d related-links:%d backlinks:%d、3つのセクションすべてが描画されていない",
			linksIndex,
			relatedIndex,
			backlinksIndex,
		)
	}
	if linksIndex >= relatedIndex || relatedIndex >= backlinksIndex {
		t.Errorf(
			"セクションの位置 = links:%d related-links:%d backlinks:%d、期待値 = links < related-links < backlinks",
			linksIndex,
			relatedIndex,
			backlinksIndex,
		)
	}
	if got := strings.Index(body, "Related Link Page"); got < relatedIndex {
		t.Errorf("リンク先ページのバックリンクが位置%dに描画され、位置%dの関連リンクセクションより前にある", got, relatedIndex)
	}

	// どのセクションも見出しの脇に説明を持ち、ページ表示画面と同じである。見出しの大きさは編集
	// 画面のものを保つため、共有コンポーネントはここでは基本のクラスだけでh2を描画する。
	for _, heading := range []string{"リンク", "関連リンク", "バックリンク"} {
		marker := fmt.Sprintf(`<h2 class="font-bold antialiased">%s</h2>`, heading)
		headingIndex := strings.Index(body, marker)
		if headingIndex == -1 {
			t.Errorf("見出し%qが描画されていない", heading)
			continue
		}
		if !strings.HasPrefix(body[headingIndex+len(marker):], `<p class="text-sm text-muted-foreground">`) {
			t.Errorf("見出し%qの後ろに説明文が無い", heading)
		}
	}
}

// 経路は編集画面自身で終わるため、末尾の項目はトピックへのリンクではなくaria-currentを持つ
// ラベルになる。ラベルは編集対象のページではなく画面自身を表す。編集画面は自身の見出しを
// 持たないためである。ページタイトルは編集画面の入力欄にも出るため、パンくず内に絞って検証する。
func TestEdit_パンくずが現在地の項目で終わる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("edit-crumb").
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
		WithTitle("Edit Crumb Page").
		Build()

	req := newRequestWithChiParams(t, http.MethodGet, "/s/edit-crumb/pages/1/edit", map[string]string{
		"space_identifier": "edit-crumb",
		"page_number":      "1",
	})
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	breadcrumb := pageEditBreadcrumb(t, rr.Body.String())
	for _, want := range []string{
		`href="/s/edit-crumb"`,
		`href="/s/edit-crumb/topics/1"`,
		`aria-current="page"`,
		"ページを編集",
	} {
		if !strings.Contains(breadcrumb, want) {
			t.Errorf("パンくずに%qが含まれていない", want)
		}
	}
	if strings.Contains(breadcrumb, `href="/s/edit-crumb/pages/1/edit"`) {
		t.Error("現在のページ編集のパンくずの項目がリンクになっている")
	}
}

// pageEditBreadcrumbはパンくずのナビゲーション部分だけのマークアップを返す。経路についての
// 検証が、画面の他の場所に出た同じ文字列で満たされてしまうのを防ぐ。
func pageEditBreadcrumb(t *testing.T, body string) string {
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
