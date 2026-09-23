package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetAccountNewDataUsecaseはアカウント作成フォーム表示用のデータ取得ユースケース
type GetAccountNewDataUsecase struct {
	emailConfirmationRepo *repository.EmailConfirmationRepository
}

// NewGetAccountNewDataUsecaseはGetAccountNewDataUsecaseを生成する
func NewGetAccountNewDataUsecase(
	emailConfirmationRepo *repository.EmailConfirmationRepository,
) *GetAccountNewDataUsecase {
	return &GetAccountNewDataUsecase{
		emailConfirmationRepo: emailConfirmationRepo,
	}
}

// GetAccountNewDataInputはアカウント作成フォームデータ取得の入力パラメータ
type GetAccountNewDataInput struct {
	EmailConfirmationID string
}

// GetAccountNewDataOutputはアカウント作成フォームデータ取得の出力
type GetAccountNewDataOutput struct {
	EmailConfirmation *model.EmailConfirmation
}

// Executeはメール確認情報を取得する
func (uc *GetAccountNewDataUsecase) Execute(ctx context.Context, input GetAccountNewDataInput) (*GetAccountNewDataOutput, error) {
	emailConfirmation, err := uc.emailConfirmationRepo.FindByID(ctx, input.EmailConfirmationID)
	if err != nil {
		return nil, fmt.Errorf("メール確認情報の取得に失敗しました: %w", err)
	}

	if emailConfirmation == nil {
		return nil, nil
	}

	return &GetAccountNewDataOutput{
		EmailConfirmation: emailConfirmation,
	}, nil
}
