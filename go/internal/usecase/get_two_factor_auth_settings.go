package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetTwoFactorAuthSettingsUsecaseは、二要素認証の設定の画面に出す今の状態を取得する
type GetTwoFactorAuthSettingsUsecase struct {
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
}

// NewGetTwoFactorAuthSettingsUsecaseはGetTwoFactorAuthSettingsUsecaseを生成する
func NewGetTwoFactorAuthSettingsUsecase(userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository) *GetTwoFactorAuthSettingsUsecase {
	return &GetTwoFactorAuthSettingsUsecase{userTwoFactorAuthRepo: userTwoFactorAuthRepo}
}

// GetTwoFactorAuthSettingsInputはGetTwoFactorAuthSettingsUsecase.Executeの入力
type GetTwoFactorAuthSettingsInput struct {
	UserID model.UserID
}

// GetTwoFactorAuthSettingsOutputは二要素認証の今の状態。秘密鍵とリカバリーコードは含めない
type GetTwoFactorAuthSettingsOutput struct {
	Enabled   bool
	EnabledAt *time.Time
}

// Executeは二要素認証が有効かと、有効にした日時を返す。登録の途中の設定は無効として扱う
func (uc *GetTwoFactorAuthSettingsUsecase) Execute(ctx context.Context, input GetTwoFactorAuthSettingsInput) (*GetTwoFactorAuthSettingsOutput, error) {
	twoFactorAuth, err := uc.userTwoFactorAuthRepo.FindByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("二要素認証の設定の取得に失敗: %w", err)
	}
	if twoFactorAuth == nil || !twoFactorAuth.Enabled {
		return &GetTwoFactorAuthSettingsOutput{}, nil
	}

	return &GetTwoFactorAuthSettingsOutput{Enabled: true, EnabledAt: twoFactorAuth.EnabledAt}, nil
}
