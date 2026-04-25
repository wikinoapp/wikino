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
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateEmailConfirmationUsecase はメール確認コード作成ユースケース
type CreateEmailConfirmationUsecase struct {
	cfg                   *config.Config
	emailConfirmationRepo *repository.EmailConfirmationRepository
	dispatcher            *dispatcher.Dispatcher
	createValidator       *validator.EmailConfirmationCreateValidator
}

// NewCreateEmailConfirmationUsecase は CreateEmailConfirmationUsecase を生成する
func NewCreateEmailConfirmationUsecase(
	cfg *config.Config,
	emailConfirmationRepo *repository.EmailConfirmationRepository,
	d *dispatcher.Dispatcher,
	createValidator *validator.EmailConfirmationCreateValidator,
) *CreateEmailConfirmationUsecase {
	return &CreateEmailConfirmationUsecase{
		cfg:                   cfg,
		emailConfirmationRepo: emailConfirmationRepo,
		dispatcher:            d,
		createValidator:       createValidator,
	}
}

// CreateEmailConfirmationInput はメール確認コード送信の入力パラメータ
type CreateEmailConfirmationInput struct {
	Email  string
	Event  model.EmailConfirmationEvent
	Locale string
}

// CreateEmailConfirmationOutput はメール確認コード送信の出力パラメータ
type CreateEmailConfirmationOutput struct {
	EmailConfirmationID string
}

// Execute はメール確認コードを生成してメール送信ジョブをエンキューする
func (uc *CreateEmailConfirmationUsecase) Execute(ctx context.Context, input CreateEmailConfirmationInput) (*CreateEmailConfirmationOutput, error) {
	// 1. バリデーション
	if err := uc.createValidator.Validate(ctx, validator.EmailConfirmationCreateValidatorInput{
		Email: input.Email,
		Event: input.Event,
	}); err != nil {
		return nil, err
	}

	// 2. 確認コードを生成
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

	// メール送信ジョブをエンキュー（テンプレートのレンダリングはWorkerで行う）
	err = uc.dispatcher.EnqueueEmailConfirmation(ctx, input.Email, code, uc.cfg.AppURL(), input.Locale)
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

	return &CreateEmailConfirmationOutput{
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
