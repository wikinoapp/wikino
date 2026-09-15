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

// staleBodyHTMLはrevision1のbody_htmlとして保存する値で、本文の再レンダリング結果と
// 意図的に異なる値にしてある。保存済みHTMLがsaveDraftPageContentで再レンダリングされずに
// 使い回された場合に、成功テストで検出できるようにするため。
const staleBodyHTML = "<p>stale html</p>"

// restoreFixtureは復元UseCaseのテストで共有するフィクスチャ一式。
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

// setupRestoreFixtureはスペース・メンバー・トピック・ページ・下書きとリビジョン2件
// (v1 → v2) をテストDBへ直接コミットして作成する (UseCaseが自前でトランザクションを管理する
// ため)。prefixはテストDBを共有する並行テスト間で識別子を一意に保つ。
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
		t.Fatalf("Execute()のエラー = %v", err)
	}

	// 下書きが復元対象リビジョンの内容で更新されていること。
	if output.DraftPage == nil {
		t.Fatal("DraftPageがnil")
	}
	if output.DraftPage.Title == nil || *output.DraftPage.Title != "Old Title" {
		t.Errorf("DraftPage.Title = %v、期待値 = %q", output.DraftPage.Title, "Old Title")
	}
	if output.DraftPage.Body != "old body" {
		t.Errorf("DraftPage.Body = %q、期待値 = %q", output.DraftPage.Body, "old body")
	}

	// 本文HTMLはsaveDraftPageContentで再レンダリングされること (リビジョン保存済みの
	// 古いbody_htmlの使い回しではないこと)。リンク先ページIDや添付参照が復元後の本文と
	// 整合するための設計判断を検証する。
	if output.DraftPage.BodyHTML == staleBodyHTML {
		t.Error("DraftPage.BodyHTMLが再描画されず、保存済みのリビジョンからコピーされている")
	}
	if output.DraftPage.BodyHTML == "" {
		t.Error("DraftPage.BodyHTMLが空")
	}

	// 復元後の状態が新しいリビジョンとして記録されていること (履歴は削除されない)。
	if output.DraftPageRevision == nil {
		t.Fatal("DraftPageRevisionがnil")
	}
	if output.DraftPageRevision.ID == fixture.revision1.ID || output.DraftPageRevision.ID == fixture.revision2.ID {
		t.Errorf("DraftPageRevision.ID = %v、期待値 = 新しいリビジョンのID", output.DraftPageRevision.ID)
	}
	if output.DraftPageRevision.Title != "Old Title" {
		t.Errorf("DraftPageRevision.Title = %q、期待値 = %q", output.DraftPageRevision.Title, "Old Title")
	}
	if output.DraftPageRevision.Body != "old body" {
		t.Errorf("DraftPageRevision.Body = %q、期待値 = %q", output.DraftPageRevision.Body, "old body")
	}
	// 新しいリビジョンにも、保存済みの古いHTMLではなく再レンダリングされたHTMLが記録されること。
	if output.DraftPageRevision.BodyHTML == staleBodyHTML {
		t.Error("DraftPageRevision.BodyHTMLが再描画されず、保存済みのリビジョンからコピーされている")
	}
	if output.DraftPageRevision.BodyHTML == "" {
		t.Error("DraftPageRevision.BodyHTMLが空")
	}

	revisionRepo := repository.NewDraftPageRevisionRepository(query.New(db))
	count, err := revisionRepo.CountByDraftPageID(ctx, fixture.draftPageID, fixture.spaceID)
	if err != nil {
		t.Fatalf("CountByDraftPageID()のエラー = %v", err)
	}
	if count != 3 {
		t.Errorf("リビジョンの件数 = %d、期待値 = 3", count)
	}

	// 新しいリビジョンが履歴一覧の先頭 (最新) であること。
	revisions, err := revisionRepo.ListByDraftPageID(ctx, fixture.draftPageID, fixture.spaceID, 1)
	if err != nil {
		t.Fatalf("ListByDraftPageID()のエラー = %v", err)
	}
	if len(revisions) != 1 || revisions[0].ID != output.DraftPageRevision.ID {
		t.Errorf("最新のリビジョン = %+v、期待値 = ID %v", revisions, output.DraftPageRevision.ID)
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
		t.Fatalf("AppErrorを期待したが、%vだった", err)
	}
	if ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("Code = %v、期待値 = %v", ae.Code, model.AppErrCodeResourceNotFound)
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
		t.Fatalf("AppErrorを期待したが、%vだった", err)
	}
	if ae.Code != model.AppErrCodeForbidden {
		t.Errorf("Code = %v、期待値 = %v", ae.Code, model.AppErrCodeForbidden)
	}
}

func TestRestoreDraftPageRevisionUsecase_Execute_OtherMembersRevision(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newRestoreUC(db)
	ctx := context.Background()

	fixture := setupRestoreFixture(t, db, "restore-rev-other")

	// 同じスペースの別メンバーが同じページに自分の下書きとリビジョンを持つ場合、
	// フィクスチャのオーナーはそれを復元できないこと (存在を隠すため404)。
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
		t.Fatalf("Create() (otherRev) のエラー = %v", err)
	}

	_, err = uc.Execute(ctx, RestoreDraftPageRevisionInput{
		SpaceIdentifier: "restore-rev-other-space",
		PageNumber:      1,
		RevisionID:      otherRev.ID,
		UserID:          fixture.userID,
	})
	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatalf("AppErrorを期待したが、%vだった", err)
	}
	if ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("Code = %v、期待値 = %v", ae.Code, model.AppErrCodeResourceNotFound)
	}
}
