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

func newDeleteRequest(t *testing.T, path string, params map[string]string, form url.Values) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodDelete, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestDelete_未ログインでサインインにリダイレクトされる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()
	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newDeleteRequest(t, "/s/test/suggestions/1/suggestion_pages/test-sp-id", map[string]string{
		"space_identifier":   "test",
		"suggestion_number":  "1",
		"suggestion_page_id": "test-sp-id",
	}, form)

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
	if loc := rr.Header().Get("Location"); loc != "/sign_in" {
		t.Errorf("wrong redirect location: got %q want %q", loc, "/sign_in")
	}
}

func TestDelete_スペースメンバーでないユーザーは404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("del-sp-forbid-owner@example.com").
		WithAtname("delspforbown").
		Build()
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("del-sp-forbid-nonmem@example.com").
		WithAtname("delspforbno").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("del-sp-forbid").
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

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newDeleteRequest(t, "/s/del-sp-forbid/suggestions/1/suggestion_pages/"+string(suggestionPageID), map[string]string{
		"space_identifier":   "del-sp-forbid",
		"suggestion_number":  "1",
		"suggestion_page_id": string(suggestionPageID),
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: nonMemberID, Atname: "delspforbno"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestDelete_正常に編集提案ページが削除される(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	queries := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("del-sp-ok@example.com").
		WithAtname("delspok").
		Build()

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("del-sp-ok-sp").
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

	// 2つのSuggestionPageを作成
	pageID1 := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Page 1").
		Build()
	pageRevisionID1 := testutil.NewPageRevisionBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID1).
		WithSpaceMemberID(spaceMemberID).
		Build()
	suggestionPageID1 := testutil.NewSuggestionPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID1).
		WithPageRevisionID(pageRevisionID1).
		Build()

	pageID2 := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		Build()
	pageRevisionID2 := testutil.NewPageRevisionBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID2).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewSuggestionPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID2).
		WithPageRevisionID(pageRevisionID2).
		Build()

	// 削除対象のSuggestionPageに紐づくDraftPageを作成
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID1).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithSuggestionPageID(suggestionPageID1).
		WithBody("下書き").
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newDeleteRequest(t, "/s/del-sp-ok-sp/suggestions/1/suggestion_pages/"+string(suggestionPageID1), map[string]string{
		"space_identifier":   "del-sp-ok-sp",
		"suggestion_number":  "1",
		"suggestion_page_id": string(suggestionPageID1),
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "delspok"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	expectedLoc := "/s/del-sp-ok-sp/suggestions/1/changes"
	if loc := rr.Header().Get("Location"); loc != expectedLoc {
		t.Errorf("wrong redirect location: got %q want %q", loc, expectedLoc)
	}

	// SuggestionPageが削除されたことを確認
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	deletedSP, err := suggestionPageRepo.FindByID(context.Background(), suggestionPageID1, spaceID)
	if err != nil {
		t.Fatalf("SuggestionPageの取得に失敗: %v", err)
	}
	if deletedSP != nil {
		t.Error("SuggestionPageが削除されていません")
	}

	// DraftPageのsuggestion_page_idがクリアされたことを確認
	draftPageRepo := repository.NewDraftPageRepository(queries)
	dp, err := draftPageRepo.FindByID(context.Background(), draftPageID, spaceID)
	if err != nil {
		t.Fatalf("DraftPageの取得に失敗: %v", err)
	}
	if dp == nil {
		t.Fatal("DraftPageが見つかりません")
	}
	if dp.SuggestionPageID != nil {
		t.Error("DraftPage.SuggestionPageID should be nil after deletion")
	}
}

func TestDelete_最後の1ページは削除できずリダイレクトされる(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	queries := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("del-sp-lastpg@example.com").
		WithAtname("delsplastpg").
		Build()

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("del-sp-lastpg").
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
		Build()
	pageRevisionID := testutil.NewPageRevisionBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newDeleteRequest(t, "/s/del-sp-lastpg/suggestions/1/suggestion_pages/"+string(suggestionPageID), map[string]string{
		"space_identifier":   "del-sp-lastpg",
		"suggestion_number":  "1",
		"suggestion_page_id": string(suggestionPageID),
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "delsplastpg"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	// Conflictエラー → フラッシュメッセージ付きリダイレクト
	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	expectedLoc := "/s/del-sp-lastpg/suggestions/1/changes"
	if loc := rr.Header().Get("Location"); loc != expectedLoc {
		t.Errorf("wrong redirect location: got %q want %q", loc, expectedLoc)
	}

	// SuggestionPageが削除されていないことを確認
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	sp, err := suggestionPageRepo.FindByID(context.Background(), suggestionPageID, spaceID)
	if err != nil {
		t.Fatalf("SuggestionPageの取得に失敗: %v", err)
	}
	if sp == nil {
		t.Error("SuggestionPageが不正に削除されています")
	}
}
