package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestAutoSaveDraftPageUsecase_Execute_NewDraftPage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	uc := NewAutoSaveDraftPageUsecase(db, spaceRepo, spaceMemberRepo, draftPageRepo, pageRepo, pageEditorRepo, topicRepo, topicMemberRepo, attachmentRepo, validator.NewDraftPageSaveValidator())

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("auto-save-new").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("auto-save-new@example.com").
		WithAtname("autosavenew").
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

	title := "テスト下書き"
	output, err := uc.Execute(context.Background(), AutoSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("auto-save-new"),
		PageNumber:      1,
		UserID:          userID,
		Title:           &title,
		Body:            "Hello, world!",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}
	if output == nil {
		t.Fatal("出力がnil")
	}
	if output.DraftPage == nil {
		t.Fatal("DraftPageがnil")
	}
	if output.DraftPage.Body != "Hello, world!" {
		t.Errorf("Body = %q、期待値 = %q", output.DraftPage.Body, "Hello, world!")
	}
	// 下書きを作るときに、ページの今の内容を起点として記録する
	wantBase := model.DraftPageContent{Title: "Test Page", Body: "Test body"}
	if got := output.DraftPage.BaseContent(); got == nil || *got != wantBase {
		t.Errorf("起点 = %+v、期待値 = %+v", got, wantBase)
	}
	if output.ModifiedAt.IsZero() {
		t.Error("ModifiedAtがゼロ値")
	}
}

func TestAutoSaveDraftPageUsecase_Execute_ExistingDraftPage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	uc := NewAutoSaveDraftPageUsecase(db, spaceRepo, spaceMemberRepo, draftPageRepo, pageRepo, pageEditorRepo, topicRepo, topicMemberRepo, attachmentRepo, validator.NewDraftPageSaveValidator())

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("auto-save-existing").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("auto-save-existing@example.com").
		WithAtname("autosaveexisting").
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

	// 1回目: DraftPage作成
	title1 := "初回タイトル"
	output1, err := uc.Execute(context.Background(), AutoSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("auto-save-existing"),
		PageNumber:      1,
		UserID:          userID,
		Title:           &title1,
		Body:            "初回本文",
	})
	if err != nil {
		t.Fatalf("1回目のExecute()のエラー = %v、期待値 = nil", err)
	}

	// 2回目: 同じDraftPageを更新
	title2 := "更新タイトル"
	output2, err := uc.Execute(context.Background(), AutoSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("auto-save-existing"),
		PageNumber:      1,
		UserID:          userID,
		Title:           &title2,
		Body:            "更新本文",
	})
	if err != nil {
		t.Fatalf("2回目のExecute()のエラー = %v、期待値 = nil", err)
	}

	// 同じDraftPageが更新されていることを確認
	if output2.DraftPage.ID != output1.DraftPage.ID {
		t.Errorf("DraftPageのIDが変わった: %v、期待値 = %v", output2.DraftPage.ID, output1.DraftPage.ID)
	}
	if output2.DraftPage.Body != "更新本文" {
		t.Errorf("Body = %q、期待値 = %q", output2.DraftPage.Body, "更新本文")
	}
	// 起点は下書きを作った後の保存で書き換えない
	wantBase := model.DraftPageContent{Title: "Test Page", Body: "Test body"}
	if got := output2.DraftPage.BaseContent(); got == nil || *got != wantBase {
		t.Errorf("起点 = %+v、期待値 = %+v", got, wantBase)
	}
}

func TestAutoSaveDraftPageUsecase_Execute_EmptyBody(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	uc := NewAutoSaveDraftPageUsecase(db, spaceRepo, spaceMemberRepo, draftPageRepo, pageRepo, pageEditorRepo, topicRepo, topicMemberRepo, attachmentRepo, validator.NewDraftPageSaveValidator())

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("auto-save-empty").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("auto-save-empty@example.com").
		WithAtname("autosaveempty").
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

	output, err := uc.Execute(context.Background(), AutoSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("auto-save-empty"),
		PageNumber:      1,
		UserID:          userID,
		Title:           nil,
		Body:            "",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}
	if output.DraftPage.Body != "" {
		t.Errorf("Body = %q、期待値 = 空文字列", output.DraftPage.Body)
	}
}

func TestAutoSaveDraftPageUsecase_Execute_WithWikilinks(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	uc := NewAutoSaveDraftPageUsecase(db, spaceRepo, spaceMemberRepo, draftPageRepo, pageRepo, pageEditorRepo, topicRepo, topicMemberRepo, attachmentRepo, validator.NewDraftPageSaveValidator())

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("auto-save-wikilink").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("auto-save-wikilink@example.com").
		WithAtname("autosavewikilink").
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

	// Wikilinkを含む本文で自動保存
	title := "Wikilink Test"
	output, err := uc.Execute(context.Background(), AutoSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("auto-save-wikilink"),
		PageNumber:      1,
		UserID:          userID,
		Title:           &title,
		Body:            "See [[リンク先ページ]]",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}

	// リンク先ページが自動作成され、LinkedPageIDsに含まれることを確認
	if len(output.DraftPage.LinkedPageIDs) == 0 {
		t.Error("LinkedPageIDsが空")
	}
}

func TestAutoSaveDraftPageUsecase_Execute_WikilinkExistingPage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	uc := NewAutoSaveDraftPageUsecase(db, spaceRepo, spaceMemberRepo, draftPageRepo, pageRepo, pageEditorRepo, topicRepo, topicMemberRepo, attachmentRepo, validator.NewDraftPageSaveValidator())

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("auto-save-existing-page").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("auto-save-existingpage@example.com").
		WithAtname("autosaveexistingpage").
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

	// リンク先ページを事前に作成
	existingPageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("既存ページ").
		Build()

	// 既存ページへのWikilinkを含む本文で自動保存
	title := "Existing Page Link Test"
	output, err := uc.Execute(context.Background(), AutoSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("auto-save-existing-page"),
		PageNumber:      1,
		UserID:          userID,
		Title:           &title,
		Body:            "See [[既存ページ]]",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}

	// LinkedPageIDsに既存ページのIDが含まれることを確認
	if len(output.DraftPage.LinkedPageIDs) != 1 {
		t.Fatalf("LinkedPageIDsの長さ = %d、期待値 = 1", len(output.DraftPage.LinkedPageIDs))
	}
	if output.DraftPage.LinkedPageIDs[0] != existingPageID {
		t.Errorf("LinkedPageIDs[0] = %v、期待値 = %v", output.DraftPage.LinkedPageIDs[0], existingPageID)
	}
}

func TestAutoSaveDraftPageUsecase_Execute_WikilinkCreatesPageEditor(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	uc := NewAutoSaveDraftPageUsecase(db, spaceRepo, spaceMemberRepo, draftPageRepo, pageRepo, pageEditorRepo, topicRepo, topicMemberRepo, attachmentRepo, validator.NewDraftPageSaveValidator())

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("auto-save-editor").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("auto-save-editor@example.com").
		WithAtname("autosaveeditor").
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

	// Wikilinkを含む本文で自動保存
	title := "Editor Test"
	output, err := uc.Execute(context.Background(), AutoSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("auto-save-editor"),
		PageNumber:      1,
		UserID:          userID,
		Title:           &title,
		Body:            "See [[自動作成ページ]]",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}

	// 自動作成されたページのIDを取得
	if len(output.DraftPage.LinkedPageIDs) == 0 {
		t.Fatal("LinkedPageIDsが空")
	}
	linkedPageID := output.DraftPage.LinkedPageIDs[0]

	// 自動作成されたページにpage_editorsが作成されていることを確認
	editor, err := pageEditorRepo.FindByPageAndSpaceMember(context.Background(), repository.FindByPageAndSpaceMemberInput{
		SpaceID:       spaceID,
		PageID:        linkedPageID,
		SpaceMemberID: spaceMemberID,
	})
	if err != nil {
		t.Fatalf("自動作成ページのpage_editorsが見つかりません: %v", err)
	}
	if editor.PageID != linkedPageID {
		t.Errorf("PageID = %v、期待値 = %v", editor.PageID, linkedPageID)
	}
	if editor.SpaceMemberID != spaceMemberID {
		t.Errorf("SpaceMemberID = %v、期待値 = %v", editor.SpaceMemberID, spaceMemberID)
	}
}

func TestAutoSaveDraftPageUsecase_Execute_WikilinkDiscardedPage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	uc := NewAutoSaveDraftPageUsecase(db, spaceRepo, spaceMemberRepo, draftPageRepo, pageRepo, pageEditorRepo, topicRepo, topicMemberRepo, attachmentRepo, validator.NewDraftPageSaveValidator())

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("auto-save-discarded").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("auto-save-discarded@example.com").
		WithAtname("autosavediscarded").
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

	// 廃棄済みページを作成
	discardedPageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("廃棄済みページ").
		WithDiscarded().
		Build()

	// 廃棄済みページと同名のWikilinkを含む本文で自動保存
	title := "Discarded Link Test"
	output, err := uc.Execute(context.Background(), AutoSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("auto-save-discarded"),
		PageNumber:      1,
		UserID:          userID,
		Title:           &title,
		Body:            "See [[廃棄済みページ]]",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}

	// LinkedPageIDsに廃棄済みページのIDが含まれることを確認
	if len(output.DraftPage.LinkedPageIDs) != 1 {
		t.Fatalf("LinkedPageIDsの長さ = %d、期待値 = 1", len(output.DraftPage.LinkedPageIDs))
	}
	if output.DraftPage.LinkedPageIDs[0] != discardedPageID {
		t.Errorf("LinkedPageIDs[0] = %v、期待値 = %v", output.DraftPage.LinkedPageIDs[0], discardedPageID)
	}
}

func TestUniqueTopicNames(t *testing.T) {
	t.Parallel()

	keys := []markup.WikilinkKey{
		{TopicName: "General", PageTitle: "Page1"},
		{TopicName: "General", PageTitle: "Page2"},
		{TopicName: "Tech", PageTitle: "Page3"},
		{TopicName: "General", PageTitle: "Page1"},
	}

	names := uniqueTopicNames(keys)

	if len(names) != 2 {
		t.Fatalf("len(names) = %d、期待値 = 2", len(names))
	}
	if names[0] != "General" {
		t.Errorf("names[0] = %q、期待値 = %q", names[0], "General")
	}
	if names[1] != "Tech" {
		t.Errorf("names[1] = %q、期待値 = %q", names[1], "Tech")
	}
}

func TestAutoSaveDraftPageUsecase_Execute_BodyTooLong(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	uc := NewAutoSaveDraftPageUsecase(db, repository.NewSpaceRepository(q), repository.NewSpaceMemberRepository(q),
		repository.NewDraftPageRepository(q), repository.NewPageRepository(q), repository.NewPageEditorRepository(q),
		repository.NewTopicRepository(q), repository.NewTopicMemberRepository(q), repository.NewAttachmentRepository(q),
		validator.NewDraftPageSaveValidator())

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("auto-save-too-long").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("auto-save-too-long@example.com").
		WithAtname("autosavetoolong").
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

	_, err := uc.Execute(context.Background(), AutoSaveDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("auto-save-too-long"),
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
