package suggestion_page_revision_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	suggestionpagerevisionhandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion_page_revision"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
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

func setupHandler(t *testing.T, queries *query.Queries, db *sql.DB) *suggestionpagerevisionhandler.Handler {
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
		suggestionRepo, suggestionPageRepo, suggestionCommentRepo, userRepo,
	)
	updateSuggestionPageUC := usecase.NewUpdateSuggestionPageUsecase(
		db, suggestionPageRepo, suggestionPageRevisionRepo, draftPageRepo, pageRepo,
	)

	return suggestionpagerevisionhandler.NewHandler(
		flashMgr,
		getSuggestionDetailUC,
		updateSuggestionPageUC,
	)
}

func TestCreate_未ログインでサインインにリダイレクトされる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()
	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")
	form.Set("page_number", "1")

	req := newPostRequest(t, "/s/test/suggestions/1/page_revisions", map[string]string{
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

func TestCreate_スペースメンバーでないユーザーは403が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("spr-forbid-owner@example.com").
		WithAtname("sprforbidowner").
		Build()
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("spr-forbid-nonmem@example.com").
		WithAtname("sprforbidnonm").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spr-forbid-sp").
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
	form.Set("page_number", "1")

	req := newPostRequest(t, "/s/spr-forbid-sp/suggestions/1/page_revisions", map[string]string{
		"space_identifier":  "spr-forbid-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: nonMemberID, Atname: "sprforbidnonm"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusForbidden)
	}
}

func TestCreate_正常に編集提案ページが更新される(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	queries := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("spr-create-ok@example.com").
		WithAtname("sprcreateok").
		Build()

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("spr-create-sp").
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
	form.Set("page_number", "3")

	req := newPostRequest(t, "/s/spr-create-sp/suggestions/1/page_revisions", map[string]string{
		"space_identifier":  "spr-create-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "sprcreateok"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	expectedLoc := "/s/spr-create-sp/suggestions/1/changes"
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

	// DraftPageのsuggestion_page_idがクリアされたことを確認
	draftPageRepo := repository.NewDraftPageRepository(queries)
	dp, err := draftPageRepo.FindByPageAndMember(context.Background(), pageID, spaceMemberID, spaceID)
	if err != nil {
		t.Fatalf("DraftPageの取得に失敗: %v", err)
	}
	if dp != nil && dp.SuggestionPageID != nil {
		t.Errorf("DraftPage.SuggestionPageID should be nil, got %v", *dp.SuggestionPageID)
	}
}

func TestCreate_クローズ済みの編集提案は更新できない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spr-closed@example.com").
		WithAtname("sprclosed").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spr-closed-sp").
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
	form.Set("page_number", "1")

	req := newPostRequest(t, "/s/spr-closed-sp/suggestions/1/page_revisions", map[string]string{
		"space_identifier":  "spr-closed-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "sprclosed"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "/suggestions/1") {
		t.Errorf("should redirect to suggestion page, got %q", loc)
	}
}
