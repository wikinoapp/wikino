package suggestion_page_edit_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	suggestionpageedithandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion_page_edit"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
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

func setupHandler(t *testing.T, queries *query.Queries, db *sql.DB) *suggestionpageedithandler.Handler {
	t.Helper()

	cfg := &config.Config{Domain: "localhost"}
	flashMgr := session.NewFlashManager("localhost", false, false)

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	suggestionRepo := repository.NewSuggestionRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	suggestionCommentRepo := repository.NewSuggestionCommentRepository(queries)
	userRepo := repository.NewUserRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	pageRepo := repository.NewPageRepository(queries)

	getSuggestionDetailUC := usecase.NewGetSuggestionDetailUsecase(
		spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo,
		suggestionRepo, suggestionPageRepo, suggestionCommentRepo, pageRepo, userRepo,
	)
	startSuggestionPageEditUC := usecase.NewStartSuggestionPageEditUsecase(
		db, spaceRepo, spaceMemberRepo, topicMemberRepo, suggestionRepo, suggestionPageRepo, draftPageRepo, pageRepo,
	)

	sidebarHelper := sidebar.NewHelper(topicRepo, draftPageRepo)

	return suggestionpageedithandler.NewHandler(
		cfg,
		flashMgr,
		getSuggestionDetailUC,
		startSuggestionPageEditUC,
		sidebarHelper,
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
	form.Set("suggestion_page_id", "test-id")

	req := newPostRequest(t, "/s/test/suggestions/1/page_edits", map[string]string{
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
		WithEmail("spe-forbid-owner@example.com").
		WithAtname("speforbidowner").
		Build()
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("spe-forbid-nonmember@example.com").
		WithAtname("speforbidnonm").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spe-forbid-sp").
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
	form.Set("suggestion_page_id", string(suggestionPageID))

	req := newPostRequest(t, "/s/spe-forbid-sp/suggestions/1/page_edits", map[string]string{
		"space_identifier":  "spe-forbid-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: nonMemberID, Atname: "speforbidnonm"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusForbidden)
	}
}

func TestCreate_下書きなしの場合にページ編集画面にリダイレクトされる(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	queries := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("spe-create-ok@example.com").
		WithAtname("specreateok").
		Build()

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("spe-create-sp").
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
		WithNumber(5).
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
		WithBody("提案本文").
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")
	form.Set("suggestion_page_id", string(suggestionPageID))

	req := newPostRequest(t, "/s/spe-create-sp/suggestions/1/page_edits", map[string]string{
		"space_identifier":  "spe-create-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "specreateok"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if loc != "/s/spe-create-sp/pages/5/edit" {
		t.Errorf("wrong redirect location: got %q want %q", loc, "/s/spe-create-sp/pages/5/edit")
	}
}

func TestCreate_通常編集の下書きがある場合に確認画面にリダイレクトされる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spe-conflict@example.com").
		WithAtname("speconflict").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spe-conflict-sp").
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
		Build()

	// 通常編集の下書きを作成
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithBody("通常の下書き").
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")
	form.Set("suggestion_page_id", string(suggestionPageID))

	req := newPostRequest(t, "/s/spe-conflict-sp/suggestions/1/page_edits", map[string]string{
		"space_identifier":  "spe-conflict-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "speconflict"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	expectedPrefix := "/s/spe-conflict-sp/suggestions/1/page_edits/"
	if !strings.HasPrefix(loc, expectedPrefix) {
		t.Errorf("wrong redirect location: got %q, want prefix %q", loc, expectedPrefix)
	}
}

func TestCreate_別の編集提案の下書きがある場合に確認画面にリダイレクトされる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spe-other-sug@example.com").
		WithAtname("speothersug").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spe-other-sg-sp").
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

	// 別の編集提案を作成
	otherSuggestionID := testutil.NewSuggestionBuilder(t, tx).
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

	// 別の編集提案のSuggestionPageを作成
	otherSuggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(otherSuggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		Build()

	// 別の編集提案にリンクされた下書きを作成
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithSuggestionPageID(otherSuggestionPageID).
		WithBody("別の編集提案の下書き").
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")
	form.Set("suggestion_page_id", string(suggestionPageID))

	req := newPostRequest(t, "/s/spe-other-sg-sp/suggestions/1/page_edits", map[string]string{
		"space_identifier":  "spe-other-sg-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "speothersug"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	expectedPrefix := "/s/spe-other-sg-sp/suggestions/1/page_edits/"
	if !strings.HasPrefix(loc, expectedPrefix) {
		t.Errorf("wrong redirect location: got %q, want prefix %q", loc, expectedPrefix)
	}
}
