package usecase

import (
	"context"
	"fmt"
	"log/slog"
)

// PasswordResetSender はパスワードリセットメールの送信を行うインターフェース
type PasswordResetSender interface {
	Send(ctx context.Context, to, resetURL, appURL, locale string) error
}

// SendPasswordResetUsecase はパスワードリセットメール送信ユースケース
type SendPasswordResetUsecase struct {
	sender PasswordResetSender
}

// NewSendPasswordResetUsecase は SendPasswordResetUsecase を生成する
func NewSendPasswordResetUsecase(sender PasswordResetSender) *SendPasswordResetUsecase {
	return &SendPasswordResetUsecase{
		sender: sender,
	}
}

// SendPasswordResetInput はパスワードリセットメール送信の入力パラメータ
type SendPasswordResetInput struct {
	Email    string
	ResetURL string
	AppURL   string
	Locale   string
}

// Execute はパスワードリセットメールを送信する
func (uc *SendPasswordResetUsecase) Execute(ctx context.Context, input SendPasswordResetInput) error {
	if input.Email == "" {
		return fmt.Errorf("メールアドレスが空です")
	}

	err := uc.sender.Send(ctx, input.Email, input.ResetURL, input.AppURL, input.Locale)
	if err != nil {
		slog.ErrorContext(ctx, "パスワードリセットメール送信に失敗しました",
			"email", input.Email,
			"error", err,
		)
		return fmt.Errorf("メール送信に失敗: %w", err)
	}

	slog.InfoContext(ctx, "パスワードリセットメールを送信しました",
		"email", input.Email,
	)

	return nil
}
