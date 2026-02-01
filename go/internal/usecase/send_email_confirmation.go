// Package usecase はアプリケーションのユースケース（ビジネスロジック）を提供します
package usecase

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/email"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/templates/emails"
)

// SendEmailConfirmationUsecase はメール確認コード送信ユースケース
type SendEmailConfirmationUsecase struct {
	cfg                   *config.Config
	emailConfirmationRepo *repository.EmailConfirmationRepository
	emailClient           EmailSender
}

// EmailSender はメール送信のインターフェース
type EmailSender interface {
	Send(ctx context.Context, input email.SendInput) error
}

// NewSendEmailConfirmationUsecase は SendEmailConfirmationUsecase を生成する
func NewSendEmailConfirmationUsecase(
	cfg *config.Config,
	emailConfirmationRepo *repository.EmailConfirmationRepository,
	emailClient EmailSender,
) *SendEmailConfirmationUsecase {
	return &SendEmailConfirmationUsecase{
		cfg:                   cfg,
		emailConfirmationRepo: emailConfirmationRepo,
		emailClient:           emailClient,
	}
}

// SendEmailConfirmationInput はメール確認コード送信の入力パラメータ
type SendEmailConfirmationInput struct {
	Email  string
	Event  model.EmailConfirmationEvent
	Locale string
}

// SendEmailConfirmationOutput はメール確認コード送信の出力パラメータ
type SendEmailConfirmationOutput struct {
	EmailConfirmationID string
}

// Execute はメール確認コードを生成して送信する
func (uc *SendEmailConfirmationUsecase) Execute(ctx context.Context, input SendEmailConfirmationInput) (*SendEmailConfirmationOutput, error) {
	// 確認コードを生成
	code, err := generateConfirmationCode()
	if err != nil {
		return nil, fmt.Errorf("確認コードの生成に失敗しました: %w", err)
	}

	// メール確認情報をDBに保存
	now := time.Now()
	confirmation, err := uc.emailConfirmationRepo.Create(ctx, repository.CreateEmailConfirmationInput{
		Email:     input.Email,
		Event:     input.Event,
		Code:      code,
		StartedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("メール確認情報の作成に失敗しました: %w", err)
	}

	// メールを送信
	if err := uc.sendConfirmationEmail(ctx, input.Email, code, input.Locale); err != nil {
		return nil, fmt.Errorf("確認メールの送信に失敗しました: %w", err)
	}

	return &SendEmailConfirmationOutput{
		EmailConfirmationID: confirmation.ID,
	}, nil
}

// sendConfirmationEmail は確認メールを送信する
func (uc *SendEmailConfirmationUsecase) sendConfirmationEmail(ctx context.Context, toEmail, code, locale string) error {
	isJapanese := locale == "ja"

	// メール本文をレンダリング
	data := emails.EmailConfirmationData{
		Email:      toEmail,
		Code:       code,
		AppURL:     uc.cfg.AppURL(),
		IsJapanese: isJapanese,
	}

	var buf bytes.Buffer
	if err := emails.EmailConfirmation(data).Render(ctx, &buf); err != nil {
		return fmt.Errorf("メールテンプレートのレンダリングに失敗しました: %w", err)
	}

	// メール件名を取得
	subject := i18n.T(ctx, "email_confirmation_subject")

	// メールを送信
	return uc.emailClient.Send(ctx, email.SendInput{
		To:      toEmail,
		Subject: subject,
		HTML:    buf.String(),
	})
}

// generateConfirmationCode は6文字のランダムな大文字英数字を生成する
func generateConfirmationCode() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 6)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[n.Int64()]
	}
	return string(code), nil
}
