package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetPageDetailUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionPageRepo := repository.NewSuggestionPageRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	uc := NewGetPageDetailUsecase(spaceRepo, spaceMemberRepo, pageRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, suggestionPageRepo, suggestionRepo)

	// テストデータを作成
	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpd-owner@example.com").
		WithAtname("gpdowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gpd-space").
		WithName("GPD Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("テストトピック").
		WithVisibility(0). // public
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("テストページ").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	t.Run("存在しないスペースでnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "nonexistent",
			PageNumber:      1,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for non-existent space")
		}
	})

	t.Run("スペースメンバーでないユーザーでnilが返る", func(t *testing.T) {
		nonMemberID := testutil.NewUserBuilder(t, tx).
			WithEmail("gpd-nonmember@example.com").
			WithAtname("gpdnonmember").
			Build()
		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "gpd-space",
			PageNumber:      1,
			UserID:          nonMemberID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for non-member user")
		}
	})

	t.Run("存在しないページでnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "gpd-space",
			PageNumber:      999,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for non-existent page")
		}
	})

	t.Run("正常系: すべてのデータが取得できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "gpd-space",
			PageNumber:      1,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Space.Name != "GPD Space" {
			t.Errorf("Space.Name = %q, want %q", output.Space.Name, "GPD Space")
		}
		if output.SpaceMember == nil {
			t.Fatal("SpaceMember should not be nil")
		}
		if output.Page == nil {
			t.Fatal("Page should not be nil")
		}
		if output.Topic == nil {
			t.Fatal("Topic should not be nil")
		}
		if output.Topic.Name != "テストトピック" {
			t.Errorf("Topic.Name = %q, want %q", output.Topic.Name, "テストトピック")
		}
		if output.TopicMember == nil {
			t.Fatal("TopicMember should not be nil")
		}
		if output.DraftPage != nil {
			t.Error("DraftPage should be nil when no draft exists")
		}
	})

	t.Run("正常系: DraftPageが存在する場合も取得できる", func(t *testing.T) {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("下書きありページ").
			WithLinkedPageIDs([]model.PageID{}).
			Build()
		testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("下書きタイトル").
			Build()

		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "gpd-space",
			PageNumber:      2,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.DraftPage == nil {
			t.Fatal("DraftPage should not be nil when draft exists")
		}
	})

	t.Run("IncludeDraftPagesがfalseの場合はDraftPagesを取得しない", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "gpd-space",
			PageNumber:      1,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.DraftPages != nil {
			t.Errorf("DraftPages should be nil when IncludeDraftPages is false, got %d entries", len(output.DraftPages))
		}
	})

	t.Run("IncludeDraftPagesがtrueの場合は同一スペースの下書き一覧を取得する", func(t *testing.T) {
		// Create a draft owned by this subtest so the assertion does not depend on other subtests' data.
		// [Ja] アサーションが他サブテストのデータに依存しないよう、このサブテスト専用の下書きを作成する。
		listPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(20).
			WithTitle("一覧確認用ページ").
			WithLinkedPageIDs([]model.PageID{}).
			Build()
		testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(listPageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("一覧確認用下書き").
			Build()

		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier:   "gpd-space",
			PageNumber:        1,
			UserID:            ownerID,
			IncludeDraftPages: true,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}

		// The draft created above must appear in the list.
		// [Ja] 上で作成した下書きが一覧に含まれることを確認する。
		found := false
		for _, d := range output.DraftPages {
			if d.Title != nil && *d.Title == "一覧確認用下書き" {
				found = true
			}
		}
		if !found {
			t.Error("created draft should be included when IncludeDraftPages is true")
		}
	})

	t.Run("IncludeDraftRevisionsがfalseの場合はリビジョンを取得しない", func(t *testing.T) {
		revPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(30).
			WithTitle("履歴フラグ確認用ページ").
			WithLinkedPageIDs([]model.PageID{}).
			Build()
		revDraftPageID := testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(revPageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("履歴フラグ確認用下書き").
			Build()
		_, err := draftPageRevisionRepo.Create(context.Background(), repository.CreateDraftPageRevisionInput{
			DraftPageID:   revDraftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "rev flag",
			Body:          "rev flag body",
			BodyHTML:      "<p>rev flag body</p>",
		})
		if err != nil {
			t.Fatalf("Create() revision error = %v", err)
		}

		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "gpd-space",
			PageNumber:      30,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.DraftPageRevisions != nil {
			t.Errorf("DraftPageRevisions should be nil when IncludeDraftRevisions is false, got %d entries", len(output.DraftPageRevisions))
		}
		if output.DraftPageRevisionTotalCount != 0 {
			t.Errorf("DraftPageRevisionTotalCount = %d, want 0 when IncludeDraftRevisions is false", output.DraftPageRevisionTotalCount)
		}
	})

	t.Run("IncludeDraftRevisionsがtrueでも下書きが無い場合はリビジョンを取得しない", func(t *testing.T) {
		// Page 1 has no draft for this member (created in the top-level fixture without a draft).
		// [Ja] ページ 1 はトップレベルの fixture で下書きなしで作成されている。
		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier:       "gpd-space",
			PageNumber:            1,
			UserID:                ownerID,
			IncludeDraftRevisions: true,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.DraftPage != nil {
			t.Fatal("precondition failed: DraftPage should be nil")
		}
		if output.DraftPageRevisions != nil {
			t.Errorf("DraftPageRevisions should be nil when no draft exists, got %d entries", len(output.DraftPageRevisions))
		}
		if output.DraftPageRevisionTotalCount != 0 {
			t.Errorf("DraftPageRevisionTotalCount = %d, want 0 when no draft exists", output.DraftPageRevisionTotalCount)
		}
	})

	t.Run("IncludeDraftRevisionsがtrueの場合はリビジョン一覧と総件数を取得する", func(t *testing.T) {
		revPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(31).
			WithTitle("履歴取得確認用ページ").
			WithLinkedPageIDs([]model.PageID{}).
			Build()
		revDraftPageID := testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(revPageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("履歴取得確認用下書き").
			Build()
		wantTitles := map[string]bool{"rev list 1": false, "rev list 2": false}
		for title := range wantTitles {
			_, err := draftPageRevisionRepo.Create(context.Background(), repository.CreateDraftPageRevisionInput{
				DraftPageID:   revDraftPageID,
				SpaceID:       spaceID,
				SpaceMemberID: spaceMemberID,
				Title:         title,
				Body:          "body of " + title,
				BodyHTML:      "<p>body of " + title + "</p>",
			})
			if err != nil {
				t.Fatalf("Create() revision error = %v", err)
			}
		}

		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier:       "gpd-space",
			PageNumber:            31,
			UserID:                ownerID,
			IncludeDraftRevisions: true,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if len(output.DraftPageRevisions) != 2 {
			t.Fatalf("len(DraftPageRevisions) = %d, want 2", len(output.DraftPageRevisions))
		}
		if output.DraftPageRevisionTotalCount != 2 {
			t.Errorf("DraftPageRevisionTotalCount = %d, want 2", output.DraftPageRevisionTotalCount)
		}

		// Both created revisions must be returned. Strict ordering (newest first) is covered by the
		// repository tests, so this test only verifies the set of entries.
		//
		// [Ja] 作成した 2 件のリビジョンが両方返ること。厳密な並び順 (新しい順) は Repository の
		// テストで担保されているため、ここではエントリの集合のみを検証する。
		for _, r := range output.DraftPageRevisions {
			if _, ok := wantTitles[r.Title]; !ok {
				t.Errorf("unexpected revision title %q", r.Title)
				continue
			}
			wantTitles[r.Title] = true
		}
		for title, seen := range wantTitles {
			if !seen {
				t.Errorf("revision %q should be included", title)
			}
		}
	})
}
