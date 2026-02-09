package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/repository"
)

// MarkEmailAsConfirmedUsecase はメール確認を完了状態に更新するユースケース
type MarkEmailAsConfirmedUsecase struct {
	emailConfirmationRepo *repository.EmailConfirmationRepository
}

// NewMarkEmailAsConfirmedUsecase は MarkEmailAsConfirmedUsecase を生成する
func NewMarkEmailAsConfirmedUsecase(
	emailConfirmationRepo *repository.EmailConfirmationRepository,
) *MarkEmailAsConfirmedUsecase {
	return &MarkEmailAsConfirmedUsecase{
		emailConfirmationRepo: emailConfirmationRepo,
	}
}

// Execute はメール確認を完了状態に更新する
func (uc *MarkEmailAsConfirmedUsecase) Execute(ctx context.Context, emailConfirmationID string) error {
	return uc.emailConfirmationRepo.Succeed(ctx, emailConfirmationID)
}
