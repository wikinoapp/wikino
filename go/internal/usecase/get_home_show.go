package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetHomeShowUsecase はホーム画面表示用ユースケース
type GetHomeShowUsecase struct {
	spaceRepo *repository.SpaceRepository
}

// NewGetHomeShowUsecase は GetHomeShowUsecase を生成する
func NewGetHomeShowUsecase(
	spaceRepo *repository.SpaceRepository,
) *GetHomeShowUsecase {
	return &GetHomeShowUsecase{
		spaceRepo: spaceRepo,
	}
}

// GetHomeShowInput はホーム画面表示の入力パラメータ
type GetHomeShowInput struct {
	UserID model.UserID
}

// GetHomeShowOutput はホーム画面表示の出力
type GetHomeShowOutput struct {
	ActiveSpaces []*model.Space
}

// Execute はホーム画面に表示する参加中スペース一覧を取得する
func (uc *GetHomeShowUsecase) Execute(ctx context.Context, input GetHomeShowInput) (*GetHomeShowOutput, error) {
	spaces, err := uc.spaceRepo.ListActiveByUser(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("参加中スペース一覧の取得に失敗: %w", err)
	}

	return &GetHomeShowOutput{
		ActiveSpaces: spaces,
	}, nil
}
