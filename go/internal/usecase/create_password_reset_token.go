package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/password_reset"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/worker"
)

// CreatePasswordResetTokenUsecase はパスワードリセットトークン作成ユースケース
type CreatePasswordResetTokenUsecase struct {
	cfg                    *config.Config
	db                     *sql.DB
	passwordResetTokenRepo *repository.PasswordResetTokenRepository
	inserter               JobInserter
}

// NewCreatePasswordResetTokenUsecase は CreatePasswordResetTokenUsecase を生成する
func NewCreatePasswordResetTokenUsecase(
	cfg *config.Config,
	db *sql.DB,
	passwordResetTokenRepo *repository.PasswordResetTokenRepository,
	inserter JobInserter,
) *CreatePasswordResetTokenUsecase {
	return &CreatePasswordResetTokenUsecase{
		cfg:                    cfg,
		db:                     db,
		passwordResetTokenRepo: passwordResetTokenRepo,
		inserter:               inserter,
	}
}

// CreatePasswordResetTokenInput はパスワードリセットトークン作成の入力パラメータ
type CreatePasswordResetTokenInput struct {
	UserID model.UserID
	Email  string
	Locale string
}

// CreatePasswordResetTokenOutput はパスワードリセットトークン作成の出力パラメータ
type CreatePasswordResetTokenOutput struct {
	TokenID string
}

// Execute はパスワードリセットトークンを生成し、メール送信ジョブをエンキューする
func (uc *CreatePasswordResetTokenUsecase) Execute(ctx context.Context, input CreatePasswordResetTokenInput) (*CreatePasswordResetTokenOutput, error) {
	// トランザクションを開始
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// トランザクション内で操作するためのリポジトリを取得
	tokenRepo := uc.passwordResetTokenRepo.WithTx(tx)

	// 既存の未使用トークンを削除
	if err := tokenRepo.DeleteUnusedByUserID(ctx, input.UserID); err != nil {
		return nil, fmt.Errorf("既存トークンの削除に失敗しました: %w", err)
	}

	// 新しいトークンを生成
	plainToken, err := password_reset.GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("トークンの生成に失敗しました: %w", err)
	}

	// トークンをハッシュ化
	tokenDigest := password_reset.HashToken(plainToken)

	// トークンをDBに保存
	expiresAt := time.Now().Add(model.PasswordResetTokenExpirationDuration)
	token, err := tokenRepo.Create(ctx, repository.CreatePasswordResetTokenInput{
		UserID:      input.UserID,
		TokenDigest: tokenDigest,
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("トークンの保存に失敗しました: %w", err)
	}

	// トランザクションをコミット
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	// パスワードリセットURL を生成
	resetURL := fmt.Sprintf("%s/password/edit?token=%s", uc.cfg.AppURL(), plainToken)

	// メール送信ジョブをエンキュー
	_, err = uc.inserter.Insert(ctx, worker.SendPasswordResetArgs{
		Email:    input.Email,
		ResetURL: resetURL,
		AppURL:   uc.cfg.AppURL(),
		Locale:   input.Locale,
	})
	if err != nil {
		// ジョブエンキューに失敗してもトークンは有効なので、エラーログを出力して続行
		slog.ErrorContext(ctx, "パスワードリセットメール送信ジョブのエンキューに失敗しました",
			"email", input.Email,
			"error", err,
		)
	} else {
		slog.InfoContext(ctx, "パスワードリセットメール送信ジョブをエンキューしました",
			"email", input.Email,
		)
	}

	return &CreatePasswordResetTokenOutput{
		TokenID: token.ID,
	}, nil
}
