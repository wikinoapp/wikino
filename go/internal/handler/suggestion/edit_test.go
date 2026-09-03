package suggestion_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestEdit_未ログインでリダイレクトされる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/test/suggestions/1/edit", map[string]string{
		"space_identifier":  "test",
		"suggestion_number": "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
}

func TestEdit_存在しない編集提案で404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("edit-404@example.com").
		WithAtname("edit404").
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/nonexistent/suggestions/999/edit", map[string]string{
		"space_identifier":  "nonexistent",
		"suggestion_number": "999",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "edit404"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestEdit_編集権限がある場合にフォームが表示される(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("edit-ok@example.com").
		WithAtname("editok").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("edit-ok-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		Build()
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("編集テスト提案").
		WithBody("テスト本文").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/edit-ok-sp/suggestions/1/edit", map[string]string{
		"space_identifier":  "edit-ok-sp",
		"suggestion_number": "1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "editok"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "編集テスト提案") {
		t.Error("response should contain suggestion title")
	}
	if !strings.Contains(body, "テスト本文") {
		t.Error("response should contain suggestion body")
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

	// The edit form shares DetailBreadcrumbHeaderData with the public change diff, but it requires
	// authentication and carries no canonical URL, so it must stay opted out of BreadcrumbList
	// JSON-LD. Moving the opt-in into the shared helper would publish this screen's labels and URLs
	// as machine-readable data.
	//
	// [Ja] 編集フォームは公開の変更差分と DetailBreadcrumbHeaderData を共有するが、認証必須で canonical
	// も持たないため BreadcrumbList JSON-LD の対象外のままでなければならない。オプトインを共有ヘルパーへ
	// 移すと、この画面のラベルと URL が機械可読なデータとして出てしまう。
	for _, notWant := range []string{
		"application/ld+json",
		"BreadcrumbList",
	} {
		if strings.Contains(body, notWant) {
			t.Errorf("認証必須の画面が構造化データを出している: %q", notWant)
		}
	}
}
