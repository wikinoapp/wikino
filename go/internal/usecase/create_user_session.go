// Package usecaseはアプリケーションのユースケース (ビジネスロジック) を提供します
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CreateUserSessionUsecaseはユーザーセッション作成ユースケース
type CreateUserSessionUsecase struct {
	userSessionRepo *repository.UserSessionRepository
}

// NewCreateUserSessionUsecaseはCreateUserSessionUsecaseを生成する
func NewCreateUserSessionUsecase(
	userSessionRepo *repository.UserSessionRepository,
) *CreateUserSessionUsecase {
	return &CreateUserSessionUsecase{
		userSessionRepo: userSessionRepo,
	}
}

// CreateUserSessionInputはセッション作成の入力パラメータ
type CreateUserSessionInput struct {
	UserID    model.UserID
	IPAddress string
	UserAgent string
}

// CreateUserSessionOutputはセッション作成の出力パラメータ
type CreateUserSessionOutput struct {
	Token string
}

// Executeはユーザーセッションを作成する
func (uc *CreateUserSessionUsecase) Execute(ctx context.Context, input CreateUserSessionInput) (*CreateUserSessionOutput, error) {
	token, err := auth.GenerateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("セッショントークンの生成に失敗しました: %w", err)
	}

	now := time.Now()
	_, err = uc.userSessionRepo.Create(ctx, repository.CreateInput{
		UserID:     input.UserID,
		Token:      token,
		IPAddress:  input.IPAddress,
		UserAgent:  input.UserAgent,
		SignedInAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("セッションの作成に失敗しました: %w", err)
	}

	return &CreateUserSessionOutput{
		Token: token,
	}, nil
}
