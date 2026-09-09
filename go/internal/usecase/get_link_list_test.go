package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestGetLinkListUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetLinkListUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewDraftPageRepository(q),
	)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gll-owner@example.com").
		WithAtname("gllowner").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gll-space").
		WithName("GLL Space").
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

	publicLinkedID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(2).
		WithTitle("Public Linked").
		Build()
	privateLinkedID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(3).
		WithTitle("Private Linked").
		Build()
	trashedLinkedID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(4).
		WithTitle("Trashed Linked").
		WithTrashed().
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Base Page").
		WithLinkedPageIDs([]model.PageID{publicLinkedID, privateLinkedID, trashedLinkedID}).
		Build()

	// Pages linking to publicLinkedID, used to check that the per-page backlinks are filtered the
	// same way as the link list itself.
	//
	// [Ja] publicLinkedID にリンクしているページ。リンク一覧に並ぶ各ページのバックリンクにも
	// 同じフィルタがかかることを検証するために作る。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(5).
		WithTitle("Public Backlinker").
		WithLinkedPageIDs([]model.PageID{publicLinkedID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(6).
		WithTitle("Private Backlinker").
		WithLinkedPageIDs([]model.PageID{publicLinkedID}).
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(10).
		WithTitle("Private Base Page").
		Build()

	t.Run("正常系: メンバーは非公開トピックのリンク先も見えるがゴミ箱のページは見えない", func(t *testing.T) {
		userID := ownerID
		output, err := uc.Execute(context.Background(), GetLinkListInput{
			SpaceIdentifier: "gll-space",
			PageNumber:      1,
			UserID:          &userID,
			CurrentPage:     1,
			LinkLimit:       15,
			BacklinkLimit:   5,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(output.LinkedPages) != 2 {
			t.Fatalf("len(LinkedPages) = %d, want 2", len(output.LinkedPages))
		}
		if output.LinkedTotalCount != 2 {
			t.Errorf("LinkedTotalCount = %d, want 2", output.LinkedTotalCount)
		}
		for _, pg := range output.LinkedPages {
			if pg.ID == trashedLinkedID {
				t.Error("LinkedPages should not contain the trashed page")
			}
		}
		if !output.CanUpdatePage {
			t.Error("CanUpdatePage should be true for a member holding page:write")
		}
		if backlinks := output.BacklinksPerPage[publicLinkedID]; backlinks == nil || backlinks.TotalCount != 2 {
			t.Errorf("backlinks of the public linked page = %v, want TotalCount 2", backlinks)
		}
	})

	t.Run("正常系: ゲストは公開トピックのリンク先とバックリンクだけを見る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetLinkListInput{
			SpaceIdentifier: "gll-space",
			PageNumber:      1,
			CurrentPage:     1,
			LinkLimit:       15,
			BacklinkLimit:   5,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.SpaceMember != nil {
			t.Error("SpaceMember should be nil for a guest")
		}
		if len(output.LinkedPages) != 1 {
			t.Fatalf("len(LinkedPages) = %d, want 1", len(output.LinkedPages))
		}
		if output.LinkedPages[0].ID != publicLinkedID {
			t.Errorf("LinkedPages[0].ID = %v, want the public linked page", output.LinkedPages[0].ID)
		}
		if output.LinkedTotalCount != 1 {
			t.Errorf("LinkedTotalCount = %d, want 1", output.LinkedTotalCount)
		}
		if output.CanUpdatePage {
			t.Error("CanUpdatePage should be false for a guest")
		}
		backlinks := output.BacklinksPerPage[publicLinkedID]
		if backlinks == nil {
			t.Fatal("backlinks of the public linked page should not be nil")
		}
		if len(backlinks.Pages) != 1 || backlinks.TotalCount != 1 {
			t.Errorf("backlinks = %d pages / TotalCount %d, want 1 / 1", len(backlinks.Pages), backlinks.TotalCount)
		}
	})

	t.Run("異常系: ゲストは非公開トピックのページのリンク一覧を取得できない", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), GetLinkListInput{
			SpaceIdentifier: "gll-space",
			PageNumber:      10,
			CurrentPage:     1,
			LinkLimit:       15,
			BacklinkLimit:   5,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: 存在しないページは取得できない", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), GetLinkListInput{
			SpaceIdentifier: "gll-space",
			PageNumber:      999,
			CurrentPage:     1,
			LinkLimit:       15,
			BacklinkLimit:   5,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})
}

func TestGetLinkListUsecase_Execute_AuthorizationBoundaries(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetLinkListUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewDraftPageRepository(q),
	)

	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gll-auth-non-member@example.com").
		WithAtname("gllauthnonmember").
		Build()
	restrictedMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gll-auth-restricted@example.com").
		WithAtname("gllauthrestricted").
		Build()
	trashMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gll-auth-trash@example.com").
		WithAtname("gllauthtrash").
		Build()
	fullMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gll-auth-full@example.com").
		WithAtname("gllauthfull").
		Build()
	topicReaderID := testutil.NewUserBuilder(t, tx).
		WithEmail("gll-auth-topic-reader@example.com").
		WithAtname("gllauthtopicreader").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gll-auth-space").
		WithName("GLL Authorization Space").
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
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Public Base Page").
		WithLinkedPageIDs([]model.PageID{publicLinkedPageID, privateLinkedPageID, privateLinkedPageBID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(10).
		WithTitle("Private Base Page").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(11).
		WithTitle("Trashed Base Page").
		WithLinkedPageIDs([]model.PageID{publicLinkedPageID, privateLinkedPageID, privateLinkedPageBID}).
		WithTrashed().
		Build()

	tests := []struct {
		name               string
		pageNumber         int32
		userID             *model.UserID
		wantNotFound       bool
		wantLinkedPageIDs  []model.PageID
		wantSpaceMemberNil bool
	}{
		{
			name:               "正常系: ゲストは公開トピックのリンク先だけを見る",
			pageNumber:         1,
			wantLinkedPageIDs:  []model.PageID{publicLinkedPageID},
			wantSpaceMemberNil: true,
		},
		{
			name:               "正常系: ログイン済み非メンバーはゲストと同じリンク先だけを見る",
			pageNumber:         1,
			userID:             &nonMemberID,
			wantLinkedPageIDs:  []model.PageID{publicLinkedPageID},
			wantSpaceMemberNil: true,
		},
		{
			name:              "正常系: topic:readを持たないメンバーには非公開トピックのリンク先が見えない",
			pageNumber:        1,
			userID:            &restrictedMemberID,
			wantLinkedPageIDs: []model.PageID{publicLinkedPageID},
		},
		{
			name:              "正常系: トピック単位のtopic:readを持つメンバーは参加中の非公開トピックだけを見る",
			pageNumber:        1,
			userID:            &topicReaderID,
			wantLinkedPageIDs: []model.PageID{publicLinkedPageID, privateLinkedPageID},
		},
		{
			name:              "正常系: 全トピックを開けるメンバーは非公開トピックのリンク先も見える",
			pageNumber:        1,
			userID:            &fullMemberID,
			wantLinkedPageIDs: []model.PageID{publicLinkedPageID, privateLinkedPageID, privateLinkedPageBID},
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
			name:              "正常系: page:trashを持つメンバーはゴミ箱のページを取得できる",
			pageNumber:        11,
			userID:            &trashMemberID,
			wantLinkedPageIDs: []model.PageID{publicLinkedPageID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(context.Background(), GetLinkListInput{
				SpaceIdentifier: "gll-auth-space",
				PageNumber:      tt.pageNumber,
				UserID:          tt.userID,
				CurrentPage:     1,
				LinkLimit:       15,
				BacklinkLimit:   5,
			})
			if tt.wantNotFound {
				assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
				return
			}
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(output.LinkedPages) != len(tt.wantLinkedPageIDs) {
				t.Fatalf("len(LinkedPages) = %d, want %d", len(output.LinkedPages), len(tt.wantLinkedPageIDs))
			}
			gotLinkedPageIDs := make(map[model.PageID]struct{}, len(output.LinkedPages))
			for _, page := range output.LinkedPages {
				gotLinkedPageIDs[page.ID] = struct{}{}
			}
			for _, wantPageID := range tt.wantLinkedPageIDs {
				if _, ok := gotLinkedPageIDs[wantPageID]; !ok {
					t.Errorf("LinkedPages does not contain page ID %v", wantPageID)
				}
			}
			if tt.wantSpaceMemberNil && output.SpaceMember != nil {
				t.Error("SpaceMember should be nil for a viewer who is not a space member")
			}
		})
	}
}

// TestGetLinkListUsecase_Execute_SourceKeepsPaginationDataset verifies that page 2 keeps using the
// same saved or draft link set selected by the caller.
//
// [Ja] TestGetLinkListUsecase_Execute_SourceKeepsPaginationDataset は、呼び出し元が選んだ保存済みまたは
// 下書きのリンク集合を 2 ページ目でも使い続けることを確認する。
func TestGetLinkListUsecase_Execute_SourceKeepsPaginationDataset(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetLinkListUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewDraftPageRepository(q),
	)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("gll-source@example.com").
		WithAtname("gllsource").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gll-source").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	baseTime := time.Now().Add(-24 * time.Hour)
	newLinkedPages := func(startNumber model.PageNumber, title string) []model.PageID {
		pageIDs := make([]model.PageID, 0, 20)
		for index := int32(0); index < 20; index++ {
			pageIDs = append(pageIDs, testutil.NewPageBuilder(t, tx).
				WithSpaceID(spaceID).
				WithTopicID(topicID).
				WithNumber(startNumber+model.PageNumber(index)).
				WithTitle(fmt.Sprintf("%s %d", title, index)).
				WithModifiedAt(baseTime.Add(time.Duration(20-index)*time.Minute)).
				Build())
		}
		return pageIDs
	}
	savedPageIDs := newLinkedPages(2, "Saved")
	draftPageIDs := newLinkedPages(22, "Draft")
	sourcePageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source").
		WithLinkedPageIDs(savedPageIDs).
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(sourcePageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithLinkedPageIDs(draftPageIDs).
		Build()

	memberID := userID
	baseInput := GetLinkListInput{
		SpaceIdentifier: "gll-source",
		PageNumber:      1,
		UserID:          &memberID,
		CurrentPage:     2,
		LinkLimit:       15,
		BacklinkLimit:   1,
	}

	savedInput := baseInput
	savedInput.Source = LinkListSourceSaved
	savedOutput, err := uc.Execute(context.Background(), savedInput)
	if err != nil {
		t.Fatalf("saved Execute() error = %v", err)
	}
	if len(savedOutput.LinkedPages) != 5 {
		t.Fatalf("len(saved page 2) = %d, want 5", len(savedOutput.LinkedPages))
	}
	for index, page := range savedOutput.LinkedPages {
		if page.ID != savedPageIDs[15+index] {
			t.Errorf("saved page 2 item %d = %v, want %v", index, page.ID, savedPageIDs[15+index])
		}
	}

	draftInput := baseInput
	draftInput.Source = LinkListSourceDraft
	draftOutput, err := uc.Execute(context.Background(), draftInput)
	if err != nil {
		t.Fatalf("draft Execute() error = %v", err)
	}
	if len(draftOutput.LinkedPages) != 5 {
		t.Fatalf("len(draft page 2) = %d, want 5", len(draftOutput.LinkedPages))
	}
	for index, page := range draftOutput.LinkedPages {
		if page.ID != draftPageIDs[15+index] {
			t.Errorf("draft page 2 item %d = %v, want %v", index, page.ID, draftPageIDs[15+index])
		}
	}
}

// TestListingWindow_KeepsCumulativeGridRemainder ties the window this package resolves to the card
// counts the view model declares. listingWindow derives a following page as one card more than the
// initial one; the constants say the same thing, and a change to either side without the other has
// to fail here rather than only shifting the rendered grid.
//
// [Ja] TestListingWindow_KeepsCumulativeGridRemainder は、本パッケージが解決する取得範囲を ViewModel
// が宣言するカード数に結び付ける。listingWindow は後続ページを初回より 1 件多いものとして求めており、
// 定数も同じことを言っている。片方だけを変えたときに、描画されるグリッドがずれるだけで済ませず、
// ここで落ちるようにする。
func TestListingWindow_KeepsCumulativeGridRemainder(t *testing.T) {
	t.Parallel()

	initialLimit := viewmodel.LinkLimit
	followingLimit := viewmodel.RelatedPageFollowingLimit

	tests := []struct {
		page               int32
		wantOffset         int32
		wantLimit          int32
		wantCumulativeSize int32
	}{
		{page: 1, wantOffset: 0, wantLimit: initialLimit, wantCumulativeSize: initialLimit},
		{page: 2, wantOffset: initialLimit, wantLimit: followingLimit, wantCumulativeSize: initialLimit + followingLimit},
		{page: 3, wantOffset: initialLimit + followingLimit, wantLimit: followingLimit, wantCumulativeSize: initialLimit + 2*followingLimit},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("page_%d", tt.page), func(t *testing.T) {
			t.Parallel()

			offset, limit := listingWindow(tt.page, initialLimit, false)
			if offset != tt.wantOffset || limit != tt.wantLimit {
				t.Errorf("listingWindow(%d, %d, false) = (%d, %d), want (%d, %d)", tt.page, initialLimit, offset, limit, tt.wantOffset, tt.wantLimit)
			}

			cumulativeOffset, cumulativeSize := listingWindow(tt.page, initialLimit, true)
			if cumulativeOffset != 0 || cumulativeSize != tt.wantCumulativeSize {
				t.Errorf("listingWindow(%d, %d, true) = (%d, %d), want (0, %d)", tt.page, initialLimit, cumulativeOffset, cumulativeSize, tt.wantCumulativeSize)
			}
			if offset+limit != cumulativeSize {
				t.Errorf("page %d leaves a gap or overlap: offset %d + limit %d != cumulative size %d", tt.page, offset, limit, cumulativeSize)
			}
			if cumulativeSize%3 != 2 {
				t.Errorf("page %d cumulative size = %d, want remainder 2 in a three-column grid", tt.page, cumulativeSize)
			}
		})
	}
}
