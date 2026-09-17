package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// pagePairsQueryNameとpageByTitleQueryNameはQueryCounterで数えるクエリの名前。
// sqlcが生成するSQLの先頭にクエリ名のコメントが残ることを利用する。
const (
	pagePairsQueryName   = "FindPagesByTopicAndTitlePairs"
	pageByTitleQueryName = "FindPageByTopicAndTitle"
)

// wikilinkKeyはテスト用にWikilinkKeyを組み立てる
func wikilinkKey(topicName string, pageTitle string) markup.WikilinkKey {
	return markup.WikilinkKey{
		Raw:       topicName + "/" + pageTitle,
		TopicName: topicName,
		PageTitle: pageTitle,
	}
}

func TestPreviewPageLocationResolver_ResolveByKeys(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	counter := testutil.NewQueryCounter(tx)
	q := counter.Queries()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("preview-resolver-space").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Alpha").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Beta").
		Build()

	resolver := &previewPageLocationResolver{
		topicRepo: repository.NewTopicRepository(q),
		pageRepo:  repository.NewPageRepository(q),
	}

	// 同じキーを繰り返し、複数のキーを混ぜた本文を想定する。
	// 存在しないトピックのキーは、一括取得の組に含める前にスキップされる。
	keys := []markup.WikilinkKey{
		wikilinkKey("General", "Alpha"),
		wikilinkKey("General", "Beta"),
		wikilinkKey("General", "Alpha"),
		wikilinkKey("General", "存在しないページ"),
		wikilinkKey("存在しないトピック", "Alpha"),
	}

	locations, err := resolver.ResolveByKeys(context.Background(), keys, spaceID)
	if err != nil {
		t.Fatalf("ResolveByKeys()のエラー = %v", err)
	}

	gotTitles := make([]string, 0, len(locations))
	for _, location := range locations {
		gotTitles = append(gotTitles, location.PageTitle)
	}
	if len(gotTitles) != 2 || gotTitles[0] != "Alpha" || gotTitles[1] != "Beta" {
		t.Errorf("解決したページタイトル = %v、期待値 = [Alpha Beta]", gotTitles)
	}

	// リンク数に依らず、トピックとページの取得はそれぞれ1クエリで済む。
	if got := counter.CountContaining("FindTopicsBySpaceAndNames"); got != 1 {
		t.Errorf("FindTopicsBySpaceAndNamesの発行回数 = %d、期待値 = 1", got)
	}
	if got := counter.CountContaining(pagePairsQueryName); got != 1 {
		t.Errorf("%sの発行回数 = %d、期待値 = 1", pagePairsQueryName, got)
	}
	if got := counter.CountContaining(pageByTitleQueryName); got != 0 {
		t.Errorf("%sの発行回数 = %d、期待値 = 0 (キーごとの単発クエリは発行しない)", pageByTitleQueryName, got)
	}
}

func TestLinkCreatingPageLocationResolver_ResolveByKeys(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	counter := testutil.NewQueryCounter(tx)
	q := counter.Queries()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("link-resolver@example.com").
		WithAtname("linkresolver").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("link-resolver-space").
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
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Alpha").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Beta").
		Build()

	resolver := &linkCreatingPageLocationResolver{
		spaceMemberID:  spaceMemberID,
		topicRepo:      repository.NewTopicRepository(q),
		pageRepo:       repository.NewPageRepository(q),
		pageEditorRepo: repository.NewPageEditorRepository(q),
	}

	keys := []markup.WikilinkKey{
		wikilinkKey("General", "Alpha"),
		wikilinkKey("General", "Beta"),
		wikilinkKey("General", "Alpha"),
	}

	locations, err := resolver.ResolveByKeys(context.Background(), keys, spaceID)
	if err != nil {
		t.Fatalf("ResolveByKeys()のエラー = %v", err)
	}
	if len(locations) != 2 {
		t.Errorf("len(locations) = %d、期待値 = 2", len(locations))
	}
	if len(resolver.linkedPageIDs) != 2 {
		t.Errorf("len(linkedPageIDs) = %d、期待値 = 2", len(resolver.linkedPageIDs))
	}

	// 既存ページだけを指す本文では、リンク数に依らずページの取得は1クエリで済む。
	if got := counter.CountContaining(pagePairsQueryName); got != 1 {
		t.Errorf("%sの発行回数 = %d、期待値 = 1", pagePairsQueryName, got)
	}
	if got := counter.CountContaining(pageByTitleQueryName); got != 0 {
		t.Errorf("%sの発行回数 = %d、期待値 = 0 (キーごとの単発クエリは発行しない)", pageByTitleQueryName, got)
	}
}

func TestLinkCreatingPageLocationResolver_ResolveByKeys_CreatesMissingPage(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("link-resolver-create@example.com").
		WithAtname("linkresolvercreate").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("link-resolver-create-space").
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
	existingPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Alpha").
		Build()

	pageRepo := repository.NewPageRepository(q)
	resolver := &linkCreatingPageLocationResolver{
		spaceMemberID:  spaceMemberID,
		topicRepo:      repository.NewTopicRepository(q),
		pageRepo:       pageRepo,
		pageEditorRepo: repository.NewPageEditorRepository(q),
	}

	keys := []markup.WikilinkKey{
		wikilinkKey("General", "Alpha"),
		wikilinkKey("General", "新しいページ"),
	}

	locations, err := resolver.ResolveByKeys(context.Background(), keys, spaceID)
	if err != nil {
		t.Fatalf("ResolveByKeys()のエラー = %v", err)
	}
	if len(locations) != 2 {
		t.Fatalf("len(locations) = %d、期待値 = 2", len(locations))
	}

	created, err := pageRepo.FindByTopicAndTitle(context.Background(), topicID, "新しいページ", spaceID)
	if err != nil {
		t.Fatalf("FindByTopicAndTitle()のエラー = %v", err)
	}
	if created == nil {
		t.Fatal("一括取得で見つからなかったリンク先ページが作成されていない")
	}

	gotIDs := []model.PageID{locations[0].PageID, locations[1].PageID}
	if gotIDs[0] != existingPageID || gotIDs[1] != created.ID {
		t.Errorf("解決したページID = %v、期待値 = [%v %v]", gotIDs, existingPageID, created.ID)
	}
}

func TestPageLocationResolvers_ResolveByKeys_PreservesDatabaseTitleMatching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		title  string
		inputs []string
	}{
		{name: "ASCIIの大小文字", title: "Alpha", inputs: []string{"alpha", "Alpha", "ALPHA"}},
		{name: "Unicodeの大小文字", title: "\u1c89", inputs: []string{"\u1c8a", "\u1c89"}},
	}

	for _, tt := range tests {
		for _, creating := range []bool{false, true} {
			mode := "プレビュー"
			if creating {
				mode = "公開"
			}
			t.Run(tt.name+"/"+mode, func(t *testing.T) {
				_, tx := testutil.SetupTx(t)
				spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("resolver-title-matching").Build()
				topicID := testutil.NewTopicBuilder(t, tx).
					WithSpaceID(spaceID).WithNumber(1).WithName("General").Build()
				pageID := testutil.NewPageBuilder(t, tx).
					WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle(tt.title).Build()

				// DBのUnicode対応は環境によって異なるため、単件検索で一致する入力を検証する。
				pageRepo := repository.NewPageRepository(testutil.QueriesWithTx(tx))
				for _, title := range tt.inputs {
					page, err := pageRepo.FindByTopicAndTitle(context.Background(), topicID, title, spaceID)
					if err != nil {
						t.Fatal(err)
					}
					if page == nil {
						t.Skipf("このDBでは %q と %q の大小文字が一致しない", tt.title, title)
					}
					if page.ID != pageID {
						t.Fatalf("単件検索のページID = %v、期待値 = %v", page.ID, pageID)
					}
				}

				counter := testutil.NewQueryCounter(tx)
				q := counter.Queries()
				var resolver markup.PageLocationResolver = &previewPageLocationResolver{
					topicRepo: repository.NewTopicRepository(q),
					pageRepo:  repository.NewPageRepository(q),
				}
				if creating {
					userID := testutil.NewUserBuilder(t, tx).
						WithEmail("resolver-title-matching@example.com").WithAtname("titlematching").Build()
					memberID := testutil.NewSpaceMemberBuilder(t, tx).
						WithSpaceID(spaceID).WithUserID(userID).Build()
					resolver = &linkCreatingPageLocationResolver{
						spaceMemberID:  memberID,
						topicRepo:      repository.NewTopicRepository(q),
						pageRepo:       repository.NewPageRepository(q),
						pageEditorRepo: repository.NewPageEditorRepository(q),
					}
				}

				keys := make([]markup.WikilinkKey, 0, len(tt.inputs)+1)
				for _, title := range tt.inputs {
					keys = append(keys, wikilinkKey("General", title))
				}
				keys = append(keys, keys[0])
				locations, err := resolver.ResolveByKeys(context.Background(), keys, spaceID)
				if err != nil {
					t.Fatal(err)
				}
				if len(locations) != len(tt.inputs) {
					t.Fatalf("解決件数 = %d、期待値 = %d", len(locations), len(tt.inputs))
				}
				for i, location := range locations {
					if location.PageID != pageID || location.PageTitle != tt.title || location.Key != keys[i] {
						t.Errorf("入力 %q の解決結果 = %+v、期待値 = ページ %v、保存タイトル %q、元の入力キー", tt.inputs[i], location, pageID, tt.title)
					}
				}
				if got := counter.CountContaining(pagePairsQueryName); got != 1 {
					t.Errorf("一括取得の発行回数 = %d、期待値 = 1", got)
				}
				if got := counter.CountContaining(pageByTitleQueryName); got != 0 {
					t.Errorf("単件取得の発行回数 = %d、期待値 = 0", got)
				}
			})
		}
	}
}
