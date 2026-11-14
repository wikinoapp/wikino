package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateEmailUpdateConfirmationUsecaseは、ログイン中のユーザーのメールアドレスの変更を始め、
// 新しいメールアドレスへ確認コードを送る
type CreateEmailUpdateConfirmationUsecase struct {
	updateValidator           *validator.SettingsEmailUpdateValidator
	createEmailConfirmationUC *CreateEmailConfirmationUsecase
}

// NewCreateEmailUpdateConfirmationUsecaseはCreateEmailUpdateConfirmationUsecaseを生成する
func NewCreateEmailUpdateConfirmationUsecase(
	updateValidator *validator.SettingsEmailUpdateValidator,
	createEmailConfirmationUC *CreateEmailConfirmationUsecase,
) *CreateEmailUpdateConfirmationUsecase {
	return &CreateEmailUpdateConfirmationUsecase{
		updateValidator:           updateValidator,
		createEmailConfirmationUC: createEmailConfirmationUC,
	}
}

// CreateEmailUpdateConfirmationInputはメールアドレスの変更を始める入力パラメータ
type CreateEmailUpdateConfirmationInput struct {
	UserID   model.UserID
	NewEmail string
	Password string
	Locale   string
}

// CreateEmailUpdateConfirmationOutputは作った確認のIDを保持する
type CreateEmailUpdateConfirmationOutput struct {
	EmailConfirmationID string
}

// Executeは現在のパスワードと新しいメールアドレスを確かめ、確認コードを作ってメールの送信を
// 投入する。確認コードの生成とメールの文面は新規登録と同じものを使う。
func (uc *CreateEmailUpdateConfirmationUsecase) Execute(ctx context.Context, input CreateEmailUpdateConfirmationInput) (*CreateEmailUpdateConfirmationOutput, error) {
	if err := uc.updateValidator.Validate(ctx, validator.SettingsEmailUpdateValidatorInput{
		UserID:   input.UserID,
		NewEmail: input.NewEmail,
		Password: input.Password,
	}); err != nil {
		return nil, err
	}

	output, err := uc.createEmailConfirmationUC.Execute(ctx, CreateEmailConfirmationInput{
		Email:  input.NewEmail,
		Event:  model.EmailConfirmationEventEmailUpdate,
		Locale: input.Locale,
	})
	if err != nil {
		return nil, err
	}

	return &CreateEmailUpdateConfirmationOutput{EmailConfirmationID: output.EmailConfirmationID}, nil
}
