package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CloseSuggestionUsecase は編集提案クローズユースケース
type CloseSuggestionUsecase struct {
	db             *sql.DB
	suggestionRepo *repository.SuggestionRepository
}

// NewCloseSuggestionUsecase は CloseSuggestionUsecase を生成する
func NewCloseSuggestionUsecase(
	db *sql.DB,
	suggestionRepo *repository.SuggestionRepository,
) *CloseSuggestionUsecase {
	return &CloseSuggestionUsecase{
		db:             db,
		suggestionRepo: suggestionRepo,
	}
}

// CloseSuggestionInput は編集提案クローズの入力パラメータ
type CloseSuggestionInput struct {
	Suggestion *model.Suggestion
}

// CloseSuggestionOutput は編集提案クローズの出力パラメータ
type CloseSuggestionOutput struct {
	Suggestion *model.Suggestion
}

// Execute は編集提案をクローズする
func (uc *CloseSuggestionUsecase) Execute(ctx context.Context, input CloseSuggestionInput) (*CloseSuggestionOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionRepo := uc.suggestionRepo.WithTx(tx)

	updatedSuggestion, err := suggestionRepo.UpdateStatus(ctx, repository.UpdateStatusInput{
		ID:      input.Suggestion.ID,
		SpaceID: input.Suggestion.SpaceID,
		Status:  model.SuggestionStatusClosed,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案のステータス更新に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &CloseSuggestionOutput{
		Suggestion: updatedSuggestion,
	}, nil
}
