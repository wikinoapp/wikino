package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// A [[...]] written as code is not a link on the screen, so publishing must not create the page
// it names. The link outside the code keeps creating its page.
//
// [Ja] コードとして書かれた [[...]] は画面上でリンクにならないため、公開時にその名前のページを
// 作ってはならない。コードの外のリンクは従来どおりページを作る。
func TestPublishPageUsecase_Execute_CodeWikilinkCreatesNoPage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	attachmentRepo := repository.NewAttachmentRepository(q)
	refRepo := repository.NewPageAttachmentReferenceRepository(q)
	uc := NewPublishPageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, pageRevisionRepo,
		pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo,
		attachmentRepo, refRepo, validator.NewPageUpdateValidator(pageRepo))

	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("publish-code-wikilink").Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("publish-code-wikilink@example.com").WithAtname("publishcodewikilink").Build()
	memberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithName("General").Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).WithTopicID(topicID).WithSpaceMemberID(memberID).Build()
	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("Code Wikilink").Build()

	body := "See [[テキストのリンク先]]\n\n```\n[[ブロックのリンク先]]\n```\n\n" +
		"Write `[[インラインのリンク先]]` and <code>[[HTMLのリンク先]]</code>"
	testutil.NewDraftPageBuilderDB(t, db).WithSpaceID(spaceID).WithPageID(pageID).
		WithSpaceMemberID(memberID).WithTopicID(topicID).WithTitle("Code Wikilink").WithBody(body).Build()

	output, err := uc.Execute(ctx, PublishPageInput{
		SpaceIdentifier: model.SpaceIdentifier("publish-code-wikilink"), PageNumber: 1, UserID: userID,
		Title: "Code Wikilink", Body: body,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, title := range []string{"ブロックのリンク先", "インラインのリンク先", "HTMLのリンク先"} {
		page, err := pageRepo.FindByTopicAndTitle(ctx, topicID, title, spaceID)
		if err != nil {
			t.Fatalf("FindByTopicAndTitle(%q) error = %v", title, err)
		}
		if page != nil {
			t.Errorf("page %q was created from a wiki link written as code", title)
		}
	}

	linked, err := pageRepo.FindByTopicAndTitle(ctx, topicID, "テキストのリンク先", spaceID)
	if err != nil {
		t.Fatalf("FindByTopicAndTitle() error = %v", err)
	}
	if linked == nil {
		t.Fatal("page テキストのリンク先 was not created from the wiki link outside code")
	}
	if len(output.Page.LinkedPageIDs) != 1 || output.Page.LinkedPageIDs[0] != linked.ID {
		t.Errorf("LinkedPageIDs = %v, want only %v", output.Page.LinkedPageIDs, linked.ID)
	}
	if !strings.Contains(output.Page.BodyHTML, "テキストのリンク先</a>") {
		t.Errorf("published HTML has no link to the created page: %s", output.Page.BodyHTML)
	}
}
