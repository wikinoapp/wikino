package usecase

import (
	"context"
	"testing"

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
			Page:              page,
			SpaceID:           spaceID,
			CurrentPage:       1,
			LinkLimit:         viewmodel.LinkLimit,
			BacklinkLimit:     viewmodel.BacklinkLimit,
			PageBacklinkLimit: viewmodel.PageBacklinkLimit,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if len(output.LinkedPages) != 0 {
			t.Errorf("len(LinkedPages) = %d, want 0", len(output.LinkedPages))
		}
		if output.LinkedTotalCount != 0 {
			t.Errorf("LinkedTotalCount = %d, want 0", output.LinkedTotalCount)
		}
		if len(output.BacklinksPerPage) != 0 {
			t.Errorf("len(BacklinksPerPage) = %d, want 0", len(output.BacklinksPerPage))
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
			Page:              page,
			SpaceID:           spaceID,
			CurrentPage:       1,
			LinkLimit:         viewmodel.LinkLimit,
			BacklinkLimit:     viewmodel.BacklinkLimit,
			PageBacklinkLimit: viewmodel.PageBacklinkLimit,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if len(output.LinkedPages) != 1 {
			t.Errorf("len(LinkedPages) = %d, want 1", len(output.LinkedPages))
		}
		if output.LinkedTotalCount != 1 {
			t.Errorf("LinkedTotalCount = %d, want 1", output.LinkedTotalCount)
		}
		if len(output.LinkTopics) == 0 {
			t.Error("LinkTopics should not be empty")
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
			Page:              page,
			DraftPage:         draftPage,
			SpaceID:           spaceID,
			CurrentPage:       1,
			LinkLimit:         viewmodel.LinkLimit,
			BacklinkLimit:     viewmodel.BacklinkLimit,
			PageBacklinkLimit: viewmodel.PageBacklinkLimit,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if len(output.LinkedPages) != 1 {
			t.Errorf("len(LinkedPages) = %d, want 1", len(output.LinkedPages))
		}
	})
}
