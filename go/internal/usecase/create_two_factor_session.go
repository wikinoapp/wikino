package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateTwoFactorSessionUsecase は2FA TOTP認証によるセッション作成ユースケース
type CreateTwoFactorSessionUsecase struct {
	twoFactorValidator *validator.SignInTwoFactorCreateValidator
	createSessionUC    *CreateUserSessionUsecase
}

// NewCreateTwoFactorSessionUsecase は CreateTwoFactorSessionUsecase を生成する
func NewCreateTwoFactorSessionUsecase(
	twoFactorValidator *validator.SignInTwoFactorCreateValidator,
	createSessionUC *CreateUserSessionUsecase,
) *CreateTwoFactorSessionUsecase {
	return &CreateTwoFactorSessionUsecase{
		twoFactorValidator: twoFactorValidator,
		createSessionUC:    createSessionUC,
	}
}

// CreateTwoFactorSessionInput は2FAセッション作成の入力パラメータ
type CreateTwoFactorSessionInput struct {
	UserID    model.UserID
	TOTPCode  string
	IPAddress string
	UserAgent string
}

// CreateTwoFactorSessionOutput は2FAセッション作成の出力パラメータ
type CreateTwoFactorSessionOutput struct {
	Token string
}

// Execute は2FA TOTPコードを検証してセッションを作成する
func (uc *CreateTwoFactorSessionUsecase) Execute(ctx context.Context, input CreateTwoFactorSessionInput) (*CreateTwoFactorSessionOutput, error) {
	// 1. バリデーション（形式チェック + TOTP検証）
	if err := uc.validate(ctx, input); err != nil {
		return nil, err
	}

	// 2. セッション作成
	sessionOutput, err := uc.createSession(ctx, input)
	if err != nil {
		return nil, err
	}

	return &CreateTwoFactorSessionOutput{
		Token: sessionOutput.Token,
	}, nil
}

func (uc *CreateTwoFactorSessionUsecase) validate(ctx context.Context, input CreateTwoFactorSessionInput) error {
	return uc.twoFactorValidator.Validate(ctx, validator.SignInTwoFactorCreateValidatorInput{
		UserID:   input.UserID,
		TOTPCode: input.TOTPCode,
	})
}

func (uc *CreateTwoFactorSessionUsecase) createSession(ctx context.Context, input CreateTwoFactorSessionInput) (*CreateUserSessionOutput, error) {
	output, err := uc.createSessionUC.Execute(ctx, CreateUserSessionInput{
		UserID:    input.UserID,
		IPAddress: input.IPAddress,
		UserAgent: input.UserAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("セッションの作成に失敗しました: %w", err)
	}
	return output, nil
}
