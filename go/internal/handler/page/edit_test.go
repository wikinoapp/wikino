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
	if !strings.Contains(body, "編集履歴はありません") {
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

	// The page editor renders neither the global sidebar nor any button that opens it (the TopNav
	// toggle and the mobile BottomNav menu button), since the sidebar is hidden here.
	// [Ja] 編集画面ではグローバルサイドバーも、それを開くボタン (TopNav の開閉ボタンとモバイルの
	// BottomNav メニューボタン) も描画しないこと (ここではサイドバーを非表示にするため)
	if strings.Contains(body, `id="sidebar"`) {
		t.Error("global sidebar should not be rendered on the page editor")
	}
	if strings.Contains(body, "サイドバーの開閉") {
		t.Error("TopNav sidebar toggle button should not be rendered on the page editor")
	}
	if strings.Contains(body, "basecoat:sidebar") {
		t.Error("no sidebar-opening button should be rendered on the page editor (TopNav or BottomNav)")
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
	if strings.Contains(body, "編集履歴はありません") {
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

	// The CSRF token is included, while hx-params="not _method" drops the enclosing form's
	// hidden _method=PATCH so the override middleware does not rewrite the preview POST into
	// a PATCH.
	// [Ja] CSRF トークンは含めつつ、hx-params="not _method" で囲みフォームの hidden な
	// _method=PATCH を除外し、override ミドルウェアがプレビュー POST を PATCH に書き換えない
	// ようにしていること
	if !strings.Contains(body, `hx-include="#page_title, #page_body, #page-edit-csrf-token"`) {
		t.Error("preview tab hx-include with the expected fields not found in response")
	}
	if !strings.Contains(body, `hx-params="not _method"`) {
		t.Error("preview tab hx-params excluding _method not found in response")
	}
	if !strings.Contains(body, `id="page-edit-csrf-token"`) {
		t.Error("csrf token input id used by hx-include not found in response")
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
			if !strings.Contains(body, "Zen モード") {
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

			// The Tailwind variant classes reacting to Zen mode are present on the layout elements
			// (hide the side columns / link lists, collapse the grid, widen the center column).
			//
			// [Ja] Zen モードに反応する Tailwind バリアントクラスがレイアウト要素に付いていること
			// (左右カラム・リンク一覧の非表示、グリッド解除、中央カラムの拡幅)。
			if !strings.Contains(body, "in-[.page-edit-zen]:lg:hidden") {
				t.Error("zen mode side column hide class not found in response")
			}
			if !strings.Contains(body, "in-[.page-edit-zen]:hidden") {
				t.Error("zen mode link list hide class not found in response")
			}
			if !strings.Contains(body, "in-[.page-edit-zen]:lg:block") {
				t.Error("zen mode grid collapse class not found in response")
			}
			if !strings.Contains(body, "in-[.page-edit-zen]:max-w-4xl") {
				t.Error("zen mode center column widen class not found in response")
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
