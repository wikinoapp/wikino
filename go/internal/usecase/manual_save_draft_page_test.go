package usecase

import (
	"context"
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newManualSaveUC(db *sql.DB) *ManualSaveDraftPageUsecase {
	q := query.New(db)
	return NewManualSaveDraftPageUsecase(
		db,
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewDraftPageRepository(q),
		repository.NewDraftPageRevisionRepository(q),
		repository.NewPageRepository(q),
		repository.NewPageEditorRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
	)
}

func TestManualSaveDraftPageUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newManualSaveUC(db)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("manual-save").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("manual-save@example.com").
		WithAtname("manualsave").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
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

	title := "下書きタイトル"
	output, err := uc.Execute(context.Background(), ManualSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("manual-save"),
		PageNumber:      1,
		UserID:          userID,
		Title:           &title,
		Body:            "下書き本文",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}
	if output == nil {
		t.Fatal("出力がnil")
	}
	if output.DraftPageRevision == nil {
		t.Fatal("DraftPageRevisionがnil")
	}
	if output.DraftPageRevision.Title != "下書きタイトル" {
		t.Errorf("Title = %q、期待値 = %q", output.DraftPageRevision.Title, "下書きタイトル")
	}
	if output.DraftPageRevision.Body != "下書き本文" {
		t.Errorf("Body = %q、期待値 = %q", output.DraftPageRevision.Body, "下書き本文")
	}
	if output.DraftPageRevision.SpaceMemberID != spaceMemberID {
		t.Errorf("SpaceMemberID = %v、期待値 = %v", output.DraftPageRevision.SpaceMemberID, spaceMemberID)
	}
	if output.DraftPageRevision.CreatedAt.IsZero() {
		t.Error("CreatedAtがゼロ値")
	}
}

func TestManualSaveDraftPageUsecase_Execute_WithoutDraftPage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newManualSaveUC(db)

	// テストデータを作成 (DraftPageは作成しない)
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("manual-save-nodraft").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("manual-save-nodraft@example.com").
		WithAtname("manualsavenodraft").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
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

	title := "新規下書き"
	output, err := uc.Execute(context.Background(), ManualSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("manual-save-nodraft"),
		PageNumber:      1,
		UserID:          userID,
		Title:           &title,
		Body:            "新規下書き本文",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}
	if output == nil {
		t.Fatal("出力がnil")
	}
	if output.DraftPageRevision == nil {
		t.Fatal("DraftPageRevisionがnil")
	}
	if output.DraftPageRevision.Title != "新規下書き" {
		t.Errorf("Title = %q、期待値 = %q", output.DraftPageRevision.Title, "新規下書き")
	}
	if output.DraftPageRevision.Body != "新規下書き本文" {
		t.Errorf("Body = %q、期待値 = %q", output.DraftPageRevision.Body, "新規下書き本文")
	}
}

// TestManualSaveDraftPageUsecase_Execute_SkipDuplicateRevisionは、タイトル・本文が
// 最新リビジョンと同一の保存ではリビジョン作成がスキップされること (保存自体は成功すること)、
// その後内容を変えた保存では再びリビジョンが作成されることを検証する。
func TestManualSaveDraftPageUsecase_Execute_SkipDuplicateRevision(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newManualSaveUC(db)

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("manual-save-skip").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("manual-save-skip@example.com").
		WithAtname("manualsaveskip").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
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

	title := "重複タイトル"
	input := ManualSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("manual-save-skip"),
		PageNumber:      1,
		UserID:          userID,
		Title:           &title,
		Body:            "重複本文",
	}
	revisionRepo := repository.NewDraftPageRevisionRepository(query.New(db))

	// 1回目の保存: リビジョンが作成される。
	first, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}
	if first.DraftPageRevision == nil {
		t.Fatal("初回の保存でリビジョンが作成されていない")
	}

	// 2回目の保存 (同一内容): リビジョン作成はスキップされ、保存自体は成功する。
	second, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}
	if second.DraftPageRevision != nil {
		t.Error("同じ内容の2回目の保存でリビジョンの作成が省かれていない")
	}
	if second.DraftPage == nil {
		t.Fatal("DraftPageがnil")
	}

	count, err := revisionRepo.CountByDraftPageID(context.Background(), second.DraftPage.ID, second.DraftPage.SpaceID)
	if err != nil {
		t.Fatalf("CountByDraftPageID()のエラー = %v、期待値 = nil", err)
	}
	if count != 1 {
		t.Errorf("リビジョンの件数 = %d、期待値 = 1", count)
	}

	// 3回目の保存 (内容変更): 新しいリビジョンが作成される。
	changedInput := input
	changedInput.Body = "変更後の本文"
	third, err := uc.Execute(context.Background(), changedInput)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}
	if third.DraftPageRevision == nil {
		t.Fatal("内容を変えて保存したのにリビジョンが作成されていない")
	}

	count, err = revisionRepo.CountByDraftPageID(context.Background(), third.DraftPage.ID, third.DraftPage.SpaceID)
	if err != nil {
		t.Fatalf("CountByDraftPageID()のエラー = %v、期待値 = nil", err)
	}
	if count != 2 {
		t.Errorf("リビジョンの件数 = %d、期待値 = 2", count)
	}
}
