package usecase

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestPublishPageUsecase_ReferenceDefinitionInsideHiddenContent(t *testing.T) {
	t.Parallel()

	for _, element := range []string{"object", "textarea"} {
		for _, existing := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/existing=%t", element, existing), func(t *testing.T) {
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

				identifier := fmt.Sprintf("pub-ref-%s-%t", element, existing)
				spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
				userID := testutil.NewUserBuilderDB(t, db).
					WithEmail(identifier + "@example.com").WithAtname(strings.ReplaceAll(identifier, "-", "")).Build()
				memberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()
				topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).Build()
				testutil.NewTopicMemberBuilderDB(t, db).
					WithSpaceID(spaceID).WithTopicID(topicID).WithSpaceMemberID(memberID).Build()
				attachmentID := testutil.NewAttachmentBuilderDB(t, db).
					WithSpaceID(spaceID).WithSpaceMemberID(memberID).WithFilename("visible.pdf").Build()
				removedID := testutil.NewAttachmentBuilderDB(t, db).
					WithSpaceID(spaceID).WithSpaceMemberID(memberID).WithFilename("removed.pdf").Build()
				pageID := testutil.NewPageBuilderDB(t, db).
					WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("References").Build()

				oldIDs := []model.AttachmentID{removedID}
				if existing {
					oldIDs = append(oldIDs, attachmentID)
				}
				oldRefs, err := refRepo.CreateBatch(ctx, pageID, spaceID, oldIDs)
				if err != nil {
					t.Fatalf("CreateBatch() error = %v", err)
				}
				body := fmt.Sprintf("[visible][r]\n\nx <%s>\n\n[r]: /attachments/%s\n\n</%s>", element, attachmentID, element)
				testutil.NewDraftPageBuilderDB(t, db).WithSpaceID(spaceID).WithPageID(pageID).
					WithSpaceMemberID(memberID).WithTopicID(topicID).WithTitle("References").WithBody(body).Build()

				output, err := uc.Execute(ctx, PublishPageInput{
					SpaceIdentifier: model.SpaceIdentifier(identifier), PageNumber: 1, UserID: userID,
					Title: "References", Body: body,
				})
				if err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				refs, err := refRepo.ListByPageID(ctx, pageID, spaceID)
				if err != nil {
					t.Fatalf("ListByPageID() error = %v", err)
				}
				if len(refs) != 1 || refs[0].AttachmentID != attachmentID {
					t.Fatalf("references = %+v, want only %s", refs, attachmentID)
				}
				if existing && refs[0].ID != oldRefs[1].ID {
					t.Errorf("existing reference was replaced: got %s, want %s", refs[0].ID, oldRefs[1].ID)
				}
				if !strings.Contains(output.Page.BodyHTML, fmt.Sprintf(`data-attachment-id="%s"`, attachmentID)) {
					t.Errorf("published HTML has no live attachment: %s", output.Page.BodyHTML)
				}
			})
		}
	}
}
