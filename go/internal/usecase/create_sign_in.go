// Package usecase はアプリケーションのユースケース（ビジネスロジック）を提供します
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateSignInUsecase はサインインユースケース
type CreateSignInUsecase struct {
	signInValidator *validator.SignInCreateValidator
	userSessionRepo *repository.UserSessionRepository
}

// NewCreateSignInUsecase は CreateSignInUsecase を生成する
func NewCreateSignInUsecase(
	signInValidator *validator.SignInCreateValidator,
	userSessionRepo *repository.UserSessionRepository,
) *CreateSignInUsecase {
	return &CreateSignInUsecase{
		signInValidator: signInValidator,
		userSessionRepo: userSessionRepo,
	}
}

// CreateSignInInput はサインインの入力パラメータ
type CreateSignInInput struct {
	Email     string
	Password  string
	IPAddress string
	UserAgent string
}

// CreateSignInOutput はサインインの出力パラメータ
type CreateSignInOutput struct {
	Token             string
	TwoFactorRequired bool
	UserID            model.UserID
}

// Execute はサインイン処理を実行する
func (uc *CreateSignInUsecase) Execute(ctx context.Context, input CreateSignInInput) (*CreateSignInOutput, error) {
	// 1. バリデーション
	validateOutput, err := uc.signInValidator.Validate(ctx, validator.SignInCreateValidatorInput{
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		return nil, err
	}

	// 2. 二要素認証が有効な場合はセッションを作成せずに返す
	if validateOutput.UserTwoFactorAuth != nil {
		return &CreateSignInOutput{
			TwoFactorRequired: true,
			UserID:            validateOutput.User.ID,
		}, nil
	}

	// 3. セッション作成
	token, err := auth.GenerateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("セッショントークンの生成に失敗しました: %w", err)
	}

	now := time.Now()
	_, err = uc.userSessionRepo.Create(ctx, repository.CreateInput{
		UserID:     validateOutput.User.ID,
		Token:      token,
		IPAddress:  input.IPAddress,
		UserAgent:  input.UserAgent,
		SignedInAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("セッションの作成に失敗しました: %w", err)
	}

	return &CreateSignInOutput{
		Token:  token,
		UserID: validateOutput.User.ID,
	}, nil
}
