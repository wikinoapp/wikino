package suggestion_page_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	suggestionpagehandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion_page"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newPatchRequest(t *testing.T, path string, params map[string]string, form url.Values) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func setupHandler(t *testing.T, queries *query.Queries, db *sql.DB) *suggestionpagehandler.Handler {
	t.Helper()

	flashMgr := session.NewFlashManager("localhost", false, false)

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	suggestionRepo := repository.NewSuggestionRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	suggestionPageRevisionRepo := repository.NewSuggestionPageRevisionRepository(queries)
	suggestionCommentRepo := repository.NewSuggestionCommentRepository(queries)
	userRepo := repository.NewUserRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)

	pageRepo := repository.NewPageRepository(queries)
	getSuggestionDetailUC := usecase.NewGetSuggestionDetailUsecase(
		spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo,
		suggestionRepo, suggestionPageRepo, suggestionCommentRepo, pageRepo, userRepo,
	)
	updateSuggestionPageUC := usecase.NewUpdateSuggestionPageUsecase(
		db, suggestionPageRepo, suggestionPageRevisionRepo,
	)
	updateValidator := validator.NewSuggestionPageUpdateValidator(draftPageRepo)

	return suggestionpagehandler.NewHandler(
		flashMgr,
		getSuggestionDetailUC,
		updateSuggestionPageUC,
		updateValidator,
	)
}

func TestUpdate_未ログインでサインインにリダイレクトされる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()
	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPatchRequest(t, "/s/test/suggestions/1/suggestion_pages/test-sp-id", map[string]string{
		"space_identifier":   "test",
		"suggestion_number":  "1",
		"suggestion_page_id": "test-sp-id",
	}, form)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
	if loc := rr.Header().Get("Location"); loc != "/sign_in" {
		t.Errorf("wrong redirect location: got %q want %q", loc, "/sign_in")
	}
}

func TestUpdate_スペースメンバーでないユーザーは403が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-forbid-owner@example.com").
		WithAtname("spforbidowner").
		Build()
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-forbid-nonmem@example.com").
		WithAtname("spforbidnonm").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-forbid-sp").
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

	req := newPatchRequest(t, "/s/sp-forbid-sp/suggestions/1/suggestion_pages/"+string(suggestionPageID), map[string]string{
		"space_identifier":   "sp-forbid-sp",
		"suggestion_number":  "1",
		"suggestion_page_id": string(suggestionPageID),
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: nonMemberID, Atname: "spforbidnonm"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusForbidden)
	}
}

func TestUpdate_正常に編集提案ページが更新される(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	queries := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("sp-update-ok@example.com").
		WithAtname("spupdateok").
		Build()

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("sp-update-sp").
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
		WithNumber(3).
		WithTitle("Test Page").
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
		WithBody("元の提案本文").
		Build()

	// 編集提案にリンクされた下書きを作成
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithSuggestionPageID(suggestionPageID).
		WithBody("更新された本文").
		WithTitle("更新されたタイトル").
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPatchRequest(t, "/s/sp-update-sp/suggestions/1/suggestion_pages/"+string(suggestionPageID), map[string]string{
		"space_identifier":   "sp-update-sp",
		"suggestion_number":  "1",
		"suggestion_page_id": string(suggestionPageID),
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "spupdateok"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	expectedLoc := "/s/sp-update-sp/suggestions/1/changes"
	if loc := rr.Header().Get("Location"); loc != expectedLoc {
		t.Errorf("wrong redirect location: got %q want %q", loc, expectedLoc)
	}

	// SuggestionPageが更新されたことを確認
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	updatedSP, err := suggestionPageRepo.FindByID(context.Background(), suggestionPageID, spaceID)
	if err != nil {
		t.Fatalf("SuggestionPageの取得に失敗: %v", err)
	}
	if updatedSP.Body != "更新された本文" {
		t.Errorf("SuggestionPage.Body = %q, want %q", updatedSP.Body, "更新された本文")
	}

	// SuggestionPageRevisionが作成されたことを確認
	revisionRepo := repository.NewSuggestionPageRevisionRepository(queries)
	revisions, err := revisionRepo.ListBySuggestionPageID(context.Background(), suggestionPageID, spaceID)
	if err != nil {
		t.Fatalf("SuggestionPageRevisionの取得に失敗: %v", err)
	}
	if len(revisions) == 0 {
		t.Error("SuggestionPageRevisionが作成されていません")
	}

	// DraftPageのsuggestion_page_idがクリアされていないことを確認
	draftPageRepo := repository.NewDraftPageRepository(queries)
	dp, err := draftPageRepo.FindByPageAndMember(context.Background(), pageID, spaceMemberID, spaceID)
	if err != nil {
		t.Fatalf("DraftPageの取得に失敗: %v", err)
	}
	if dp == nil {
		t.Fatal("DraftPageが見つかりません")
	}
	if dp.SuggestionPageID == nil {
		t.Error("DraftPage.SuggestionPageID should not be nil")
	}
	if dp.SuggestionPageID != nil && *dp.SuggestionPageID != suggestionPageID {
		t.Errorf("DraftPage.SuggestionPageID = %q, want %q", *dp.SuggestionPageID, suggestionPageID)
	}
}

func TestUpdate_下書きステータスの編集提案ページが更新される(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	queries := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("sp-draft-ok@example.com").
		WithAtname("spdraftok").
		Build()

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("sp-draft-sp").
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
		WithStatus(model.SuggestionStatusDraft).
		Build()

	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Draft Test Page").
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
		WithBody("元の提案本文").
		Build()

	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithSuggestionPageID(suggestionPageID).
		WithBody("下書き更新された本文").
		WithTitle("下書き更新されたタイトル").
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPatchRequest(t, "/s/sp-draft-sp/suggestions/1/suggestion_pages/"+string(suggestionPageID), map[string]string{
		"space_identifier":   "sp-draft-sp",
		"suggestion_number":  "1",
		"suggestion_page_id": string(suggestionPageID),
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "spdraftok"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	expectedLoc := "/s/sp-draft-sp/suggestions/1/changes"
	if loc := rr.Header().Get("Location"); loc != expectedLoc {
		t.Errorf("wrong redirect location: got %q want %q", loc, expectedLoc)
	}

	// SuggestionPageが更新されたことを確認
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	updatedSP, err := suggestionPageRepo.FindByID(context.Background(), suggestionPageID, spaceID)
	if err != nil {
		t.Fatalf("SuggestionPageの取得に失敗: %v", err)
	}
	if updatedSP.Body != "下書き更新された本文" {
		t.Errorf("SuggestionPage.Body = %q, want %q", updatedSP.Body, "下書き更新された本文")
	}
}

func TestUpdate_反映済みの編集提案は更新できない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-applied@example.com").
		WithAtname("spapplied").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-applied-sp").
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
		WithStatus(model.SuggestionStatusApplied).
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPatchRequest(t, "/s/sp-applied-sp/suggestions/1/suggestion_pages/test-sp-id", map[string]string{
		"space_identifier":   "sp-applied-sp",
		"suggestion_number":  "1",
		"suggestion_page_id": "test-sp-id",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "spapplied"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "/suggestions/1") {
		t.Errorf("should redirect to suggestion page, got %q", loc)
	}
}

func TestUpdate_クローズ済みの編集提案は更新できない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-closed@example.com").
		WithAtname("spclosed").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-closed-sp").
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

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPatchRequest(t, "/s/sp-closed-sp/suggestions/1/suggestion_pages/test-sp-id", map[string]string{
		"space_identifier":   "sp-closed-sp",
		"suggestion_number":  "1",
		"suggestion_page_id": "test-sp-id",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "spclosed"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "/suggestions/1") {
		t.Errorf("should redirect to suggestion page, got %q", loc)
	}
}
