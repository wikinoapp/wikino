package usecase

import (
	"context"
	"fmt"
	"log/slog"
)

// EmailConfirmationSender はメール確認コードの送信を行うインターフェース
type EmailConfirmationSender interface {
	Send(ctx context.Context, to, code, appURL, locale string) error
}

// SendEmailConfirmationUsecase はメール確認コードのメール送信ユースケース
type SendEmailConfirmationUsecase struct {
	sender EmailConfirmationSender
}

// NewSendEmailConfirmationUsecase は SendEmailConfirmationUsecase を生成する
func NewSendEmailConfirmationUsecase(sender EmailConfirmationSender) *SendEmailConfirmationUsecase {
	return &SendEmailConfirmationUsecase{
		sender: sender,
	}
}

// SendEmailConfirmationInput はメール確認コード送信の入力パラメータ
type SendEmailConfirmationInput struct {
	Email  string
	Code   string
	AppURL string
	Locale string
}

// Execute はメール確認コードのメールを送信する
func (uc *SendEmailConfirmationUsecase) Execute(ctx context.Context, input SendEmailConfirmationInput) error {
	if input.Email == "" {
		return fmt.Errorf("メールアドレスが空です")
	}

	err := uc.sender.Send(ctx, input.Email, input.Code, input.AppURL, input.Locale)
	if err != nil {
		slog.ErrorContext(ctx, "メール送信に失敗しました",
			"email", input.Email,
			"error", err,
		)
		return fmt.Errorf("メール送信に失敗: %w", err)
	}

	slog.InfoContext(ctx, "メール確認コードを送信しました",
		"email", input.Email,
	)

	return nil
}
