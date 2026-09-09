package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newCreatePageUC(db *sql.DB) *CreatePageUsecase {
	q := query.New(db)
	return NewCreatePageUsecase(
		db,
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewPageEditorRepository(q),
		repository.NewDraftPageRepository(q),
		repository.NewAttachmentRepository(q),
	)
}

// createPageFixture is the fixture set shared by the page creation UseCase tests.
//
// [Ja] createPageFixture はページ作成 UseCase のテストで共有するフィクスチャ一式。
type createPageFixture struct {
	userID          model.UserID
	spaceID         model.SpaceID
	spaceIdentifier model.SpaceIdentifier
	spaceMemberID   model.SpaceMemberID
	topicID         model.TopicID
}

// setupCreatePageFixture creates a space, member, topic and topic member committed directly to
// the test DB, because the UseCase manages its own transaction. prefix and atname both keep
// identifiers unique across parallel tests sharing the test DB; atname is passed separately
// because it must also stay within validator.AtnameMaxLength, which the prefixes exceed. Nil
// scope arguments use each builder's defaults.
//
// [Ja] setupCreatePageFixture はスペース・メンバー・トピック・トピックメンバーをテスト DB へ
// 直接コミットして作成する (UseCase が自前でトランザクションを管理するため)。prefix と atname は
// どちらもテスト DB を共有する並行テスト間で識別子を一意に保つ。atname を別に受け取るのは、
// prefix では超えてしまう validator.AtnameMaxLength にも収める必要があるため。スコープの引数が
// nil の場合は各ビルダーの既定値を使う。
func setupCreatePageFixture(
	t *testing.T,
	db *sql.DB,
	prefix string,
	atname string,
	spaceMemberScopes []model.Scope,
	topicMemberScopes []model.Scope,
) createPageFixture {
	t.Helper()

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail(prefix + "@example.com").
		WithAtname(atname).
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier(prefix + "-space").
		Build()

	spaceMemberBuilder := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID)
	if spaceMemberScopes != nil {
		spaceMemberBuilder = spaceMemberBuilder.WithScopes(spaceMemberScopes)
	}
	spaceMemberID := spaceMemberBuilder.Build()

	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	topicMemberBuilder := testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID)
	if topicMemberScopes != nil {
		topicMemberBuilder = topicMemberBuilder.WithScopes(topicMemberScopes)
	}
	topicMemberBuilder.Build()

	return createPageFixture{
		userID:          userID,
		spaceID:         spaceID,
		spaceIdentifier: model.SpaceIdentifier(prefix + "-space"),
		spaceMemberID:   spaceMemberID,
		topicID:         topicID,
	}
}

// findDraftPage returns the fixture member's draft for a page, or nil when none exists.
//
// [Ja] findDraftPage はフィクスチャのメンバーが作成したページの下書きを取得する。
// 下書きが存在しない場合は nil を返す。
func findDraftPage(t *testing.T, db *sql.DB, f createPageFixture, pageID model.PageID) *model.DraftPage {
	t.Helper()

	draftPage, err := repository.NewDraftPageRepository(query.New(db)).
		FindByPageAndMember(context.Background(), pageID, f.spaceMemberID, f.spaceID)
	if err != nil {
		t.Fatalf("FindByPageAndMember() error = %v", err)
	}
	return draftPage
}

func TestCreatePageUsecase_Execute_WithoutPrefilledContent(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newCreatePageUC(db)
	f := setupCreatePageFixture(t, db, "create-page-blank", "cpblank", nil, nil)

	output, err := uc.Execute(context.Background(), CreatePageInput{
		SpaceIdentifier: f.spaceIdentifier,
		TopicNumber:     1,
		UserID:          f.userID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if output.Page == nil {
		t.Fatal("Page should not be nil")
	}
	if output.Page.Title != nil {
		t.Errorf("Page.Title = %v, want nil", output.Page.Title)
	}
	if output.Page.Body != "" {
		t.Errorf("Page.Body = %q, want empty string", output.Page.Body)
	}
	if output.Page.PublishedAt != nil {
		t.Errorf("Page.PublishedAt = %v, want nil", output.Page.PublishedAt)
	}
	if output.Page.TopicID != f.topicID {
		t.Errorf("Page.TopicID = %v, want %v", output.Page.TopicID, f.topicID)
	}

	if draftPage := findDraftPage(t, db, f, output.Page.ID); draftPage != nil {
		t.Errorf("DraftPage = %v, want nil", draftPage)
	}

	// The creator is also registered as an editor.
	//
	// [Ja] 作成者は編集者としても登録される。
	pageEditor, err := repository.NewPageEditorRepository(query.New(db)).
		FindByPageAndSpaceMember(context.Background(), repository.FindByPageAndSpaceMemberInput{
			SpaceID:       f.spaceID,
			PageID:        output.Page.ID,
			SpaceMemberID: f.spaceMemberID,
		})
	if err != nil {
		t.Fatalf("FindByPageAndSpaceMember() error = %v", err)
	}
	if pageEditor == nil {
		t.Fatal("PageEditor should not be nil")
	}
}

func TestCreatePageUsecase_Execute_WithTitleOnly(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newCreatePageUC(db)
	f := setupCreatePageFixture(t, db, "create-page-title", "cptitle", nil, nil)

	output, err := uc.Execute(context.Background(), CreatePageInput{
		SpaceIdentifier: f.spaceIdentifier,
		TopicNumber:     1,
		UserID:          f.userID,
		Title:           "事前入力タイトル",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	draftPage := findDraftPage(t, db, f, output.Page.ID)
	if draftPage == nil {
		t.Fatal("DraftPage should not be nil")
	}
	if draftPage.Title == nil || *draftPage.Title != "事前入力タイトル" {
		t.Errorf("DraftPage.Title = %v, want %q", draftPage.Title, "事前入力タイトル")
	}
	if draftPage.Body != "" {
		t.Errorf("DraftPage.Body = %q, want empty string", draftPage.Body)
	}

	// The page itself stays blank because publishing requires an explicit action on the edit page.
	//
	// [Ja] 公開には編集画面での明示的な操作が必要なため、ページ本体は空のままになる。
	if output.Page.Title != nil {
		t.Errorf("Page.Title = %v, want nil", output.Page.Title)
	}
}

func TestCreatePageUsecase_Execute_WithBodyOnly(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newCreatePageUC(db)
	f := setupCreatePageFixture(t, db, "create-page-body", "cpbody", nil, nil)

	output, err := uc.Execute(context.Background(), CreatePageInput{
		SpaceIdentifier: f.spaceIdentifier,
		TopicNumber:     1,
		UserID:          f.userID,
		Body:            "事前入力された本文",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	draftPage := findDraftPage(t, db, f, output.Page.ID)
	if draftPage == nil {
		t.Fatal("DraftPage should not be nil")
	}
	if draftPage.Title != nil {
		t.Errorf("DraftPage.Title = %v, want nil", draftPage.Title)
	}
	if draftPage.Body != "事前入力された本文" {
		t.Errorf("DraftPage.Body = %q, want %q", draftPage.Body, "事前入力された本文")
	}
}

func TestCreatePageUsecase_Execute_WithTitleAndBody(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newCreatePageUC(db)
	f := setupCreatePageFixture(t, db, "create-page-both", "cpboth", nil, nil)

	output, err := uc.Execute(context.Background(), CreatePageInput{
		SpaceIdentifier: f.spaceIdentifier,
		TopicNumber:     1,
		UserID:          f.userID,
		Title:           "記事タイトル - example.com",
		Body:            "https://example.com/article\n\n> 引用文",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	draftPage := findDraftPage(t, db, f, output.Page.ID)
	if draftPage == nil {
		t.Fatal("DraftPage should not be nil")
	}
	if draftPage.Title == nil || *draftPage.Title != "記事タイトル - example.com" {
		t.Errorf("DraftPage.Title = %v, want %q", draftPage.Title, "記事タイトル - example.com")
	}
	if draftPage.Body != "https://example.com/article\n\n> 引用文" {
		t.Errorf("DraftPage.Body = %q, want the prefilled body", draftPage.Body)
	}
	// The body is rendered through the same path as auto save.
	//
	// [Ja] 本文は自動保存と同じ経路でレンダリングされる。
	if !strings.Contains(draftPage.BodyHTML, "blockquote") {
		t.Errorf("DraftPage.BodyHTML = %q, want it to contain a rendered blockquote", draftPage.BodyHTML)
	}
}

func TestCreatePageUsecase_Execute_WithFeaturedImage(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newCreatePageUC(db)
	f := setupCreatePageFixture(t, db, "create-page-featured-image", "cpfeaturedimg", nil, nil)
	attachmentID := testutil.NewAttachmentBuilderDB(t, db).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.spaceMemberID).
		WithFilename("featured-image.png").
		Build()
	body := fmt.Sprintf("![image](/attachments/%s)\n\n本文", attachmentID)

	output, err := uc.Execute(context.Background(), CreatePageInput{
		SpaceIdentifier: f.spaceIdentifier,
		TopicNumber:     1,
		UserID:          f.userID,
		Body:            body,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	draftPage := findDraftPage(t, db, f, output.Page.ID)
	if draftPage == nil {
		t.Fatal("DraftPage should not be nil")
	}
	if draftPage.FeaturedImageAttachmentID == nil {
		t.Fatal("DraftPage.FeaturedImageAttachmentID should not be nil")
	}
	if *draftPage.FeaturedImageAttachmentID != attachmentID {
		t.Errorf(
			"DraftPage.FeaturedImageAttachmentID = %v, want %v",
			*draftPage.FeaturedImageAttachmentID,
			attachmentID,
		)
	}
}

func TestCreatePageUsecase_Execute_WithWikilink(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newCreatePageUC(db)
	f := setupCreatePageFixture(t, db, "create-page-wikilink", "cpwikilink", nil, nil)

	output, err := uc.Execute(context.Background(), CreatePageInput{
		SpaceIdentifier: f.spaceIdentifier,
		TopicNumber:     1,
		UserID:          f.userID,
		Body:            "タグ: [[リンク先ページ]]",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	draftPage := findDraftPage(t, db, f, output.Page.ID)
	if draftPage == nil {
		t.Fatal("DraftPage should not be nil")
	}
	if len(draftPage.LinkedPageIDs) == 0 {
		t.Error("DraftPage.LinkedPageIDs should not be empty")
	}

	linkedPage, err := repository.NewPageRepository(query.New(db)).
		FindByTopicAndTitle(context.Background(), f.topicID, "リンク先ページ", f.spaceID)
	if err != nil {
		t.Fatalf("FindByTopicAndTitle() error = %v", err)
	}
	if linkedPage == nil {
		t.Fatal("リンク先ページ should be created")
	}
}

func TestCreatePageUsecase_Execute_SpaceNotFound(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newCreatePageUC(db)
	f := setupCreatePageFixture(t, db, "create-page-nospace", "cpnospace", nil, nil)

	_, err := uc.Execute(context.Background(), CreatePageInput{
		SpaceIdentifier: model.SpaceIdentifier("create-page-nospace-missing"),
		TopicNumber:     1,
		UserID:          f.userID,
	})
	assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
}

func TestCreatePageUsecase_Execute_TopicNotFound(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newCreatePageUC(db)
	f := setupCreatePageFixture(t, db, "create-page-notopic", "cpnotopic", nil, nil)

	_, err := uc.Execute(context.Background(), CreatePageInput{
		SpaceIdentifier: f.spaceIdentifier,
		TopicNumber:     999,
		UserID:          f.userID,
	})
	assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
}

func TestCreatePageUsecase_Execute_NotSpaceMember(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newCreatePageUC(db)
	f := setupCreatePageFixture(t, db, "create-page-nonmember", "cpnonmember", nil, nil)

	otherUserID := testutil.NewUserBuilderDB(t, db).
		WithEmail("create-page-nonmember-other@example.com").
		WithAtname("cpnonmemberother").
		Build()

	_, err := uc.Execute(context.Background(), CreatePageInput{
		SpaceIdentifier: f.spaceIdentifier,
		TopicNumber:     1,
		UserID:          otherUserID,
	})
	assertAppErrCode(t, err, model.AppErrCodeForbidden)
}

func TestCreatePageUsecase_Execute_WithTopicPageWriteScope(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newCreatePageUC(db)
	f := setupCreatePageFixture(
		t,
		db,
		"create-page-topic-writer",
		"cptopicwriter",
		[]model.Scope{model.ScopePageRead},
		[]model.Scope{model.ScopePageWrite},
	)

	output, err := uc.Execute(context.Background(), CreatePageInput{
		SpaceIdentifier: f.spaceIdentifier,
		TopicNumber:     1,
		UserID:          f.userID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if output.Page == nil {
		t.Fatal("Page should not be nil")
	}
}

func TestCreatePageUsecase_Execute_WithSpacePageWriteScopeWithoutTopicMembership(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newCreatePageUC(db)
	f := setupCreatePageFixture(
		t,
		db,
		"create-page-space-writer",
		"cpspacewriter",
		[]model.Scope{model.ScopePageWrite},
		nil,
	)
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(f.spaceID).
		WithNumber(2).
		WithName("Without membership").
		Build()

	output, err := uc.Execute(context.Background(), CreatePageInput{
		SpaceIdentifier: f.spaceIdentifier,
		TopicNumber:     2,
		UserID:          f.userID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if output.Page == nil {
		t.Fatal("Page should not be nil")
	}
	if output.Page.TopicID != topicID {
		t.Errorf("Page.TopicID = %v, want %v", output.Page.TopicID, topicID)
	}
}

func TestCreatePageUsecase_Execute_WithoutPageWriteScope(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc := newCreatePageUC(db)
	f := setupCreatePageFixture(
		t,
		db,
		"create-page-readonly",
		"cpreadonly",
		[]model.Scope{model.ScopePageRead},
		[]model.Scope{model.ScopePageRead},
	)

	_, err := uc.Execute(context.Background(), CreatePageInput{
		SpaceIdentifier: f.spaceIdentifier,
		TopicNumber:     1,
		UserID:          f.userID,
	})
	assertAppErrCode(t, err, model.AppErrCodeForbidden)
}
