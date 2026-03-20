package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CreateAccountUsecase はアカウント作成ユースケース
type CreateAccountUsecase struct {
	db               *sql.DB
	userRepo         *repository.UserRepository
	userPasswordRepo *repository.UserPasswordRepository
}

// NewCreateAccountUsecase は CreateAccountUsecase を生成する
func NewCreateAccountUsecase(
	db *sql.DB,
	userRepo *repository.UserRepository,
	userPasswordRepo *repository.UserPasswordRepository,
) *CreateAccountUsecase {
	return &CreateAccountUsecase{
		db:               db,
		userRepo:         userRepo,
		userPasswordRepo: userPasswordRepo,
	}
}

// CreateAccountInput はアカウント作成の入力パラメータ
type CreateAccountInput struct {
	EmailConfirmation *model.EmailConfirmation
	Atname            string
	Password          string
	Locale            model.Locale
	TimeZone          string
}

// CreateAccountOutput はアカウント作成の出力パラメータ
type CreateAccountOutput struct {
	UserID model.UserID
}

// Execute はアカウントを作成する
func (uc *CreateAccountUsecase) Execute(ctx context.Context, input CreateAccountInput) (*CreateAccountOutput, error) {
	// トランザクションを開始
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// トランザクション内で操作するためのリポジトリを取得
	userRepo := uc.userRepo.WithTx(tx)
	userPasswordRepo := uc.userPasswordRepo.WithTx(tx)

	// パスワードをbcryptでハッシュ化
	passwordDigest, err := hashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("パスワードのハッシュ化に失敗しました: %w", err)
	}

	// ユーザーを作成
	now := time.Now()
	user, err := userRepo.Create(ctx, repository.CreateUserInput{
		Email:       input.EmailConfirmation.Email,
		Atname:      input.Atname,
		Name:        "",
		Description: "",
		Locale:      input.Locale,
		TimeZone:    input.TimeZone,
		JoinedAt:    now,
	})
	if err != nil {
		return nil, fmt.Errorf("ユーザーの作成に失敗しました: %w", err)
	}

	// ユーザーパスワードを作成
	_, err = userPasswordRepo.Create(ctx, repository.CreateUserPasswordInput{
		UserID:         user.ID,
		PasswordDigest: passwordDigest,
	})
	if err != nil {
		return nil, fmt.Errorf("ユーザーパスワードの作成に失敗しました: %w", err)
	}

	// トランザクションをコミット
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &CreateAccountOutput{
		UserID: user.ID,
	}, nil
}

// hashPassword はパスワードをbcryptでハッシュ化する
func hashPassword(password string) (string, error) {
	// bcrypt.DefaultCostを使用（現在は10）
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
