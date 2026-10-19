package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CleanupEmailConfirmationsUsecaseは古いメール確認情報を削除するユースケース
type CleanupEmailConfirmationsUsecase struct {
	emailConfirmationRepo *repository.EmailConfirmationRepository
}

// NewCleanupEmailConfirmationsUsecaseはCleanupEmailConfirmationsUsecaseを生成する
func NewCleanupEmailConfirmationsUsecase(emailConfirmationRepo *repository.EmailConfirmationRepository) *CleanupEmailConfirmationsUsecase {
	return &CleanupEmailConfirmationsUsecase{
		emailConfirmationRepo: emailConfirmationRepo,
	}
}

// CleanupEmailConfirmationsInputは古いメール確認情報の削除の入力パラメータ
type CleanupEmailConfirmationsInput struct {
	RetentionHours int
}

// Executeは、開始から保持期間を過ぎたメール確認情報を削除する。確認コードの有効期限 (15分) を
// 過ぎた行はGo版・Rails版のどちらからも参照されないため、確認の成否は問わない。
//
// 保持期間が0以下のときは24時間とみなす。引数を省いて投入したジョブで、確認の途中の行まで
// 消さないためである。
func (uc *CleanupEmailConfirmationsUsecase) Execute(ctx context.Context, input CleanupEmailConfirmationsInput) error {
	retention := time.Duration(input.RetentionHours) * time.Hour
	if retention <= 0 {
		retention = 24 * time.Hour
	}

	deletedCount, err := uc.emailConfirmationRepo.DeleteStartedBefore(ctx, time.Now().UTC().Add(-retention))
	if err != nil {
		return fmt.Errorf("古いメール確認情報の削除に失敗: %w", err)
	}

	slog.InfoContext(ctx, "古いメール確認情報を削除しました",
		"retention_hours", retention.Hours(),
		"deleted_count", deletedCount,
	)
	return nil
}
