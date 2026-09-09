package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetPageShowUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetPageShowUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
	)

	// Space owner (holds the space:admin scope by default, so it can edit pages).
	//
	// [Ja] スペースオーナー (デフォルトで space:admin スコープを持つため、ページを編集できる)。
	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gps-owner@example.com").
		WithAtname("gpsowner").
		Build()
	// Member holding page:trash but not page:write (verifies the trash alert path without edit rights).
	//
	// [Ja] page:write を持たず page:trash を持つメンバー (編集権限なしでゴミ箱表示経路を検証する)。
	trashMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gps-trash@example.com").
		WithAtname("gpstrash").
		Build()
	// Read-only member (verifies that page:read alone reveals neither a trashed page nor a page in
	// a private topic).
	//
	// [Ja] 読み取り専用メンバー (page:read だけではゴミ箱のページも非公開トピックのページも見えない
	// ことを検証する)。
	readerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gps-reader@example.com").
		WithAtname("gpsreader").
		Build()
	// Member whose private-topic and trash permissions come only from topic memberships.
	//
	// [Ja] 非公開トピックとゴミ箱の権限をトピックメンバーからだけ得るメンバー。
	topicScopedMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gps-topic-scoped@example.com").
		WithAtname("gpstopicscoped").
		Build()
	// Logged-in user who has not joined the space (verifies that a signed-in non-member is treated
	// like a guest: public pages are readable but not editable, and the trash stays hidden).
	//
	// [Ja] スペースに参加していないログイン済みユーザー (ログイン済み非メンバーがゲストと同じ扱いに
	// なること = 公開ページは読めるが編集できず、ゴミ箱は見えないことを検証する)。
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gps-nonmember@example.com").
		WithAtname("gpsnonmember").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gps-space").
		WithName("GPS Space").
		Build()
	ownerSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(trashMemberID).
		WithScopes([]model.Scope{model.ScopePageTrash}).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(readerID).
		WithScopes([]model.Scope{model.ScopePageRead}).
		Build()
	topicScopedSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(topicScopedMemberID).
		WithScopes([]model.Scope{}).
		Build()
	otherSpaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gps-other-space").
		WithName("GPS Other Space").
		Build()
	otherSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(otherSpaceID).
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
	discardedTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(3).
		WithName("Discarded").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		WithDiscarded().
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithSpaceMemberID(topicScopedSpaceMemberID).
		WithScopes([]model.Scope{model.ScopePageTrash}).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithSpaceMemberID(topicScopedSpaceMemberID).
		WithScopes([]model.Scope{model.ScopeTopicRead}).
		Build()

	attachmentID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(ownerSpaceMemberID).
		WithFilename("cover.png").
		Build()
	otherSpaceAttachmentID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(otherSpaceID).
		WithSpaceMemberID(otherSpaceMemberID).
		WithFilename("other-cover.png").
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Public Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(2).
		WithTitle("Private Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(3).
		WithTitle("Trashed Page").
		WithLinkedPageIDs([]model.PageID{}).
		WithTrashed().
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(discardedTopicID).
		WithNumber(4).
		WithTitle("Page In Discarded Topic").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(5).
		WithTitle("Page With Cover Image").
		WithLinkedPageIDs([]model.PageID{}).
		WithFeaturedImageAttachmentID(attachmentID).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(6).
		WithTitle("Unpublished Page").
		WithLinkedPageIDs([]model.PageID{}).
		WithUnpublished().
		Build()
	// A cover image pointing at another space's attachment is inconsistent data that normal
	// operation never creates. It is built on purpose so that the repository returns (nil, nil)
	// here, pinning both the nil fallback and the space_id scope of FindByIDAndSpace.
	//
	// [Ja] 別スペースの添付ファイルを指すアイキャッチ画像は、通常の運用では作られない不整合データ。
	// Repository が (nil, nil) を返す経路を通すために意図的に作り、nil フォールバックと
	// FindByIDAndSpace の space_id スコープの両方を固定する。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(7).
		WithTitle("Page With Cross-Space Cover Image").
		WithLinkedPageIDs([]model.PageID{}).
		WithFeaturedImageAttachmentID(otherSpaceAttachmentID).
		Build()

	// Page 10 links to a public-topic page and a private-topic page, and is linked to from one page
	// of each kind. It exercises the link list and the backlink list of one page against both
	// visibilities at once.
	//
	// [Ja] ページ 10 は公開トピックのページと非公開トピックのページの双方へリンクし、双方から
	// リンクされている。1 ページのリンク一覧とバックリンク一覧を、両方の公開設定に対して同時に
	// 検証できるようにするためである。
	publicLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(8).
		WithTitle("Public Linked Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	privateLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(9).
		WithTitle("Private Linked Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	linkSourcePageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(10).
		WithTitle("Link Source Page").
		WithLinkedPageIDs([]model.PageID{publicLinkedPageID, privateLinkedPageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(11).
		WithTitle("Public Backlink Source").
		WithLinkedPageIDs([]model.PageID{linkSourcePageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(12).
		WithTitle("Private Backlink Source").
		WithLinkedPageIDs([]model.PageID{linkSourcePageID}).
		Build()

	// Two pages linking to page 8 give its nested backlink list something to page through, which is
	// the listing the full-page fallback advances one card at a time.
	//
	// [Ja] ページ 8 へリンクする 2 ページを置き、そのネストしたバックリンク一覧をページ送りできるように
	// する。フルページフォールバックがカード単位で進めるのはこの一覧である。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(13).
		WithTitle("Nested Backlink Newer").
		WithModifiedAt(time.Now()).
		WithLinkedPageIDs([]model.PageID{publicLinkedPageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(14).
		WithTitle("Nested Backlink Older").
		WithModifiedAt(time.Now().Add(-time.Hour)).
		WithLinkedPageIDs([]model.PageID{publicLinkedPageID}).
		Build()

	// linkListInput builds the input of the page carrying both listings, with the limits the
	// handler passes.
	//
	// [Ja] linkListInput は 2 つの一覧を持つページの入力を、Handler が渡すのと同じ件数上限で組み立てる。
	linkListInput := func(userID *model.UserID) GetPageShowInput {
		return GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             10,
			UserID:                 userID,
			LinkLimit:              15,
			BacklinkLimit:          13,
			PageBacklinkLimit:      14,
		}
	}

	t.Run("正常系: ゲストのリンク一覧・バックリンク一覧は公開トピックのページだけになる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), linkListInput(nil))
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if got := pageNumbersOf(output.LinkedPages); len(got) != 1 || got[0] != 8 {
			t.Errorf("LinkedPages = %v, want [8]", got)
		}
		if output.LinkedTotalCount != 1 {
			t.Errorf("LinkedTotalCount = %d, want 1", output.LinkedTotalCount)
		}
		if got := pageNumbersOf(output.PageBacklinks); len(got) != 1 || got[0] != 11 {
			t.Errorf("PageBacklinks = %v, want [11]", got)
		}
		if output.PageBacklinkCount != 1 {
			t.Errorf("PageBacklinkCount = %d, want 1", output.PageBacklinkCount)
		}
		// The topics are resolved for the cards' labels, so the private topic must not leak here
		// either.
		//
		// [Ja] トピックはカードのラベル用に解決するため、ここでも非公開トピックが漏れてはならない。
		for _, topic := range output.LinkTopics {
			if topic.ID == privateTopicID {
				t.Error("LinkTopics should not contain the private topic for a guest")
			}
		}
	})

	t.Run("正常系: 非公開トピックを開けるメンバーの一覧には非公開トピックのページも並ぶ", func(t *testing.T) {
		userID := ownerID
		output, err := uc.Execute(context.Background(), linkListInput(&userID))
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if got := len(output.LinkedPages); got != 2 {
			t.Errorf("len(LinkedPages) = %d, want 2", got)
		}
		if got := len(output.PageBacklinks); got != 2 {
			t.Errorf("len(PageBacklinks) = %d, want 2", got)
		}
		// The linked pages carry their own backlinks, which the link list renders next to each card.
		//
		// [Ja] リンク先ページは自身のバックリンクを伴い、リンク一覧が各カードの隣に描画する。
		if output.BacklinksPerPage == nil {
			t.Error("BacklinksPerPage should not be nil when the page has links")
		}
	})

	// The full-page fallback advances the two independent top-level listings without htmx.
	//
	// [Ja] フルページフォールバックが htmx なしでも、独立した 2 つの最上位一覧を進めることを確認する。
	t.Run("正常系: フルページフォールバックで2ページ目を取得できる", func(t *testing.T) {
		userID := ownerID
		firstInput := linkListInput(&userID)
		firstInput.LinkLimit = 1
		firstInput.PageBacklinkLimit = 1

		firstOutput, err := uc.Execute(context.Background(), firstInput)
		if err != nil {
			t.Fatalf("first Execute() error = %v", err)
		}

		secondInput := firstInput
		secondInput.LinkPage = 2
		secondInput.PageBacklinkPage = 2
		secondOutput, err := uc.Execute(context.Background(), secondInput)
		if err != nil {
			t.Fatalf("second Execute() error = %v", err)
		}

		if len(firstOutput.LinkedPages) != 1 || len(secondOutput.LinkedPages) != 1 {
			t.Fatalf("linked page lengths = (%d, %d), want (1, 1)", len(firstOutput.LinkedPages), len(secondOutput.LinkedPages))
		}
		if firstOutput.LinkedPages[0].ID == secondOutput.LinkedPages[0].ID {
			t.Error("the second full-page link slice should differ from the first")
		}
		if firstOutput.LinkedTotalCount != 2 || secondOutput.LinkedTotalCount != 2 {
			t.Errorf("linked totals = (%d, %d), want (2, 2)", firstOutput.LinkedTotalCount, secondOutput.LinkedTotalCount)
		}

		if len(firstOutput.PageBacklinks) != 1 || len(secondOutput.PageBacklinks) != 1 {
			t.Fatalf("backlink page lengths = (%d, %d), want (1, 1)", len(firstOutput.PageBacklinks), len(secondOutput.PageBacklinks))
		}
		if firstOutput.PageBacklinks[0].ID == secondOutput.PageBacklinks[0].ID {
			t.Error("the second full-page backlink slice should differ from the first")
		}
		if firstOutput.PageBacklinkCount != 2 || secondOutput.PageBacklinkCount != 2 {
			t.Errorf("backlink totals = (%d, %d), want (2, 2)", firstOutput.PageBacklinkCount, secondOutput.PageBacklinkCount)
		}
	})

	// The nested backlink list of one listed card is the third listing the fallback can advance, and
	// the only one that has to pick out a single card without disturbing the others.
	//
	// [Ja] リンク先カードのネストしたバックリンク一覧は、フォールバックが進められる 3 つ目の一覧であり、
	// 他に影響を与えずに 1 枚のカードだけを選び出す必要がある唯一の一覧である。
	t.Run("正常系: フルページフォールバックでリンク先ページのバックリンク2ページ目を取得できる", func(t *testing.T) {
		userID := ownerID
		input := linkListInput(&userID)
		input.BacklinkLimit = 1
		input.LinkedPageNumber = 8
		input.LinkedPageBacklinkPage = 2

		output, err := uc.Execute(context.Background(), input)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		selected := output.BacklinksPerPage[publicLinkedPageID]
		if selected == nil {
			t.Fatal("BacklinksPerPage should contain the selected linked page")
		}
		if got := pageNumbersOf(selected.Pages); len(got) != 1 || got[0] != 14 {
			t.Errorf("selected card's backlinks = %v, want [14]", got)
		}
		if selected.TotalCount != 2 {
			t.Errorf("selected card's backlink total = %d, want 2", selected.TotalCount)
		}

		// Advancing one card must not move another card's listing off its first page.
		//
		// [Ja] 1 枚のカードを進めても、他のカードの一覧を 1 ページ目から動かしてはならない。
		if other := output.BacklinksPerPage[privateLinkedPageID]; other != nil && len(other.Pages) > 0 {
			if got := pageNumbersOf(other.Pages); got[0] == 14 {
				t.Error("the unselected card should keep its own first page")
			}
		}
	})

	t.Run("正常系: ゲストは公開トピックのページを閲覧できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             1,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.SpaceMember != nil {
			t.Error("SpaceMember should be nil for a guest")
		}
		if output.Page == nil || output.Page.Number != 1 {
			t.Errorf("Page.Number = %v, want 1", output.Page)
		}
		if output.Topic == nil || output.Topic.ID != publicTopicID {
			t.Errorf("Topic = %v, want the public topic", output.Topic)
		}
		if output.IsTrashed {
			t.Error("IsTrashed should be false for a page that is not in the trash")
		}
		if output.CanUpdatePage {
			t.Error("CanUpdatePage should be false for a guest")
		}
		if output.CanTrashPage {
			t.Error("CanTrashPage should be false for a guest")
		}
		if output.FeaturedImageAttachment != nil {
			t.Error("FeaturedImageAttachment should be nil for a page without a cover image")
		}
	})

	// This screen does not filter on published_at: an unpublished page is returned instead of 404.
	// Other page queries do filter on it, so pin the difference here.
	//
	// [Ja] 本画面は published_at でフィルタせず、未公開ページも 404 ではなく取得する。
	// 他のページ取得クエリは published_at を条件に持つため、その差分をここで固定する。
	t.Run("正常系: 未公開ページも 404 にならない", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             6,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Page == nil || output.Page.Number != 6 {
			t.Errorf("Page = %v, want the unpublished page", output.Page)
		}
		if output.Page.PublishedAt != nil {
			t.Error("PublishedAt should be nil for an unpublished page")
		}
	})

	t.Run("正常系: ページを編集できるメンバーは CanUpdatePage が true になる", func(t *testing.T) {
		userID := ownerID
		output, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             1,
			UserID:                 &userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.SpaceMember == nil {
			t.Fatal("SpaceMember should not be nil for a space member")
		}
		if !output.CanUpdatePage {
			t.Error("CanUpdatePage should be true for a member holding page:write")
		}
	})

	// The header's action dropdown offers editing and trashing on two different scopes, so the two
	// flags are pinned per scope. page:write must not open the trash item: an editor who may rewrite
	// a page is not thereby allowed to take it out of the space's visible content.
	//
	// [Ja] ヘッダーの操作ドロップダウンは編集とゴミ箱を別々のスコープで出し分けるため、2 つのフラグ
	// をスコープごとに固定する。page:write でゴミ箱項目が開いてはならない。ページを書き換えてよい
	// 編集者が、そのページをスペースの可視な内容から外してよいとは限らないためである。
	t.Run("正常系: CanTrashPage は page:write ではなく page:trash で決まる", func(t *testing.T) {
		tests := []struct {
			name              string
			userID            model.UserID
			wantCanUpdatePage bool
			wantCanTrashPage  bool
		}{
			{
				name:              "space:admin を持つオーナーは両方できる",
				userID:            ownerID,
				wantCanUpdatePage: true,
				wantCanTrashPage:  true,
			},
			{
				name:              "page:trash だけを持つメンバーはゴミ箱へ入れるだけできる",
				userID:            trashMemberID,
				wantCanUpdatePage: false,
				wantCanTrashPage:  true,
			},
			{
				name:              "page:read だけを持つメンバーはどちらもできない",
				userID:            readerID,
				wantCanUpdatePage: false,
				wantCanTrashPage:  false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				userID := tt.userID
				output, err := uc.Execute(context.Background(), GetPageShowInput{
					LinkPage:               1,
					LinkedPageBacklinkPage: 1,
					PageBacklinkPage:       1,
					SpaceIdentifier:        "gps-space",
					PageNumber:             1,
					UserID:                 &userID,
				})
				if err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				if output == nil {
					t.Fatal("output should not be nil")
				}
				if output.CanUpdatePage != tt.wantCanUpdatePage {
					t.Errorf("CanUpdatePage = %v, want %v", output.CanUpdatePage, tt.wantCanUpdatePage)
				}
				if output.CanTrashPage != tt.wantCanTrashPage {
					t.Errorf("CanTrashPage = %v, want %v", output.CanTrashPage, tt.wantCanTrashPage)
				}
			})
		}
	})

	t.Run("正常系: ログイン済み非メンバーは公開ページを閲覧できるが編集はできない", func(t *testing.T) {
		userID := nonMemberID
		output, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             1,
			UserID:                 &userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.SpaceMember != nil {
			t.Error("SpaceMember should be nil for a signed-in non-member")
		}
		if output.CanUpdatePage {
			t.Error("CanUpdatePage should be false for a signed-in non-member")
		}
	})

	t.Run("正常系: メンバーは非公開トピックのページを閲覧できる", func(t *testing.T) {
		userID := ownerID
		output, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             2,
			UserID:                 &userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Topic == nil || output.Topic.ID != privateTopicID {
			t.Errorf("Topic = %v, want the private topic", output.Topic)
		}
	})

	t.Run("正常系: トピックの topic:read を持つメンバーは非公開ページを閲覧できる", func(t *testing.T) {
		userID := topicScopedMemberID
		output, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             2,
			UserID:                 &userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Topic == nil || output.Topic.ID != privateTopicID {
			t.Errorf("Topic = %v, want the private topic", output.Topic)
		}
	})

	t.Run("正常系: アイキャッチ画像を持つページでは添付ファイルが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             5,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.FeaturedImageAttachment == nil {
			t.Fatal("FeaturedImageAttachment should not be nil for a page with a cover image")
		}
		if output.FeaturedImageAttachment.ID != attachmentID {
			t.Errorf("FeaturedImageAttachment.ID = %v, want %v", output.FeaturedImageAttachment.ID, attachmentID)
		}
		// The filename is what the og:image output needs to detect GIFs (see the Rails version).
		//
		// [Ja] ファイル名は og:image 出力側で GIF を判定するために必要になる (Rails 版と同じ判定)。
		if output.FeaturedImageAttachment.Filename != "cover.png" {
			t.Errorf("FeaturedImageAttachment.Filename = %q, want %q", output.FeaturedImageAttachment.Filename, "cover.png")
		}
	})

	t.Run("正常系: 別スペースのアイキャッチ画像は未解決として nil を返す", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             7,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.FeaturedImageAttachment != nil {
			t.Errorf("FeaturedImageAttachment = %v, want nil for an attachment in another space", output.FeaturedImageAttachment)
		}
	})

	t.Run("正常系: page:trash を持つメンバーはゴミ箱のページを閲覧できる", func(t *testing.T) {
		userID := trashMemberID
		output, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             3,
			UserID:                 &userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if !output.IsTrashed {
			t.Error("IsTrashed should be true for a page in the trash")
		}
		if output.CanUpdatePage {
			t.Error("CanUpdatePage should be false for a member without page:write")
		}
	})

	t.Run("正常系: トピックの page:trash を持つメンバーはゴミ箱のページを閲覧できる", func(t *testing.T) {
		userID := topicScopedMemberID
		output, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             3,
			UserID:                 &userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if !output.IsTrashed {
			t.Error("IsTrashed should be true for a page in the trash")
		}
		if output.CanUpdatePage {
			t.Error("CanUpdatePage should be false for a member without page:write")
		}
	})

	t.Run("異常系: ゲストはゴミ箱のページを閲覧できない", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             3,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: ログイン済み非メンバーはゴミ箱のページを閲覧できない", func(t *testing.T) {
		userID := nonMemberID
		_, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             3,
			UserID:                 &userID,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	// page:read alone must not reveal the trashed page: the check is on page:trash, and page:write
	// implies page:read, so pinning both keeps the requirement from silently breaking.
	//
	// [Ja] page:read だけではゴミ箱のページを見せない。判定軸は page:trash であり、page:write は
	// 含意で page:read を得るため、両方を固定して要件が静かに壊れないようにする。
	t.Run("異常系: page:read だけのメンバーはゴミ箱のページを閲覧できない", func(t *testing.T) {
		userID := readerID
		_, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             3,
			UserID:                 &userID,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: ゲストは非公開トピックのページを閲覧できない", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             2,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	// Being a space member is not enough for a private topic: the check is on topic:read, which
	// space:admin and topic-level grants both expand to. Pinning the false side keeps a member from
	// silently gaining access to every private topic.
	//
	// [Ja] 非公開トピックはスペースメンバーであるだけでは見せない。判定軸は topic:read であり、
	// space:admin とトピック単位の付与のどちらからも得られる。false 側を固定して、メンバーが
	// すべての非公開トピックに静かにアクセスできるようになる退行を防ぐ。
	t.Run("異常系: topic:read を持たないメンバーは非公開トピックのページを閲覧できない", func(t *testing.T) {
		userID := readerID
		_, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             2,
			UserID:                 &userID,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: 論理削除済みトピックのページは閲覧できない", func(t *testing.T) {
		userID := ownerID
		_, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             4,
			UserID:                 &userID,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: 存在しないスペースは AppErrCodeResourceNotFound を返す", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-nonexistent",
			PageNumber:             1,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: 存在しないページ番号は AppErrCodeResourceNotFound を返す", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             999,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})
}

// pageNumbersOf lists the numbers of the given pages, so that an assertion can name the expected
// pages by their number instead of by their generated ID.
//
// [Ja] pageNumbersOf は渡したページの番号を並べる。生成された ID ではなく番号で期待するページを
// 書けるようにするためである。
func pageNumbersOf(pages []*model.Page) []model.PageNumber {
	numbers := make([]model.PageNumber, 0, len(pages))
	for _, pg := range pages {
		numbers = append(numbers, pg.Number)
	}
	return numbers
}
