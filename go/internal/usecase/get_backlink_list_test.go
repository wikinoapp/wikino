package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetBacklinkListUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetBacklinkListUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
	)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gbl-owner@example.com").
		WithAtname("gblowner").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gbl-space").
		WithName("GBL Space").
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

	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(2).
		WithTitle("Linked Page").
		Build()
	privateLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(3).
		WithTitle("Private Linked Page").
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Base Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID, privateLinkedPageID}).
		Build()

	publicLinkerID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(4).
		WithTitle("Public Linker").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(5).
		WithTitle("Private Linker").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(6).
		WithTitle("Trashed Linker").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		WithTrashed().
		Build()

	t.Run("正常系: メンバーは非公開トピックのバックリンクも見えるがゴミ箱のページは見えない", func(t *testing.T) {
		userID := ownerID
		output, err := uc.Execute(context.Background(), GetBacklinkListInput{
			SpaceIdentifier:  "gbl-space",
			PageNumber:       1,
			LinkedPageNumber: 2,
			UserID:           &userID,
			CurrentPage:      1,
			Limit:            15,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.LinkedPage == nil || output.LinkedPage.ID != linkedPageID {
			t.Errorf("LinkedPage = %v, want the linked page", output.LinkedPage)
		}
		if len(output.Backlinks) != 2 {
			t.Fatalf("len(Backlinks) = %d, want 2", len(output.Backlinks))
		}
		if output.TotalCount != 2 {
			t.Errorf("TotalCount = %d, want 2", output.TotalCount)
		}
	})

	t.Run("正常系: ゲストは公開トピックのバックリンクだけを見る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetBacklinkListInput{
			SpaceIdentifier:  "gbl-space",
			PageNumber:       1,
			LinkedPageNumber: 2,
			CurrentPage:      1,
			Limit:            15,
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

	t.Run("異常系: ゲストは非公開トピックのリンク先ページのバックリンクを取得できない", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), GetBacklinkListInput{
			SpaceIdentifier:  "gbl-space",
			PageNumber:       1,
			LinkedPageNumber: 3,
			CurrentPage:      1,
			Limit:            15,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: 存在しないリンク先ページは取得できない", func(t *testing.T) {
		userID := ownerID
		_, err := uc.Execute(context.Background(), GetBacklinkListInput{
			SpaceIdentifier:  "gbl-space",
			PageNumber:       1,
			LinkedPageNumber: 999,
			UserID:           &userID,
			CurrentPage:      1,
			Limit:            15,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})
}

func TestGetBacklinkListUsecase_Execute_AuthorizationBoundaries(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetBacklinkListUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
	)

	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gbl-auth-non-member@example.com").
		WithAtname("gblauthnonmember").
		Build()
	restrictedMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gbl-auth-restricted@example.com").
		WithAtname("gblauthrestricted").
		Build()
	trashMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gbl-auth-trash@example.com").
		WithAtname("gblauthtrash").
		Build()
	fullMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gbl-auth-full@example.com").
		WithAtname("gblauthfull").
		Build()
	topicReaderID := testutil.NewUserBuilder(t, tx).
		WithEmail("gbl-auth-topic-reader@example.com").
		WithAtname("gblauthtopicreader").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gbl-auth-space").
		WithName("GBL Authorization Space").
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

	publicLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(2).
		WithTitle("Public Linked Page").
		Build()
	privateLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(3).
		WithTitle("Private Linked Page").
		Build()
	privateLinkedPageBID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicBID).
		WithNumber(4).
		WithTitle("Private B Linked Page").
		Build()
	publicBasePageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Public Base Page").
		WithLinkedPageIDs([]model.PageID{publicLinkedPageID, privateLinkedPageID, privateLinkedPageBID}).
		Build()
	privateBasePageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(10).
		WithTitle("Private Base Page").
		WithLinkedPageIDs([]model.PageID{publicLinkedPageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(11).
		WithTitle("Trashed Base Page").
		WithLinkedPageIDs([]model.PageID{publicLinkedPageID}).
		WithTrashed().
		Build()
	trashedLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(12).
		WithTitle("Trashed Linked Page").
		WithTrashed().
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(13).
		WithTitle("Trashed Target Base Page").
		WithLinkedPageIDs([]model.PageID{trashedLinkedPageID}).
		Build()
	publicLinkerID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(14).
		WithTitle("Public Linker").
		WithLinkedPageIDs([]model.PageID{publicLinkedPageID}).
		Build()
	privateLinkerID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(15).
		WithTitle("Private Linker").
		WithLinkedPageIDs([]model.PageID{publicLinkedPageID}).
		Build()
	privateLinkerBID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicBID).
		WithNumber(16).
		WithTitle("Private B Linker").
		WithLinkedPageIDs([]model.PageID{publicLinkedPageID}).
		Build()

	tests := []struct {
		name               string
		pageNumber         int32
		linkedPageNumber   int32
		userID             *model.UserID
		wantNotFound       bool
		wantBacklinkIDs    []model.PageID
		wantSpaceMemberNil bool
	}{
		{
			name:               "正常系: ゲストは公開トピックのバックリンクだけを見る",
			pageNumber:         1,
			linkedPageNumber:   2,
			wantBacklinkIDs:    []model.PageID{publicLinkerID},
			wantSpaceMemberNil: true,
		},
		{
			name:               "正常系: ログイン済み非メンバーはゲストと同じバックリンクだけを見る",
			pageNumber:         1,
			linkedPageNumber:   2,
			userID:             &nonMemberID,
			wantBacklinkIDs:    []model.PageID{publicLinkerID},
			wantSpaceMemberNil: true,
		},
		{
			name:             "正常系: topic:readを持たないメンバーには非公開トピックのバックリンクが見えない",
			pageNumber:       1,
			linkedPageNumber: 2,
			userID:           &restrictedMemberID,
			wantBacklinkIDs:  []model.PageID{publicLinkerID},
		},
		{
			name:             "正常系: トピック単位のtopic:readを持つメンバーは参加中の非公開トピックだけを見る",
			pageNumber:       1,
			linkedPageNumber: 2,
			userID:           &topicReaderID,
			wantBacklinkIDs:  []model.PageID{privateBasePageID, publicLinkerID, privateLinkerID},
		},
		{
			name:             "正常系: 全トピックを開けるメンバーは非公開トピックのバックリンクも見える",
			pageNumber:       1,
			linkedPageNumber: 2,
			userID:           &fullMemberID,
			wantBacklinkIDs:  []model.PageID{privateBasePageID, publicLinkerID, privateLinkerID, privateLinkerBID},
		},
		{
			name:             "異常系: ログイン済み非メンバーは非公開の元ページを取得できない",
			pageNumber:       10,
			linkedPageNumber: 2,
			userID:           &nonMemberID,
			wantNotFound:     true,
		},
		{
			name:             "異常系: ログイン済み非メンバーは非公開のリンク先ページを取得できない",
			pageNumber:       1,
			linkedPageNumber: 3,
			userID:           &nonMemberID,
			wantNotFound:     true,
		},
		{
			name:             "異常系: topic:readを持たないメンバーは非公開の元ページを取得できない",
			pageNumber:       10,
			linkedPageNumber: 2,
			userID:           &restrictedMemberID,
			wantNotFound:     true,
		},
		{
			name:             "異常系: topic:readを持たないメンバーは非公開のリンク先ページを取得できない",
			pageNumber:       1,
			linkedPageNumber: 3,
			userID:           &restrictedMemberID,
			wantNotFound:     true,
		},
		{
			name:             "正常系: トピック単位のtopic:readを持つメンバーは参加中の非公開リンク先を取得できる",
			pageNumber:       1,
			linkedPageNumber: 3,
			userID:           &topicReaderID,
		},
		{
			name:             "異常系: トピック単位のtopic:readを持つメンバーは未参加の非公開リンク先を取得できない",
			pageNumber:       1,
			linkedPageNumber: 4,
			userID:           &topicReaderID,
			wantNotFound:     true,
		},
		{
			name:             "異常系: ゲストはゴミ箱の元ページを取得できない",
			pageNumber:       11,
			linkedPageNumber: 2,
			wantNotFound:     true,
		},
		{
			name:             "異常系: page:trashを持たないメンバーはゴミ箱の元ページを取得できない",
			pageNumber:       11,
			linkedPageNumber: 2,
			userID:           &restrictedMemberID,
			wantNotFound:     true,
		},
		{
			name:             "正常系: page:trashを持つメンバーはゴミ箱の元ページを取得できる",
			pageNumber:       11,
			linkedPageNumber: 2,
			userID:           &trashMemberID,
			wantBacklinkIDs:  []model.PageID{publicBasePageID, publicLinkerID},
		},
		{
			name:             "異常系: ゲストはゴミ箱のリンク先ページを取得できない",
			pageNumber:       13,
			linkedPageNumber: 12,
			wantNotFound:     true,
		},
		{
			name:             "異常系: page:trashを持たないメンバーはゴミ箱のリンク先ページを取得できない",
			pageNumber:       13,
			linkedPageNumber: 12,
			userID:           &restrictedMemberID,
			wantNotFound:     true,
		},
		{
			name:             "正常系: page:trashを持つメンバーはゴミ箱のリンク先ページを取得できる",
			pageNumber:       13,
			linkedPageNumber: 12,
			userID:           &trashMemberID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(context.Background(), GetBacklinkListInput{
				SpaceIdentifier:  "gbl-auth-space",
				PageNumber:       tt.pageNumber,
				LinkedPageNumber: tt.linkedPageNumber,
				UserID:           tt.userID,
				CurrentPage:      1,
				Limit:            15,
			})
			if tt.wantNotFound {
				assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
				return
			}
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if output.LinkedPage == nil || output.LinkedPage.Number != model.PageNumber(tt.linkedPageNumber) {
				t.Errorf("LinkedPage = %v, want page number %d", output.LinkedPage, tt.linkedPageNumber)
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
