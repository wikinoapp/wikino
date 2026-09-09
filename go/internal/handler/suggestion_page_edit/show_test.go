package suggestion_page_edit_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newGetRequest(t *testing.T, path string, params map[string]string, body io.Reader) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, body)

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestShow_未ログインでサインインにリダイレクトされる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()
	handler := setupHandler(t, queries, db)

	req := newGetRequest(t, "/s/test/suggestions/1/page_edits/sp-1", map[string]string{
		"space_identifier":   "test",
		"suggestion_number":  "1",
		"suggestion_page_id": "sp-1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
	if loc := rr.Header().Get("Location"); loc != "/sign_in" {
		t.Errorf("wrong redirect location: got %q want %q", loc, "/sign_in")
	}
}

func TestShow_存在しない編集提案で404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spe-show-noexist@example.com").
		WithAtname("speshownoexist").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spe-show-noexist").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()

	handler := setupHandler(t, queries, db)

	req := newGetRequest(t, "/s/spe-show-noexist/suggestions/999/page_edits/sp-1", map[string]string{
		"space_identifier":   "spe-show-noexist",
		"suggestion_number":  "999",
		"suggestion_page_id": "sp-1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "speshownoexist"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_存在しないSuggestionPageIDで404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spe-show-nosp@example.com").
		WithAtname("speshownosp").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spe-show-nosp").
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
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, queries, db)

	req := newGetRequest(t, "/s/spe-show-nosp/suggestions/1/page_edits/nonexistent-sp-id", map[string]string{
		"space_identifier":   "spe-show-nosp",
		"suggestion_number":  "1",
		"suggestion_page_id": "nonexistent-sp-id",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "speshownosp"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_スペースメンバーでないユーザーは403が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("spe-show-owner@example.com").
		WithAtname("speshowowner").
		Build()
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("spe-show-nonmem@example.com").
		WithAtname("speshownonmem").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spe-show-forbid").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		Build()
	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	suggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		Build()

	handler := setupHandler(t, queries, db)

	req := newGetRequest(t, "/s/spe-show-forbid/suggestions/1/page_edits/"+string(suggestionPageID), map[string]string{
		"space_identifier":   "spe-show-forbid",
		"suggestion_number":  "1",
		"suggestion_page_id": string(suggestionPageID),
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: nonMemberID, Atname: "speshownonmem"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusForbidden)
	}
}

func TestShow_クローズ済み提案は変更差分画面にリダイレクトされる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spe-show-closed@example.com").
		WithAtname("speshowclosed").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spe-show-closed").
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
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusClosed).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		Build()
	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	suggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		Build()

	handler := setupHandler(t, queries, db)

	req := newGetRequest(t, "/s/spe-show-closed/suggestions/1/page_edits/"+string(suggestionPageID), map[string]string{
		"space_identifier":   "spe-show-closed",
		"suggestion_number":  "1",
		"suggestion_page_id": string(suggestionPageID),
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "speshowclosed"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	expected := "/s/spe-show-closed/suggestions/1/changes"
	if loc != expected {
		t.Errorf("wrong redirect location: got %q want %q", loc, expected)
	}
}

func TestShow_正常に確認画面が表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spe-show-ok@example.com").
		WithAtname("speshowok").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spe-show-ok-sp").
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
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		Build()
	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	suggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		WithTitle("確認テストページ").
		Build()

	handler := setupHandler(t, queries, db)

	req := newGetRequest(t, "/s/spe-show-ok-sp/suggestions/1/page_edits/"+string(suggestionPageID), map[string]string{
		"space_identifier":   "spe-show-ok-sp",
		"suggestion_number":  "1",
		"suggestion_page_id": string(suggestionPageID),
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "speshowok"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

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
