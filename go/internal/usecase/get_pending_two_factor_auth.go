package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPendingTwoFactorAuthUsecaseは、認証アプリへの登録の途中の秘密鍵を画面に出す形で取得する。
//
// コードやパスワードの誤りで登録の画面を描き直すときに使う。秘密鍵を作り直すと、
// 認証アプリへ登録した秘密鍵が無駄になり、QRコードを読み取り直すことになるため。
type GetPendingTwoFactorAuthUsecase struct {
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
}

// NewGetPendingTwoFactorAuthUsecaseはGetPendingTwoFactorAuthUsecaseを生成する
func NewGetPendingTwoFactorAuthUsecase(userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository) *GetPendingTwoFactorAuthUsecase {
	return &GetPendingTwoFactorAuthUsecase{userTwoFactorAuthRepo: userTwoFactorAuthRepo}
}

// GetPendingTwoFactorAuthInputはGetPendingTwoFactorAuthUsecase.Executeの入力
type GetPendingTwoFactorAuthInput struct {
	UserID model.UserID
	// Emailは認証アプリに表示するアカウントの名前
	Email string
}

// GetPendingTwoFactorAuthOutputはGetPendingTwoFactorAuthUsecase.Executeの結果
type GetPendingTwoFactorAuthOutput struct {
	// Setupは登録の途中の秘密鍵。登録の途中の設定が無いとき (有効にしたときを含む) はnil
	Setup *TwoFactorAuthSetup
}

// Executeは登録の途中の秘密鍵を返す
func (uc *GetPendingTwoFactorAuthUsecase) Execute(ctx context.Context, input GetPendingTwoFactorAuthInput) (*GetPendingTwoFactorAuthOutput, error) {
	twoFactorAuth, err := uc.userTwoFactorAuthRepo.FindByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("二要素認証の設定の取得に失敗: %w", err)
	}
	if twoFactorAuth == nil || twoFactorAuth.Enabled {
		return &GetPendingTwoFactorAuthOutput{}, nil
	}

	setup, err := newTwoFactorAuthSetup(twoFactorAuth.Secret, input.Email)
	if err != nil {
		return nil, err
	}

	return &GetPendingTwoFactorAuthOutput{Setup: setup}, nil
}
