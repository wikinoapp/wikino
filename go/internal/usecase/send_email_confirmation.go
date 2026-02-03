// Package usecase はアプリケーションのユースケース（ビジネスロジック）を提供します
package usecase

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/templates/emails/email_confirmation"
	"github.com/wikinoapp/wikino/go/internal/worker"
)

// JobInserter はジョブをキューに追加するインターフェース
type JobInserter interface {
	Insert(ctx context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error)
}

// SendEmailConfirmationUsecase はメール確認コード送信ユースケース
type SendEmailConfirmationUsecase struct {
	cfg                   *config.Config
	emailConfirmationRepo *repository.EmailConfirmationRepository
	inserter              JobInserter
}

// NewSendEmailConfirmationUsecase は SendEmailConfirmationUsecase を生成する
func NewSendEmailConfirmationUsecase(
	cfg *config.Config,
	emailConfirmationRepo *repository.EmailConfirmationRepository,
	inserter JobInserter,
) *SendEmailConfirmationUsecase {
	return &SendEmailConfirmationUsecase{
		cfg:                   cfg,
		emailConfirmationRepo: emailConfirmationRepo,
		inserter:              inserter,
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

// Execute はメール確認コードを生成してメール送信ジョブをエンキューする
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

	// テンプレートをレンダリング
	htmlBody, textBody, err := uc.renderEmailTemplates(ctx, input.Email, code, input.Locale)
	if err != nil {
		slog.ErrorContext(ctx, "メールテンプレートのレンダリングに失敗しました",
			"email", input.Email,
			"error", err,
		)
		// レンダリングに失敗してもコードは有効なので、処理を続行
		return &SendEmailConfirmationOutput{
			EmailConfirmationID: confirmation.ID,
		}, nil
	}

	// メール件名を取得
	subject := i18n.T(ctx, "email_confirmation_subject")

	// メール送信ジョブをエンキュー（事前レンダリング済み文字列を渡す）
	_, err = uc.inserter.Insert(ctx, worker.SendEmailArgs{
		To:       input.Email,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	})
	if err != nil {
		// ジョブエンキューに失敗してもコードは有効なので、エラーログを出力して続行
		slog.ErrorContext(ctx, "メール送信ジョブのエンキューに失敗しました",
			"email", input.Email,
			"error", err,
		)
	} else {
		slog.InfoContext(ctx, "メール送信ジョブをエンキューしました",
			"email", input.Email,
		)
	}

	return &SendEmailConfirmationOutput{
		EmailConfirmationID: confirmation.ID,
	}, nil
}

// renderEmailTemplates はロケールに基づいてメールテンプレートをレンダリングする
func (uc *SendEmailConfirmationUsecase) renderEmailTemplates(ctx context.Context, emailAddr, code, locale string) (htmlBody, textBody string, err error) {
	data := email_confirmation.Data{
		Email:  emailAddr,
		Code:   code,
		AppURL: uc.cfg.AppURL(),
	}

	// HTMLテンプレートをレンダリング
	var htmlBuf bytes.Buffer
	var textBuf bytes.Buffer

	switch locale {
	case "ja":
		if err := email_confirmation.JaHTML(data).Render(ctx, &htmlBuf); err != nil {
			return "", "", fmt.Errorf("HTMLテンプレートのレンダリングに失敗しました: %w", err)
		}
		if err := email_confirmation.JaText(data).Render(ctx, &textBuf); err != nil {
			return "", "", fmt.Errorf("テキストテンプレートのレンダリングに失敗しました: %w", err)
		}
	default:
		if err := email_confirmation.EnHTML(data).Render(ctx, &htmlBuf); err != nil {
			return "", "", fmt.Errorf("HTMLテンプレートのレンダリングに失敗しました: %w", err)
		}
		if err := email_confirmation.EnText(data).Render(ctx, &textBuf); err != nil {
			return "", "", fmt.Errorf("テキストテンプレートのレンダリングに失敗しました: %w", err)
		}
	}

	return htmlBuf.String(), textBuf.String(), nil
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
