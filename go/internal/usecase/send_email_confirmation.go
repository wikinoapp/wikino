// Package usecase はアプリケーションのユースケース（ビジネスロジック）を提供します
package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
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

	// メール送信ジョブをエンキュー（テンプレートのレンダリングはWorkerで行う）
	_, err = uc.inserter.Insert(ctx, worker.SendEmailConfirmationArgs{
		Email:  input.Email,
		Code:   code,
		AppURL: uc.cfg.AppURL(),
		Locale: input.Locale,
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
