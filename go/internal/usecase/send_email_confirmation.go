// Package usecase はアプリケーションのユースケース（ビジネスロジック）を提供します
package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/worker"
)

// EmailConfirmationEnqueuer はメール確認送信ジョブをエンキューするインターフェース
type EmailConfirmationEnqueuer interface {
	EnqueueEmailConfirmation(ctx context.Context, input worker.EnqueueEmailConfirmationInput) error
}

// SendEmailConfirmationUsecase はメール確認コード送信ユースケース
type SendEmailConfirmationUsecase struct {
	cfg                   *config.Config
	emailConfirmationRepo *repository.EmailConfirmationRepository
	enqueuer              EmailConfirmationEnqueuer
}

// NewSendEmailConfirmationUsecase は SendEmailConfirmationUsecase を生成する
func NewSendEmailConfirmationUsecase(
	cfg *config.Config,
	emailConfirmationRepo *repository.EmailConfirmationRepository,
	enqueuer EmailConfirmationEnqueuer,
) *SendEmailConfirmationUsecase {
	return &SendEmailConfirmationUsecase{
		cfg:                   cfg,
		emailConfirmationRepo: emailConfirmationRepo,
		enqueuer:              enqueuer,
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

	// メール送信ジョブをエンキュー
	if err := uc.enqueuer.EnqueueEmailConfirmation(ctx, worker.EnqueueEmailConfirmationInput{
		Email:  input.Email,
		Code:   code,
		AppURL: uc.cfg.AppURL(),
		Locale: input.Locale,
	}); err != nil {
		// ジョブエンキューに失敗してもコードは有効なので、エラーログを出力して続行
		slog.ErrorContext(ctx, "メール送信ジョブのエンキューに失敗しました",
			"email", input.Email,
			"error", err,
		)
	}

	return &SendEmailConfirmationOutput{
		EmailConfirmationID: confirmation.ID,
	}, nil
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
