// Package usecase はアプリケーションのユースケース（ビジネスロジック）を提供します
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// CreateUserSessionUsecase はユーザーセッション作成ユースケース
type CreateUserSessionUsecase struct {
	userSessionRepo *repository.UserSessionRepository
}

// NewCreateUserSessionUsecase は CreateUserSessionUsecase を生成する
func NewCreateUserSessionUsecase(
	userSessionRepo *repository.UserSessionRepository,
) *CreateUserSessionUsecase {
	return &CreateUserSessionUsecase{
		userSessionRepo: userSessionRepo,
	}
}

// CreateUserSessionInput はセッション作成の入力パラメータ
type CreateUserSessionInput struct {
	UserID    string
	IPAddress string
	UserAgent string
}

// CreateUserSessionOutput はセッション作成の出力パラメータ
type CreateUserSessionOutput struct {
	Token string
}

// Execute はユーザーセッションを作成する
func (uc *CreateUserSessionUsecase) Execute(ctx context.Context, input CreateUserSessionInput) (*CreateUserSessionOutput, error) {
	// セッショントークンを生成
	token, err := session.GenerateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("セッショントークンの生成に失敗しました: %w", err)
	}

	// セッションをDBに保存
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
