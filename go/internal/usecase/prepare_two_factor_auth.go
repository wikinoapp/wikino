package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// PrepareTwoFactorAuthUsecaseは、認証アプリへ登録する新しい秘密鍵を作り、登録の途中の設定として保存する。
//
// 登録の画面を開くたびに秘密鍵を作り直す。前に表示した秘密鍵が画面の外に残っていても、
// 有効にする前に使えなくなる。
type PrepareTwoFactorAuthUsecase struct {
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
}

// NewPrepareTwoFactorAuthUsecaseはPrepareTwoFactorAuthUsecaseを生成する
func NewPrepareTwoFactorAuthUsecase(userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository) *PrepareTwoFactorAuthUsecase {
	return &PrepareTwoFactorAuthUsecase{userTwoFactorAuthRepo: userTwoFactorAuthRepo}
}

// PrepareTwoFactorAuthInputはPrepareTwoFactorAuthUsecase.Executeの入力
type PrepareTwoFactorAuthInput struct {
	UserID model.UserID
	// Emailは認証アプリに表示するアカウントの名前
	Email string
}

// PrepareTwoFactorAuthOutputはPrepareTwoFactorAuthUsecase.Executeの結果
type PrepareTwoFactorAuthOutput struct {
	// Setupは画面に出す秘密鍵。既に有効にしているときはnil
	Setup *TwoFactorAuthSetup
}

// Executeは新しい秘密鍵を保存し、画面に出す形で返す。
// 既に有効にしているときは、登録済みの認証アプリのコードが通らなくならないよう、秘密鍵を差し替えない。
func (uc *PrepareTwoFactorAuthUsecase) Execute(ctx context.Context, input PrepareTwoFactorAuthInput) (*PrepareTwoFactorAuthOutput, error) {
	secret, err := auth.GenerateTOTPSecret()
	if err != nil {
		return nil, fmt.Errorf("TOTPの秘密鍵の生成に失敗: %w", err)
	}
	setup, err := newTwoFactorAuthSetup(secret, input.Email)
	if err != nil {
		return nil, err
	}

	pending, err := uc.userTwoFactorAuthRepo.UpsertPending(ctx, input.UserID, secret)
	if err != nil {
		return nil, fmt.Errorf("登録の途中の二要素認証の設定の保存に失敗: %w", err)
	}
	if pending == nil {
		return &PrepareTwoFactorAuthOutput{}, nil
	}

	return &PrepareTwoFactorAuthOutput{Setup: setup}, nil
}
