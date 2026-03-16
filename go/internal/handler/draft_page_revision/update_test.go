package draft_page_revision_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/draft_page_revision"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// setupHandler はテスト用のハンドラーを生成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *draft_page_revision.Handler {
	t.Helper()

	db := testutil.GetTestDB()

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)

	flashMgr := session.NewFlashManager("", false, false)

	return draft_page_revision.NewHandler(
		usecase.NewGetPageDetailUsecase(
			spaceRepo,
			spaceMemberRepo,
			pageRepo,
			draftPageRepo,
			topicRepo,
			topicMemberRepo,
		),
		flashMgr,
		usecase.NewManualSaveDraftPageUsecase(
			db,
			draftPageRepo,
			repository.NewDraftPageRevisionRepository(queries),
			pageRepo,
			repository.NewPageEditorRepository(queries),
			topicRepo,
			repository.NewAttachmentRepository(queries),
		),
	)
}

// newRequestWithChiParams はchiのURLパラメータ付きPATCHリクエストを作成するヘルパーです
func newRequestWithChiParams(t *testing.T, path string, params map[string]string, formData url.Values) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestUpdate_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, "/s/my-space/pages/1/draft_page_revision", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	}, url.Values{})

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnauthorized)
	}
}

func TestUpdate_InvalidPageNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpr-invalidnum@example.com").
		WithAtname("dprinvalidnum").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, "/s/my-space/pages/abc/draft_page_revision", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "abc",
	}, url.Values{})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_SpaceNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpr-spacenotfound@example.com").
		WithAtname("dprspacenotfound").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, "/s/nonexistent/pages/1/draft_page_revision", map[string]string{
		"space_identifier": "nonexistent",
		"page_number":      "1",
	}, url.Values{})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()

	// UseCaseが独自トランザクションを管理するため、DB直接書き込みを使用
	db := testutil.GetTestDB()
	q := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("dpr-update-success@example.com").
		WithAtname("dprupdatesuccess").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("dpr-update-success-space").
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

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	handler := draft_page_revision.NewHandler(
		usecase.NewGetPageDetailUsecase(
			spaceRepo,
			spaceMemberRepo,
			pageRepo,
			draftPageRepo,
			topicRepo,
			topicMemberRepo,
		),
		session.NewFlashManager("", false, false),
		usecase.NewManualSaveDraftPageUsecase(
			db,
			draftPageRepo,
			repository.NewDraftPageRevisionRepository(q),
			pageRepo,
			repository.NewPageEditorRepository(q),
			topicRepo,
			repository.NewAttachmentRepository(q),
		),
	)

	formData := url.Values{}
	formData.Set("title", "Draft Title")
	formData.Set("body", "Draft body")
	req := newRequestWithChiParams(t, "/s/dpr-update-success-space/pages/1/draft_page_revision", map[string]string{
		"space_identifier": "dpr-update-success-space",
		"page_number":      "1",
	}, formData)

	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}

	location := rr.Header().Get("Location")
	if location != "/drafts" {
		t.Errorf("wrong redirect location: got %v want /drafts", location)
	}
}

func TestUpdate_WithoutDraftPage(t *testing.T) {
	t.Parallel()

	// DraftPageが存在しない場合でも、find_or_createにより成功する
	db := testutil.GetTestDB()
	q := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("dpr-nodraft@example.com").
		WithAtname("dprnodraft").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("dpr-nodraft-space").
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
	testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	handler := draft_page_revision.NewHandler(
		usecase.NewGetPageDetailUsecase(
			spaceRepo,
			spaceMemberRepo,
			pageRepo,
			draftPageRepo,
			topicRepo,
			topicMemberRepo,
		),
		session.NewFlashManager("", false, false),
		usecase.NewManualSaveDraftPageUsecase(
			db,
			draftPageRepo,
			repository.NewDraftPageRevisionRepository(q),
			pageRepo,
			repository.NewPageEditorRepository(q),
			topicRepo,
			repository.NewAttachmentRepository(q),
		),
	)

	formData := url.Values{}
	formData.Set("title", "新規下書き")
	formData.Set("body", "新規下書き本文")
	req := newRequestWithChiParams(t, "/s/dpr-nodraft-space/pages/1/draft_page_revision", map[string]string{
		"space_identifier": "dpr-nodraft-space",
		"page_number":      "1",
	}, formData)

	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}

	location := rr.Header().Get("Location")
	if location != "/drafts" {
		t.Errorf("wrong redirect location: got %v want /drafts", location)
	}
}
