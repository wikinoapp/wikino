package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetProfileShowUsecaseはプロフィールの画面に表示するデータを取得する
type GetProfileShowUsecase struct {
	userRepo  *repository.UserRepository
	spaceRepo *repository.SpaceRepository
}

// NewGetProfileShowUsecaseはGetProfileShowUsecaseを生成する
func NewGetProfileShowUsecase(
	userRepo *repository.UserRepository,
	spaceRepo *repository.SpaceRepository,
) *GetProfileShowUsecase {
	return &GetProfileShowUsecase{
		userRepo:  userRepo,
		spaceRepo: spaceRepo,
	}
}

// GetProfileShowInputはプロフィールの画面の入力パラメータ
type GetProfileShowInput struct {
	Atname string

	// ViewerUserIDはプロフィールを開いた閲覧者のユーザーID。ログインしていなければnil。
	ViewerUserID *model.UserID
}

// GetProfileShowOutputはプロフィールの画面に表示するデータ
type GetProfileShowOutput struct {
	User         *model.User
	JoinedSpaces []*model.Space

	// CanEditProfileは、閲覧者が本人で、プロフィールの編集と設定へのリンクを出すかを表す
	CanEditProfile bool
}

// Executeはアットネームで削除されていないユーザーと、そのユーザーが参加しているスペースを取得する。
// ユーザーが見つからない場合は (nil, nil) を返す。
func (uc *GetProfileShowUsecase) Execute(ctx context.Context, input GetProfileShowInput) (*GetProfileShowOutput, error) {
	user, err := uc.userRepo.FindByAtname(ctx, input.Atname)
	if err != nil {
		return nil, fmt.Errorf("ユーザーの取得に失敗: %w", err)
	}
	if user == nil {
		return nil, nil
	}

	spaces, err := uc.spaceRepo.ListActiveByUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("参加中スペース一覧の取得に失敗: %w", err)
	}

	return &GetProfileShowOutput{
		User:           user,
		JoinedSpaces:   spaces,
		CanEditProfile: input.ViewerUserID != nil && *input.ViewerUserID == user.ID,
	}, nil
}
