package usecase

import (
	"context"
	"errors"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/password_reset"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// パスワードリセットトークン検証のエラー定義
var (
	ErrPasswordResetTokenNotFound = errors.New("トークンが見つかりません")
	ErrPasswordResetTokenUsed     = errors.New("トークンは使用済みです")
	ErrPasswordResetTokenExpired  = errors.New("トークンの有効期限が切れています")
)

// GetPasswordResetTokenDataUsecase はパスワードリセットトークンの検証ユースケース
type GetPasswordResetTokenDataUsecase struct {
	passwordResetTokenRepo *repository.PasswordResetTokenRepository
}

// NewGetPasswordResetTokenDataUsecase は GetPasswordResetTokenDataUsecase を生成する
func NewGetPasswordResetTokenDataUsecase(
	passwordResetTokenRepo *repository.PasswordResetTokenRepository,
) *GetPasswordResetTokenDataUsecase {
	return &GetPasswordResetTokenDataUsecase{
		passwordResetTokenRepo: passwordResetTokenRepo,
	}
}

// GetPasswordResetTokenDataInput はパスワードリセットトークン検証の入力パラメータ
type GetPasswordResetTokenDataInput struct {
	Token string
}

// GetPasswordResetTokenDataOutput はパスワードリセットトークン検証の出力
type GetPasswordResetTokenDataOutput struct {
	PasswordResetToken *model.PasswordResetToken
}

// Execute はパスワードリセットトークンを検証する
func (uc *GetPasswordResetTokenDataUsecase) Execute(ctx context.Context, input GetPasswordResetTokenDataInput) (*GetPasswordResetTokenDataOutput, error) {
	tokenDigest := password_reset.HashToken(input.Token)
	token, err := uc.passwordResetTokenRepo.FindByTokenDigest(ctx, tokenDigest)
	if err != nil {
		return nil, err
	}

	if token == nil {
		return nil, ErrPasswordResetTokenNotFound
	}

	if token.IsUsed() {
		return nil, ErrPasswordResetTokenUsed
	}

	if token.IsExpired() {
		return nil, ErrPasswordResetTokenExpired
	}

	return &GetPasswordResetTokenDataOutput{
		PasswordResetToken: token,
	}, nil
}
