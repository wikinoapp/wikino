package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetOAuthDeviceVerificationNewUsecaseは、デバイス認可のユーザーコードの入力画面を開けるかを確かめる。
type GetOAuthDeviceVerificationNewUsecase struct {
	featureFlagRepo *repository.FeatureFlagRepository
}

// NewGetOAuthDeviceVerificationNewUsecaseはGetOAuthDeviceVerificationNewUsecaseを生成する
func NewGetOAuthDeviceVerificationNewUsecase(featureFlagRepo *repository.FeatureFlagRepository) *GetOAuthDeviceVerificationNewUsecase {
	return &GetOAuthDeviceVerificationNewUsecase{featureFlagRepo: featureFlagRepo}
}

// GetOAuthDeviceVerificationNewInputはユーザーコードの入力画面の入力パラメータ
type GetOAuthDeviceVerificationNewInput struct {
	UserID model.UserID
}

// Executeは、フィーチャーフラグが無効なら*model.AppErrorを返す
func (uc *GetOAuthDeviceVerificationNewUsecase) Execute(ctx context.Context, input GetOAuthDeviceVerificationNewInput) error {
	return checkPublicAPIEnabled(ctx, uc.featureFlagRepo, input.UserID)
}
