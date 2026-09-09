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
	// Usecaseの動作は usecase/publish_page_test.go でテストされている。
	t.Skip("Usecase uses separate transaction, tested in usecase package")
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
	// Send the Zen mode cookie to verify the validation-error re-render also keeps Zen mode on.
	// [Ja] バリデーションエラーの再描画でも Zenモードが維持されることを検証するため、Zenモードクッキーを送る。
	req.AddCookie(&http.Cookie{Name: "wikino_zen_mode", Value: "1"})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// バリデーションエラーで422が返ること
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()

	// タイトル必須エラーメッセージが表示されること
	if !strings.Contains(body, "タイトルを入力してください") {
		t.Error("title required error message not found in response")
	}

	// 編集フォームが再表示されること
	if !strings.Contains(body, `name="title"`) {
		t.Error("title input not found in response")
	}

	// 本文の入力値が保持されていること
	if !strings.Contains(body, "Updated body") {
		t.Error("body value not preserved in response")
	}

	// パンくずリストが表示されていること
	if !strings.Contains(body, "General") {
		t.Error("topic name not found in breadcrumb")
	}

	// The validation-error re-render supplies the same breadcrumb header as the editor itself, so
	// it renders outside <main> with the editor's max-w-6xl content width.
	//
	// [Ja] バリデーションエラーの再描画でも編集画面と同じパンくずヘッダーを供給するため、
	// <main> の外に編集画面の本文幅 max-w-6xl で描画される。
	if !strings.Contains(body, `<div class="max-w-6xl mx-auto flex w-full items-center justify-between gap-2 px-4">`) {
		t.Error("shared breadcrumb header should keep the max-w-6xl content width after validation error re-render")
	}
	header, main := strings.Index(body, "<header"), strings.Index(body, `<main id="main" tabindex="-1">`)
	if header == -1 || main == -1 || header > main {
		t.Errorf("shared breadcrumb header (index %d) must precede <main> (index %d)", header, main)
	}

	// The re-rendered editor keeps the Zen mode class read from the cookie. Assert on the full
	// class attribute because the bare "page-edit-zen" substring also matches the always-present
	// Tailwind variant classes (in-[.page-edit-zen]:lg:hidden etc.).
	//
	// [Ja] 再描画されたエディタにクッキー由来の Zenモードクラスが付くこと。"page-edit-zen" の
	// 部分一致では常に存在する Tailwind バリアントクラス (in-[.page-edit-zen]:lg:hidden など) にも
	// マッチしてしまうため、class 属性全体で検証する。
	if !strings.Contains(body, `class="max-w-6xl w-full mx-auto lg:px-4 page-edit-zen"`) {
		t.Error("zen mode class not found on the editor container after validation error re-render")
	}

	// The validation-error re-render keeps the global navigation, so the top bar and the bottom bar
	// still swap at md after a failed submission.
	//
	// [Ja] バリデーションエラーの再描画でもグローバルナビを維持し、送信失敗後も上部バーと下部バーが
	// md で入れ替わること。
	if !strings.Contains(body, `<nav class="shrink-0 hidden md:flex"`) {
		t.Error("global navigation top bar should switch at md after validation error re-render")
	}
	if !strings.Contains(body, `<nav class="md:hidden"`) {
		t.Error("global navigation bottom bar should switch at md after validation error re-render")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()

	// タイトル禁止文字エラーが表示されること
	if !strings.Contains(body, `aria-invalid="true"`) {
		t.Error("aria-invalid attribute not found in response")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()

	// 重複エラーメッセージ（HTMLリンク付き）が表示されること
	if !strings.Contains(body, "/s/dup-space/pages/1/edit") {
		t.Error("edit link for existing page not found in response")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
	location := rr.Header().Get("Location")
	if location != "/sign_in" {
		t.Errorf("wrong redirect location: got %v want /sign_in", location)
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

// toReadCloser は文字列をio.ReadCloserに変換するヘルパーです
func toReadCloser(s string) *readCloser {
	return &readCloser{Reader: strings.NewReader(s)}
}

type readCloser struct {
	*strings.Reader
}

func (rc *readCloser) Close() error { return nil }
