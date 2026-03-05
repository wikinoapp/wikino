package draft_page_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/draft_page"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// newPostRequestWithChiParams はchiのURLパラメータ付きPOSTリクエストを作成するヘルパーです
func newPostRequestWithChiParams(t *testing.T, path string, params map[string]string) *http.Request {
	t.Helper()

	formData := url.Values{}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestCreate_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newPostRequestWithChiParams(t, "/s/my-space/pages/1/draft_page", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	})

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnauthorized)
	}
}

func TestCreate_InvalidPageNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("create-invalidnum@example.com").
		WithAtname("createinvalidnum").
		Build()

	handler := setupHandler(t, queries)

	req := newPostRequestWithChiParams(t, "/s/my-space/pages/abc/draft_page", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "abc",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestCreate_SpaceNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("create-spacenotfound@example.com").
		WithAtname("createspacenotfound").
		Build()

	handler := setupHandler(t, queries)

	req := newPostRequestWithChiParams(t, "/s/nonexistent/pages/1/draft_page", map[string]string{
		"space_identifier": "nonexistent",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	// UseCaseが独自トランザクションを管理するため、DB直接書き込みを使用
	db := testutil.GetTestDB()
	q := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("create-success@example.com").
		WithAtname("createsuccess").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("create-success-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
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
		WithTitle("Test Page").
		Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Title").
		WithBody("Draft body").
		Build()

	draftPageRepo := repository.NewDraftPageRepository(q)
	handler := draft_page.NewHandler(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		draftPageRepo,
		usecase.NewAutoSaveDraftPageUsecase(
			db,
			draftPageRepo,
			repository.NewPageRepository(q),
			repository.NewPageEditorRepository(q),
			repository.NewTopicRepository(q),
			repository.NewAttachmentRepository(q),
		),
		usecase.NewManualSaveDraftPageUsecase(
			db,
			draftPageRepo,
			repository.NewDraftPageRevisionRepository(q),
		),
	)

	req := newPostRequestWithChiParams(t, "/s/create-success-space/pages/1/draft_page", map[string]string{
		"space_identifier": "create-success-space",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNoContent)
	}
}

func TestCreate_DraftPageNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("create-nodraft@example.com").
		WithAtname("createnodraft").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("create-nodraft-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	handler := setupHandler(t, queries)

	req := newPostRequestWithChiParams(t, "/s/create-nodraft-space/pages/1/draft_page", map[string]string{
		"space_identifier": "create-nodraft-space",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// 下書きが存在しない場合は404
	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}
