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
	SuggestionID model.SuggestionID
	SpaceID      model.SpaceID
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

	suggestion, err := suggestionRepo.FindByID(ctx, input.SuggestionID, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("編集提案の取得に失敗しました: %w", err)
	}
	if suggestion == nil {
		return nil, fmt.Errorf("編集提案が見つかりません: %s", input.SuggestionID)
	}

	if suggestion.Status != model.SuggestionStatusOpen {
		return nil, fmt.Errorf("オープンステータスの編集提案のみクローズできます（現在のステータス: %d）", suggestion.Status)
	}

	updatedSuggestion, err := suggestionRepo.UpdateStatus(ctx, repository.UpdateStatusInput{
		ID:      suggestion.ID,
		SpaceID: input.SpaceID,
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
