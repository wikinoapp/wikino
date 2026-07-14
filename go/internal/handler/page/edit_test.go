package page_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
)

// setupHandler はテスト用のハンドラーを生成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *page.Handler {
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
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(queries)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)

	publishPageUC := usecase.NewPublishPageUsecase(
		nil,
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

	return page.NewHandler(
		cfg,
		flashMgr,
		getPageDetailUC,
		getEditLinkDataUC,
		publishPageUC,
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// フォームアクションが含まれているか確認
	if !strings.Contains(body, `/s/my-space/pages/1`) {
		t.Error("form action not found in response")
	}

	// CSRFトークンが含まれているか確認
	if !strings.Contains(body, "test-csrf-token") {
		t.Error("CSRF token not found in response")
	}

	// タイトルが表示されているか確認
	if !strings.Contains(body, "Test Page Title") {
		t.Error("page title not found in response")
	}

	// 本文が表示されているか確認
	if !strings.Contains(body, "Test page body content") {
		t.Error("page body not found in response")
	}

	// _methodがPATCHであることを確認
	if !strings.Contains(body, `value="PATCH"`) {
		t.Error("method override PATCH not found in response")
	}

	// The default layout's content wrapper reserves no left space for the rail (a floating pill that
	// overlays the content, so the content stays centered), and reserves bottom-nav height plus the
	// bottom safe-area inset as bottom padding below md so the fixed nav doesn't cover the last
	// content even when a PWA standalone display lifts the nav above the home indicator. Match the
	// full opening tag so the assertion stays pinned to the wrapper.
	//
	// [Ja] default レイアウトのコンテンツラッパーが、レール用の左余白を確保せず (レールは本文の上に
	// オーバーレイする浮遊ピルのため本文は中央のまま)、md 未満で固定ナビに最下部コンテンツが隠れない
	// よう下部ナビの高さ + 下端 safe-area 分の下部余白を確保していること (PWA スタンドアロン表示で
	// ナビをホームインジケータの上へ押し上げても足りるようにする)。ラッパーに固定するため開始タグ
	// 全体で照合する。
	if !strings.Contains(body, `<div class="flex-1 flex flex-col min-h-screen pb-[calc(var(--app-bottom-nav-max-height)+0.5rem+env(safe-area-inset-bottom))] md:pb-0">`) {
		t.Error("content wrapper padding class not found in response")
	}

	// The fixed bottom-nav wrapper carries pb-safe so a PWA standalone display lifts the nav pill
	// above the home indicator. Match the full opening tag so the assertion stays pinned to the
	// wrapper.
	//
	// [Ja] 下部ナビの固定ラッパーが pb-safe を持ち、PWA スタンドアロン表示でナビのピルをホーム
	// インジケータの上へ押し上げること。ラッパーに固定するため開始タグ全体で照合する。
	if !strings.Contains(body, `<div class="fixed bottom-2 left-1/2 z-sticky-bar flex w-full -translate-x-1/2 flex-col items-center px-2 pb-safe">`) {
		t.Error("bottom nav fixed wrapper pb-safe class not found in response")
	}

	// 日本語のラベルが含まれているか確認
	if !strings.Contains(body, "タイトル") {
		t.Error("Japanese title label not found in response")
	}

	// 公開ボタンが含まれているか確認
	if !strings.Contains(body, "トピックに公開") {
		t.Error("Japanese publish button not found in response")
	}

	// キャンセルリンクが含まれているか確認
	if !strings.Contains(body, "/s/my-space/pages/1") {
		t.Error("cancel link not found in response")
	}

	// パンくずリストにトピック名が含まれているか確認
	if !strings.Contains(body, "General") {
		t.Error("topic name not found in breadcrumb")
	}

	// パンくずリストにスペースへのリンクが含まれているか確認
	if !strings.Contains(body, "/s/my-space") {
		t.Error("space link not found in breadcrumb")
	}

	// 下書きがない場合、下書きアラートが表示されないことを確認
	if strings.Contains(body, "現在下書きを表示しています") {
		t.Error("draft alert should not be shown when no draft exists")
	}

	// The draft list column shows the empty state when the member has no drafts.
	// [Ja] メンバーに下書きが 1 件もないとき、下書き一覧カラムに空状態テキストが表示されること
	if !strings.Contains(body, "下書きはありません") {
		t.Error("draft list empty state text not found when no draft exists")
	}

	// The edit history column shows the empty state when there is no draft (hence no revisions).
	// [Ja] 下書きが無い (= リビジョンも無い) とき、編集履歴カラムに空状態テキストが表示されること
	if !strings.Contains(body, "下書きを保存すると、ここに編集履歴が表示されます") {
		t.Error("edit history empty state text not found when no draft exists")
	}

	// パンくずリストにトピックへのリンクが含まれているか確認
	if !strings.Contains(body, "/s/my-space/topics/1") {
		t.Error("topic link not found in breadcrumb")
	}

	// トピックのアイコン（公開トピックのためglobe-regular）が表示されているか確認
	// globe-regularのSVGパスデータに含まれる固有の文字列で検証
	if !strings.Contains(body, "a87.61,87.61") {
		t.Error("topic visibility icon (globe) not found in breadcrumb")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// DraftPageの内容が表示されていることを確認
	if !strings.Contains(body, "Draft Title") {
		t.Error("draft title not found in response")
	}
	if !strings.Contains(body, "Draft body content") {
		t.Error("draft body not found in response")
	}

	// 元のページの内容が表示されていないことを確認
	if strings.Contains(body, "Original Title") {
		t.Error("original title should not be shown when draft exists")
	}
	if strings.Contains(body, "Original body") {
		t.Error("original body should not be shown when draft exists")
	}

	// 下書きアラートが表示されていることを確認
	if !strings.Contains(body, "現在下書きを表示しています") {
		t.Error("draft alert message not found in response")
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

	// The page being edited (page 1).
	// [Ja] 編集対象のページ (page 1)
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Edited Page").
		WithBody("body").
		Build()

	// Another draft within the same space (listed in the left column).
	// [Ja] 同一スペース内の別の下書き (左カラムに一覧表示される)
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// The other draft appears in the left column and links to its editor.
	// [Ja] 左カラムに別の下書きが表示され、その編集画面へのリンクを持つこと
	if !strings.Contains(body, "Sidebar Column Draft") {
		t.Error("draft list column does not contain the other draft")
	}
	if !strings.Contains(body, "/s/draftcol-space/pages/2/edit") {
		t.Error("draft card link to the other page's editor not found")
	}

	// The draft list "view all" link is shown in the heading.
	// [Ja] 下書き一覧の「すべて表示」リンクが見出しに表示されること
	if !strings.Contains(body, "すべて表示") {
		t.Error("draft list view-all link not found")
	}

	// The narrow-screen drawer (panel and open button) is wired up.
	// [Ja] 狭幅時のドロワー (本体と開閉ボタン) が結線されていること
	if !strings.Contains(body, `id="page-edit-draft-pages-drawer"`) {
		t.Error("draft list drawer not found")
	}
	if !strings.Contains(body, `data-drawer-open="page-edit-draft-pages-drawer"`) {
		t.Error("draft list drawer open button not found")
	}

	// The removed sidebar leaves no trace: no off-canvas sidebar element, no TopNav toggle, and no
	// sidebar-opening dispatch. The in-screen draft navigation is the left column, not a sidebar.
	// [Ja] 廃止したサイドバーの痕跡が残っていないこと。off-canvas のサイドバー要素・TopNav の
	// 開閉ボタン・サイドバーを開く dispatch のいずれも無い。画面内の下書きナビゲーションは
	// サイドバーではなく左カラムが担う。
	if strings.Contains(body, `id="sidebar"`) {
		t.Error("global sidebar should not be rendered on the page editor")
	}
	if strings.Contains(body, "サイドバーの開閉") {
		t.Error("TopNav sidebar toggle button should not be rendered on the page editor")
	}
	if strings.Contains(body, "basecoat:sidebar") {
		t.Error("no sidebar-opening button should be rendered on the page editor")
	}

	// The editor is wired to the global navigation like every other page (rail + bottom bar).
	// [Ja] 編集画面も他のページと同様にグローバルナビ (レール + 下部バー) へ結線されていること
	if !strings.Contains(body, `aria-label="グローバルナビゲーション"`) {
		t.Error("global navigation rail should be rendered on the page editor")
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

	// Two saved revisions: the column should list v1 and v2.
	// [Ja] 保存済みリビジョン 2 件: カラムに v1 と v2 が表示されること
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
			t.Fatalf("Create() revision error = %v", err)
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// The edit history column heading is shown.
	// [Ja] 編集履歴カラムの見出しが表示されること
	if !strings.Contains(body, "編集履歴") {
		t.Error("edit history heading not found")
	}

	// Version numbers derived from the total count appear (oldest = v1).
	// [Ja] 総件数から算出したバージョン番号が表示されること (最古 = v1)
	if !strings.Contains(body, "v2") {
		t.Error("version v2 not found in the edit history column")
	}
	if !strings.Contains(body, "v1") {
		t.Error("version v1 not found in the edit history column")
	}

	// The newest revision gets the "current" badge.
	// [Ja] 最新リビジョンに「現在」バッジが付くこと
	if !strings.Contains(body, "現在") {
		t.Error("current badge not found in the edit history column")
	}

	// The narrow-screen drawer (panel and open button) is wired up.
	// [Ja] 狭幅時のドロワー (本体と開閉ボタン) が結線されていること
	if !strings.Contains(body, `id="page-edit-draft-revisions-drawer"`) {
		t.Error("edit history drawer not found")
	}
	if !strings.Contains(body, `data-drawer-open="page-edit-draft-revisions-drawer"`) {
		t.Error("edit history drawer open button not found")
	}

	// The OOB swap targets of the manual save response wrap both column instances with
	// distinct ids.
	//
	// [Ja] 手動保存レスポンスの OOB スワップターゲットが、カラム 2 箇所を別々の id で
	// 包んでいること
	if !strings.Contains(body, `id="page-revision-list"`) {
		t.Error("static revision list OOB target not found")
	}
	if !strings.Contains(body, `id="page-revision-list-drawer"`) {
		t.Error("drawer revision list OOB target not found")
	}

	// The save-draft button sends an htmx PATCH so saving does not navigate away.
	// [Ja] 「下書き保存」ボタンが htmx の PATCH で送信され、保存で画面遷移しないこと
	if !strings.Contains(body, `id="page-edit-save-draft-button"`) {
		t.Error("save draft button not found")
	}
	if !strings.Contains(body, `hx-patch="/s/revcol-space/pages/1/draft_page_revision"`) {
		t.Error("save draft button hx-patch attribute not found")
	}

	// The empty state must not be shown when revisions exist.
	// [Ja] リビジョンが存在するときは空状態テキストが表示されないこと
	if strings.Contains(body, "下書きを保存すると、ここに編集履歴が表示されます") {
		t.Error("edit history empty state should not be shown when revisions exist")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// タイトルが空のとき、タイトル入力欄にautofocusが設定されていることを確認
	if !strings.Contains(body, `id="page_title"`) {
		t.Error("page title input not found")
	}
	if !strings.Contains(body, "autofocus") {
		t.Error("autofocus attribute not found in page_title input")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
	location := rr.Header().Get("Location")
	if location != "/sign_in" {
		t.Errorf("wrong redirect location: got %v want /sign_in", location)
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// リンク一覧セクションのコンテナが存在すること
	if !strings.Contains(body, `id="page-link-list"`) {
		t.Error("page-link-list container not found in response")
	}

	// htmxのhx-trigger属性が含まれていること
	// この属性により、下書き保存後にリンク一覧がOOBスワップで自動再読み込みされる
	if !strings.Contains(body, `hx-trigger="draft-autosaved from:window"`) {
		t.Error("hx-trigger attribute not found - link list auto-reload will not work")
	}

	// hx-getでエンドポイントが指定されていること
	if !strings.Contains(body, "/s/linklist-reload-space/pages/1/draft_page") {
		t.Error("draft_page endpoint URL not found in response")
	}

	// hx-swap="none"が指定されていること（OOBスワップのみで更新するため）
	if !strings.Contains(body, `hx-swap="none"`) {
		t.Error("hx-swap=none not found - OOB swap will not work correctly")
	}

	// The link heading (h2) sits outside #page-link-list (out of the OOB swap target) and is always
	// rendered: it stays in the DOM even when there are no links (this case) and the not-has CSS
	// hides the section instead.
	//
	// [Ja] 見出し (h2) は #page-link-list の外 (OOB スワップ対象外) にあり、常にレンダリングされる。
	// リンクが無いこのケースでも DOM 上には存在し、CSS の not-has でセクションごと非表示になる。
	if !strings.Contains(body, `<h2 class="font-bold antialiased">`) {
		t.Error("リンク見出しの h2 が見つからない (呼び出し側でのレンダリングが壊れている可能性)")
	}

	// The section hides itself via the not-has variant while its list is empty. With this CSS hook,
	// heading visibility tracks the content without re-rendering the heading.
	//
	// [Ja] セクションは not-has バリアントでリストが空のとき非表示になる。
	// この CSS フックにより、見出しを再描画せずに表示・非表示が内容に追従する。
	if !strings.Contains(body, `not-has-[#page-link-list>*]:hidden`) {
		t.Error("リンクセクションの表示制御クラス not-has-[#page-link-list>*]:hidden が見つからない")
	}
	if !strings.Contains(body, `not-has-[#page-backlink-list>*]:hidden`) {
		t.Error("バックリンクセクションの表示制御クラス not-has-[#page-backlink-list>*]:hidden が見つからない")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// The tabs container and both tabs are rendered.
	// [Ja] タブのコンテナと編集/プレビューの両タブが描画されること
	if !strings.Contains(body, `id="page-edit-tabs"`) {
		t.Error("tabs container not found in response")
	}

	// The tabs container carries min-w-0 so the preview's <pre> overflow-auto contains long
	// code lines instead of the grid item widening past the center column. Matching the whole
	// start tag ties min-w-0 to this container (min-w-0 also appears on other elements).
	//
	// [Ja] タブコンテナが min-w-0 を持ち、プレビューの <pre> の overflow-auto が長いコード行を
	// 収める (grid アイテムが中央カラムを超えて広がらない) こと。開始タグ全体で照合し、min-w-0 が
	// このコンテナに付いていることを担保する (min-w-0 は他の要素にも現れる)。
	if !strings.Contains(body, `<div class="tabs gap-4 min-w-0" id="page-edit-tabs">`) {
		t.Error("tabs container is missing min-w-0 (guards preview code block overflow)")
	}
	if !strings.Contains(body, `id="page-edit-tab-edit"`) {
		t.Error("edit tab not found in response")
	}
	if !strings.Contains(body, `id="page-edit-tab-preview"`) {
		t.Error("preview tab not found in response")
	}

	// The preview tab POSTs the current form values to the preview endpoint via htmx.
	// [Ja] プレビュータブが htmx でフォームの現在値をプレビューエンドポイントに POST すること
	if !strings.Contains(body, `hx-post="/s/preview-tab-space/pages/1/preview"`) {
		t.Error("preview tab hx-post to the preview endpoint not found in response")
	}

	// htmx 4 collects the whole edit form (including the hidden _method=PATCH) when the preview
	// tab fires, so the button blanks _method via hx-vals. The override middleware ignores an
	// empty _method, so the preview request stays a POST instead of being rewritten to PATCH.
	// The rendered double quotes are HTML-escaped to &#34; (the browser decodes them back).
	//
	// [Ja] プレビュータブ発火時、htmx 4 は編集フォーム全体 (hidden の _method=PATCH を含む) を
	// 収集するため、ボタンは hx-vals で _method を空にする。override ミドルウェアは空の _method を
	// 無視するので、プレビュー要求は PATCH に書き換えられず POST のままになる。レンダリングされる
	// 二重引用符は &#34; に HTML エスケープされる (ブラウザがデコードして元に戻す)。
	if !strings.Contains(body, `hx-vals="{&#34;_method&#34;: &#34;&#34;}"`) {
		t.Error("preview tab hx-vals blanking _method not found in response")
	}
	// The CSRF token input stays in the form, so the auto-collected preview POST carries it and
	// passes the CSRF middleware.
	// [Ja] CSRF トークン入力はフォーム内に残るため、自動収集されるプレビュー POST に含まれ、
	// CSRF ミドルウェアを通過する。
	if !strings.Contains(body, `id="page-edit-csrf-token"`) {
		t.Error("csrf token input not found in response")
	}

	// The result is swapped into the preview panel, with a loading indicator.
	// [Ja] 結果はプレビューパネルにスワップされ、ローディング表示が用意されていること
	if !strings.Contains(body, `hx-target="#page-edit-preview-content"`) {
		t.Error("preview tab hx-target not found in response")
	}
	if !strings.Contains(body, `id="page-edit-preview-content"`) {
		t.Error("preview content panel not found in response")
	}
	if !strings.Contains(body, `id="page-edit-preview-loading"`) {
		t.Error("preview loading indicator not found in response")
	}

	// The tab labels are localized. Match each label adjacent to its closing </button> tag
	// so the preview assertion is not satisfied by the loading text ("プレビューを生成中...").
	//
	// [Ja] タブのラベルが翻訳されていること。プレビューのアサーションがローディングテキスト
	// ("プレビューを生成中...") で成立しないよう、各ラベルを閉じ </button> タグ隣接で検証する。
	if !strings.Contains(body, "編集</button>") {
		t.Error("edit tab label not found in response")
	}
	if !strings.Contains(body, "プレビュー</button>") {
		t.Error("preview tab label not found in response")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// Both the ⌘ (Mac) and Ctrl+ (other) variants are rendered so CSS can switch by html[data-os].
	// The Mac modifier glues to the key with no separator to keep each chip narrow, while the non-Mac
	// modifier is "Ctrl+" so it reads with a "+" before the key. The publish button hints Mod-Enter,
	// rendered as the return-arrow icon after the modifier; the save button hints Mod-s as text.
	//
	// [Ja] ⌘ (Mac) と Ctrl+ (それ以外) の両版が描画され、CSS が html[data-os] で切り替えられること。
	// Mac の修飾キーは区切り無しでキーに詰めてチップを細く保ち、非 Mac の修飾キーは "Ctrl+" で
	// キーの前に "+" が入る。公開ボタンは Mod-Enter を修飾キーの直後の折り返し矢印アイコンで、
	// 保存ボタンは Mod-s をテキストで表記すること。
	if !strings.Contains(body, `>⌘<svg class="size-3.5"`) {
		t.Error("publish button Mac shortcut hint (⌘ + return-arrow icon) not found in response")
	}
	if !strings.Contains(body, `>Ctrl+<svg class="size-3.5"`) {
		t.Error("publish button non-Mac shortcut hint (Ctrl+ and return-arrow icon) not found in response")
	}
	// The return-arrow key is rendered with the arrow-elbow-down-left icon (its path's leading
	// move/vertical-line command), not the small ↵ glyph.
	//
	// [Ja] 折り返しキーは小さな ↵ グリフではなく arrow-elbow-down-left アイコン (パス先頭の
	// move/vertical-line コマンド) で描画すること。
	if !strings.Contains(body, "M200,32V176") {
		t.Error("publish button shortcut hint does not use the arrow-elbow-down-left icon")
	}
	if !strings.Contains(body, ">⌘S</kbd>") {
		t.Error("save button Mac shortcut hint (⌘S) not found in response")
	}
	if !strings.Contains(body, ">Ctrl+S</kbd>") {
		t.Error("save button non-Mac shortcut hint (Ctrl+S) not found in response")
	}

	// The hint chips toggle by OS only on non-touch devices via the platform-attribute variants.
	// [Ja] 表記チップはタッチ以外の端末で、プラットフォーム属性バリアントにより OS で出し分けること。
	if !strings.Contains(body, "non-touch:in-[[data-os=mac]]:inline-flex") {
		t.Error("Mac keyboard hint display-control class not found in response")
	}
	if !strings.Contains(body, "non-touch:in-[[data-os=other]]:inline-flex") {
		t.Error("non-Mac keyboard hint display-control class not found in response")
	}

	// Each chip tints itself in its host button's foreground color (foreground-colored text over a
	// low-opacity foreground background): publish tints btn-primary, save tints btn-secondary.
	//
	// [Ja] 各チップはホストボタンの前景色で淡く染める (前景色の文字 + 低不透明度の前景色背景)。
	// 公開は btn-primary を、保存は btn-secondary を染めること。
	if !strings.Contains(body, "bg-primary-foreground/20 text-primary-foreground") {
		t.Error("publish keyboard hint tinted color classes (bg-primary-foreground/20 text-primary-foreground) not found in response")
	}
	if !strings.Contains(body, "bg-secondary-foreground/8 text-secondary-foreground") {
		t.Error("save keyboard hint tinted color classes (bg-secondary-foreground/8 text-secondary-foreground) not found in response")
	}

	// The chips are decorative, so screen readers read only the button label.
	// [Ja] チップは装飾的なため、スクリーンリーダーはボタンラベルのみを読み上げること。
	if !strings.Contains(body, `<kbd class="kbd hidden non-touch:in-[[data-os=mac]]:inline-flex bg-secondary-foreground/8 text-secondary-foreground" aria-hidden="true">`) {
		t.Error("keyboard hint kbd chip with aria-hidden not found in response")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// The action row is one line on wide screens and collapses to two tiers below the 1152px
	// container width, where the fixed center column is too narrow for everything on one line.
	//
	// [Ja] 操作行は広い画面では 1 行、1152px のコンテナ幅未満では 2 段に折り返すこと (この幅未満では
	// 中央カラムが狭く 1 行に収まらないため)。
	if !strings.Contains(body, `<div class="flex flex-col gap-1 min-[1152px]:flex-row min-[1152px]:items-center">`) {
		t.Error("action row responsive layout container not found in response")
	}

	// The saved-at indicator and cancel sit in one container, un-reversed (saved-at then cancel).
	// On the two-tier layout they group at the right (justify-end); on one line they spread
	// (min-[1152px]:justify-between). The time never wraps.
	//
	// [Ja] 保存時刻表示とキャンセルは 1 つのコンテナに反転させず (保存時刻→キャンセル順) 並べる。
	// 2 段時は右揃えでまとめ (justify-end)、1 行時は左右へ振り分ける (min-[1152px]:justify-between)。
	// 時刻は折り返さないこと。
	if !strings.Contains(body, `<div class="flex items-center justify-end gap-2 min-[1152px]:flex-1 min-[1152px]:justify-between">`) {
		t.Error("saved-at + cancel container (un-reversed, right-aligned on two tiers, responsive) not found in response")
	}
	if !strings.Contains(body, `<div id="page-draft-saved-at" class="text-xs text-muted-foreground whitespace-nowrap">`) {
		t.Error("saved-at indicator with whitespace-nowrap not found in response")
	}
	if strings.Contains(body, "flex-row-reverse") {
		t.Error("action row should not reverse saved-at and cancel order (flex-row-reverse found)")
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

	// The "page-edit-zen" substring alone matches the always-present Tailwind variant classes
	// (in-[.page-edit-zen]:lg:hidden etc.), so assert on the full class attribute of the container.
	//
	// [Ja] "page-edit-zen" の部分一致では常に存在する Tailwind バリアントクラス
	// (in-[.page-edit-zen]:lg:hidden など) にもマッチしてしまうため、コンテナの class 属性全体で
	// 検証する。
	const containerClassOff = `class="max-w-6xl w-full mx-auto lg:px-4"`
	const containerClassOn = `class="max-w-6xl w-full mx-auto lg:px-4 page-edit-zen"`

	tests := []struct {
		name        string
		cookieValue string // empty = no cookie. [Ja] 空はクッキーなし
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
				t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()

			// The toggle button is rendered with its label and the server-rendered pressed state.
			// [Ja] トグルボタンがラベルとサーバー描画の押下状態付きで描画されること
			if !strings.Contains(body, "data-zen-mode-toggle") {
				t.Error("zen mode toggle button not found in response")
			}
			if !strings.Contains(body, "Zenモード") {
				t.Error("zen mode button label not found in response")
			}

			if tt.wantZen {
				if !strings.Contains(body, containerClassOn) {
					t.Error("zen mode class not found on the editor container")
				}
				if !strings.Contains(body, `aria-pressed="true"`) {
					t.Error(`aria-pressed="true" not found on the zen mode toggle`)
				}
			} else {
				if !strings.Contains(body, containerClassOff) {
					t.Error("editor container without the zen mode class not found")
				}
				if strings.Contains(body, containerClassOn) {
					t.Error("zen mode class should not be set on the editor container")
				}
				if !strings.Contains(body, `aria-pressed="false"`) {
					t.Error(`aria-pressed="false" not found on the zen mode toggle`)
				}
			}

			// Zen mode only reshapes the desktop (lg+) layout, so every Zen variant is gated to lg:
			// the side columns and link/backlink lists hide, the grid collapses, and the center
			// column widens only at lg. On mobile there are no side columns to collapse.
			//
			// [Ja] Zenモードはデスクトップ (lg 以上) のレイアウトだけを変えるため、Zenバリアントは
			// すべて lg 限定になる (左右カラム・リンク / バックリンク一覧の非表示、グリッド解除、
			// 中央カラムの拡幅はいずれも lg でのみ効く)。モバイルには畳む対象のサイドカラムが無い。

			// The side columns wrap their Zen hide with lg, asserted on the full class to pin it to
			// the always-lg-hidden side columns rather than the link/backlink section, which now
			// shares the same in-[.page-edit-zen]:lg:hidden variant.
			//
			// [Ja] 左右サイドカラムは Zen非表示を lg でラップする。同じ in-[.page-edit-zen]:lg:hidden
			// バリアントを持つようになったリンク / バックリンクセクションではなく、常に lg 非表示の
			// サイドカラムに固定するため、class 属性全体で検証する。
			if !strings.Contains(body, `class="hidden lg:block in-[.page-edit-zen]:lg:hidden"`) {
				t.Error("zen mode side column hide class not found in response")
			}
			// The link/backlink section wraps its Zen hide with lg so the lists stay visible on
			// mobile even when the Zen cookie is set (asserted on the full class to pin it to the
			// link section, not the always-lg-hidden side columns).
			//
			// [Ja] リンク / バックリンク一覧セクションは Zen非表示を lg でラップし、Zenクッキーが
			// 設定されていてもモバイルでは一覧を表示したままにする (常に lg 非表示のサイドカラムでは
			// なくリンクセクションに固定するため、class 属性全体で検証する)。
			if !strings.Contains(body, `class="flex flex-col gap-4 px-4 in-[.page-edit-zen]:lg:hidden"`) {
				t.Error("zen mode link list lg-only hide class not found in response")
			}
			if !strings.Contains(body, "in-[.page-edit-zen]:lg:block") {
				t.Error("zen mode grid collapse class not found in response")
			}
			if !strings.Contains(body, "in-[.page-edit-zen]:lg:max-w-4xl") {
				t.Error("zen mode center column widen class not found in response")
			}
			// The Zen toggle is hidden below lg (mobile has no side columns to collapse, and a
			// hidden toggle with Zen on would trap the mobile user), and restored to inline-flex at
			// lg. Assert on the full class so it stays pinned to the toggle button.
			//
			// [Ja] Zenトグルは lg 未満で非表示にし (モバイルには畳むサイドカラムが無く、Zen ON の
			// ままトグルが無いとモバイルのユーザーが戻せなくなる)、lg で inline-flex に戻す。
			// トグルボタンに固定するため class 属性全体で検証する。
			if !strings.Contains(body, `class="hidden lg:inline-flex btn-sm-outline rounded-full w-fit"`) {
				t.Error("zen mode toggle lg-only display class not found in response")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// 英語のラベルが含まれているか確認
	if !strings.Contains(body, "Title") {
		t.Error("English title label not found in response")
	}
	if !strings.Contains(body, "Publish") {
		t.Error("English publish button not found in response")
	}
	if !strings.Contains(body, "Cancel") {
		t.Error("English cancel link not found in response")
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

	// ページリビジョンを作成（SuggestionPageのベース用）
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// 編集提案モードのメッセージが表示されていることを確認
	if !strings.Contains(body, "編集提案 #") {
		t.Error("suggestion editing message not found in response")
	}
	if !strings.Contains(body, "のページを編集中です") {
		t.Error("suggestion editing suffix not found in response")
	}

	// 編集提案へのリンクが含まれていることを確認
	if !strings.Contains(body, `/s/suggestion-edit-space/suggestions/`) {
		t.Error("suggestion show link not found in response")
	}

	// 「編集提案を更新」ボタンが表示されていることを確認
	if !strings.Contains(body, "編集提案を更新") {
		t.Error("update suggestion button not found in response")
	}

	// 「トピックに公開」ボタンが表示されていないことを確認
	if strings.Contains(body, "トピックに公開") {
		t.Error("publish to topic button should not be shown in suggestion mode")
	}

	// フォームのアクションが編集提案ページのURLであることを確認
	if !strings.Contains(body, "/suggestions/") {
		t.Error("suggestion page URL not found in form action")
	}
	if !strings.Contains(body, "/suggestion_pages/") {
		t.Error("suggestion_pages path not found in form action")
	}

	// _method=PATCH が含まれていることを確認（PATCHメソッドで送信）
	if !strings.Contains(body, `value="PATCH"`) {
		t.Error("_method=PATCH should be present in suggestion mode")
	}

	// The edit history column is rendered in suggestion mode too (suggestion edits are also backed
	// by draft pages and their revisions).
	//
	// [Ja] 編集提案モードでも編集履歴カラムが描画されること (編集提案の編集も実体は下書きページと
	// そのリビジョンであるため)
	if !strings.Contains(body, "編集履歴") {
		t.Error("edit history heading not found in suggestion mode")
	}
	if !strings.Contains(body, `id="page-edit-draft-revisions-drawer"`) {
		t.Error("edit history drawer not found in suggestion mode")
	}

	// 下書き保存ボタンが表示されていることを確認
	if !strings.Contains(body, "下書き保存") {
		t.Error("save draft button should be shown in suggestion mode")
	}

	// The suggestion-mode save button also sends an htmx PATCH without navigation.
	// [Ja] 編集提案モードの「下書き保存」ボタンも htmx の PATCH で画面遷移なしに送信されること
	if !strings.Contains(body, `hx-patch="/s/suggestion-edit-space/pages/1/draft_page_revision"`) {
		t.Error("save draft button hx-patch attribute not found in suggestion mode")
	}

	// 通常の下書きアラートが表示されていないことを確認（編集提案メッセージが代わりに表示される）
	if strings.Contains(body, "現在下書きを表示しています") {
		t.Error("normal draft alert should not be shown in suggestion mode")
	}
}
