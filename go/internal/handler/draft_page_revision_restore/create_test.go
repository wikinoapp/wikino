package draft_page_revision_restore_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/draft_page_revision_restore"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// setupHandler builds the restore handler backed by the shared test DB. The UseCase manages its
// own transaction, so the repositories read straight from the committed test DB.
//
// [Ja] setupHandler は共有テスト DB を使う復元ハンドラーを生成する。UseCase が自前で
// トランザクションを管理するため、リポジトリはコミット済みのテスト DB を直接読む。
func setupHandler(t *testing.T, db *sql.DB) *draft_page_revision_restore.Handler {
	t.Helper()

	q := query.New(db)

	return draft_page_revision_restore.NewHandler(
		session.NewFlashManager("", false, false),
		usecase.NewRestoreDraftPageRevisionUsecase(
			db,
			repository.NewSpaceRepository(q),
			repository.NewSpaceMemberRepository(q),
			repository.NewPageRepository(q),
			repository.NewPageEditorRepository(q),
			repository.NewTopicRepository(q),
			repository.NewTopicMemberRepository(q),
			repository.NewDraftPageRepository(q),
			repository.NewDraftPageRevisionRepository(q),
			repository.NewAttachmentRepository(q),
		),
	)
}

// restoreFixture is the fixture set shared by the Create handler tests.
// [Ja] restoreFixture は Create ハンドラーのテストで共有するフィクスチャ一式。
type restoreFixture struct {
	userID      model.UserID
	spaceID     model.SpaceID
	draftPageID model.DraftPageID
	revision1   *model.DraftPageRevision
	revision2   *model.DraftPageRevision
}

// setupRestoreFixture creates a space, member, topic, page, draft and two revisions (v1 then v2)
// committed directly to the test DB. prefix keeps identifiers unique across parallel tests.
//
// [Ja] setupRestoreFixture はスペース・メンバー・トピック・ページ・下書きとリビジョン 2 件
// (v1 → v2) をテスト DB へ直接コミットして作成する。prefix は並行テスト間で識別子を一意に保つ。
func setupRestoreFixture(t *testing.T, db *sql.DB, prefix string) restoreFixture {
	t.Helper()

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail(prefix + "@example.com").
		WithAtname(strings.ReplaceAll(prefix, "-", "")).
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier(prefix + "-space").
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
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Current Title").
		WithBody("current body").
		Build()

	revisionRepo := repository.NewDraftPageRevisionRepository(query.New(db))
	revision1, err := revisionRepo.Create(context.Background(), repository.CreateDraftPageRevisionInput{
		DraftPageID:   draftPageID,
		SpaceID:       spaceID,
		SpaceMemberID: spaceMemberID,
		Title:         "Old Title",
		Body:          "old body",
		BodyHTML:      "<p>old body</p>",
	})
	if err != nil {
		t.Fatalf("Create() revision1 error = %v", err)
	}
	revision2, err := revisionRepo.Create(context.Background(), repository.CreateDraftPageRevisionInput{
		DraftPageID:   draftPageID,
		SpaceID:       spaceID,
		SpaceMemberID: spaceMemberID,
		Title:         "Current Title",
		Body:          "current body",
		BodyHTML:      "<p>current body</p>",
	})
	if err != nil {
		t.Fatalf("Create() revision2 error = %v", err)
	}

	return restoreFixture{
		userID:      userID,
		spaceID:     spaceID,
		draftPageID: draftPageID,
		revision1:   revision1,
		revision2:   revision2,
	}
}

// newCreateRequest builds the restore POST request carrying chi URL parameters.
// [Ja] newCreateRequest は chi の URL パラメータ付き復元 POST リクエストを作成する。
func newCreateRequest(t *testing.T, spaceIdentifier string, revisionID string) *http.Request {
	t.Helper()

	path := "/s/" + spaceIdentifier + "/pages/1/draft_page_revisions/" + revisionID + "/restore"
	req := httptest.NewRequest(http.MethodPost, path, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", spaceIdentifier)
	rctx.URLParams.Add("page_number", "1")
	rctx.URLParams.Add("draft_page_revision_id", revisionID)

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestCreate_NotLoggedIn(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	handler := setupHandler(t, db)

	req := newCreateRequest(t, "restore-h-anon-space", "00000000-0000-0000-0000-000000000000")

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnauthorized)
	}
}

func TestCreate_InvalidPageNumber(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	handler := setupHandler(t, db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("restore-h-invalidnum@example.com").
		WithAtname("restorehinvalidnum").
		Build()

	req := httptest.NewRequest(http.MethodPost, "/s/restore-h-invalid-space/pages/abc/draft_page_revisions/00000000-0000-0000-0000-000000000000/restore", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", "restore-h-invalid-space")
	rctx.URLParams.Add("page_number", "abc")
	rctx.URLParams.Add("draft_page_revision_id", "00000000-0000-0000-0000-000000000000")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(middleware.SetUserToContext(req.Context(), &model.User{ID: userID}))

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	handler := setupHandler(t, db)
	ctx := context.Background()

	fixture := setupRestoreFixture(t, db, "restore-h-success")

	req := newCreateRequest(t, "restore-h-success-space", string(fixture.revision1.ID))
	req = req.WithContext(middleware.SetUserToContext(req.Context(), &model.User{ID: fixture.userID}))

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("wrong status code: got %v want %v (body: %s)", rr.Code, http.StatusSeeOther, rr.Body.String())
	}

	// The handler must redirect back to the page editor so it reloads with the restored content.
	// [Ja] 復元後の内容でエディタが再読み込みされるよう、ページ編集画面へリダイレクトすること。
	location := rr.Header().Get("Location")
	wantLocation := "/s/restore-h-success-space/pages/1/edit"
	if location != wantLocation {
		t.Errorf("wrong redirect location: got %v want %v", location, wantLocation)
	}

	// The draft must be updated and the restored state recorded as a new revision.
	// [Ja] 下書きが更新され、復元後の状態が新しいリビジョンとして記録されていること。
	q := query.New(db)
	draftPage, err := repository.NewDraftPageRepository(q).FindByID(ctx, fixture.draftPageID, fixture.spaceID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if draftPage == nil {
		t.Fatal("draftPage should not be nil")
	}
	if draftPage.Title == nil || *draftPage.Title != "Old Title" {
		t.Errorf("DraftPage.Title = %v, want %q", draftPage.Title, "Old Title")
	}
	if draftPage.Body != "old body" {
		t.Errorf("DraftPage.Body = %q, want %q", draftPage.Body, "old body")
	}

	count, err := repository.NewDraftPageRevisionRepository(q).CountByDraftPageID(ctx, fixture.draftPageID, fixture.spaceID)
	if err != nil {
		t.Fatalf("CountByDraftPageID() error = %v", err)
	}
	if count != 3 {
		t.Errorf("revision count = %d, want 3", count)
	}
}

func TestCreate_RevisionNotFound(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	handler := setupHandler(t, db)

	fixture := setupRestoreFixture(t, db, "restore-h-notfound")

	req := newCreateRequest(t, "restore-h-notfound-space", "00000000-0000-0000-0000-000000000000")
	req = req.WithContext(middleware.SetUserToContext(req.Context(), &model.User{ID: fixture.userID}))

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestCreate_NotSpaceMember(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	handler := setupHandler(t, db)

	fixture := setupRestoreFixture(t, db, "restore-h-nonmember")
	strangerID := testutil.NewUserBuilderDB(t, db).
		WithEmail("restore-h-stranger@example.com").
		WithAtname("restorehstranger").
		Build()

	req := newCreateRequest(t, "restore-h-nonmember-space", string(fixture.revision1.ID))
	req = req.WithContext(middleware.SetUserToContext(req.Context(), &model.User{ID: strangerID}))

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}
