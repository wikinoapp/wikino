package suggestion_comment_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	suggestioncommenthandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion_comment"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
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

// setupHandler はテスト用の編集提案コメントハンドラーを作成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *suggestioncommenthandler.Handler {
	t.Helper()

	flashMgr := session.NewFlashManager("localhost", false, false)

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	suggestionRepo := repository.NewSuggestionRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	suggestionCommentRepo := repository.NewSuggestionCommentRepository(queries)
	userRepo := repository.NewUserRepository(queries)

	pageRepo := repository.NewPageRepository(queries)
	getSuggestionDetailUC := usecase.NewGetSuggestionDetailUsecase(
		spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo,
		suggestionRepo, suggestionPageRepo, suggestionCommentRepo, pageRepo, userRepo,
	)
	createSuggestionCommentUC := usecase.NewCreateSuggestionCommentUsecase(suggestionCommentRepo)
	commentCreateValidator := validator.NewSuggestionCommentCreateValidator()

	return suggestioncommenthandler.NewHandler(
		flashMgr,
		getSuggestionDetailUC,
		createSuggestionCommentUC,
		commentCreateValidator,
	)
}

func TestCreate_未ログインでサインインにリダイレクトされる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, queries)

	form := url.Values{}
	form.Set("body", "コメント本文")

	req := newPostRequest(t, "/s/test/suggestions/1/comments", map[string]string{
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

func TestCreate_本文が空の場合バリデーションエラーでリダイレクトされる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("comment-empty@example.com").
		WithAtname("commentempty").
		Build()

	handler := setupHandler(t, queries)

	form := url.Values{}
	form.Set("body", "")
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/any-sp/suggestions/1/comments", map[string]string{
		"space_identifier":  "any-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "commentempty"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if loc != "/s/any-sp/suggestions/1" {
		t.Errorf("wrong redirect location: got %q", loc)
	}
}

func TestCreate_存在しない編集提案で404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("comment-nosugg@example.com").
		WithAtname("commentnosugg").
		Build()

	handler := setupHandler(t, queries)

	form := url.Values{}
	form.Set("body", "コメント本文")
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/nonexistent/suggestions/999/comments", map[string]string{
		"space_identifier":  "nonexistent",
		"suggestion_number": "999",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "commentnosugg"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestCreate_正常にコメントが作成されリダイレクトされる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("comment-ok@example.com").
		WithAtname("commentok").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("comment-ok-sp").
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
		Build()

	handler := setupHandler(t, queries)

	form := url.Values{}
	form.Set("body", "テストコメント")
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/comment-ok-sp/suggestions/1/comments", map[string]string{
		"space_identifier":  "comment-ok-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "commentok"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if loc != "/s/comment-ok-sp/suggestions/1" {
		t.Errorf("wrong redirect location: got %q", loc)
	}
}

func TestCreate_スペースメンバーでないユーザーは403が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("comment-owner@example.com").
		WithAtname("commentowner").
		Build()
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("comment-nonmember@example.com").
		WithAtname("commentnonmember").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("comment-forbid-sp").
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
		Build()

	handler := setupHandler(t, queries)

	form := url.Values{}
	form.Set("body", "コメント本文")
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/comment-forbid-sp/suggestions/1/comments", map[string]string{
		"space_identifier":  "comment-forbid-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: nonMemberID, Atname: "commentnonmember"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusForbidden)
	}
}
