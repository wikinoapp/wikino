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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output != nil {
			t.Error("存在しないスペースなのに出力がnilではない")
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output != nil {
			t.Error("非メンバーのユーザーなのに出力がnilではない")
		}
	})

	t.Run("存在しないページでnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "gpd-space",
			PageNumber:      999,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output != nil {
			t.Error("存在しないページなのに出力がnilではない")
		}
	})

	t.Run("正常系: すべてのデータが取得できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "gpd-space",
			PageNumber:      1,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.Space.Name != "GPD Space" {
			t.Errorf("Space.Name = %q、期待値 = %q", output.Space.Name, "GPD Space")
		}
		if output.SpaceMember == nil {
			t.Fatal("SpaceMemberがnil")
		}
		if output.Page == nil {
			t.Fatal("Pageがnil")
		}
		if output.Topic == nil {
			t.Fatal("Topicがnil")
		}
		if output.Topic.Name != "テストトピック" {
			t.Errorf("Topic.Name = %q、期待値 = %q", output.Topic.Name, "テストトピック")
		}
		if output.TopicMember == nil {
			t.Fatal("TopicMemberがnil")
		}
		if output.DraftPage != nil {
			t.Error("下書きが無いのにDraftPageがnilではない")
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.DraftPage == nil {
			t.Fatal("下書きがあるのにDraftPageがnil")
		}
	})

	t.Run("IncludeDraftPagesがfalseの場合はDraftPagesを取得しない", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "gpd-space",
			PageNumber:      1,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.DraftPages != nil {
			t.Errorf("IncludeDraftPagesがfalseなのにDraftPagesがnilではない: %d件", len(output.DraftPages))
		}
	})

	t.Run("IncludeDraftPagesがtrueの場合は同一スペースの下書き一覧を取得する", func(t *testing.T) {
		// アサーションが他サブテストのデータに依存しないよう、このサブテスト専用の下書きを作成する。
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}

		// 上で作成した下書きが一覧に含まれることを確認する。
		found := false
		for _, d := range output.DraftPages {
			if d.Title != nil && *d.Title == "一覧確認用下書き" {
				found = true
			}
		}
		if !found {
			t.Error("IncludeDraftPagesがtrueなのに作成した下書きが含まれていない")
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
			t.Fatalf("Create() (revision) のエラー = %v", err)
		}

		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier: "gpd-space",
			PageNumber:      30,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.DraftPageRevisions != nil {
			t.Errorf("IncludeDraftRevisionsがfalseなのにDraftPageRevisionsがnilではない: %d件", len(output.DraftPageRevisions))
		}
		if output.DraftPageRevisionTotalCount != 0 {
			t.Errorf("IncludeDraftRevisionsがfalseのときのDraftPageRevisionTotalCount = %d、期待値 = 0", output.DraftPageRevisionTotalCount)
		}
	})

	t.Run("IncludeDraftRevisionsがtrueでも下書きが無い場合はリビジョンを取得しない", func(t *testing.T) {
		// ページ1はトップレベルのfixtureで下書きなしで作成されている。
		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier:       "gpd-space",
			PageNumber:            1,
			UserID:                ownerID,
			IncludeDraftRevisions: true,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.DraftPage != nil {
			t.Fatal("前提条件を満たしていない: DraftPageがnilではない")
		}
		if output.DraftPageRevisions != nil {
			t.Errorf("下書きが無いのにDraftPageRevisionsがnilではない: %d件", len(output.DraftPageRevisions))
		}
		if output.DraftPageRevisionTotalCount != 0 {
			t.Errorf("下書きが無いときのDraftPageRevisionTotalCount = %d、期待値 = 0", output.DraftPageRevisionTotalCount)
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
				t.Fatalf("Create() (revision) のエラー = %v", err)
			}
		}

		output, err := uc.Execute(context.Background(), GetPageDetailInput{
			SpaceIdentifier:       "gpd-space",
			PageNumber:            31,
			UserID:                ownerID,
			IncludeDraftRevisions: true,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if len(output.DraftPageRevisions) != 2 {
			t.Fatalf("len(DraftPageRevisions) = %d、期待値 = 2", len(output.DraftPageRevisions))
		}
		if output.DraftPageRevisionTotalCount != 2 {
			t.Errorf("DraftPageRevisionTotalCount = %d、期待値 = 2", output.DraftPageRevisionTotalCount)
		}

		// 作成した2件のリビジョンが両方返ること。厳密な並び順 (新しい順) はRepositoryの
		// テストで担保されているため、ここではエントリの集合のみを検証する。
		for _, r := range output.DraftPageRevisions {
			if _, ok := wantTitles[r.Title]; !ok {
				t.Errorf("想定外のリビジョンのタイトル%q", r.Title)
				continue
			}
			wantTitles[r.Title] = true
		}
		for title, seen := range wantTitles {
			if !seen {
				t.Errorf("リビジョン%qが含まれていない", title)
			}
		}
	})
}
