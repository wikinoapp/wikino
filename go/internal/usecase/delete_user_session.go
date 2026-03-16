package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/repository"
)

// DeleteUserSessionUsecase はユーザーセッション削除ユースケース
type DeleteUserSessionUsecase struct {
	userSessionRepo *repository.UserSessionRepository
}

// NewDeleteUserSessionUsecase は DeleteUserSessionUsecase を生成する
func NewDeleteUserSessionUsecase(
	userSessionRepo *repository.UserSessionRepository,
) *DeleteUserSessionUsecase {
	return &DeleteUserSessionUsecase{
		userSessionRepo: userSessionRepo,
	}
}

// DeleteUserSessionInput はセッション削除の入力パラメータ
type DeleteUserSessionInput struct {
	Token string
}

// Execute はユーザーセッションをトークンで削除する
func (uc *DeleteUserSessionUsecase) Execute(ctx context.Context, input DeleteUserSessionInput) error {
	if err := uc.userSessionRepo.DeleteByToken(ctx, input.Token); err != nil {
		return fmt.Errorf("セッションの削除に失敗しました: %w", err)
	}

	return nil
}
