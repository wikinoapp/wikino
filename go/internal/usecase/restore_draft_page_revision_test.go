package usecase

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newRestoreUC(db *sql.DB) *RestoreDraftPageRevisionUsecase {
	q := query.New(db)
	return NewRestoreDraftPageRevisionUsecase(
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
	)
}

// staleBodyHTML is stored as revision1's body_html and is deliberately different from what
// re-rendering its body produces, so the success test can detect the stored HTML being copied
// instead of re-rendered through saveDraftPageContent.
//
// [Ja] staleBodyHTML は revision1 の body_html として保存する値で、本文の再レンダリング結果と
// 意図的に異なる値にしてある。保存済み HTML が saveDraftPageContent で再レンダリングされずに
// 使い回された場合に、成功テストで検出できるようにするため。
const staleBodyHTML = "<p>stale html</p>"

// restoreFixture is the fixture set shared by the restore UseCase tests.
// [Ja] restoreFixture は復元 UseCase のテストで共有するフィクスチャ一式。
type restoreFixture struct {
	userID        model.UserID
	spaceID       model.SpaceID
	spaceMemberID model.SpaceMemberID
	topicID       model.TopicID
	pageID        model.PageID
	draftPageID   model.DraftPageID
	revision1     *model.DraftPageRevision
	revision2     *model.DraftPageRevision
}

// setupRestoreFixture creates a space, member, topic, page, draft and two revisions (v1 then v2)
// committed directly to the test DB, because the UseCase manages its own transaction. prefix
// keeps identifiers unique across parallel tests sharing the test DB.
//
// [Ja] setupRestoreFixture はスペース・メンバー・トピック・ページ・下書きとリビジョン 2 件
// (v1 → v2) をテスト DB へ直接コミットして作成する (UseCase が自前でトランザクションを管理する
// ため)。prefix はテスト DB を共有する並行テスト間で識別子を一意に保つ。
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
		BodyHTML:      staleBodyHTML,
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
		userID:        userID,
		spaceID:       spaceID,
		spaceMemberID: spaceMemberID,
		topicID:       topicID,
		pageID:        pageID,
		draftPageID:   draftPageID,
		revision1:     revision1,
		revision2:     revision2,
	}
}

func TestRestoreDraftPageRevisionUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newRestoreUC(db)
	ctx := context.Background()

	fixture := setupRestoreFixture(t, db, "restore-rev")

	output, err := uc.Execute(ctx, RestoreDraftPageRevisionInput{
		SpaceIdentifier: "restore-rev-space",
		PageNumber:      1,
		RevisionID:      fixture.revision1.ID,
		UserID:          fixture.userID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// The draft must be updated with the restored revision's content.
	// [Ja] 下書きが復元対象リビジョンの内容で更新されていること。
	if output.DraftPage == nil {
		t.Fatal("DraftPage should not be nil")
	}
	if output.DraftPage.Title == nil || *output.DraftPage.Title != "Old Title" {
		t.Errorf("DraftPage.Title = %v, want %q", output.DraftPage.Title, "Old Title")
	}
	if output.DraftPage.Body != "old body" {
		t.Errorf("DraftPage.Body = %q, want %q", output.DraftPage.Body, "old body")
	}

	// The body HTML must be re-rendered through saveDraftPageContent, not copied from the
	// revision's stored (stale) body_html, so linked page IDs and attachment references stay
	// consistent with the restored body.
	//
	// [Ja] 本文 HTML は saveDraftPageContent で再レンダリングされること (リビジョン保存済みの
	// 古い body_html の使い回しではないこと)。リンク先ページ ID や添付参照が復元後の本文と
	// 整合するための設計判断を検証する。
	if output.DraftPage.BodyHTML == staleBodyHTML {
		t.Error("DraftPage.BodyHTML must be re-rendered, not copied from the stored revision")
	}
	if output.DraftPage.BodyHTML == "" {
		t.Error("DraftPage.BodyHTML should not be empty")
	}

	// The restored state must be recorded as a new revision (history is kept, not rewritten).
	// [Ja] 復元後の状態が新しいリビジョンとして記録されていること (履歴は削除されない)。
	if output.DraftPageRevision == nil {
		t.Fatal("DraftPageRevision should not be nil")
	}
	if output.DraftPageRevision.ID == fixture.revision1.ID || output.DraftPageRevision.ID == fixture.revision2.ID {
		t.Errorf("DraftPageRevision.ID = %v, want a new revision ID", output.DraftPageRevision.ID)
	}
	if output.DraftPageRevision.Title != "Old Title" {
		t.Errorf("DraftPageRevision.Title = %q, want %q", output.DraftPageRevision.Title, "Old Title")
	}
	if output.DraftPageRevision.Body != "old body" {
		t.Errorf("DraftPageRevision.Body = %q, want %q", output.DraftPageRevision.Body, "old body")
	}
	// The new revision must also record the re-rendered HTML, not the stored (stale) one.
	// [Ja] 新しいリビジョンにも、保存済みの古い HTML ではなく再レンダリングされた HTML が記録されること。
	if output.DraftPageRevision.BodyHTML == staleBodyHTML {
		t.Error("DraftPageRevision.BodyHTML must be re-rendered, not copied from the stored revision")
	}
	if output.DraftPageRevision.BodyHTML == "" {
		t.Error("DraftPageRevision.BodyHTML should not be empty")
	}

	revisionRepo := repository.NewDraftPageRevisionRepository(query.New(db))
	count, err := revisionRepo.CountByDraftPageID(ctx, fixture.draftPageID, fixture.spaceID)
	if err != nil {
		t.Fatalf("CountByDraftPageID() error = %v", err)
	}
	if count != 3 {
		t.Errorf("revision count = %d, want 3", count)
	}

	// The new revision must be the newest one in the history list.
	// [Ja] 新しいリビジョンが履歴一覧の先頭 (最新) であること。
	revisions, err := revisionRepo.ListByDraftPageID(ctx, fixture.draftPageID, fixture.spaceID, 1)
	if err != nil {
		t.Fatalf("ListByDraftPageID() error = %v", err)
	}
	if len(revisions) != 1 || revisions[0].ID != output.DraftPageRevision.ID {
		t.Errorf("newest revision = %+v, want ID %v", revisions, output.DraftPageRevision.ID)
	}
}

func TestRestoreDraftPageRevisionUsecase_Execute_RevisionNotFound(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newRestoreUC(db)

	fixture := setupRestoreFixture(t, db, "restore-rev-notfound")

	_, err := uc.Execute(context.Background(), RestoreDraftPageRevisionInput{
		SpaceIdentifier: "restore-rev-notfound-space",
		PageNumber:      1,
		RevisionID:      model.DraftPageRevisionID("00000000-0000-0000-0000-000000000000"),
		UserID:          fixture.userID,
	})
	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatalf("expected AppError, got %v", err)
	}
	if ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("Code = %v, want %v", ae.Code, model.AppErrCodeResourceNotFound)
	}
}

func TestRestoreDraftPageRevisionUsecase_Execute_NotSpaceMember(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newRestoreUC(db)

	fixture := setupRestoreFixture(t, db, "restore-rev-nonmember")
	strangerID := testutil.NewUserBuilderDB(t, db).
		WithEmail("restore-rev-stranger@example.com").
		WithAtname("restorerevstranger").
		Build()

	_, err := uc.Execute(context.Background(), RestoreDraftPageRevisionInput{
		SpaceIdentifier: "restore-rev-nonmember-space",
		PageNumber:      1,
		RevisionID:      fixture.revision1.ID,
		UserID:          strangerID,
	})
	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatalf("expected AppError, got %v", err)
	}
	if ae.Code != model.AppErrCodeForbidden {
		t.Errorf("Code = %v, want %v", ae.Code, model.AppErrCodeForbidden)
	}
}

func TestRestoreDraftPageRevisionUsecase_Execute_OtherMembersRevision(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newRestoreUC(db)
	ctx := context.Background()

	fixture := setupRestoreFixture(t, db, "restore-rev-other")

	// Another member of the same space with their own draft and revision for the same page:
	// the fixture owner must not be able to restore it (404, hiding its existence).
	//
	// [Ja] 同じスペースの別メンバーが同じページに自分の下書きとリビジョンを持つ場合、
	// フィクスチャのオーナーはそれを復元できないこと (存在を隠すため 404)。
	otherUserID := testutil.NewUserBuilderDB(t, db).
		WithEmail("restore-rev-other-member@example.com").
		WithAtname("restorerevothermember").
		Build()
	otherMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(fixture.spaceID).
		WithUserID(otherUserID).
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(fixture.spaceID).
		WithTopicID(fixture.topicID).
		WithSpaceMemberID(otherMemberID).
		Build()
	otherDraftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(fixture.spaceID).
		WithPageID(fixture.pageID).
		WithSpaceMemberID(otherMemberID).
		WithTopicID(fixture.topicID).
		WithTitle("Other Draft").
		WithBody("other draft body").
		Build()
	otherRev, err := repository.NewDraftPageRevisionRepository(query.New(db)).Create(ctx, repository.CreateDraftPageRevisionInput{
		DraftPageID:   otherDraftPageID,
		SpaceID:       fixture.spaceID,
		SpaceMemberID: otherMemberID,
		Title:         "Other Member Title",
		Body:          "other member body",
		BodyHTML:      "<p>other member body</p>",
	})
	if err != nil {
		t.Fatalf("Create() otherRev error = %v", err)
	}

	_, err = uc.Execute(ctx, RestoreDraftPageRevisionInput{
		SpaceIdentifier: "restore-rev-other-space",
		PageNumber:      1,
		RevisionID:      otherRev.ID,
		UserID:          fixture.userID,
	})
	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatalf("expected AppError, got %v", err)
	}
	if ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("Code = %v, want %v", ae.Code, model.AppErrCodeResourceNotFound)
	}
}
