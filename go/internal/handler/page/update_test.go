package page_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUpdate_Success(t *testing.T) {
	// PublishPageUsecaseは内部でトランザクションを管理するため、
	// テスト用トランザクションと競合する。
	// Usecaseの動作はusecase/publish_page_test.goでテストされている。
	t.Skip("UseCaseが別のトランザクションを使うため、usecaseパッケージでテストする")
}

func TestUpdate_ValidationError_EmptyTitle(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("update-empty@example.com").
		WithAtname("updateempty").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("update-space").
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
		WithBody("Test body").
		Build()

	handler := setupHandler(t, queries)

	// タイトルが空のフォームデータ
	form := url.Values{}
	form.Set("title", "")
	form.Set("body", "Updated body")
	form.Set("draft_page_digest", "submitted-digest")
	form.Set("csrf_token", "test-csrf-token")

	req := newRequestWithChiParams(t, http.MethodPost, "/s/update-space/pages/1", map[string]string{
		"space_identifier": "update-space",
		"page_number":      "1",
	})
	req.Body = toReadCloser(form.Encode())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// バリデーションエラーの再描画でもZenモードが維持されることを検証するため、Zenモードクッキーを送る。
	req.AddCookie(&http.Cookie{Name: "wikino_zen_mode", Value: "1"})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// バリデーションエラーで422が返ること
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()

	// タイトル必須エラーメッセージが表示されること
	if !strings.Contains(body, "タイトルを入力してください") {
		t.Error("レスポンスにタイトル必須のエラーメッセージが見つからない")
	}

	// バリデーションエラーの再描画でも編集画面と同じ文書のタイトルになること
	if !strings.Contains(body, "<title>ページを編集 | General | Test Space</title>") {
		t.Error("再描画したレスポンスにトピック名を含む文書のタイトルが見つからない")
	}

	// 再表示したエディタも動くよう、編集画面と同じくeditor.jsを読むこと
	if !strings.Contains(body, `<script type="module" src="/static/js/editor.js?v=`) {
		t.Error("再描画したレスポンスにeditor.jsを読み込むscriptが見つからない")
	}

	// 編集フォームが再表示されること
	if !strings.Contains(body, `name="title"`) {
		t.Error("レスポンスにタイトルの入力欄が見つからない")
	}

	// 本文の入力値が保持されていること
	if !strings.Contains(body, "Updated body") {
		t.Error("レスポンスで本文の値が保持されていない")
	}

	// 公開は下書きを変えずに失敗したため、再表示した画面も送信された版を前提にし続けること
	for _, want := range []string{
		`name="draft_page_digest" value="submitted-digest"`,
		`data-markdown-editor-draft-page-digest="submitted-digest"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	// バリデーションエラーでは競合の知らせを出さないこと
	if strings.Contains(body, "data-draft-conflict-publish") || strings.Contains(body, "data-markdown-editor-draft-conflict-digest") {
		t.Error("バリデーションエラーの再表示に公開の競合の印が含まれている")
	}

	breadcrumb := pageEditBreadcrumb(t, body)
	for _, want := range []string{
		`href="/s/update-space"`,
		`href="/s/update-space/topics/1"`,
		`aria-current="page"`,
		"ページを編集",
	} {
		if !strings.Contains(breadcrumb, want) {
			t.Errorf("パンくずに%qが含まれていない", want)
		}
	}
	if strings.Contains(breadcrumb, `href="/s/update-space/pages/1/edit"`) {
		t.Error("バリデーションエラーの後に現在のページ編集のパンくずの項目がリンクになっている")
	}

	// バリデーションエラーの再描画でも編集画面と同じパンくずヘッダーを供給するため、
	// <main> の外に編集画面の本文幅max-w-6xlで描画される。
	if !strings.Contains(body, `<div class="max-w-6xl mx-auto flex w-full items-center justify-between gap-2 px-4">`) {
		t.Error("バリデーションエラーの再描画後に共通のパンくずヘッダーがmax-w-6xlのコンテンツ幅を保っていない")
	}
	header, main := strings.Index(body, "<header"), strings.Index(body, `<main id="main" tabindex="-1">`)
	if header == -1 || main == -1 || header > main {
		t.Errorf("共通のパンくずヘッダー (位置%d) が <main> (位置%d) より前にない", header, main)
	}

	// 再描画されたエディタにクッキー由来のZenモードクラスが付くこと。"page-edit-zen" の
	// 部分一致では常に存在するTailwindバリアントクラス (in-[.page-edit-zen]:lg:hiddenなど) にも
	// マッチしてしまうため、class属性全体で検証する。
	if !strings.Contains(body, `class="max-w-6xl w-full mx-auto lg:px-4 page-edit-zen"`) {
		t.Error("バリデーションエラーの再描画後にエディタのコンテナにZenモードのクラスが見つからない")
	}

	// バリデーションエラーの再描画でもグローバルナビを維持し、送信失敗後も上部バーと下部バーが
	// mdで入れ替わること。
	if !strings.Contains(body, `<nav class="shrink-0 hidden md:flex"`) {
		t.Error("バリデーションエラーの再描画後にグローバルナビゲーションの上部バーがmdで切り替わっていない")
	}
	if !strings.Contains(body, `<nav class="md:hidden"`) {
		t.Error("バリデーションエラーの再描画後にグローバルナビゲーションの下部バーがmdで切り替わっていない")
	}
}

func TestUpdate_ValidationError_InvalidChars(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("update-invalid@example.com").
		WithAtname("updateinvalid").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("invalid-space").
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
		WithTitle("Test Page").
		WithBody("Test body").
		Build()

	handler := setupHandler(t, queries)

	form := url.Values{}
	form.Set("title", "foo/bar")
	form.Set("body", "body content")
	form.Set("draft_page_digest", "")

	req := newRequestWithChiParams(t, http.MethodPost, "/s/invalid-space/pages/1", map[string]string{
		"space_identifier": "invalid-space",
		"page_number":      "1",
	})
	req.Body = toReadCloser(form.Encode())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()

	// タイトル禁止文字エラーが表示されること
	if !strings.Contains(body, `aria-invalid="true"`) {
		t.Error("レスポンスにaria-invalid属性が見つからない")
	}
}

func TestUpdate_ValidationError_DuplicateTitle(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("update-dup@example.com").
		WithAtname("updatedup").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dup-space").
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
		WithTitle("Existing Title").
		WithBody("body1").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Another Page").
		WithBody("body2").
		Build()

	handler := setupHandler(t, queries)

	// ページ2のタイトルを既存のページ1のタイトルに変更
	form := url.Values{}
	form.Set("title", "Existing Title")
	form.Set("body", "body2")
	form.Set("draft_page_digest", "")

	req := newRequestWithChiParams(t, http.MethodPost, "/s/dup-space/pages/2", map[string]string{
		"space_identifier": "dup-space",
		"page_number":      "2",
	})
	req.Body = toReadCloser(form.Encode())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()

	// 重複エラーメッセージ (HTMLリンク付き) が表示されること
	if !strings.Contains(body, "/s/dup-space/pages/1/edit") {
		t.Error("レスポンスに既存ページの編集リンクが見つからない")
	}
}

func TestUpdate_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	form := url.Values{}
	form.Set("title", "New Title")
	form.Set("body", "New body")
	form.Set("draft_page_digest", "")

	req := newRequestWithChiParams(t, http.MethodPost, "/s/my-space/pages/1", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	})
	req.Body = toReadCloser(form.Encode())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}
	location := rr.Header().Get("Location")
	if location != "/sign_in" {
		t.Errorf("リダイレクト先 = %v、期待値 = /sign_in", location)
	}
}

func TestUpdate_SpaceNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("update-nosp@example.com").
		WithAtname("updatenosp").
		Build()

	handler := setupHandler(t, queries)

	form := url.Values{}
	form.Set("title", "New Title")
	form.Set("body", "New body")
	form.Set("draft_page_digest", "")

	req := newRequestWithChiParams(t, http.MethodPost, "/s/nonexistent/pages/1", map[string]string{
		"space_identifier": "nonexistent",
		"page_number":      "1",
	})
	req.Body = toReadCloser(form.Encode())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_PageNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("update-nopg@example.com").
		WithAtname("updatenopg").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("nopg-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	handler := setupHandler(t, queries)

	form := url.Values{}
	form.Set("title", "New Title")
	form.Set("body", "New body")
	form.Set("draft_page_digest", "")

	req := newRequestWithChiParams(t, http.MethodPost, "/s/nopg-space/pages/999", map[string]string{
		"space_identifier": "nopg-space",
		"page_number":      "999",
	})
	req.Body = toReadCloser(form.Encode())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

// toReadCloserは文字列をio.ReadCloserに変換するヘルパーです
func toReadCloser(s string) *readCloser {
	return &readCloser{Reader: strings.NewReader(s)}
}

type readCloser struct {
	*strings.Reader
}

func (rc *readCloser) Close() error { return nil }

func TestUpdate_ValidationError_BodyTooLong(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("page-body-too-long@example.com").
		WithAtname("pagebodytoolong").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-body-too-long").
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
		WithTitle("Test Page").
		Build()

	handler := setupHandler(t, queries)

	form := url.Values{}
	form.Set("title", "Test Page")
	form.Set("body", strings.Repeat("a", model.PageBodyMaxLength+1))
	form.Set("draft_page_digest", "")

	req := newRequestWithChiParams(t, http.MethodPost, "/s/page-body-too-long/pages/1", map[string]string{
		"space_identifier": "page-body-too-long",
		"page_number":      "1",
	})
	req.Body = toReadCloser(form.Encode())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}
	body := rr.Body.String()
	want := "本文は500,000文字以内で入力してください (現在500,001文字)"
	if !strings.Contains(body, want) {
		t.Errorf("レスポンスに %qが含まれていない", want)
	}

	// 本文の入力面はCodeMirrorのため、エラーはエディタの説明として関連付ける
	for _, want := range []string{
		`<div id="page-body-error" class="grid gap-2"><p class="text-sm text-error">` + want,
		`data-markdown-editor-describedby="page-body-error page-draft-save-error"`,
		`data-markdown-editor-invalid="true"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %qが含まれていない", want)
		}
	}

	// 再描画した編集画面の下書き保存は、編集画面を表示したときと同じ手動保存の送り先を指す
	for _, want := range []string{
		`hx-patch="/s/page-body-too-long/pages/1/draft_page_revision"`,
		`hx-patch="/s/page-body-too-long/pages/1/draft_page_revision?redirect_to=suggestion_new"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %qが含まれていない", want)
		}
	}
}

// publishDigestFixtureは、公開の版の照合を確かめるための下書きを持つメンバーとスペース。
// 公開のUseCaseは自前でトランザクションを管理するため、データをテストDBへ直接コミットして作る
type publishDigestFixture struct {
	identifier  string
	userID      model.UserID
	spaceID     model.SpaceID
	pageID      model.PageID
	draftPageID model.DraftPageID
}

func setupPublishDigestFixture(t *testing.T, identifier string) publishDigestFixture {
	t.Helper()

	db := testutil.GetTestDB()
	atname := strings.ReplaceAll(identifier, "-", "")
	userID := testutil.NewUserBuilderDB(t, db).WithEmail(identifier + "@example.com").WithAtname(atname).Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("General").Build()
	testutil.NewTopicMemberBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithSpaceMemberID(spaceMemberID).Build()
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("Test Page").Build()
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).WithPageID(pageID).WithSpaceMemberID(spaceMemberID).WithTopicID(topicID).
		WithTitle("Draft Title").WithBody("Draft body").Build()

	return publishDigestFixture{identifier: identifier, userID: userID, spaceID: spaceID, pageID: pageID, draftPageID: draftPageID}
}

// currentDigestは下書きの今の版を返す
func (f publishDigestFixture) currentDigest(t *testing.T) string {
	t.Helper()

	draftPage, err := repository.NewDraftPageRepository(query.New(testutil.GetTestDB())).FindByID(t.Context(), f.draftPageID, f.spaceID)
	if err != nil || draftPage == nil {
		t.Fatalf("下書きの取得 = (%v, %v)、期待値 = 下書き", draftPage, err)
	}
	return draftPage.ContentDigest()
}

// publishはuserIDのメンバーとしてformを公開のフォームから送り、記録したレスポンスを返す
func (f publishDigestFixture) publish(t *testing.T, userID model.UserID, form url.Values) *httptest.ResponseRecorder {
	t.Helper()

	req := newRequestWithChiParams(t, http.MethodPost, "/s/"+f.identifier+"/pages/1", map[string]string{
		"space_identifier": f.identifier,
		"page_number":      "1",
	})
	req.Body = toReadCloser(form.Encode())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	db := testutil.GetTestDB()
	rr := httptest.NewRecorder()
	setupHandlerWithDB(t, db, query.New(db)).Update(rr, req)
	return rr
}

// countPublishedTitleは、ページのタイトルがtitleになっている (公開された) 数を返す
func (f publishDigestFixture) countPublishedTitle(t *testing.T, title string) int {
	t.Helper()

	var n int
	if err := testutil.GetTestDB().QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM pages WHERE id = $1 AND space_id = $2 AND title = $3", f.pageID, f.spaceID, title).Scan(&n); err != nil {
		t.Fatalf("ページの数の取得: %v", err)
	}
	return n
}

func TestUpdate_DraftPageDigest(t *testing.T) {
	t.Parallel()

	f := setupPublishDigestFixture(t, "page-update-digest-match")
	rr := f.publish(t, f.userID, url.Values{
		"title":             {"Published Title"},
		"body":              {"Published body"},
		"draft_page_digest": {f.currentDigest(t)},
	})

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusSeeOther)
	}
	if n := f.countPublishedTitle(t, "Published Title"); n != 1 {
		t.Errorf("公開したタイトルのページの数 = %d、期待値 = 1", n)
	}
}

// TestUpdate_DraftPageDigestConflictは、公開の前提にした版の後に下書きが変わっていれば公開せず、
// 送信されたタイトルと本文を残した編集画面を、競合の知らせを出した状態で409で返すことを確かめる
func TestUpdate_DraftPageDigestConflict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		// digestsは送る版の項目。nilなら項目を送らない
		digests func(staleDigest string) []string
		// wantFormDigestは再表示した公開のフォームが持つ前提の版
		wantFormDigest func(staleDigest string) string
	}{
		{
			name:           "前提の版の後に下書きが別の場所で更新されていた",
			identifier:     "page-update-digest-conflict",
			digests:        func(stale string) []string { return []string{stale} },
			wantFormDigest: func(stale string) string { return stale },
		},
		{
			name:           "版の項目を送らない (版を持たない古い編集画面からの) 公開",
			identifier:     "page-update-digest-missing",
			digests:        func(string) []string { return nil },
			wantFormDigest: func(string) string { return "" },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupPublishDigestFixture(t, tt.identifier)
			staleDigest := f.currentDigest(t)
			if _, err := testutil.GetTestDB().ExecContext(t.Context(),
				"UPDATE draft_pages SET body = 'API body', modified_at = modified_at + interval '1 second' WHERE id = $1 AND space_id = $2",
				f.draftPageID, f.spaceID); err != nil {
				t.Fatalf("下書きの更新: %v", err)
			}
			currentDigest := f.currentDigest(t)

			form := url.Values{"title": {"Published Title"}, "body": {"Screen body"}}
			if digests := tt.digests(staleDigest); digests != nil {
				form["draft_page_digest"] = digests
			}
			rr := f.publish(t, f.userID, form)

			if rr.Code != http.StatusConflict {
				t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusConflict)
			}
			body := rr.Body.String()
			for _, want := range []string{
				// 送信されたタイトルと本文を画面に残す
				`value="Published Title"`,
				`data-markdown-editor-body="Screen body"`,
				// 知らせを表示した状態で描画し、上書きを選んだら公開し直させる
				`data-draft-conflict data-draft-conflict-publish data-reload-confirm=`,
				`data-reload-url="/s/` + tt.identifier + `/pages/1/edit"`,
				"下書きが別の場所で更新されています",
				// エディターは送信された版を前提にしたまま、今の版を競合の版として持つ
				`name="draft_page_digest" value="` + tt.wantFormDigest(staleDigest) + `"`,
				`data-markdown-editor-draft-page-digest="` + tt.wantFormDigest(staleDigest) + `"`,
				`data-markdown-editor-draft-conflict-digest="` + currentDigest + `"`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}
			_, conflictTag, _ := strings.Cut(body, "data-draft-conflict data-draft-conflict-publish ")
			conflictTag, _, _ = strings.Cut(conflictTag, ">")
			if strings.Contains(conflictTag, " hidden") {
				t.Error("公開の競合の知らせが隠れている")
			}
			if n := f.countPublishedTitle(t, "Published Title"); n != 0 {
				t.Errorf("公開したタイトルのページの数 = %d、期待値 = 0", n)
			}
			if got := f.currentDigest(t); got != currentDigest {
				t.Errorf("競合した公開の後の版 = %q、期待値 = 公開前の版 %q", got, currentDigest)
			}
		})
	}
}

// TestUpdate_DraftPageDigestMissing_NotEditorは、版の項目を送らない公開でも、ページを編集できない利用者には
// 編集画面を再表示せず404を返すことを確かめる
func TestUpdate_DraftPageDigestMissing_NotEditor(t *testing.T) {
	t.Parallel()

	f := setupPublishDigestFixture(t, "page-update-digest-not-editor")
	outsiderID := testutil.NewUserBuilderDB(t, testutil.GetTestDB()).
		WithEmail("page-update-digest-outsider@example.com").WithAtname("pageupdatedigestoutsider").Build()

	rr := f.publish(t, outsiderID, url.Values{"title": {"Published Title"}, "body": {"Screen body"}})

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}
