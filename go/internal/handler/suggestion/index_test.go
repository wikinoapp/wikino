package suggestion_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	suggestionhandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// newIndexRequest はchiのURLパラメータ付きGETリクエストを作成するヘルパーです
func newIndexRequest(t *testing.T, path string, params map[string]string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// setupHandler はテスト用の編集提案ハンドラーを作成するヘルパーです
func setupHandler(t *testing.T, db *sql.DB, queries *query.Queries) *suggestionhandler.Handler {
	t.Helper()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	suggestionRepo := repository.NewSuggestionRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	suggestionPageRevisionRepo := repository.NewSuggestionPageRevisionRepository(queries)
	userRepo := repository.NewUserRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	pageRevisionRepo := repository.NewPageRevisionRepository(queries)
	sidebarHelper := sidebar.NewHelper(topicRepo, draftPageRepo)
	flashMgr := session.NewFlashManager("localhost", false, false)

	suggestionCommentRepo := repository.NewSuggestionCommentRepository(queries)

	getSuggestionListUC := usecase.NewGetSuggestionListUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, userRepo)
	getSuggestionDetailUC := usecase.NewGetSuggestionDetailUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, suggestionPageRepo, suggestionCommentRepo, pageRepo, userRepo)
	getSuggestionNewUC := usecase.NewGetSuggestionNewUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, draftPageRepo)
	createSuggestionUC := usecase.NewCreateSuggestionUsecase(db, suggestionRepo, suggestionPageRepo, suggestionPageRevisionRepo, pageRevisionRepo, topicRepo, pageRepo, draftPageRepo)
	suggestionCreateValidator := validator.NewSuggestionCreateValidator(draftPageRepo)

	return suggestionhandler.NewHandler(
		cfg,
		flashMgr,
		getSuggestionListUC,
		getSuggestionDetailUC,
		getSuggestionNewUC,
		createSuggestionUC,
		sidebarHelper,
		suggestionCreateValidator,
	)
}

func TestIndex_存在しないスペースで404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, db, queries)

	req := newIndexRequest(t, "/s/nonexistent/topics/1/suggestions", map[string]string{
		"space_identifier": "nonexistent",
		"topic_number":     "1",
	})

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestIndex_不正なトピック番号で404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, db, queries)

	req := newIndexRequest(t, "/s/test-space/topics/abc/suggestions", map[string]string{
		"space_identifier": "test-space",
		"topic_number":     "abc",
	})

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestIndex_存在しないトピックで404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("si-noexist").
		Build()

	handler := setupHandler(t, db, queries)

	req := newIndexRequest(t, "/s/si-noexist/topics/999/suggestions", map[string]string{
		"space_identifier": "si-noexist",
		"topic_number":     "999",
	})

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestIndex_公開トピックの編集提案一覧を未ログインで閲覧できる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("si-pub@example.com").
		WithAtname("sipub").
		WithName("提案者").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("si-pub-space").
		WithName("Public Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("公開トピック").
		WithVisibility(0). // public
		Build()
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("テスト提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, db, queries)

	req := newIndexRequest(t, "/s/si-pub-space/topics/1/suggestions", map[string]string{
		"space_identifier": "si-pub-space",
		"topic_number":     "1",
	})

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "テスト提案") {
		t.Error("response should contain suggestion title")
	}
	if !strings.Contains(body, "提案者") {
		t.Error("response should contain creator name")
	}
}

func TestIndex_非公開トピックを未ログインで閲覧すると404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("si-priv1").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(1). // private
		Build()

	handler := setupHandler(t, db, queries)

	req := newIndexRequest(t, "/s/si-priv1/topics/1/suggestions", map[string]string{
		"space_identifier": "si-priv1",
		"topic_number":     "1",
	})

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestIndex_クローズタブで反映済みとクローズの提案が表示される(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("si-closed@example.com").
		WithAtname("siclosed").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("si-closed-sp").
		WithName("Closed Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("クローズテスト").
		WithVisibility(0).
		Build()
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("オープン提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("クローズ提案").
		WithStatus(model.SuggestionStatusClosed).
		Build()

	handler := setupHandler(t, db, queries)

	req := newIndexRequest(t, "/s/si-closed-sp/topics/1/suggestions?tab=closed", map[string]string{
		"space_identifier": "si-closed-sp",
		"topic_number":     "1",
	})

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "クローズ提案") {
		t.Error("response should contain closed suggestion title")
	}
	if strings.Contains(body, "オープン提案") {
		t.Error("response should not contain open suggestion title on closed tab")
	}
}

func TestIndex_非公開トピックをスペースオーナーが閲覧できる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("si-owner@example.com").
		WithAtname("siowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("si-priv2").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		WithRole(0). // owner
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("オーナー閲覧").
		WithVisibility(1). // private
		Build()

	handler := setupHandler(t, db, queries)

	req := newIndexRequest(t, "/s/si-priv2/topics/1/suggestions", map[string]string{
		"space_identifier": "si-priv2",
		"topic_number":     "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "siowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}
}
