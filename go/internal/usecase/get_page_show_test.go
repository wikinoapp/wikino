package usecase

import (
	"context"
	"testing"

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

	t.Run("正常系: ゲストは公開トピックのページを閲覧できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageShowInput{
			SpaceIdentifier: "gps-space",
			PageNumber:      1,
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
			SpaceIdentifier: "gps-space",
			PageNumber:      6,
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
			SpaceIdentifier: "gps-space",
			PageNumber:      1,
			UserID:          &userID,
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

	t.Run("正常系: ログイン済み非メンバーは公開ページを閲覧できるが編集はできない", func(t *testing.T) {
		userID := nonMemberID
		output, err := uc.Execute(context.Background(), GetPageShowInput{
			SpaceIdentifier: "gps-space",
			PageNumber:      1,
			UserID:          &userID,
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
			SpaceIdentifier: "gps-space",
			PageNumber:      2,
			UserID:          &userID,
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
			SpaceIdentifier: "gps-space",
			PageNumber:      2,
			UserID:          &userID,
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
			SpaceIdentifier: "gps-space",
			PageNumber:      5,
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
			SpaceIdentifier: "gps-space",
			PageNumber:      7,
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
			SpaceIdentifier: "gps-space",
			PageNumber:      3,
			UserID:          &userID,
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
			SpaceIdentifier: "gps-space",
			PageNumber:      3,
			UserID:          &userID,
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
			SpaceIdentifier: "gps-space",
			PageNumber:      3,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: ログイン済み非メンバーはゴミ箱のページを閲覧できない", func(t *testing.T) {
		userID := nonMemberID
		_, err := uc.Execute(context.Background(), GetPageShowInput{
			SpaceIdentifier: "gps-space",
			PageNumber:      3,
			UserID:          &userID,
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
			SpaceIdentifier: "gps-space",
			PageNumber:      3,
			UserID:          &userID,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: ゲストは非公開トピックのページを閲覧できない", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), GetPageShowInput{
			SpaceIdentifier: "gps-space",
			PageNumber:      2,
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
			SpaceIdentifier: "gps-space",
			PageNumber:      2,
			UserID:          &userID,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: 論理削除済みトピックのページは閲覧できない", func(t *testing.T) {
		userID := ownerID
		_, err := uc.Execute(context.Background(), GetPageShowInput{
			SpaceIdentifier: "gps-space",
			PageNumber:      4,
			UserID:          &userID,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: 存在しないスペースは AppErrCodeResourceNotFound を返す", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), GetPageShowInput{
			SpaceIdentifier: "gps-nonexistent",
			PageNumber:      1,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("異常系: 存在しないページ番号は AppErrCodeResourceNotFound を返す", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), GetPageShowInput{
			SpaceIdentifier: "gps-space",
			PageNumber:      999,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})
}
