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

// setupHandlerは共有テストDBを使う復元ハンドラーを生成する。UseCaseが自前で
// トランザクションを管理するため、リポジトリはコミット済みのテストDBを直接読む。
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

// restoreFixtureはCreateハンドラーのテストで共有するフィクスチャ一式。
type restoreFixture struct {
	userID      model.UserID
	spaceID     model.SpaceID
	draftPageID model.DraftPageID
	revision1   *model.DraftPageRevision
	revision2   *model.DraftPageRevision
}

// setupRestoreFixtureはスペース・メンバー・トピック・ページ・下書きとリビジョン2件
// (v1 → v2) をテストDBへ直接コミットして作成する。prefixは並行テスト間で識別子を一意に保つ。
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
		t.Fatalf("Create() (revision1) のエラー = %v", err)
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
		t.Fatalf("Create() (revision2) のエラー = %v", err)
	}

	return restoreFixture{
		userID:      userID,
		spaceID:     spaceID,
		draftPageID: draftPageID,
		revision1:   revision1,
		revision2:   revision2,
	}
}

// newCreateRequestはchiのURLパラメータ付き復元POSTリクエストを作成する。
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnauthorized)
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
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
		t.Fatalf("ステータスコード = %v、期待値 = %v (本文: %s)", rr.Code, http.StatusSeeOther, rr.Body.String())
	}

	// 復元後の内容でエディタが再読み込みされるよう、ページ編集画面へリダイレクトすること。
	location := rr.Header().Get("Location")
	wantLocation := "/s/restore-h-success-space/pages/1/edit"
	if location != wantLocation {
		t.Errorf("リダイレクト先 = %v、期待値 = %v", location, wantLocation)
	}

	// 下書きが更新され、復元後の状態が新しいリビジョンとして記録されていること。
	q := query.New(db)
	draftPage, err := repository.NewDraftPageRepository(q).FindByID(ctx, fixture.draftPageID, fixture.spaceID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if draftPage == nil {
		t.Fatal("draftPageがnil")
	}
	if draftPage.Title == nil || *draftPage.Title != "Old Title" {
		t.Errorf("DraftPage.Title = %v、期待値 = %q", draftPage.Title, "Old Title")
	}
	if draftPage.Body != "old body" {
		t.Errorf("DraftPage.Body = %q、期待値 = %q", draftPage.Body, "old body")
	}

	count, err := repository.NewDraftPageRevisionRepository(q).CountByDraftPageID(ctx, fixture.draftPageID, fixture.spaceID)
	if err != nil {
		t.Fatalf("CountByDraftPageID()のエラー = %v", err)
	}
	if count != 3 {
		t.Errorf("リビジョンの件数 = %d、期待値 = 3", count)
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}
