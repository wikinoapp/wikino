package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestGetEditLinkDataUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	uc := NewGetEditLinkDataUsecase(pageRepo, topicRepo)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("geld-space").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("テストトピック").
		WithVisibility(0).
		Build()

	t.Run("リンクがないページで空のデータが返る", func(t *testing.T) {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("リンクなしページ").
			WithLinkedPageIDs([]model.PageID{}).
			Build()

		page := &model.Page{
			ID:            pageID,
			LinkedPageIDs: []model.PageID{},
		}

		output, err := uc.Execute(context.Background(), GetEditLinkDataInput{
			Page:                   page,
			SpaceID:                spaceID,
			CurrentPage:            1,
			LinkLimit:              viewmodel.LinkLimit,
			BacklinkLimit:          viewmodel.BacklinkLimit,
			PageBacklinkLimit:      viewmodel.PageBacklinkLimit,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if len(output.LinkedPages) != 0 {
			t.Errorf("len(LinkedPages) = %d、期待値 = 0", len(output.LinkedPages))
		}
		if output.LinkedTotalCount != 0 {
			t.Errorf("LinkedTotalCount = %d、期待値 = 0", output.LinkedTotalCount)
		}
		if len(output.BacklinksPerPage) != 0 {
			t.Errorf("len(BacklinksPerPage) = %d、期待値 = 0", len(output.BacklinksPerPage))
		}
	})

	t.Run("リンクがあるページでリンク先データが取得できる", func(t *testing.T) {
		linkedPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("リンク先ページ").
			WithLinkedPageIDs([]model.PageID{}).
			Build()

		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(3).
			WithTitle("リンク元ページ").
			WithLinkedPageIDs([]model.PageID{linkedPageID}).
			Build()

		page := &model.Page{
			ID:            pageID,
			TopicID:       topicID,
			LinkedPageIDs: []model.PageID{linkedPageID},
		}

		output, err := uc.Execute(context.Background(), GetEditLinkDataInput{
			Page:                   page,
			SpaceID:                spaceID,
			CurrentPage:            1,
			LinkLimit:              viewmodel.LinkLimit,
			BacklinkLimit:          viewmodel.BacklinkLimit,
			PageBacklinkLimit:      viewmodel.PageBacklinkLimit,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if len(output.LinkedPages) != 1 {
			t.Errorf("len(LinkedPages) = %d、期待値 = 1", len(output.LinkedPages))
		}
		if output.LinkedTotalCount != 1 {
			t.Errorf("LinkedTotalCount = %d、期待値 = 1", output.LinkedTotalCount)
		}
		if len(output.LinkTopics) == 0 {
			t.Error("LinkTopicsが空")
		}
	})

	t.Run("DraftPageがある場合はDraftPageのリンクを使用する", func(t *testing.T) {
		draftLinkedPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(4).
			WithTitle("下書きリンク先ページ").
			WithLinkedPageIDs([]model.PageID{}).
			Build()

		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(5).
			WithTitle("下書きページ").
			WithLinkedPageIDs([]model.PageID{}).
			Build()

		page := &model.Page{
			ID:            pageID,
			TopicID:       topicID,
			LinkedPageIDs: []model.PageID{},
		}
		draftPage := &model.DraftPage{
			LinkedPageIDs: []model.PageID{draftLinkedPageID},
		}

		output, err := uc.Execute(context.Background(), GetEditLinkDataInput{
			Page:                   page,
			DraftPage:              draftPage,
			SpaceID:                spaceID,
			CurrentPage:            1,
			LinkLimit:              viewmodel.LinkLimit,
			BacklinkLimit:          viewmodel.BacklinkLimit,
			PageBacklinkLimit:      viewmodel.PageBacklinkLimit,
			LinkedPageBacklinkPage: 1,
			PageBacklinkPage:       1,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if len(output.LinkedPages) != 1 {
			t.Errorf("len(LinkedPages) = %d、期待値 = 1", len(output.LinkedPages))
		}
	})

	// 編集画面のフルページフォールバックが、ページ表示画面と同じページネーション契約で
	// 3種類すべての関連ページ一覧を進めることを確認する。
	t.Run("フルページフォールバックで各一覧の2ページ目を取得できる", func(t *testing.T) {
		baseTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		linkedNewerID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(100).
			WithTitle("Linked Newer").
			WithModifiedAt(baseTime.Add(2 * time.Hour)).
			WithLinkedPageIDs([]model.PageID{}).
			Build()
		linkedOlderID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(101).
			WithTitle("Linked Older").
			WithModifiedAt(baseTime.Add(time.Hour)).
			WithLinkedPageIDs([]model.PageID{}).
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(102).
			WithTitle("Paginated Editor Page").
			WithLinkedPageIDs([]model.PageID{linkedNewerID, linkedOlderID}).
			Build()

		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(103).
			WithTitle("Nested Backlink Newer").
			WithModifiedAt(baseTime.Add(4 * time.Hour)).
			WithLinkedPageIDs([]model.PageID{linkedOlderID}).
			Build()
		nestedOlderID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(104).
			WithTitle("Nested Backlink Older").
			WithModifiedAt(baseTime.Add(3 * time.Hour)).
			WithLinkedPageIDs([]model.PageID{linkedOlderID}).
			Build()

		otherNestedNewerID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(107).
			WithTitle("Other Nested Backlink Newer").
			WithModifiedAt(baseTime.Add(8 * time.Hour)).
			WithLinkedPageIDs([]model.PageID{linkedNewerID}).
			Build()
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(108).
			WithTitle("Other Nested Backlink Older").
			WithModifiedAt(baseTime.Add(7 * time.Hour)).
			WithLinkedPageIDs([]model.PageID{linkedNewerID}).
			Build()

		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(105).
			WithTitle("Page Backlink Newer").
			WithModifiedAt(baseTime.Add(6 * time.Hour)).
			WithLinkedPageIDs([]model.PageID{pageID}).
			Build()
		pageBacklinkOlderID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(106).
			WithTitle("Page Backlink Older").
			WithModifiedAt(baseTime.Add(5 * time.Hour)).
			WithLinkedPageIDs([]model.PageID{pageID}).
			Build()

		output, err := uc.Execute(context.Background(), GetEditLinkDataInput{
			Page: &model.Page{
				ID:            pageID,
				TopicID:       topicID,
				LinkedPageIDs: []model.PageID{linkedNewerID, linkedOlderID},
			},
			SpaceID:                spaceID,
			CurrentPage:            2,
			LinkLimit:              1,
			BacklinkLimit:          1,
			PageBacklinkLimit:      1,
			LinkedPageNumber:       101,
			LinkedPageBacklinkPage: 2,
			PageBacklinkPage:       2,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if output.LinkedTotalCount != 2 || len(output.LinkedPages) != 1 || output.LinkedPages[0].ID != linkedOlderID {
			t.Errorf("リンク先の2ページ目 = (%d, %v)、期待値 = (2, [%s])", output.LinkedTotalCount, output.LinkedPages, linkedOlderID)
		}
		selected := output.BacklinksPerPage[linkedOlderID]
		if selected == nil {
			t.Fatal("BacklinksPerPageに選択中のリンク先ページが含まれていない")
		}
		if selected.TotalCount != 2 || len(selected.Pages) != 1 || selected.Pages[0].ID != nestedOlderID {
			t.Errorf("入れ子のバックリンクの2ページ目 = (%d, %v)、期待値 = (2, [%s])", selected.TotalCount, selected.Pages, nestedOlderID)
		}
		if output.PageBacklinkCount != 2 || len(output.PageBacklinks) != 1 || output.PageBacklinks[0].ID != pageBacklinkOlderID {
			t.Errorf("ページのバックリンクの2ページ目 = (%d, %v)、期待値 = (2, [%s])", output.PageBacklinkCount, output.PageBacklinks, pageBacklinkOlderID)
		}

		// 選択したカードのネスト一覧だけを差し替え、別カードは1ページ目に残す。
		multiCardOutput, err := uc.Execute(context.Background(), GetEditLinkDataInput{
			Page: &model.Page{
				ID:            pageID,
				TopicID:       topicID,
				LinkedPageIDs: []model.PageID{linkedNewerID, linkedOlderID},
			},
			SpaceID:                spaceID,
			CurrentPage:            1,
			LinkLimit:              2,
			BacklinkLimit:          1,
			PageBacklinkLimit:      1,
			LinkedPageNumber:       101,
			LinkedPageBacklinkPage: 2,
			PageBacklinkPage:       1,
		})
		if err != nil {
			t.Fatalf("複数カードでのExecute()のエラー = %v", err)
		}

		selected = multiCardOutput.BacklinksPerPage[linkedOlderID]
		if selected == nil || len(selected.Pages) != 1 || selected.Pages[0].ID != nestedOlderID {
			t.Errorf("選択中のカードの入れ子のページ = %#v、期待値 = [%s]", selected, nestedOlderID)
		}
		other := multiCardOutput.BacklinksPerPage[linkedNewerID]
		if other == nil {
			t.Fatal("BacklinksPerPageに選択していないリンク先ページが含まれていない")
		}
		if other.TotalCount != 2 || len(other.Pages) != 1 || other.Pages[0].ID != otherNestedNewerID {
			t.Errorf("選択していないカードの入れ子のページ = (%d, %v)、期待値 = 1ページ目[%s]", other.TotalCount, other.Pages, otherNestedNewerID)
		}
	})

	// 下書き再取得は一覧のコンテナごと差し替えるため、要求ページだけではなく閲覧者が読み込み
	// 済みの範囲すべてを要求する。
	t.Run("IncludePrecedingPagesで各一覧の1ページ目から要求ページまでをまとめて取得できる", func(t *testing.T) {
		baseTime := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
		linkedNewerID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(200).
			WithTitle("Cumulative Linked Newer").
			WithModifiedAt(baseTime.Add(2 * time.Hour)).
			WithLinkedPageIDs([]model.PageID{}).
			Build()
		linkedOlderID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(201).
			WithTitle("Cumulative Linked Older").
			WithModifiedAt(baseTime.Add(time.Hour)).
			WithLinkedPageIDs([]model.PageID{}).
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(202).
			WithTitle("Cumulative Editor Page").
			WithLinkedPageIDs([]model.PageID{linkedNewerID, linkedOlderID}).
			Build()

		nestedNewerID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(203).
			WithTitle("Cumulative Nested Newer").
			WithModifiedAt(baseTime.Add(4 * time.Hour)).
			WithLinkedPageIDs([]model.PageID{linkedOlderID}).
			Build()
		nestedOlderID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(204).
			WithTitle("Cumulative Nested Older").
			WithModifiedAt(baseTime.Add(3 * time.Hour)).
			WithLinkedPageIDs([]model.PageID{linkedOlderID}).
			Build()

		pageBacklinkNewerID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(205).
			WithTitle("Cumulative Page Backlink Newer").
			WithModifiedAt(baseTime.Add(6 * time.Hour)).
			WithLinkedPageIDs([]model.PageID{pageID}).
			Build()
		pageBacklinkOlderID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(206).
			WithTitle("Cumulative Page Backlink Older").
			WithModifiedAt(baseTime.Add(5 * time.Hour)).
			WithLinkedPageIDs([]model.PageID{pageID}).
			Build()

		input := GetEditLinkDataInput{
			Page: &model.Page{
				ID:            pageID,
				TopicID:       topicID,
				LinkedPageIDs: []model.PageID{linkedNewerID, linkedOlderID},
			},
			SpaceID:                spaceID,
			CurrentPage:            2,
			LinkLimit:              1,
			BacklinkLimit:          1,
			PageBacklinkLimit:      1,
			LinkedPageNumber:       201,
			LinkedPageBacklinkPage: 2,
			PageBacklinkPage:       2,
			IncludePrecedingPages:  true,
		}

		output, err := uc.Execute(context.Background(), input)
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		wantLinked := []model.PageID{linkedNewerID, linkedOlderID}
		if !samePageIDs(output.LinkedPages, wantLinked) {
			t.Errorf("リンク先のページ = %v、期待値 = 両方のページ%v", pageIDsOf(output.LinkedPages), wantLinked)
		}
		selected := output.BacklinksPerPage[linkedOlderID]
		if selected == nil {
			t.Fatal("BacklinksPerPageに選択中のリンク先ページが含まれていない")
		}
		if !samePageIDs(selected.Pages, []model.PageID{nestedNewerID, nestedOlderID}) {
			t.Errorf("入れ子のバックリンク = %v、期待値 = 両方のページ", pageIDsOf(selected.Pages))
		}
		if !samePageIDs(output.PageBacklinks, []model.PageID{pageBacklinkNewerID, pageBacklinkOlderID}) {
			t.Errorf("ページのバックリンク = %v、期待値 = 両方のページ", pageIDsOf(output.PageBacklinks))
		}

		// 一覧が縮んだあとに残った古いページ番号でも、空ではなく残っているものをすべて返す。
		// 保存後に編集画面の一覧が空になることがないようにするためである。
		input.CurrentPage = 5
		input.PageBacklinkPage = 5
		staleOutput, err := uc.Execute(context.Background(), input)
		if err != nil {
			t.Fatalf("古い状態でのExecute()のエラー = %v", err)
		}
		if !samePageIDs(staleOutput.LinkedPages, wantLinked) {
			t.Errorf("範囲外のページでのリンク先のページ = %v、期待値 = 両方のページ", pageIDsOf(staleOutput.LinkedPages))
		}
		if !samePageIDs(staleOutput.PageBacklinks, []model.PageID{pageBacklinkNewerID, pageBacklinkOlderID}) {
			t.Errorf("範囲外のページでのページのバックリンク = %v、期待値 = 両方のページ", pageIDsOf(staleOutput.PageBacklinks))
		}
	})
}

func TestCumulativeRelatedPagePagesInRange(t *testing.T) {
	t.Parallel()

	atLimit := relatedPageListInput{
		LinkPage:               MaxCumulativeRelatedPagePages,
		LinkedPageBacklinkPage: MaxCumulativeRelatedPagePages,
		PageBacklinkPage:       MaxCumulativeRelatedPagePages,
	}
	if !cumulativeRelatedPagePagesInRange(atLimit) {
		t.Fatal("累積の上限ちょうどの関連ページの一覧が受け付けられていない")
	}

	tests := []struct {
		name   string
		mutate func(*relatedPageListInput)
	}{
		{
			name: "リンクの一覧",
			mutate: func(input *relatedPageListInput) {
				input.LinkPage++
			},
		},
		{
			name: "入れ子のバックリンクの一覧",
			mutate: func(input *relatedPageListInput) {
				input.LinkedPageBacklinkPage++
			},
		},
		{
			name: "ページのバックリンクの一覧",
			mutate: func(input *relatedPageListInput) {
				input.PageBacklinkPage++
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := atLimit
			tt.mutate(&input)
			if cumulativeRelatedPagePagesInRange(input) {
				t.Error("累積の上限を超えた関連ページの一覧が拒否されていない")
			}
		})
	}
}

// samePageIDsは、渡されたページが期待するIDと順序どおり一致するかを返す。
func samePageIDs(pages []*model.Page, want []model.PageID) bool {
	if len(pages) != len(want) {
		return false
	}
	for i, pg := range pages {
		if pg.ID != want[i] {
			return false
		}
	}
	return true
}

// pageIDsOfは失敗メッセージ用にページをIDだけへ落とす。
func pageIDsOf(pages []*model.Page) []model.PageID {
	ids := make([]model.PageID, 0, len(pages))
	for _, pg := range pages {
		ids = append(ids, pg.ID)
	}
	return ids
}
