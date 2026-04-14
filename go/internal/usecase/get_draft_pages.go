package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetDraftPagesUsecase は下書きページ一覧取得ユースケース
type GetDraftPagesUsecase struct {
	draftPageRepo *repository.DraftPageRepository
}

// NewGetDraftPagesUsecase は GetDraftPagesUsecase を生成する
func NewGetDraftPagesUsecase(
	draftPageRepo *repository.DraftPageRepository,
) *GetDraftPagesUsecase {
	return &GetDraftPagesUsecase{
		draftPageRepo: draftPageRepo,
	}
}

// GetDraftPagesInput は下書きページ一覧取得の入力パラメータ
type GetDraftPagesInput struct {
	UserID model.UserID
}

// GetDraftPagesOutput は下書きページ一覧取得の出力
type GetDraftPagesOutput struct {
	DraftPages []*model.DraftPage
}

// Execute は下書きページ一覧を取得する
func (uc *GetDraftPagesUsecase) Execute(ctx context.Context, input GetDraftPagesInput) (*GetDraftPagesOutput, error) {
	drafts, err := uc.draftPageRepo.ListByUserForIndex(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("下書き一覧の取得に失敗: %w", err)
	}

	return &GetDraftPagesOutput{
		DraftPages: drafts,
	}, nil
}
