package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// MarkEmailAsConfirmedUsecase はメール確認を完了状態に更新するユースケース
type MarkEmailAsConfirmedUsecase struct {
	emailConfirmationRepo *repository.EmailConfirmationRepository
	updateValidator       *validator.EmailConfirmationUpdateValidator
}

// NewMarkEmailAsConfirmedUsecase は MarkEmailAsConfirmedUsecase を生成する
func NewMarkEmailAsConfirmedUsecase(
	emailConfirmationRepo *repository.EmailConfirmationRepository,
	updateValidator *validator.EmailConfirmationUpdateValidator,
) *MarkEmailAsConfirmedUsecase {
	return &MarkEmailAsConfirmedUsecase{
		emailConfirmationRepo: emailConfirmationRepo,
		updateValidator:       updateValidator,
	}
}

// MarkEmailAsConfirmedInput はメール確認完了の入力パラメータ
type MarkEmailAsConfirmedInput struct {
	EmailConfirmationID string
	Code                string
}

// Execute はメール確認コードを検証し、確認を完了状態に更新する
func (uc *MarkEmailAsConfirmedUsecase) Execute(ctx context.Context, input MarkEmailAsConfirmedInput) error {
	// 1. バリデーション
	_, err := uc.updateValidator.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: input.EmailConfirmationID,
		Code:                input.Code,
	})
	if err != nil {
		return err
	}

	// 2. 永続化
	if err := uc.emailConfirmationRepo.Succeed(ctx, input.EmailConfirmationID); err != nil {
		return fmt.Errorf("メール確認の完了状態更新に失敗しました: %w", err)
	}

	return nil
}
