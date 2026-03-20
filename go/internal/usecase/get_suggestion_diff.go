package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSuggestionDiffUsecase は編集提案の差分取得ユースケース
type GetSuggestionDiffUsecase struct {
	pageRevisionRepo *repository.PageRevisionRepository
}

// NewGetSuggestionDiffUsecase は GetSuggestionDiffUsecase を生成する
func NewGetSuggestionDiffUsecase(
	pageRevisionRepo *repository.PageRevisionRepository,
) *GetSuggestionDiffUsecase {
	return &GetSuggestionDiffUsecase{
		pageRevisionRepo: pageRevisionRepo,
	}
}

// GetSuggestionDiffInput は編集提案の差分取得の入力パラメータ
type GetSuggestionDiffInput struct {
	SpaceID         model.SpaceID
	SuggestionPages []*model.SuggestionPage
}

// GetSuggestionDiffOutput は編集提案の差分取得の出力
type GetSuggestionDiffOutput struct {
	// BaseRevisions は SuggestionPageID をキーとして、ベースとなるページリビジョンをマップで返す
	BaseRevisions map[model.SuggestionPageID]*model.PageRevision
}

// Execute は各SuggestionPageのベースリビジョンを取得する
func (uc *GetSuggestionDiffUsecase) Execute(ctx context.Context, input GetSuggestionDiffInput) (*GetSuggestionDiffOutput, error) {
	baseRevisions := make(map[model.SuggestionPageID]*model.PageRevision, len(input.SuggestionPages))

	for _, sp := range input.SuggestionPages {
		rev, err := uc.pageRevisionRepo.FindByID(ctx, sp.PageRevisionID, input.SpaceID)
		if err != nil {
			return nil, fmt.Errorf("ベースリビジョンの取得に失敗: %w", err)
		}
		if rev == nil {
			return nil, fmt.Errorf("ベースリビジョンが見つかりません: pageRevisionID=%s, spaceID=%s", sp.PageRevisionID, input.SpaceID)
		}
		baseRevisions[sp.ID] = rev
	}

	return &GetSuggestionDiffOutput{
		BaseRevisions: baseRevisions,
	}, nil
}
