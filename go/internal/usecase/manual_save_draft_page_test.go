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
	"github.com/wikinoapp/wikino/go/internal/validator"
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
		validator.NewDraftPageSaveValidator(),
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
	// 下書きを作るときに、ページの今の内容を起点として記録する
	wantBase := model.DraftPageContent{Title: "Test Page", Body: "Test body"}
	if got := output.DraftPage.BaseContent(); got == nil || *got != wantBase {
		t.Errorf("起点 = %+v、期待値 = %+v", got, wantBase)
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
	// 編集画面は、直前の保存の応答で受け取った版を前提にして次を保存する
	input.DraftPageDigest = first.DraftPage.ContentDigest()
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
	changedInput.DraftPageDigest = second.DraftPage.ContentDigest()
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

// TestManualSaveDraftPageUsecase_Execute_SkipRevisionSameAsBaseは、リビジョンの無い下書きで、
// 書き始めた時点の内容 (起点) と同じ保存ではリビジョンを作らないこと、起点から変えた保存では作ることを検証する。
// 起点が分からない下書きでは、リビジョンの無いときの保存でリビジョンを作る。
func TestManualSaveDraftPageUsecase_Execute_SkipRevisionSameAsBase(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newManualSaveUC(db)
	revisionRepo := repository.NewDraftPageRevisionRepository(query.New(db))

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("manual-save-base").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("manual-save-base@example.com").
		WithAtname("manualsavebase").
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
		WithTitle("Base Page").
		WithBody("元の本文").
		Build()
	legacyPageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Legacy Page").
		WithBody("元の本文").
		Build()
	// 起点を記録する前からある下書き (起点の列がNULL)
	legacyDraftID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(legacyPageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Legacy Page").
		WithBody("元の本文").
		Build()

	countRevisions := func(t *testing.T, draftPage *model.DraftPage) int64 {
		t.Helper()
		count, err := revisionRepo.CountByDraftPageID(context.Background(), draftPage.ID, draftPage.SpaceID)
		if err != nil {
			t.Fatalf("CountByDraftPageID()のエラー = %v、期待値 = nil", err)
		}
		return count
	}

	title := "Base Page"
	input := ManualSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("manual-save-base"),
		PageNumber:      1,
		UserID:          userID,
		Title:           &title,
		Body:            "元の本文",
	}

	// 下書きの無いページを、ページの今の内容のまま保存する: 下書きは作るが、リビジョンは作らない。
	first, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}
	if first.DraftPageRevision != nil {
		t.Error("起点と同じ内容の保存でリビジョンが作成された")
	}
	if first.DraftPage == nil {
		t.Fatal("DraftPageがnil")
	}
	if n := countRevisions(t, first.DraftPage); n != 0 {
		t.Errorf("リビジョンの件数 = %d、期待値 = 0", n)
	}

	// リビジョンの無い下書きを、起点と同じ内容でもう一度保存する: リビジョンは作らない。
	// 編集画面は、直前の保存の応答で受け取った版を前提にして次を保存する
	input.DraftPageDigest = first.DraftPage.ContentDigest()
	second, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}
	if second.DraftPageRevision != nil {
		t.Error("リビジョンの無い下書きを起点と同じ内容で保存したのにリビジョンが作成された")
	}

	// 起点から変えて保存する: リビジョンを作る。
	changedInput := input
	changedInput.Body = "元の本文\n足した行"
	changedInput.DraftPageDigest = second.DraftPage.ContentDigest()
	third, err := uc.Execute(context.Background(), changedInput)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}
	if third.DraftPageRevision == nil {
		t.Fatal("起点から変えて保存したのにリビジョンが作成されていない")
	}

	// 起点の内容に戻して保存する: 直前のリビジョンとは違うため、リビジョンを作る。
	input.DraftPageDigest = third.DraftPage.ContentDigest()
	fourth, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}
	if fourth.DraftPageRevision == nil {
		t.Fatal("直前のリビジョンと違う内容で保存したのにリビジョンが作成されていない")
	}
	if n := countRevisions(t, fourth.DraftPage); n != 2 {
		t.Errorf("リビジョンの件数 = %d、期待値 = 2", n)
	}

	// 起点が分からない下書きは、リビジョンの無いときの保存でリビジョンを作る。
	legacyTitle := "Legacy Page"
	legacy, err := uc.Execute(context.Background(), ManualSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("manual-save-base"),
		PageNumber:      2,
		UserID:          userID,
		Title:           &legacyTitle,
		Body:            "元の本文",
		DraftPageDigest: draftPageDigest(t, db, legacyDraftID, spaceID),
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}
	if legacy.DraftPageRevision == nil {
		t.Error("起点が分からない下書きの保存でリビジョンが作成されていない")
	}
}

func TestManualSaveDraftPageUsecase_Execute_BodyTooLong(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newManualSaveUC(db)

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("manual-save-too-long").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("manual-save-too-long@example.com").
		WithAtname("manualsavetoolong").
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
	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		WithBody("元の本文").
		Build()

	_, err := uc.Execute(context.Background(), ManualSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("manual-save-too-long"),
		PageNumber:      1,
		UserID:          userID,
		Body:            strings.Repeat("a", model.PageBodyMaxLength+1),
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatalf("ValidationErrorを期待したが、%vだった", err)
	}
	if !ve.HasFieldError("body") {
		t.Error("bodyのフィールドエラーが無い")
	}

	draftPage, err := repository.NewDraftPageRepository(query.New(db)).FindByPageAndMember(context.Background(), pageID, spaceMemberID, spaceID)
	if err != nil {
		t.Fatalf("下書きの取得のエラー = %v", err)
	}
	if draftPage != nil {
		t.Error("本文が上限を超えたのに下書きが保存された")
	}
}

// TestManualSaveDraftPageUsecase_Execute_DraftPageDigestは、手動保存も編集画面が前提にした下書きの版を照合し、
// 食い違うときは下書きもリビジョンも書かないことを確かめる
func TestManualSaveDraftPageUsecase_Execute_DraftPageDigest(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()

	t.Run("前提の版が今の版と一致すれば、保存してリビジョンを作る", func(t *testing.T) {
		t.Parallel()
		f := setupCreateAPIPageFixture(t, db, "manual-save-digest-match", apiTopicRegularMemberRole)
		pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[2]).WithNumber(1).WithTitle("ページ").Build()
		draftID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[2]).
			WithTitle("下書き").WithBody("下書きの本文").Build()
		digest := draftPageDigest(t, db, draftID, f.space.ID)

		output, err := newManualSaveUC(db).Execute(t.Context(), ManualSaveDraftPageInput{
			SpaceIdentifier: f.space.Identifier, PageNumber: 1, UserID: f.userID,
			Title: ptr("下書き"), Body: "画面の本文", DraftPageDigest: digest,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.DraftPageRevision == nil {
			t.Error("リビジョン = nil、期待値 = 作成したリビジョン")
		}
		if got := draftPageDigest(t, db, draftID, f.space.ID); got != output.DraftPage.ContentDigest() {
			t.Errorf("返した版 = %q、期待値 = 保存後の今の版 %q", output.DraftPage.ContentDigest(), got)
		}
	})

	t.Run("前提の版の後に下書きが変わっていれば、保存もリビジョンの作成もせずに競合を返す", func(t *testing.T) {
		t.Parallel()
		f := setupCreateAPIPageFixture(t, db, "manual-save-digest-mismatch", apiTopicRegularMemberRole)
		pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[2]).WithNumber(1).WithTitle("ページ").Build()
		draftID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[2]).
			WithTitle("下書き").WithBody("下書きの本文").Build()
		staleDigest := draftPageDigest(t, db, draftID, f.space.ID)
		execForTest(t, db, "UPDATE draft_pages SET body = 'APIの本文', modified_at = modified_at + interval '1 second' WHERE id = $1 AND space_id = $2", draftID, f.space.ID)
		currentDigest := draftPageDigest(t, db, draftID, f.space.ID)

		_, err := newManualSaveUC(db).Execute(t.Context(), ManualSaveDraftPageInput{
			SpaceIdentifier: f.space.Identifier, PageNumber: 1, UserID: f.userID,
			Title: ptr("下書き"), Body: "画面の本文", DraftPageDigest: staleDigest,
		})
		assertDraftPageConflict(t, err, currentDigest)
		if n := countRows(t, db, "draft_pages", f.space.ID, "id = $2 AND body = 'APIの本文'", draftID); n != 1 {
			t.Errorf("APIの本文を保った下書きの数 = %d、期待値 = 1", n)
		}
		if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2", draftID); n != 0 {
			t.Errorf("リビジョンの数 = %d、期待値 = 0", n)
		}
	})
}
