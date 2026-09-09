package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetPageBacklinksUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetPageBacklinksUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
	)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpb-owner@example.com").
		WithAtname("gpbowner").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gpb-space").
		WithName("GPB Space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	publicTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Public").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()
	privateTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Private").
		WithVisibility(int32(model.TopicVisibilityPrivate)).
		Build()

	targetPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Target Page").
		Build()

	publicLinkerID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(2).
		WithTitle("Public Linker").
		WithLinkedPageIDs([]model.PageID{targetPageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(3).
		WithTitle("Private Linker").
		WithLinkedPageIDs([]model.PageID{targetPageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(4).
		WithTitle("Trashed Linker").
		WithLinkedPageIDs([]model.PageID{targetPageID}).
		WithTrashed().
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(10).
		WithTitle("Private Target Page").
		Build()

	t.Run("正常系: メンバーは非公開トピックのバックリンクも見えるがゴミ箱のページは見えない", func(t *testing.T) {
		userID := ownerID
		output, err := uc.Execute(context.Background(), GetPageBacklinksInput{
			SpaceIdentifier: "gpb-space",
			PageNumber:      1,
			UserID:          &userID,
			CurrentPage:     1,
			Limit:           15,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(output.Backlinks) != 2 {
			t.Fatalf("len(Backlinks) = %d, want 2", len(output.Backlinks))
		}
		if output.TotalCount != 2 {
			t.Errorf("TotalCount = %d, want 2", output.TotalCount)
		}
		if !output.CanUpdatePage {
			t.Error("CanUpdatePage should be true for a member holding page:write")
		}
	})

	t.Run("正常系: ゲストは公開トピックのバックリンクだけを見る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageBacklinksInput{
			SpaceIdentifier: "gpb-space",
			PageNumber:      1,
			CurrentPage:     1,
			Limit:           15,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.SpaceMember != nil {
			t.Error("SpaceMember should be nil for a guest")
		}
		if len(output.Backlinks) != 1 {
			t.Fatalf("len(Backlinks) = %d, want 1", len(output.Backlinks))
		}
		if output.Backlinks[0].ID != publicLinkerID {
			t.Errorf("Backlinks[0].ID = %v, want the public linker", output.Backlinks[0].ID)
		}
		if output.TotalCount != 1 {
			t.Errorf("TotalCount = %d, want 1", output.TotalCount)
		}
		if output.CanUpdatePage {
			t.Error("CanUpdatePage should be false for a guest")
		}
	})

	t.Run("異常系: ゲストは非公開トピックのページのバックリンクを取得できない", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), GetPageBacklinksInput{
			SpaceIdentifier: "gpb-space",
			PageNumber:      10,
			CurrentPage:     1,
			Limit:           15,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})
}

func TestGetPageBacklinksUsecase_Execute_AuthorizationBoundaries(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetPageBacklinksUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
	)

	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpb-auth-non-member@example.com").
		WithAtname("gpbauthnonmember").
		Build()
	restrictedMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpb-auth-restricted@example.com").
		WithAtname("gpbauthrestricted").
		Build()
	trashMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpb-auth-trash@example.com").
		WithAtname("gpbauthtrash").
		Build()
	fullMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpb-auth-full@example.com").
		WithAtname("gpbauthfull").
		Build()
	topicReaderID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpb-auth-topic-reader@example.com").
		WithAtname("gpbauthtopicreader").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gpb-auth-space").
		WithName("GPB Authorization Space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(restrictedMemberID).
		WithScopes([]model.Scope{model.ScopePageWrite}).
		Build()
	topicReaderSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(topicReaderID).
		WithScopes([]model.Scope{}).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(trashMemberID).
		WithScopes([]model.Scope{model.ScopePageTrash}).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(fullMemberID).
		Build()

	publicTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Public").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()
	privateTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Private").
		WithVisibility(int32(model.TopicVisibilityPrivate)).
		Build()
	privateTopicBID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(3).
		WithName("Private B").
		WithVisibility(int32(model.TopicVisibilityPrivate)).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithSpaceMemberID(topicReaderSpaceMemberID).
		WithScopes([]model.Scope{model.ScopeTopicRead}).
		Build()
	publicTargetID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Public Target").
		Build()
	publicLinkerID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(2).
		WithTitle("Public Linker").
		WithLinkedPageIDs([]model.PageID{publicTargetID}).
		Build()
	privateLinkerID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(3).
		WithTitle("Private Linker").
		WithLinkedPageIDs([]model.PageID{publicTargetID}).
		Build()
	privateLinkerBID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicBID).
		WithNumber(4).
		WithTitle("Private B Linker").
		WithLinkedPageIDs([]model.PageID{publicTargetID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(10).
		WithTitle("Private Target").
		Build()
	trashedTargetID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(11).
		WithTitle("Trashed Target").
		WithTrashed().
		Build()
	trashedTargetLinkerID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(12).
		WithTitle("Trashed Target Linker").
		WithLinkedPageIDs([]model.PageID{trashedTargetID}).
		Build()

	tests := []struct {
		name               string
		pageNumber         int32
		userID             *model.UserID
		wantNotFound       bool
		wantBacklinkIDs    []model.PageID
		wantSpaceMemberNil bool
	}{
		{
			name:               "正常系: ゲストは公開トピックのバックリンクだけを見る",
			pageNumber:         1,
			wantBacklinkIDs:    []model.PageID{publicLinkerID},
			wantSpaceMemberNil: true,
		},
		{
			name:               "正常系: ログイン済み非メンバーはゲストと同じバックリンクだけを見る",
			pageNumber:         1,
			userID:             &nonMemberID,
			wantBacklinkIDs:    []model.PageID{publicLinkerID},
			wantSpaceMemberNil: true,
		},
		{
			name:            "正常系: topic:readを持たないメンバーには非公開トピックのバックリンクが見えない",
			pageNumber:      1,
			userID:          &restrictedMemberID,
			wantBacklinkIDs: []model.PageID{publicLinkerID},
		},
		{
			name:            "正常系: トピック単位のtopic:readを持つメンバーは参加中の非公開トピックだけを見る",
			pageNumber:      1,
			userID:          &topicReaderID,
			wantBacklinkIDs: []model.PageID{publicLinkerID, privateLinkerID},
		},
		{
			name:            "正常系: 全トピックを開けるメンバーは非公開トピックのバックリンクも見える",
			pageNumber:      1,
			userID:          &fullMemberID,
			wantBacklinkIDs: []model.PageID{publicLinkerID, privateLinkerID, privateLinkerBID},
		},
		{
			name:         "異常系: ログイン済み非メンバーは非公開ページを取得できない",
			pageNumber:   10,
			userID:       &nonMemberID,
			wantNotFound: true,
		},
		{
			name:         "異常系: topic:readを持たないメンバーは非公開ページを取得できない",
			pageNumber:   10,
			userID:       &restrictedMemberID,
			wantNotFound: true,
		},
		{
			name:         "異常系: ゲストはゴミ箱のページを取得できない",
			pageNumber:   11,
			wantNotFound: true,
		},
		{
			name:         "異常系: page:trashを持たないメンバーはゴミ箱のページを取得できない",
			pageNumber:   11,
			userID:       &restrictedMemberID,
			wantNotFound: true,
		},
		{
			name:            "正常系: page:trashを持つメンバーはゴミ箱のページを取得できる",
			pageNumber:      11,
			userID:          &trashMemberID,
			wantBacklinkIDs: []model.PageID{trashedTargetLinkerID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(context.Background(), GetPageBacklinksInput{
				SpaceIdentifier: "gpb-auth-space",
				PageNumber:      tt.pageNumber,
				UserID:          tt.userID,
				CurrentPage:     1,
				Limit:           15,
			})
			if tt.wantNotFound {
				assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
				return
			}
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(output.Backlinks) != len(tt.wantBacklinkIDs) {
				t.Fatalf("len(Backlinks) = %d, want %d", len(output.Backlinks), len(tt.wantBacklinkIDs))
			}
			gotBacklinkIDs := make(map[model.PageID]struct{}, len(output.Backlinks))
			for _, page := range output.Backlinks {
				gotBacklinkIDs[page.ID] = struct{}{}
			}
			for _, wantPageID := range tt.wantBacklinkIDs {
				if _, ok := gotBacklinkIDs[wantPageID]; !ok {
					t.Errorf("Backlinks does not contain page ID %v", wantPageID)
				}
			}
			if tt.wantSpaceMemberNil && output.SpaceMember != nil {
				t.Error("SpaceMember should be nil for a viewer who is not a space member")
			}
		})
	}
}
