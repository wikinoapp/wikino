package suggestion_apply_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	suggestionapplyhandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion_apply"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// newPostRequest はchiのURLパラメータ付きPOSTリクエストを作成するヘルパーです
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

// setupHandler はテスト用の編集提案反映ハンドラーを作成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries, db *sql.DB) *suggestionapplyhandler.Handler {
	t.Helper()

	flashMgr := session.NewFlashManager("localhost", false, false)

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	suggestionRepo := repository.NewSuggestionRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	pageRevisionRepo := repository.NewPageRevisionRepository(queries)
	pageEditorRepo := repository.NewPageEditorRepository(queries)
	attachmentRepo := repository.NewAttachmentRepository(queries)
	pageAttachmentRefRepo := repository.NewPageAttachmentReferenceRepository(queries)

	draftPageRepo := repository.NewDraftPageRepository(queries)
	applySuggestionUC := usecase.NewApplySuggestionUsecase(
		db, spaceRepo, spaceMemberRepo, topicMemberRepo,
		suggestionRepo, suggestionPageRepo, pageRepo, pageRevisionRepo,
		pageEditorRepo, attachmentRepo, pageAttachmentRefRepo, draftPageRepo,
	)

	return suggestionapplyhandler.NewHandler(
		flashMgr,
		applySuggestionUC,
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

	req := newPostRequest(t, "/s/test/suggestions/1/apply", map[string]string{
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

func TestCreate_存在しない編集提案で404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("apply-nosugg@example.com").
		WithAtname("applynosugg").
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/nonexistent/suggestions/999/apply", map[string]string{
		"space_identifier":  "nonexistent",
		"suggestion_number": "999",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "applynosugg"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestCreate_スペースメンバーでないユーザーは403が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("apply-owner@example.com").
		WithAtname("applyowner").
		Build()
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("apply-nonmember@example.com").
		WithAtname("applynonmember").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("apply-forbid-sp").
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

	req := newPostRequest(t, "/s/apply-forbid-sp/suggestions/1/apply", map[string]string{
		"space_identifier":  "apply-forbid-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: nonMemberID, Atname: "applynonmember"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusForbidden)
	}
}

func TestCreate_一般メンバーは403が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("apply-ownerX@example.com").
		WithAtname("applyownerX").
		Build()
	memberID := testutil.NewUserBuilder(t, tx).
		WithEmail("apply-member@example.com").
		WithAtname("applymember").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("apply-member-sp").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(memberID).
		WithRole(int32(model.SpaceMemberRoleMember)).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		WithRole(int32(model.TopicMemberRoleMember)).
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

	req := newPostRequest(t, "/s/apply-member-sp/suggestions/1/apply", map[string]string{
		"space_identifier":  "apply-member-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: memberID, Atname: "applymember"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusForbidden)
	}
}

func TestCreate_スペースオーナーが反映できる(t *testing.T) {
	t.Parallel()

	// usecaseが独自トランザクションを管理するためDB直接書き込みを使用
	db := testutil.GetTestDB()
	queries := query.New(db)

	ownerID := testutil.NewUserBuilderDB(t, db).
		WithEmail("apply-ok-owner@example.com").
		WithAtname("applyokowner").
		Build()

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("apply-ok-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
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
	suggestionID := testutil.NewSuggestionBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()
	testutil.NewSuggestionPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		WithTitle("更新タイトル").
		WithBody("更新本文").
		WithBodyHTML("<p>更新本文</p>").
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/apply-ok-sp/suggestions/1/apply", map[string]string{
		"space_identifier":  "apply-ok-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "applyokowner"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if loc != "/s/apply-ok-sp/suggestions/1" {
		t.Errorf("wrong redirect location: got %q", loc)
	}
}

func TestCreate_クローズ済みの編集提案はエラーリダイレクトされる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("apply-closed@example.com").
		WithAtname("applyclosed").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("apply-closed-sp").
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
		WithStatus(model.SuggestionStatusClosed).
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/apply-closed-sp/suggestions/1/apply", map[string]string{
		"space_identifier":  "apply-closed-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "applyclosed"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if loc != "/s/apply-closed-sp/suggestions/1" {
		t.Errorf("wrong redirect location: got %q", loc)
	}
}

func TestCreate_反映済みの編集提案はべき等に成功する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("apply-idem@example.com").
		WithAtname("applyidem").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("apply-idem-sp").
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
		WithStatus(model.SuggestionStatusApplied).
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/apply-idem-sp/suggestions/1/apply", map[string]string{
		"space_identifier":  "apply-idem-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "applyidem"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if loc != "/s/apply-idem-sp/suggestions/1" {
		t.Errorf("wrong redirect location: got %q", loc)
	}
}
