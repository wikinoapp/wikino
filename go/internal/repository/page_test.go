package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestPageRepository_FindBySpaceAndNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-find-space").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		WithBody("Hello").
		WithBodyHTML("<p>Hello</p>").
		Build()

	t.Run("存在するページをスペースIDとページ番号で取得できる", func(t *testing.T) {
		page, err := repo.FindBySpaceAndNumber(context.Background(), spaceID, 1)
		if err != nil {
			t.Fatalf("FindBySpaceAndNumber() error = %v", err)
		}
		if page == nil {
			t.Fatal("FindBySpaceAndNumber() returned nil, want page")
		}
		if page.ID != pageID {
			t.Errorf("page.ID = %v, want %v", page.ID, pageID)
		}
		if page.SpaceID != spaceID {
			t.Errorf("page.SpaceID = %v, want %v", page.SpaceID, spaceID)
		}
		if page.TopicID != topicID {
			t.Errorf("page.TopicID = %v, want %v", page.TopicID, topicID)
		}
		if page.Number != 1 {
			t.Errorf("page.Number = %v, want 1", page.Number)
		}
		if page.Title == nil || *page.Title != "Test Page" {
			t.Errorf("page.Title = %v, want 'Test Page'", page.Title)
		}
		if page.Body != "Hello" {
			t.Errorf("page.Body = %v, want 'Hello'", page.Body)
		}
		if page.BodyHTML != "<p>Hello</p>" {
			t.Errorf("page.BodyHTML = %v, want '<p>Hello</p>'", page.BodyHTML)
		}
		if page.PublishedAt == nil {
			t.Error("page.PublishedAt should not be nil")
		}
		if page.DiscardedAt != nil {
			t.Errorf("page.DiscardedAt = %v, want nil", page.DiscardedAt)
		}
	})

	t.Run("存在しないページ番号はnilを返す", func(t *testing.T) {
		page, err := repo.FindBySpaceAndNumber(context.Background(), spaceID, 999)
		if err != nil {
			t.Fatalf("FindBySpaceAndNumber() error = %v", err)
		}
		if page != nil {
			t.Errorf("FindBySpaceAndNumber() = %v, want nil", page)
		}
	})

	t.Run("廃棄されたページは取得できない", func(t *testing.T) {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(99).
			WithTitle("Discarded Page").
			WithDiscarded().
			Build()

		page, err := repo.FindBySpaceAndNumber(context.Background(), spaceID, 99)
		if err != nil {
			t.Fatalf("FindBySpaceAndNumber() error = %v", err)
		}
		if page != nil {
			t.Errorf("FindBySpaceAndNumber() = %v, want nil (discarded page should not be returned)", page)
		}
	})
}

func TestPageRepository_FindPinnedByTopic(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-pinned-topic").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// ピン留めページ2件（pinned_at DESCでソートされることを検証）
	pinnedID1 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Pinned Old").
		WithPinnedAt(baseTime).
		Build()

	pinnedID2 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Pinned New").
		WithPinnedAt(baseTime.Add(1 * time.Hour)).
		Build()

	// 通常ページ（ピン留めなし）
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Regular Page").
		Build()

	// 非公開ページ（ピン留めあり）
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(4).
		WithTitle("Unpublished Pinned").
		WithPinnedAt(baseTime.Add(2 * time.Hour)).
		WithUnpublished().
		Build()

	// 廃棄済みページ（ピン留めあり）
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(5).
		WithTitle("Discarded Pinned").
		WithPinnedAt(baseTime.Add(3 * time.Hour)).
		WithDiscarded().
		Build()

	// ゴミ箱ページ（ピン留めあり）
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(6).
		WithTitle("Trashed Pinned").
		WithPinnedAt(baseTime.Add(4 * time.Hour)).
		WithTrashed().
		Build()

	t.Run("ピン留めページをpinned_at DESCで取得できる", func(t *testing.T) {
		pages, err := repo.FindPinnedByTopic(context.Background(), topicID, spaceID)
		if err != nil {
			t.Fatalf("FindPinnedByTopic() error = %v", err)
		}
		if len(pages) != 2 {
			t.Fatalf("len(pages) = %d, want 2", len(pages))
		}
		// pinned_at DESC でソートされるため、新しい順
		if pages[0].ID != pinnedID2 {
			t.Errorf("pages[0].ID = %v, want %v", pages[0].ID, pinnedID2)
		}
		if pages[1].ID != pinnedID1 {
			t.Errorf("pages[1].ID = %v, want %v", pages[1].ID, pinnedID1)
		}
	})

	t.Run("別トピックのピン留めページは取得されない", func(t *testing.T) {
		otherTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(2).
			WithName("Other").
			Build()

		pages, err := repo.FindPinnedByTopic(context.Background(), otherTopicID, spaceID)
		if err != nil {
			t.Fatalf("FindPinnedByTopic() error = %v", err)
		}
		if len(pages) != 0 {
			t.Errorf("len(pages) = %d, want 0", len(pages))
		}
	})
}

func TestPageRepository_FindRegularByTopicPaginated(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-regular-topic").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// 通常ページ3件（modified_at DESCでソートされることを検証）
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Regular Old").
		WithModifiedAt(baseTime).
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Regular Middle").
		WithModifiedAt(baseTime.Add(1 * time.Hour)).
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Regular New").
		WithModifiedAt(baseTime.Add(2 * time.Hour)).
		Build()

	// ピン留めページ（通常ページには含まれない）
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(4).
		WithTitle("Pinned Page").
		WithPinnedAt(baseTime).
		Build()

	// 非公開ページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(5).
		WithTitle("Unpublished").
		WithUnpublished().
		Build()

	// 廃棄済みページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(6).
		WithTitle("Discarded").
		WithDiscarded().
		Build()

	// ゴミ箱ページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(7).
		WithTitle("Trashed").
		WithTrashed().
		Build()

	t.Run("1ページ目を取得できる（limit=2）", func(t *testing.T) {
		result, err := repo.FindRegularByTopicPaginated(context.Background(), topicID, spaceID, 1, 2)
		if err != nil {
			t.Fatalf("FindRegularByTopicPaginated() error = %v", err)
		}
		if len(result.Pages) != 2 {
			t.Fatalf("len(pages) = %d, want 2", len(result.Pages))
		}
		// modified_at DESC でソートされるため、新しい順
		if *result.Pages[0].Title != "Regular New" {
			t.Errorf("pages[0].Title = %v, want 'Regular New'", *result.Pages[0].Title)
		}
		if *result.Pages[1].Title != "Regular Middle" {
			t.Errorf("pages[1].Title = %v, want 'Regular Middle'", *result.Pages[1].Title)
		}
		if result.TotalCount != 3 {
			t.Errorf("TotalCount = %d, want 3", result.TotalCount)
		}
	})

	t.Run("2ページ目を取得できる（limit=2）", func(t *testing.T) {
		result, err := repo.FindRegularByTopicPaginated(context.Background(), topicID, spaceID, 2, 2)
		if err != nil {
			t.Fatalf("FindRegularByTopicPaginated() error = %v", err)
		}
		if len(result.Pages) != 1 {
			t.Fatalf("len(pages) = %d, want 1", len(result.Pages))
		}
		if *result.Pages[0].Title != "Regular Old" {
			t.Errorf("pages[0].Title = %v, want 'Regular Old'", *result.Pages[0].Title)
		}
		if result.TotalCount != 3 {
			t.Errorf("TotalCount = %d, want 3", result.TotalCount)
		}
	})

	t.Run("ピン留め・非公開・廃棄済み・ゴミ箱ページは含まれない", func(t *testing.T) {
		result, err := repo.FindRegularByTopicPaginated(context.Background(), topicID, spaceID, 1, 100)
		if err != nil {
			t.Fatalf("FindRegularByTopicPaginated() error = %v", err)
		}
		if len(result.Pages) != 3 {
			t.Errorf("len(pages) = %d, want 3", len(result.Pages))
		}
		if result.TotalCount != 3 {
			t.Errorf("TotalCount = %d, want 3", result.TotalCount)
		}
	})
}

func TestPageRepository_FindPinnedBySpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-pinned-space").
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
		WithDiscarded().
		Build()

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Two pinned pages in a public topic (verifies pinned_at DESC ordering).
	// [Ja] 公開トピックのピン留めページ 2 件 (pinned_at DESC でソートされることを検証)。
	pinnedPublicOldID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Pinned Public Old").
		WithPinnedAt(baseTime.Add(1 * time.Hour)).
		Build()

	pinnedPublicNewID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(2).
		WithTitle("Pinned Public New").
		WithPinnedAt(baseTime.Add(3 * time.Hour)).
		Build()

	// A pinned page in a private topic (visible to members, hidden from non-members).
	// [Ja] 非公開トピックのピン留めページ (メンバーには見えるが非メンバーには見えない)。
	pinnedPrivateID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(3).
		WithTitle("Pinned Private").
		WithPinnedAt(baseTime.Add(2 * time.Hour)).
		Build()

	// A pinned page in a discarded topic (latest pinned_at, but always excluded).
	// [Ja] 廃棄済みトピックのピン留めページ (pinned_at は最新だが常に除外される)。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(discardedTopicID).
		WithNumber(4).
		WithTitle("Pinned In Discarded Topic").
		WithPinnedAt(baseTime.Add(4 * time.Hour)).
		Build()

	// A regular (non-pinned) page.
	// [Ja] 通常ページ (ピン留めなし)。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(5).
		WithTitle("Regular Page").
		Build()

	// An unpublished page (pinned).
	// [Ja] 非公開ページ (ピン留めあり)。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(6).
		WithTitle("Unpublished Pinned").
		WithPinnedAt(baseTime.Add(5 * time.Hour)).
		WithUnpublished().
		Build()

	// A discarded page (pinned).
	// [Ja] 廃棄済みページ (ピン留めあり)。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(7).
		WithTitle("Discarded Pinned").
		WithPinnedAt(baseTime.Add(6 * time.Hour)).
		WithDiscarded().
		Build()

	// A trashed page (pinned).
	// [Ja] ゴミ箱ページ (ピン留めあり)。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(8).
		WithTitle("Trashed Pinned").
		WithPinnedAt(baseTime.Add(7 * time.Hour)).
		WithTrashed().
		Build()

	t.Run("メンバー (publicOnly=false) は全トピックのピン留めページをpinned_at DESCで取得できる", func(t *testing.T) {
		pages, err := repo.FindPinnedBySpace(context.Background(), spaceID, false)
		if err != nil {
			t.Fatalf("FindPinnedBySpace() error = %v", err)
		}
		if len(pages) != 3 {
			t.Fatalf("len(pages) = %d, want 3", len(pages))
		}
		// Expected pinned_at DESC order: Public New (3h) → Private (2h) → Public Old (1h).
		// [Ja] 期待される pinned_at DESC 順: Public New (3h) → Private (2h) → Public Old (1h)。
		if pages[0].ID != pinnedPublicNewID {
			t.Errorf("pages[0].ID = %v, want %v", pages[0].ID, pinnedPublicNewID)
		}
		if pages[1].ID != pinnedPrivateID {
			t.Errorf("pages[1].ID = %v, want %v", pages[1].ID, pinnedPrivateID)
		}
		if pages[2].ID != pinnedPublicOldID {
			t.Errorf("pages[2].ID = %v, want %v", pages[2].ID, pinnedPublicOldID)
		}
	})

	t.Run("非メンバー (publicOnly=true) は公開トピックのピン留めページのみ取得できる", func(t *testing.T) {
		pages, err := repo.FindPinnedBySpace(context.Background(), spaceID, true)
		if err != nil {
			t.Fatalf("FindPinnedBySpace() error = %v", err)
		}
		if len(pages) != 2 {
			t.Fatalf("len(pages) = %d, want 2", len(pages))
		}
		// Private-topic pages are excluded; only public-topic pages remain, in pinned_at DESC.
		// [Ja] 非公開トピックのページが除外され、公開トピックのみ pinned_at DESC。
		if pages[0].ID != pinnedPublicNewID {
			t.Errorf("pages[0].ID = %v, want %v", pages[0].ID, pinnedPublicNewID)
		}
		if pages[1].ID != pinnedPublicOldID {
			t.Errorf("pages[1].ID = %v, want %v", pages[1].ID, pinnedPublicOldID)
		}
	})
}

func TestPageRepository_FindRegularBySpacePaginated(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-regular-space").
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
		WithDiscarded().
		Build()

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Two regular pages in a public topic (verifies modified_at DESC ordering).
	// [Ja] 公開トピックの通常ページ 2 件 (modified_at DESC でソートされることを検証)。
	regularPublicOldID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Regular Public Old").
		WithModifiedAt(baseTime).
		Build()

	regularPublicNewID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(2).
		WithTitle("Regular Public New").
		WithModifiedAt(baseTime.Add(2 * time.Hour)).
		Build()

	// A regular page in a private topic (visible to members, hidden from non-members).
	// [Ja] 非公開トピックの通常ページ (メンバーには見えるが非メンバーには見えない)。
	regularPrivateID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(3).
		WithTitle("Regular Private").
		WithModifiedAt(baseTime.Add(1 * time.Hour)).
		Build()

	// A regular page in a discarded topic (always excluded).
	// [Ja] 廃棄済みトピックの通常ページ (常に除外される)。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(discardedTopicID).
		WithNumber(4).
		WithTitle("Regular In Discarded Topic").
		WithModifiedAt(baseTime.Add(3 * time.Hour)).
		Build()

	// A pinned page (excluded from the regular page list).
	// [Ja] ピン留めページ (通常ページには含まれない)。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(5).
		WithTitle("Pinned Page").
		WithPinnedAt(baseTime).
		Build()

	// An unpublished page.
	// [Ja] 非公開ページ。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(6).
		WithTitle("Unpublished").
		WithUnpublished().
		Build()

	// A discarded page.
	// [Ja] 廃棄済みページ。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(7).
		WithTitle("Discarded").
		WithDiscarded().
		Build()

	// A trashed page.
	// [Ja] ゴミ箱ページ。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(8).
		WithTitle("Trashed").
		WithTrashed().
		Build()

	t.Run("メンバー (publicOnly=false) は全トピックの通常ページをページネーションで取得できる", func(t *testing.T) {
		// Page 1 (limit=2): modified_at DESC → Public New (2h) → Private (1h).
		// [Ja] 1 ページ目 (limit=2): modified_at DESC で Public New (2h) → Private (1h)。
		result, err := repo.FindRegularBySpacePaginated(context.Background(), spaceID, false, 1, 2)
		if err != nil {
			t.Fatalf("FindRegularBySpacePaginated() error = %v", err)
		}
		if len(result.Pages) != 2 {
			t.Fatalf("len(pages) = %d, want 2", len(result.Pages))
		}
		if result.Pages[0].ID != regularPublicNewID {
			t.Errorf("pages[0].ID = %v, want %v", result.Pages[0].ID, regularPublicNewID)
		}
		if result.Pages[1].ID != regularPrivateID {
			t.Errorf("pages[1].ID = %v, want %v", result.Pages[1].ID, regularPrivateID)
		}
		if result.TotalCount != 3 {
			t.Errorf("TotalCount = %d, want 3", result.TotalCount)
		}

		// Page 2 (limit=2): Public Old (0h).
		// [Ja] 2 ページ目 (limit=2): Public Old (0h)。
		result, err = repo.FindRegularBySpacePaginated(context.Background(), spaceID, false, 2, 2)
		if err != nil {
			t.Fatalf("FindRegularBySpacePaginated() error = %v", err)
		}
		if len(result.Pages) != 1 {
			t.Fatalf("len(pages) = %d, want 1", len(result.Pages))
		}
		if result.Pages[0].ID != regularPublicOldID {
			t.Errorf("pages[0].ID = %v, want %v", result.Pages[0].ID, regularPublicOldID)
		}
		if result.TotalCount != 3 {
			t.Errorf("TotalCount = %d, want 3", result.TotalCount)
		}
	})

	t.Run("非メンバー (publicOnly=true) は公開トピックの通常ページのみ取得できる", func(t *testing.T) {
		result, err := repo.FindRegularBySpacePaginated(context.Background(), spaceID, true, 1, 100)
		if err != nil {
			t.Fatalf("FindRegularBySpacePaginated() error = %v", err)
		}
		if len(result.Pages) != 2 {
			t.Fatalf("len(pages) = %d, want 2", len(result.Pages))
		}
		// Private-topic pages are excluded; only public-topic pages remain, in modified_at DESC.
		// [Ja] 非公開トピックのページが除外され、公開トピックのみ modified_at DESC。
		if result.Pages[0].ID != regularPublicNewID {
			t.Errorf("pages[0].ID = %v, want %v", result.Pages[0].ID, regularPublicNewID)
		}
		if result.Pages[1].ID != regularPublicOldID {
			t.Errorf("pages[1].ID = %v, want %v", result.Pages[1].ID, regularPublicOldID)
		}
		if result.TotalCount != 2 {
			t.Errorf("TotalCount = %d, want 2", result.TotalCount)
		}
	})
}

func TestPageRepository_FindBySpace_IDDescTiebreak(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-space-id-tiebreak").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()

	sameTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Two pinned pages sharing the same pinned_at, to verify the id DESC tiebreaker.
	// [Ja] 同一 pinned_at のピン留めページ 2 件。id DESC のタイブレークを検証する。
	pinnedA := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Pinned A").
		WithPinnedAt(sameTime).
		Build()

	pinnedB := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Pinned B").
		WithPinnedAt(sameTime).
		Build()

	// Two regular pages sharing the same modified_at, to verify the id DESC tiebreaker.
	// [Ja] 同一 modified_at の通常ページ 2 件。id DESC のタイブレークを検証する。
	regularA := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Regular A").
		WithModifiedAt(sameTime).
		Build()

	regularB := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(4).
		WithTitle("Regular B").
		WithModifiedAt(sameTime).
		Build()

	t.Run("ピン留めページは同一 pinned_at のとき id DESC で並ぶ", func(t *testing.T) {
		pages, err := repo.FindPinnedBySpace(context.Background(), spaceID, false)
		if err != nil {
			t.Fatalf("FindPinnedBySpace() error = %v", err)
		}
		if len(pages) != 2 {
			t.Fatalf("len(pages) = %d, want 2", len(pages))
		}
		first, second := expectedIDDescOrder(pinnedA, pinnedB)
		if pages[0].ID != first {
			t.Errorf("pages[0].ID = %v, want %v", pages[0].ID, first)
		}
		if pages[1].ID != second {
			t.Errorf("pages[1].ID = %v, want %v", pages[1].ID, second)
		}
	})

	t.Run("通常ページは同一 modified_at のとき id DESC で並ぶ", func(t *testing.T) {
		result, err := repo.FindRegularBySpacePaginated(context.Background(), spaceID, false, 1, 100)
		if err != nil {
			t.Fatalf("FindRegularBySpacePaginated() error = %v", err)
		}
		if len(result.Pages) != 2 {
			t.Fatalf("len(pages) = %d, want 2", len(result.Pages))
		}
		first, second := expectedIDDescOrder(regularA, regularB)
		if result.Pages[0].ID != first {
			t.Errorf("pages[0].ID = %v, want %v", result.Pages[0].ID, first)
		}
		if result.Pages[1].ID != second {
			t.Errorf("pages[1].ID = %v, want %v", result.Pages[1].ID, second)
		}
	})
}

// expectedIDDescOrder returns the two page IDs in the order an "id DESC" sort would
// produce, i.e. the lexicographically larger id first. Page IDs are UUID (ULID) values
// whose canonical string form sorts identically to the underlying bytes, so a string
// comparison matches the database's ORDER BY id DESC.
//
// [Ja] expectedIDDescOrder は 2 つのページ ID を "id DESC" ソートが返す順 (辞書順で
// 大きい方を先頭) に並べて返す。ページ ID は UUID (ULID) 値で、その正準文字列表現は
// 元のバイト列と同じ順序でソートされるため、文字列比較は DB の ORDER BY id DESC と一致する。
func expectedIDDescOrder(a, b model.PageID) (first, second model.PageID) {
	if string(a) >= string(b) {
		return a, b
	}
	return b, a
}

func TestPageRepository_FindLinkedPagesPaginated(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-linked-paginated").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	// 3つの公開ページを作成（modified_atで降順にソートされることを検証するため異なる日時を設定）
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	pageID1 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Old Page").
		WithModifiedAt(baseTime).
		Build()

	pageID2 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Middle Page").
		WithModifiedAt(baseTime.Add(1 * time.Hour)).
		Build()

	pageID3 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("New Page").
		WithModifiedAt(baseTime.Add(2 * time.Hour)).
		Build()

	// 非公開ページ（Wikiリンクで自動作成されたページを想定）
	unpublishedID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(4).
		WithTitle("Unpublished").
		WithUnpublished().
		WithModifiedAt(baseTime.Add(-1 * time.Hour)).
		Build()

	allIDs := []model.PageID{pageID1, pageID2, pageID3, unpublishedID}

	t.Run("1ページ目を取得できる（limit=2）", func(t *testing.T) {
		result, err := repo.FindLinkedPagesPaginated(context.Background(), allIDs, spaceID, AllTopicsVisible(), 1, 2)
		if err != nil {
			t.Fatalf("FindLinkedPagesPaginated() error = %v", err)
		}
		if len(result.Pages) != 2 {
			t.Fatalf("len(pages) = %d, want 2", len(result.Pages))
		}
		// modified_at DESC でソートされるため、新しい順
		if *result.Pages[0].Title != "New Page" {
			t.Errorf("pages[0].Title = %v, want 'New Page'", *result.Pages[0].Title)
		}
		if *result.Pages[1].Title != "Middle Page" {
			t.Errorf("pages[1].Title = %v, want 'Middle Page'", *result.Pages[1].Title)
		}
		// 非公開ページも含めて4件
		if result.TotalCount != 4 {
			t.Errorf("TotalCount = %d, want 4", result.TotalCount)
		}
	})

	t.Run("2ページ目を取得できる（limit=2）", func(t *testing.T) {
		result, err := repo.FindLinkedPagesPaginated(context.Background(), allIDs, spaceID, AllTopicsVisible(), 2, 2)
		if err != nil {
			t.Fatalf("FindLinkedPagesPaginated() error = %v", err)
		}
		if len(result.Pages) != 2 {
			t.Fatalf("len(pages) = %d, want 2", len(result.Pages))
		}
		if result.TotalCount != 4 {
			t.Errorf("TotalCount = %d, want 4", result.TotalCount)
		}
	})

	t.Run("非公開ページも件数に含まれる", func(t *testing.T) {
		result, err := repo.FindLinkedPagesPaginated(context.Background(), allIDs, spaceID, AllTopicsVisible(), 1, 100)
		if err != nil {
			t.Fatalf("FindLinkedPagesPaginated() error = %v", err)
		}
		if len(result.Pages) != 4 {
			t.Errorf("len(pages) = %d, want 4", len(result.Pages))
		}
		if result.TotalCount != 4 {
			t.Errorf("TotalCount = %d, want 4", result.TotalCount)
		}
	})

	t.Run("空のIDリストは空の結果を返す", func(t *testing.T) {
		result, err := repo.FindLinkedPagesPaginated(context.Background(), []model.PageID{}, spaceID, AllTopicsVisible(), 1, 15)
		if err != nil {
			t.Fatalf("FindLinkedPagesPaginated() error = %v", err)
		}
		if len(result.Pages) != 0 {
			t.Errorf("len(pages) = %d, want 0", len(result.Pages))
		}
	})

	t.Run("ゴミ箱に入ったページは除外される", func(t *testing.T) {
		trashedID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(20).
			WithTitle("Trashed").
			WithTrashed().
			Build()

		result, err := repo.FindLinkedPagesPaginated(context.Background(), []model.PageID{pageID1, trashedID}, spaceID, AllTopicsVisible(), 1, 15)
		if err != nil {
			t.Fatalf("FindLinkedPagesPaginated() error = %v", err)
		}
		if len(result.Pages) != 1 {
			t.Fatalf("len(pages) = %d, want 1", len(result.Pages))
		}
		if *result.Pages[0].Title != "Old Page" {
			t.Errorf("pages[0].Title = %v, want 'Old Page'", *result.Pages[0].Title)
		}
		if result.TotalCount != 1 {
			t.Errorf("TotalCount = %d, want 1", result.TotalCount)
		}
	})

	t.Run("廃棄済みトピックのページは除外される", func(t *testing.T) {
		discardedTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(90).
			WithName("Discarded Topic").
			WithDiscarded().
			Build()

		pageInDiscardedTopicID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(discardedTopicID).
			WithNumber(21).
			WithTitle("In Discarded Topic").
			Build()

		result, err := repo.FindLinkedPagesPaginated(context.Background(), []model.PageID{pageID1, pageInDiscardedTopicID}, spaceID, AllTopicsVisible(), 1, 15)
		if err != nil {
			t.Fatalf("FindLinkedPagesPaginated() error = %v", err)
		}
		if len(result.Pages) != 1 {
			t.Fatalf("len(pages) = %d, want 1", len(result.Pages))
		}
		if result.TotalCount != 1 {
			t.Errorf("TotalCount = %d, want 1", result.TotalCount)
		}
	})

	t.Run("閲覧可能トピックに含まれないページは除外される", func(t *testing.T) {
		otherTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(91).
			WithName("Other Topic").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()

		pageInOtherTopicID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(otherTopicID).
			WithNumber(22).
			WithTitle("In Other Topic").
			Build()

		ids := []model.PageID{pageID1, pageInOtherTopicID}

		allResult, err := repo.FindLinkedPagesPaginated(context.Background(), ids, spaceID, AllTopicsVisible(), 1, 15)
		if err != nil {
			t.Fatalf("FindLinkedPagesPaginated() error = %v", err)
		}
		if len(allResult.Pages) != 2 {
			t.Errorf("AllTopicsVisible: len(pages) = %d, want 2", len(allResult.Pages))
		}
		if allResult.TotalCount != 2 {
			t.Errorf("AllTopicsVisible: TotalCount = %d, want 2", allResult.TotalCount)
		}

		limitedResult, err := repo.FindLinkedPagesPaginated(context.Background(), ids, spaceID, VisibleTopics([]model.TopicID{topicID}), 1, 15)
		if err != nil {
			t.Fatalf("FindLinkedPagesPaginated() error = %v", err)
		}
		if len(limitedResult.Pages) != 1 {
			t.Fatalf("VisibleTopics: len(pages) = %d, want 1", len(limitedResult.Pages))
		}
		if *limitedResult.Pages[0].Title != "Old Page" {
			t.Errorf("VisibleTopics: pages[0].Title = %v, want 'Old Page'", *limitedResult.Pages[0].Title)
		}
		if limitedResult.TotalCount != 1 {
			t.Errorf("VisibleTopics: TotalCount = %d, want 1", limitedResult.TotalCount)
		}
	})

	t.Run("閲覧可能トピックが空のときは何も返さない", func(t *testing.T) {
		result, err := repo.FindLinkedPagesPaginated(context.Background(), allIDs, spaceID, VisibleTopics(nil), 1, 15)
		if err != nil {
			t.Fatalf("FindLinkedPagesPaginated() error = %v", err)
		}
		if len(result.Pages) != 0 {
			t.Errorf("len(pages) = %d, want 0", len(result.Pages))
		}
		if result.TotalCount != 0 {
			t.Errorf("TotalCount = %d, want 0", result.TotalCount)
		}
	})
}

func TestPageRepository_FindBacklinkedPagesPaginated(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-backlink-paginated").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	// 被リンクページ
	targetPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Target Page").
		Build()

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// targetPageIDをリンクしているページ3件
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Linker Old").
		WithLinkedPageIDs([]model.PageID{targetPageID}).
		WithModifiedAt(baseTime).
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Linker Middle").
		WithLinkedPageIDs([]model.PageID{targetPageID}).
		WithModifiedAt(baseTime.Add(1 * time.Hour)).
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(4).
		WithTitle("Linker New").
		WithLinkedPageIDs([]model.PageID{targetPageID}).
		WithModifiedAt(baseTime.Add(2 * time.Hour)).
		Build()

	// targetPageIDをリンクしていないページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(5).
		WithTitle("No Link").
		Build()

	t.Run("1ページ目を取得できる（limit=2）", func(t *testing.T) {
		result, err := repo.FindBacklinkedPagesPaginated(context.Background(), targetPageID, spaceID, AllTopicsVisible(), 1, 2, nil)
		if err != nil {
			t.Fatalf("FindBacklinkedPagesPaginated() error = %v", err)
		}
		if len(result.Pages) != 2 {
			t.Fatalf("len(pages) = %d, want 2", len(result.Pages))
		}
		// modified_at DESC でソートされるため、新しい順
		if *result.Pages[0].Title != "Linker New" {
			t.Errorf("pages[0].Title = %v, want 'Linker New'", *result.Pages[0].Title)
		}
		if *result.Pages[1].Title != "Linker Middle" {
			t.Errorf("pages[1].Title = %v, want 'Linker Middle'", *result.Pages[1].Title)
		}
		if result.TotalCount != 3 {
			t.Errorf("TotalCount = %d, want 3", result.TotalCount)
		}
	})

	t.Run("2ページ目を取得できる（limit=2）", func(t *testing.T) {
		result, err := repo.FindBacklinkedPagesPaginated(context.Background(), targetPageID, spaceID, AllTopicsVisible(), 2, 2, nil)
		if err != nil {
			t.Fatalf("FindBacklinkedPagesPaginated() error = %v", err)
		}
		if len(result.Pages) != 1 {
			t.Fatalf("len(pages) = %d, want 1", len(result.Pages))
		}
		if *result.Pages[0].Title != "Linker Old" {
			t.Errorf("pages[0].Title = %v, want 'Linker Old'", *result.Pages[0].Title)
		}
		if result.TotalCount != 3 {
			t.Errorf("TotalCount = %d, want 3", result.TotalCount)
		}
	})

	t.Run("バックリンクがないページは空の結果を返す", func(t *testing.T) {
		isolatedPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(10).
			WithTitle("Isolated").
			Build()

		result, err := repo.FindBacklinkedPagesPaginated(context.Background(), isolatedPageID, spaceID, AllTopicsVisible(), 1, 14, nil)
		if err != nil {
			t.Fatalf("FindBacklinkedPagesPaginated() error = %v", err)
		}
		if len(result.Pages) != 0 {
			t.Errorf("len(pages) = %d, want 0", len(result.Pages))
		}
		if result.TotalCount != 0 {
			t.Errorf("TotalCount = %d, want 0", result.TotalCount)
		}
	})

	t.Run("ゴミ箱に入ったリンク元ページは除外される", func(t *testing.T) {
		trashedTargetID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(20).
			WithTitle("Trash Target").
			Build()

		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(21).
			WithTitle("Trashed Linker").
			WithLinkedPageIDs([]model.PageID{trashedTargetID}).
			WithTrashed().
			Build()

		result, err := repo.FindBacklinkedPagesPaginated(context.Background(), trashedTargetID, spaceID, AllTopicsVisible(), 1, 14, nil)
		if err != nil {
			t.Fatalf("FindBacklinkedPagesPaginated() error = %v", err)
		}
		if len(result.Pages) != 0 {
			t.Errorf("len(pages) = %d, want 0", len(result.Pages))
		}
		if result.TotalCount != 0 {
			t.Errorf("TotalCount = %d, want 0", result.TotalCount)
		}
	})

	t.Run("廃棄済みトピックのリンク元ページは除外される", func(t *testing.T) {
		discardedTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(91).
			WithName("Discarded Topic").
			WithDiscarded().
			Build()
		targetID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(30).
			WithTitle("Discarded Topic Target").
			Build()
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(discardedTopicID).
			WithNumber(31).
			WithTitle("Discarded Topic Linker").
			WithLinkedPageIDs([]model.PageID{targetID}).
			Build()

		result, err := repo.FindBacklinkedPagesPaginated(context.Background(), targetID, spaceID, AllTopicsVisible(), 1, 14, nil)
		if err != nil {
			t.Fatalf("FindBacklinkedPagesPaginated() error = %v", err)
		}
		if len(result.Pages) != 0 {
			t.Errorf("len(pages) = %d, want 0", len(result.Pages))
		}
		if result.TotalCount != 0 {
			t.Errorf("TotalCount = %d, want 0", result.TotalCount)
		}
	})

	t.Run("閲覧可能トピックに含まれないリンク元ページは除外される", func(t *testing.T) {
		otherTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(90).
			WithName("Other Topic").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()

		mixedTargetID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(22).
			WithTitle("Mixed Target").
			Build()

		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(23).
			WithTitle("Visible Linker").
			WithLinkedPageIDs([]model.PageID{mixedTargetID}).
			Build()

		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(otherTopicID).
			WithNumber(24).
			WithTitle("Hidden Linker").
			WithLinkedPageIDs([]model.PageID{mixedTargetID}).
			Build()

		allResult, err := repo.FindBacklinkedPagesPaginated(context.Background(), mixedTargetID, spaceID, AllTopicsVisible(), 1, 14, nil)
		if err != nil {
			t.Fatalf("FindBacklinkedPagesPaginated() error = %v", err)
		}
		if len(allResult.Pages) != 2 {
			t.Errorf("AllTopicsVisible: len(pages) = %d, want 2", len(allResult.Pages))
		}
		if allResult.TotalCount != 2 {
			t.Errorf("AllTopicsVisible: TotalCount = %d, want 2", allResult.TotalCount)
		}

		guestResult, err := repo.FindBacklinkedPagesPaginated(context.Background(), mixedTargetID, spaceID, VisibleTopics([]model.TopicID{topicID}), 1, 14, nil)
		if err != nil {
			t.Fatalf("FindBacklinkedPagesPaginated() error = %v", err)
		}
		if len(guestResult.Pages) != 1 {
			t.Fatalf("VisibleTopics: len(pages) = %d, want 1", len(guestResult.Pages))
		}
		if *guestResult.Pages[0].Title != "Visible Linker" {
			t.Errorf("VisibleTopics: pages[0].Title = %v, want 'Visible Linker'", *guestResult.Pages[0].Title)
		}
		if guestResult.TotalCount != 1 {
			t.Errorf("VisibleTopics: TotalCount = %d, want 1", guestResult.TotalCount)
		}
	})
}

func TestPageRepository_FindByIDs(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-find-ids-space").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	pageID1 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Page 1").
		Build()

	pageID2 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Page 2").
		Build()

	// 非公開ページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Unpublished").
		WithUnpublished().
		Build()

	t.Run("IDリストに含まれる公開済みページを取得できる", func(t *testing.T) {
		pages, err := repo.FindByIDs(context.Background(), []model.PageID{pageID1, pageID2}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDs() error = %v", err)
		}
		if len(pages) != 2 {
			t.Fatalf("len(pages) = %v, want 2", len(pages))
		}
		if pages[0].Number != 1 {
			t.Errorf("pages[0].Number = %v, want 1", pages[0].Number)
		}
		if pages[1].Number != 2 {
			t.Errorf("pages[1].Number = %v, want 2", pages[1].Number)
		}
	})

	t.Run("非公開ページも結果に含まれる", func(t *testing.T) {
		unpublishedID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(4).
			WithTitle("Also Unpublished").
			WithUnpublished().
			Build()

		pages, err := repo.FindByIDs(context.Background(), []model.PageID{unpublishedID}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDs() error = %v", err)
		}
		if len(pages) != 1 {
			t.Errorf("len(pages) = %v, want 1", len(pages))
		}
	})

	t.Run("空のIDリストは空のスライスを返す", func(t *testing.T) {
		pages, err := repo.FindByIDs(context.Background(), []model.PageID{}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDs() error = %v", err)
		}
		if len(pages) != 0 {
			t.Errorf("len(pages) = %v, want 0", len(pages))
		}
	})
}

func TestPageRepository_FindBacklinkedByPageID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-backlink-space").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	// 被リンクページ
	targetPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Target Page").
		Build()

	// targetPageIDをリンクしているページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Linking Page").
		WithLinkedPageIDs([]model.PageID{targetPageID}).
		Build()

	// targetPageIDをリンクしていないページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("No Link Page").
		Build()

	t.Run("バックリンクページを取得できる", func(t *testing.T) {
		pages, err := repo.FindBacklinkedByPageID(context.Background(), targetPageID, spaceID)
		if err != nil {
			t.Fatalf("FindBacklinkedByPageID() error = %v", err)
		}
		if len(pages) != 1 {
			t.Fatalf("len(pages) = %v, want 1", len(pages))
		}
		if *pages[0].Title != "Linking Page" {
			t.Errorf("pages[0].Title = %v, want 'Linking Page'", *pages[0].Title)
		}
	})

	t.Run("バックリンクがないページは空のスライスを返す", func(t *testing.T) {
		noLinkPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(4).
			WithTitle("Isolated Page").
			Build()

		pages, err := repo.FindBacklinkedByPageID(context.Background(), noLinkPageID, spaceID)
		if err != nil {
			t.Fatalf("FindBacklinkedByPageID() error = %v", err)
		}
		if len(pages) != 0 {
			t.Errorf("len(pages) = %v, want 0", len(pages))
		}
	})
}

func TestPageRepository_FindByTopicAndTitle(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-topic-title-space").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("My Page").
		Build()

	t.Run("トピックIDとタイトルでページを取得できる", func(t *testing.T) {
		page, err := repo.FindByTopicAndTitle(context.Background(), topicID, "My Page", spaceID)
		if err != nil {
			t.Fatalf("FindByTopicAndTitle() error = %v", err)
		}
		if page == nil {
			t.Fatal("FindByTopicAndTitle() returned nil, want page")
		}
		if page.ID != pageID {
			t.Errorf("page.ID = %v, want %v", page.ID, pageID)
		}
	})

	t.Run("存在しないタイトルはnilを返す", func(t *testing.T) {
		page, err := repo.FindByTopicAndTitle(context.Background(), topicID, "Not Exist", spaceID)
		if err != nil {
			t.Fatalf("FindByTopicAndTitle() error = %v", err)
		}
		if page != nil {
			t.Errorf("FindByTopicAndTitle() = %v, want nil", page)
		}
	})

	t.Run("廃棄済みページも取得できる", func(t *testing.T) {
		discardedPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Discarded Page").
			WithDiscarded().
			Build()

		page, err := repo.FindByTopicAndTitle(context.Background(), topicID, "Discarded Page", spaceID)
		if err != nil {
			t.Fatalf("FindByTopicAndTitle() error = %v", err)
		}
		if page == nil {
			t.Fatal("FindByTopicAndTitle() returned nil, want discarded page")
		}
		if page.ID != discardedPageID {
			t.Errorf("page.ID = %v, want %v", page.ID, discardedPageID)
		}
	})
}

func TestPageRepository_Update(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-update-space").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Before Update").
		WithBody("old body").
		WithBodyHTML("<p>old body</p>").
		Build()

	t.Run("ページを更新できる", func(t *testing.T) {
		now := time.Now()
		newTitle := "After Update"
		page, err := repo.Update(context.Background(), UpdatePageInput{
			ID:            pageID,
			SpaceID:       spaceID,
			TopicID:       topicID,
			Title:         &newTitle,
			Body:          "new body",
			BodyHTML:      "<p>new body</p>",
			LinkedPageIDs: []model.PageID{},
			ModifiedAt:    now,
			PublishedAt:   &now,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if page == nil {
			t.Fatal("Update() returned nil, want page")
		}
		if page.Title == nil || *page.Title != "After Update" {
			t.Errorf("page.Title = %v, want 'After Update'", page.Title)
		}
		if page.Body != "new body" {
			t.Errorf("page.Body = %v, want 'new body'", page.Body)
		}
		if page.BodyHTML != "<p>new body</p>" {
			t.Errorf("page.BodyHTML = %v, want '<p>new body</p>'", page.BodyHTML)
		}
	})
}

func TestPageRepository_FindBacklinksForPages(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-backlinks-batch").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// ターゲットページ2つ
	targetPage1ID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Target 1").
		Build()

	targetPage2ID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Target 2").
		Build()

	// targetPage1をリンクしているページ3件
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(10).
		WithTitle("Linker1-Old").
		WithLinkedPageIDs([]model.PageID{targetPage1ID}).
		WithModifiedAt(baseTime).
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(11).
		WithTitle("Linker1-Mid").
		WithLinkedPageIDs([]model.PageID{targetPage1ID}).
		WithModifiedAt(baseTime.Add(1 * time.Hour)).
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(12).
		WithTitle("Linker1-New").
		WithLinkedPageIDs([]model.PageID{targetPage1ID}).
		WithModifiedAt(baseTime.Add(2 * time.Hour)).
		Build()

	// targetPage2をリンクしているページ1件
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(20).
		WithTitle("Linker2-Only").
		WithLinkedPageIDs([]model.PageID{targetPage2ID}).
		WithModifiedAt(baseTime).
		Build()

	targetPage1 := &model.Page{ID: targetPage1ID}
	targetPage2 := &model.Page{ID: targetPage2ID}

	t.Run("複数ターゲットのバックリンクを一括取得できる", func(t *testing.T) {
		result, err := repo.FindBacklinksForPages(context.Background(), []*model.Page{targetPage1, targetPage2}, spaceID, AllTopicsVisible(), 100, nil)
		if err != nil {
			t.Fatalf("FindBacklinksForPages() error = %v", err)
		}

		// targetPage1のバックリンクは3件
		if len(result[targetPage1ID].Pages) != 3 {
			t.Errorf("targetPage1 backlinks = %d, want 3", len(result[targetPage1ID].Pages))
		}
		if result[targetPage1ID].TotalCount != 3 {
			t.Errorf("targetPage1 TotalCount = %d, want 3", result[targetPage1ID].TotalCount)
		}

		// targetPage2のバックリンクは1件
		if len(result[targetPage2ID].Pages) != 1 {
			t.Errorf("targetPage2 backlinks = %d, want 1", len(result[targetPage2ID].Pages))
		}
		if result[targetPage2ID].TotalCount != 1 {
			t.Errorf("targetPage2 TotalCount = %d, want 1", result[targetPage2ID].TotalCount)
		}
	})

	t.Run("limitで取得件数を制限できる", func(t *testing.T) {
		result, err := repo.FindBacklinksForPages(context.Background(), []*model.Page{targetPage1}, spaceID, AllTopicsVisible(), 2, nil)
		if err != nil {
			t.Fatalf("FindBacklinksForPages() error = %v", err)
		}

		// limitが2なのでページは2件のみ
		if len(result[targetPage1ID].Pages) != 2 {
			t.Errorf("targetPage1 backlinks = %d, want 2", len(result[targetPage1ID].Pages))
		}
		// TotalCountは全件数の3
		if result[targetPage1ID].TotalCount != 3 {
			t.Errorf("targetPage1 TotalCount = %d, want 3", result[targetPage1ID].TotalCount)
		}
	})

	t.Run("バックリンクがないターゲットは空の結果を返す", func(t *testing.T) {
		isolatedPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(50).
			WithTitle("Isolated").
			Build()
		isolatedPage := &model.Page{ID: isolatedPageID}

		result, err := repo.FindBacklinksForPages(context.Background(), []*model.Page{isolatedPage}, spaceID, AllTopicsVisible(), 100, nil)
		if err != nil {
			t.Fatalf("FindBacklinksForPages() error = %v", err)
		}

		if len(result[isolatedPageID].Pages) != 0 {
			t.Errorf("isolated backlinks = %d, want 0", len(result[isolatedPageID].Pages))
		}
		if result[isolatedPageID].TotalCount != 0 {
			t.Errorf("isolated TotalCount = %d, want 0", result[isolatedPageID].TotalCount)
		}
	})

	t.Run("空のターゲットリストはnilを返す", func(t *testing.T) {
		result, err := repo.FindBacklinksForPages(context.Background(), []*model.Page{}, spaceID, AllTopicsVisible(), 100, nil)
		if err != nil {
			t.Fatalf("FindBacklinksForPages() error = %v", err)
		}
		if result != nil {
			t.Errorf("result = %v, want nil", result)
		}
	})

	t.Run("廃棄済みトピックのリンク元ページは一括取得の件数からも除外される", func(t *testing.T) {
		discardedTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(91).
			WithName("Discarded Topic").
			WithDiscarded().
			Build()
		targetID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(70).
			WithTitle("Discarded Topic Batch Target").
			Build()
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(discardedTopicID).
			WithNumber(71).
			WithTitle("Discarded Topic Batch Linker").
			WithLinkedPageIDs([]model.PageID{targetID}).
			Build()
		target := &model.Page{ID: targetID}

		result, err := repo.FindBacklinksForPages(context.Background(), []*model.Page{target}, spaceID, AllTopicsVisible(), 100, nil)
		if err != nil {
			t.Fatalf("FindBacklinksForPages() error = %v", err)
		}
		if len(result[targetID].Pages) != 0 {
			t.Errorf("backlinks = %d, want 0", len(result[targetID].Pages))
		}
		if result[targetID].TotalCount != 0 {
			t.Errorf("TotalCount = %d, want 0", result[targetID].TotalCount)
		}
	})

	t.Run("ゴミ箱に入ったリンク元ページと閲覧可能トピックのフィルタが件数にも効く", func(t *testing.T) {
		otherTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(90).
			WithName("Other Topic").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()

		targetPage3ID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(60).
			WithTitle("Target 3").
			Build()
		targetPage3 := &model.Page{ID: targetPage3ID}

		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(61).
			WithTitle("Linker3-Visible").
			WithLinkedPageIDs([]model.PageID{targetPage3ID}).
			Build()

		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(62).
			WithTitle("Linker3-Trashed").
			WithLinkedPageIDs([]model.PageID{targetPage3ID}).
			WithTrashed().
			Build()

		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(otherTopicID).
			WithNumber(63).
			WithTitle("Linker3-Hidden").
			WithLinkedPageIDs([]model.PageID{targetPage3ID}).
			Build()

		allResult, err := repo.FindBacklinksForPages(context.Background(), []*model.Page{targetPage3}, spaceID, AllTopicsVisible(), 100, nil)
		if err != nil {
			t.Fatalf("FindBacklinksForPages() error = %v", err)
		}
		if len(allResult[targetPage3ID].Pages) != 2 {
			t.Errorf("AllTopicsVisible: backlinks = %d, want 2", len(allResult[targetPage3ID].Pages))
		}
		if allResult[targetPage3ID].TotalCount != 2 {
			t.Errorf("AllTopicsVisible: TotalCount = %d, want 2", allResult[targetPage3ID].TotalCount)
		}

		limitedResult, err := repo.FindBacklinksForPages(context.Background(), []*model.Page{targetPage3}, spaceID, VisibleTopics([]model.TopicID{topicID}), 100, nil)
		if err != nil {
			t.Fatalf("FindBacklinksForPages() error = %v", err)
		}
		if len(limitedResult[targetPage3ID].Pages) != 1 {
			t.Fatalf("VisibleTopics: backlinks = %d, want 1", len(limitedResult[targetPage3ID].Pages))
		}
		if *limitedResult[targetPage3ID].Pages[0].Title != "Linker3-Visible" {
			t.Errorf("VisibleTopics: backlinks[0].Title = %v, want 'Linker3-Visible'", *limitedResult[targetPage3ID].Pages[0].Title)
		}
		if limitedResult[targetPage3ID].TotalCount != 1 {
			t.Errorf("VisibleTopics: TotalCount = %d, want 1", limitedResult[targetPage3ID].TotalCount)
		}
	})
}

func TestPageRepository_CreateLinkedPage(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-create-linked-space").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	t.Run("Wikiリンクからページを作成できる", func(t *testing.T) {
		page, err := repo.CreateLinkedPage(context.Background(), CreateLinkedPageInput{
			SpaceID: spaceID,
			TopicID: topicID,
			Number:  100,
			Title:   "Linked Page",
		})
		if err != nil {
			t.Fatalf("CreateLinkedPage() error = %v", err)
		}
		if page == nil {
			t.Fatal("CreateLinkedPage() returned nil, want page")
		}
		if page.SpaceID != spaceID {
			t.Errorf("page.SpaceID = %v, want %v", page.SpaceID, spaceID)
		}
		if page.TopicID != topicID {
			t.Errorf("page.TopicID = %v, want %v", page.TopicID, topicID)
		}
		if page.Number != 100 {
			t.Errorf("page.Number = %v, want 100", page.Number)
		}
		if page.Title == nil || *page.Title != "Linked Page" {
			t.Errorf("page.Title = %v, want 'Linked Page'", page.Title)
		}
		if page.Body != "" {
			t.Errorf("page.Body = %v, want empty string", page.Body)
		}
		if page.BodyHTML != "" {
			t.Errorf("page.BodyHTML = %v, want empty string", page.BodyHTML)
		}
		if page.PublishedAt != nil {
			t.Errorf("page.PublishedAt = %v, want nil", page.PublishedAt)
		}
	})
}
