package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CreateAccountUsecase はアカウント作成ユースケース
type CreateAccountUsecase struct {
	db *sql.DB
}

// NewCreateAccountUsecase は CreateAccountUsecase を生成する
func NewCreateAccountUsecase(db *sql.DB) *CreateAccountUsecase {
	return &CreateAccountUsecase{
		db: db,
	}
}

// CreateAccountInput はアカウント作成の入力パラメータ
type CreateAccountInput struct {
	EmailConfirmationID string
	Email               string
	Atname              string
	Password            string
	Locale              model.Locale
	TimeZone            string
}

// CreateAccountOutput はアカウント作成の出力パラメータ
type CreateAccountOutput struct {
	UserID string
}

// エラー定義
var (
	// ErrEmailNotConfirmed はメール確認が完了していない場合のエラー
	ErrEmailNotConfirmed = errors.New("メール確認が完了していません")
	// ErrAtnameAlreadyTaken はアットネームが既に使用されている場合のエラー
	ErrAtnameAlreadyTaken = errors.New("このアットネームは既に使用されています")
)

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

	// トランザクション内でクエリを実行するためのQueriesを生成
	q := query.New(tx)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(q)
	userRepo := repository.NewUserRepository(q)
	userPasswordRepo := repository.NewUserPasswordRepository(q)

	// メール確認が完了しているかチェック
	confirmation, err := emailConfirmationRepo.FindByID(ctx, input.EmailConfirmationID)
	if err != nil {
		return nil, fmt.Errorf("メール確認情報の取得に失敗しました: %w", err)
	}
	if confirmation == nil || !confirmation.IsSucceeded() {
		return nil, ErrEmailNotConfirmed
	}

	// アットネームの重複チェック
	existingUser, err := userRepo.FindByAtname(ctx, input.Atname)
	if err != nil {
		return nil, fmt.Errorf("ユーザーの検索に失敗しました: %w", err)
	}
	if existingUser != nil {
		return nil, ErrAtnameAlreadyTaken
	}

	// パスワードをbcryptでハッシュ化
	passwordDigest, err := hashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("パスワードのハッシュ化に失敗しました: %w", err)
	}

	// ユーザーを作成
	now := time.Now()
	user, err := userRepo.Create(ctx, repository.CreateUserInput{
		Email:       input.Email,
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
