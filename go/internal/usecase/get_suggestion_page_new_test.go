package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetSuggestionPageNewUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	uc := NewGetSuggestionPageNewUsecase(spaceRepo, spaceMemberRepo, topicMemberRepo, suggestionRepo, topicRepo, draftPageRepo)

	// テストデータのセットアップ
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sug-page-new@example.com").
		WithAtname("sugpagenew").
		Build()
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("sug-page-new-non@example.com").
		WithAtname("sugpagenewnon").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sug-page-new-sp").
		WithName("SugPageNew Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("テストトピック").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("ページ追加テスト提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	// 下書きページ（編集提案に未リンク）
	pageID1 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("未リンクページ").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID1).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithBody("未リンク下書き").
		Build()

	// 下書きページ（編集提案にリンク済み）
	pageID2 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("リンク済みページ").
		Build()
	pageRevisionID2 := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID2).
		WithSpaceMemberID(spaceMemberID).
		Build()
	suggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID2).
		WithPageRevisionID(pageRevisionID2).
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID2).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithBody("リンク済み下書き").
		WithSuggestionPageID(suggestionPageID).
		Build()

	t.Run("正常系: スペースメンバーがページ追加画面のデータを取得できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionPageNewInput{
			SpaceIdentifier:  "sug-page-new-sp",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Space == nil {
			t.Fatal("Space should not be nil")
		}
		if output.Topic == nil {
			t.Fatal("Topic should not be nil")
		}
		if output.Suggestion == nil {
			t.Fatal("Suggestion should not be nil")
		}
		if output.DraftPages == nil {
			t.Fatal("DraftPages should not be nil")
		}
	})

	t.Run("正常系: 編集提案にリンク済みの下書きはDraftPagesに含まれない", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionPageNewInput{
			SpaceIdentifier:  "sug-page-new-sp",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		for _, dp := range output.DraftPages {
			if dp.SuggestionPageID != nil {
				t.Error("DraftPages should not contain drafts linked to a suggestion page")
			}
		}
	})

	t.Run("異常系: 存在しないスペースでAppErrCodeResourceNotFoundが返る", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), GetSuggestionPageNewInput{
			SpaceIdentifier:  "nonexistent-space",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		if err == nil {
			t.Fatal("expected error but got nil")
		}
		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError but got %T: %v", err, err)
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("error code = %d, want %d", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("異常系: スペースメンバーでないユーザーはAppErrCodeForbiddenが返る", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), GetSuggestionPageNewInput{
			SpaceIdentifier:  "sug-page-new-sp",
			SuggestionNumber: 1,
			UserID:           nonMemberID,
		})
		if err == nil {
			t.Fatal("expected error but got nil")
		}
		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError but got %T: %v", err, err)
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("error code = %d, want %d", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("異常系: クローズ済み提案はAppErrCodeForbiddenが返る", func(t *testing.T) {
		testutil.NewSuggestionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("クローズ済み提案").
			WithStatus(model.SuggestionStatusClosed).
			Build()

		_, err := uc.Execute(context.Background(), GetSuggestionPageNewInput{
			SpaceIdentifier:  "sug-page-new-sp",
			SuggestionNumber: 2,
			UserID:           userID,
		})
		if err == nil {
			t.Fatal("expected error but got nil")
		}
		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError but got %T: %v", err, err)
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("error code = %d, want %d", ae.Code, model.AppErrCodeForbidden)
		}
	})
}
