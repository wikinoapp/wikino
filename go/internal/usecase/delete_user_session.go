package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/repository"
)

// DeleteUserSessionUsecaseはユーザーセッション削除ユースケース
type DeleteUserSessionUsecase struct {
	userSessionRepo *repository.UserSessionRepository
}

// NewDeleteUserSessionUsecaseはDeleteUserSessionUsecaseを生成する
func NewDeleteUserSessionUsecase(
	userSessionRepo *repository.UserSessionRepository,
) *DeleteUserSessionUsecase {
	return &DeleteUserSessionUsecase{
		userSessionRepo: userSessionRepo,
	}
}

// DeleteUserSessionInputはセッション削除の入力パラメータ
type DeleteUserSessionInput struct {
	Token string
}

// Executeはユーザーセッションをトークンで削除する
func (uc *DeleteUserSessionUsecase) Execute(ctx context.Context, input DeleteUserSessionInput) error {
	if err := uc.userSessionRepo.DeleteByToken(ctx, input.Token); err != nil {
		return fmt.Errorf("セッションの削除に失敗しました: %w", err)
	}

	return nil
}
