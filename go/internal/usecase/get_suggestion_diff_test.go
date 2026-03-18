package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetSuggestionDiffUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	uc := NewGetSuggestionDiffUsecase(pageRevisionRepo)

	// テストデータのセットアップ
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sug-diff@example.com").
		WithAtname("sugdiffuser").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sug-diff-space").
		WithName("SugDiff Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("差分テストトピック").
		Build()

	// ページとリビジョンを作成
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		Build()
	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithPageID(pageID).
		WithTitle("元のタイトル").
		WithBody("元の本文\n2行目\n3行目").
		Build()

	// 編集提案を作成
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("差分テスト提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	// 編集提案ページを作成（ベースリビジョンを参照）
	suggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		WithTitle("変更後タイトル").
		WithBody("元の本文\n変更された2行目\n3行目\n追加行").
		Build()

	t.Run("ベースリビジョンが取得できる", func(t *testing.T) {
		suggestionPages := []*model.SuggestionPage{
			{
				ID:             suggestionPageID,
				SpaceID:        spaceID,
				SuggestionID:   suggestionID,
				PageID:         pageID,
				PageRevisionID: pageRevisionID,
				Title:          strPtr("変更後タイトル"),
				Body:           "元の本文\n変更された2行目\n3行目\n追加行",
			},
		}

		output, err := uc.Execute(context.Background(), GetSuggestionDiffInput{
			SpaceID:         spaceID,
			SuggestionPages: suggestionPages,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}

		baseRev, ok := output.BaseRevisions[suggestionPageID]
		if !ok {
			t.Fatal("BaseRevisions should contain the suggestion page ID")
		}
		if baseRev == nil {
			t.Fatal("base revision should not be nil")
		}
		if baseRev.Title != "元のタイトル" {
			t.Errorf("baseRev.Title = %q, want %q", baseRev.Title, "元のタイトル")
		}
		if baseRev.Body != "元の本文\n2行目\n3行目" {
			t.Errorf("baseRev.Body = %q, want %q", baseRev.Body, "元の本文\n2行目\n3行目")
		}
	})

	t.Run("空のSuggestionPagesの場合は空のマップが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionDiffInput{
			SpaceID:         spaceID,
			SuggestionPages: []*model.SuggestionPage{},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(output.BaseRevisions) != 0 {
			t.Errorf("len(BaseRevisions) = %d, want 0", len(output.BaseRevisions))
		}
	})
}

func strPtr(s string) *string {
	return &s
}
