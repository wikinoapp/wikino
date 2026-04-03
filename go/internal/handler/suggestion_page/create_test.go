package suggestion_page_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newPostRequest(t *testing.T, path string, params map[string]string, form url.Values) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestCreate_未ログインでサインインにリダイレクトされる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()
	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/test/suggestions/1/suggestion_pages", map[string]string{
		"space_identifier":  "test",
		"suggestion_number": "1",
	}, form)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
	if loc := rr.Header().Get("Location"); loc != "/sign_in" {
		t.Errorf("wrong redirect location: got %q want %q", loc, "/sign_in")
	}
}

func TestCreate_スペースメンバーでないユーザーは404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-cr-forbid-own@example.com").
		WithAtname("spcrforbidow").
		Build()
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-cr-forbid-nonm@example.com").
		WithAtname("spcrforbidnm").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-cr-forbid").
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

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/sp-cr-forbid/suggestions/1/suggestion_pages", map[string]string{
		"space_identifier":  "sp-cr-forbid",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: nonMemberID, Atname: "spcrforbidnm"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestCreate_正常にページが追加される(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	queries := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("sp-cr-ok@example.com").
		WithAtname("spcrok").
		Build()

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("sp-cr-ok").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithName("Topic").
		Build()
	suggestionID := testutil.NewSuggestionBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()

	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()
	testutil.NewPageRevisionBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	// 編集提案にリンクされていない下書きを作成
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithBody("下書き本文").
		WithTitle("下書きタイトル").
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")
	form.Add("draft_page_ids", string(draftPageID))

	req := newPostRequest(t, "/s/sp-cr-ok/suggestions/1/suggestion_pages", map[string]string{
		"space_identifier":  "sp-cr-ok",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "spcrok"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	expectedLoc := "/s/sp-cr-ok/suggestions/1/changes"
	if loc := rr.Header().Get("Location"); loc != expectedLoc {
		t.Errorf("wrong redirect location: got %q want %q", loc, expectedLoc)
	}

	// SuggestionPageが作成されたことを確認
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	pages, err := suggestionPageRepo.ListBySuggestionID(context.Background(), suggestionID, spaceID)
	if err != nil {
		t.Fatalf("SuggestionPageの取得に失敗: %v", err)
	}
	if len(pages) != 1 {
		t.Errorf("SuggestionPage count = %d, want 1", len(pages))
	}
}

func TestCreate_下書きページ未選択でバリデーションエラーが返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-cr-nosel@example.com").
		WithAtname("spcrnosel").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-cr-nosel").
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

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/sp-cr-nosel/suggestions/1/suggestion_pages", map[string]string{
		"space_identifier":  "sp-cr-nosel",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "spcrnosel"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}
}
