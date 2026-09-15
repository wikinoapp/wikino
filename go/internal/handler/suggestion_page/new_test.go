package suggestion_page_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newGetRequest(t *testing.T, path string, params map[string]string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestNew_未ログインでサインインにリダイレクトされる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()
	handler := setupHandler(t, queries, db)

	req := newGetRequest(t, "/s/test/suggestions/1/suggestion_pages/new", map[string]string{
		"space_identifier":  "test",
		"suggestion_number": "1",
	})

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}
	if loc := rr.Header().Get("Location"); loc != "/sign_in" {
		t.Errorf("リダイレクト先 = %q、期待値 = %q", loc, "/sign_in")
	}
}

func TestNew_存在しないスペースで404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-new-notfound@example.com").
		WithAtname("spnewnotfound").
		Build()

	handler := setupHandler(t, queries, db)

	req := newGetRequest(t, "/s/nonexistent/suggestions/1/suggestion_pages/new", map[string]string{
		"space_identifier":  "nonexistent",
		"suggestion_number": "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "spnewnotfound"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestNew_スペースメンバーでないユーザーは404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-new-forbid-owner@example.com").
		WithAtname("spnewforbidowner").
		Build()
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-new-forbid-nonmem@example.com").
		WithAtname("spnewforbidnonm").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-new-forbid").
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
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, queries, db)

	req := newGetRequest(t, "/s/sp-new-forbid/suggestions/1/suggestion_pages/new", map[string]string{
		"space_identifier":  "sp-new-forbid",
		"suggestion_number": "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: nonMemberID, Atname: "spnewforbidnonm"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestNew_正常にフォームが表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-new-ok@example.com").
		WithAtname("spnewok").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-new-ok").
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

	req := newGetRequest(t, "/s/sp-new-ok/suggestions/1/suggestion_pages/new", map[string]string{
		"space_identifier":  "sp-new-ok",
		"suggestion_number": "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "spnewok"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

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

func TestNew_クローズ済みの編集提案は404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-new-closed@example.com").
		WithAtname("spnewclosed").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-new-closed").
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
		WithStatus(model.SuggestionStatusClosed).
		Build()

	handler := setupHandler(t, queries, db)

	req := newGetRequest(t, "/s/sp-new-closed/suggestions/1/suggestion_pages/new", map[string]string{
		"space_identifier":  "sp-new-closed",
		"suggestion_number": "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "spnewclosed"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}
