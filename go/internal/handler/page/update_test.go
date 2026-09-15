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

	// 編集フォームが再表示されること
	if !strings.Contains(body, `name="title"`) {
		t.Error("レスポンスにタイトルの入力欄が見つからない")
	}

	// 本文の入力値が保持されていること
	if !strings.Contains(body, "Updated body") {
		t.Error("レスポンスで本文の値が保持されていない")
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
