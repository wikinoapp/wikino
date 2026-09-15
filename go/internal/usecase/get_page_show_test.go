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

	// スペースオーナー (デフォルトでspace:adminスコープを持つため、ページを編集できる)。
	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gps-owner@example.com").
		WithAtname("gpsowner").
		Build()
	// page:writeを持たずpage:trashを持つメンバー (編集権限なしでゴミ箱表示経路を検証する)。
	trashMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gps-trash@example.com").
		WithAtname("gpstrash").
		Build()
	// 読み取り専用メンバー (page:readだけではゴミ箱のページも非公開トピックのページも見えない
	// ことを検証する)。
	readerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gps-reader@example.com").
		WithAtname("gpsreader").
		Build()
	// 非公開トピックとゴミ箱の権限をトピックメンバーからだけ得るメンバー。
	topicScopedMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gps-topic-scoped@example.com").
		WithAtname("gpstopicscoped").
		Build()
	// スペースに参加していないログイン済みユーザー (ログイン済み非メンバーがゲストと同じ扱いに
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
	// 別スペースの添付ファイルを指すアイキャッチ画像は、通常の運用では作られない不整合データ。
	// Repositoryが (nil, nil) を返す経路を通すために意図的に作り、nilフォールバックと
	// FindByIDAndSpaceのspace_idスコープの両方を固定する。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(7).
		WithTitle("Page With Cross-Space Cover Image").
		WithLinkedPageIDs([]model.PageID{}).
		WithFeaturedImageAttachmentID(otherSpaceAttachmentID).
		Build()

	// ページ10は公開トピックのページと非公開トピックのページの双方へリンクし、双方から
	// リンクされている。1ページのリンク一覧とバックリンク一覧を、両方の公開設定に対して同時に
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

	// ページ8へリンクする2ページを置き、そのネストしたバックリンク一覧をページ送りできるように
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

	// linkListInputは2つの一覧を持つページの入力を、Handlerが渡すのと同じ件数上限で組み立てる。
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if got := pageNumbersOf(output.LinkedPages); len(got) != 1 || got[0] != 8 {
			t.Errorf("LinkedPages = %v、期待値 = [8]", got)
		}
		if output.LinkedTotalCount != 1 {
			t.Errorf("LinkedTotalCount = %d、期待値 = 1", output.LinkedTotalCount)
		}
		if got := pageNumbersOf(output.PageBacklinks); len(got) != 1 || got[0] != 11 {
			t.Errorf("PageBacklinks = %v、期待値 = [11]", got)
		}
		if output.PageBacklinkCount != 1 {
			t.Errorf("PageBacklinkCount = %d、期待値 = 1", output.PageBacklinkCount)
		}
		// トピックはカードのラベル用に解決するため、ここでも非公開トピックが漏れてはならない。
		for _, topic := range output.LinkTopics {
			if topic.ID == privateTopicID {
				t.Error("ゲストなのにLinkTopicsに非公開トピックが含まれている")
			}
		}
	})

	t.Run("正常系: 非公開トピックを開けるメンバーの一覧には非公開トピックのページも並ぶ", func(t *testing.T) {
		userID := ownerID
		output, err := uc.Execute(context.Background(), linkListInput(&userID))
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if got := len(output.LinkedPages); got != 2 {
			t.Errorf("len(LinkedPages) = %d、期待値 = 2", got)
		}
		if got := len(output.PageBacklinks); got != 2 {
			t.Errorf("len(PageBacklinks) = %d、期待値 = 2", got)
		}
		// リンク先ページは自身のバックリンクを伴い、リンク一覧が各カードの隣に描画する。
		if output.BacklinksPerPage == nil {
			t.Error("ページにリンクがあるのにBacklinksPerPageがnil")
		}
	})

	// フルページフォールバックがhtmxなしでも、独立した2つの最上位一覧を進めることを確認する。
	t.Run("正常系: フルページフォールバックで2ページ目を取得できる", func(t *testing.T) {
		userID := ownerID
		firstInput := linkListInput(&userID)
		firstInput.LinkLimit = 1
		firstInput.PageBacklinkLimit = 1

		firstOutput, err := uc.Execute(context.Background(), firstInput)
		if err != nil {
			t.Fatalf("1回目のExecute()のエラー = %v", err)
		}

		secondInput := firstInput
		secondInput.LinkPage = 2
		secondInput.PageBacklinkPage = 2
		secondOutput, err := uc.Execute(context.Background(), secondInput)
		if err != nil {
			t.Fatalf("2回目のExecute()のエラー = %v", err)
		}

		if len(firstOutput.LinkedPages) != 1 || len(secondOutput.LinkedPages) != 1 {
			t.Fatalf("リンク先のページの長さ = (%d, %d)、期待値 = (1, 1)", len(firstOutput.LinkedPages), len(secondOutput.LinkedPages))
		}
		if firstOutput.LinkedPages[0].ID == secondOutput.LinkedPages[0].ID {
			t.Error("フルページフォールバックで取得したリンク一覧の2ページ目が1ページ目と同じ")
		}
		if firstOutput.LinkedTotalCount != 2 || secondOutput.LinkedTotalCount != 2 {
			t.Errorf("リンク先の総数 = (%d, %d)、期待値 = (2, 2)", firstOutput.LinkedTotalCount, secondOutput.LinkedTotalCount)
		}

		if len(firstOutput.PageBacklinks) != 1 || len(secondOutput.PageBacklinks) != 1 {
			t.Fatalf("バックリンクのページの長さ = (%d, %d)、期待値 = (1, 1)", len(firstOutput.PageBacklinks), len(secondOutput.PageBacklinks))
		}
		if firstOutput.PageBacklinks[0].ID == secondOutput.PageBacklinks[0].ID {
			t.Error("フルページフォールバックで取得したバックリンク一覧の2ページ目が1ページ目と同じ")
		}
		if firstOutput.PageBacklinkCount != 2 || secondOutput.PageBacklinkCount != 2 {
			t.Errorf("バックリンクの総数 = (%d, %d)、期待値 = (2, 2)", firstOutput.PageBacklinkCount, secondOutput.PageBacklinkCount)
		}
	})

	// リンク先カードのネストしたバックリンク一覧は、フォールバックが進められる3つ目の一覧であり、
	// 他に影響を与えずに1枚のカードだけを選び出す必要がある唯一の一覧である。
	t.Run("正常系: フルページフォールバックでリンク先ページのバックリンク2ページ目を取得できる", func(t *testing.T) {
		userID := ownerID
		input := linkListInput(&userID)
		input.BacklinkLimit = 1
		input.LinkedPageNumber = 8
		input.LinkedPageBacklinkPage = 2

		output, err := uc.Execute(context.Background(), input)
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		selected := output.BacklinksPerPage[publicLinkedPageID]
		if selected == nil {
			t.Fatal("BacklinksPerPageに選択中のリンク先ページが含まれていない")
		}
		if got := pageNumbersOf(selected.Pages); len(got) != 1 || got[0] != 14 {
			t.Errorf("選択中のカードのバックリンク = %v、期待値 = [14]", got)
		}
		if selected.TotalCount != 2 {
			t.Errorf("選択中のカードのバックリンクの総数 = %d、期待値 = 2", selected.TotalCount)
		}

		// 1枚のカードを進めても、他のカードの一覧を1ページ目から動かしてはならない。
		if other := output.BacklinksPerPage[privateLinkedPageID]; other != nil && len(other.Pages) > 0 {
			if got := pageNumbersOf(other.Pages); got[0] == 14 {
				t.Error("選択していないカードが自身の1ページ目を保っていない")
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.SpaceMember != nil {
			t.Error("ゲストなのにSpaceMemberがnilではない")
		}
		if output.Page == nil || output.Page.Number != 1 {
			t.Errorf("Page.Number = %v、期待値 = 1", output.Page)
		}
		if output.Topic == nil || output.Topic.ID != publicTopicID {
			t.Errorf("Topic = %v、期待値 = 公開トピック", output.Topic)
		}
		if output.IsTrashed {
			t.Error("ゴミ箱に無いページなのにIsTrashedがtrue")
		}
		if output.CanUpdatePage {
			t.Error("ゲストなのにCanUpdatePageがtrue")
		}
		if output.CanTrashPage {
			t.Error("ゲストなのにCanTrashPageがtrue")
		}
		if output.FeaturedImageAttachment != nil {
			t.Error("アイキャッチ画像の無いページなのにFeaturedImageAttachmentがnilではない")
		}
	})

	// 本画面はpublished_atでフィルタせず、未公開ページも404ではなく取得する。
	// 他のページ取得クエリはpublished_atを条件に持つため、その差分をここで固定する。
	t.Run("正常系: 未公開ページも404にならない", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             6,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Page == nil || output.Page.Number != 6 {
			t.Errorf("Page = %v、期待値 = 未公開のページ", output.Page)
		}
		if output.Page.PublishedAt != nil {
			t.Error("未公開のページなのにPublishedAtがnilではない")
		}
	})

	t.Run("正常系: ページを編集できるメンバーはCanUpdatePageがtrueになる", func(t *testing.T) {
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.SpaceMember == nil {
			t.Fatal("スペースメンバーなのにSpaceMemberがnil")
		}
		if !output.CanUpdatePage {
			t.Error("page:writeを持つメンバーなのにCanUpdatePageがfalse")
		}
	})

	// ヘッダーの操作ドロップダウンは編集とゴミ箱を別々のスコープで出し分けるため、2つのフラグ
	// をスコープごとに固定する。page:writeでゴミ箱項目が開いてはならない。ページを書き換えてよい
	// 編集者が、そのページをスペースの可視な内容から外してよいとは限らないためである。
	t.Run("正常系: CanTrashPageはpage:writeではなくpage:trashで決まる", func(t *testing.T) {
		tests := []struct {
			name              string
			userID            model.UserID
			wantCanUpdatePage bool
			wantCanTrashPage  bool
		}{
			{
				name:              "space:adminを持つオーナーは両方できる",
				userID:            ownerID,
				wantCanUpdatePage: true,
				wantCanTrashPage:  true,
			},
			{
				name:              "page:trashだけを持つメンバーはゴミ箱へ入れるだけできる",
				userID:            trashMemberID,
				wantCanUpdatePage: false,
				wantCanTrashPage:  true,
			},
			{
				name:              "page:readだけを持つメンバーはどちらもできない",
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
					t.Fatalf("Execute()のエラー = %v", err)
				}
				if output == nil {
					t.Fatal("出力がnil")
				}
				if output.CanUpdatePage != tt.wantCanUpdatePage {
					t.Errorf("CanUpdatePage = %v、期待値 = %v", output.CanUpdatePage, tt.wantCanUpdatePage)
				}
				if output.CanTrashPage != tt.wantCanTrashPage {
					t.Errorf("CanTrashPage = %v、期待値 = %v", output.CanTrashPage, tt.wantCanTrashPage)
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.SpaceMember != nil {
			t.Error("ログイン中の非メンバーなのにSpaceMemberがnilではない")
		}
		if output.CanUpdatePage {
			t.Error("ログイン中の非メンバーなのにCanUpdatePageがtrue")
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.Topic == nil || output.Topic.ID != privateTopicID {
			t.Errorf("Topic = %v、期待値 = 非公開トピック", output.Topic)
		}
	})

	t.Run("正常系: トピックのtopic:readを持つメンバーは非公開ページを閲覧できる", func(t *testing.T) {
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.Topic == nil || output.Topic.ID != privateTopicID {
			t.Errorf("Topic = %v、期待値 = 非公開トピック", output.Topic)
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.FeaturedImageAttachment == nil {
			t.Fatal("アイキャッチ画像のあるページなのにFeaturedImageAttachmentがnil")
		}
		if output.FeaturedImageAttachment.ID != attachmentID {
			t.Errorf("FeaturedImageAttachment.ID = %v、期待値 = %v", output.FeaturedImageAttachment.ID, attachmentID)
		}
		// ファイル名はog:image出力側でGIFを判定するために必要になる (Rails版と同じ判定)。
		if output.FeaturedImageAttachment.Filename != "cover.png" {
			t.Errorf("FeaturedImageAttachment.Filename = %q、期待値 = %q", output.FeaturedImageAttachment.Filename, "cover.png")
		}
	})

	t.Run("正常系: 別スペースのアイキャッチ画像は未解決としてnilを返す", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-space",
			PageNumber:             7,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.FeaturedImageAttachment != nil {
			t.Errorf("別スペースの添付ファイルでのFeaturedImageAttachment = %v、期待値 = nil", output.FeaturedImageAttachment)
		}
	})

	t.Run("正常系: page:trashを持つメンバーはゴミ箱のページを閲覧できる", func(t *testing.T) {
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if !output.IsTrashed {
			t.Error("ゴミ箱にあるページなのにIsTrashedがfalse")
		}
		if output.CanUpdatePage {
			t.Error("page:writeを持たないメンバーなのにCanUpdatePageがtrue")
		}
	})

	t.Run("正常系: トピックのpage:trashを持つメンバーはゴミ箱のページを閲覧できる", func(t *testing.T) {
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if !output.IsTrashed {
			t.Error("ゴミ箱にあるページなのにIsTrashedがfalse")
		}
		if output.CanUpdatePage {
			t.Error("page:writeを持たないメンバーなのにCanUpdatePageがtrue")
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

	// page:readだけではゴミ箱のページを見せない。判定軸はpage:trashであり、page:writeは
	// 含意でpage:readを得るため、両方を固定して要件が静かに壊れないようにする。
	t.Run("異常系: page:readだけのメンバーはゴミ箱のページを閲覧できない", func(t *testing.T) {
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

	// 非公開トピックはスペースメンバーであるだけでは見せない。判定軸はtopic:readであり、
	// space:adminとトピック単位の付与のどちらからも得られる。false側を固定して、メンバーが
	// すべての非公開トピックに静かにアクセスできるようになる退行を防ぐ。
	t.Run("異常系: topic:readを持たないメンバーは非公開トピックのページを閲覧できない", func(t *testing.T) {
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

	t.Run("異常系: 存在しないスペースはAppErrCodeResourceNotFoundを返す", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), GetPageShowInput{
			LinkPage:               1,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
			SpaceIdentifier:        "gps-nonexistent",
			PageNumber:             1,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: 存在しないページ番号はAppErrCodeResourceNotFoundを返す", func(t *testing.T) {
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

// pageNumbersOfは渡したページの番号を並べる。生成されたIDではなく番号で期待するページを
// 書けるようにするためである。
func pageNumbersOf(pages []*model.Page) []model.PageNumber {
	numbers := make([]model.PageNumber, 0, len(pages))
	for _, pg := range pages {
		numbers = append(numbers, pg.Number)
	}
	return numbers
}
