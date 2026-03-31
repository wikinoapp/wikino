package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateAccountUsecase はアカウント作成ユースケース
type CreateAccountUsecase struct {
	db                    *sql.DB
	emailConfirmationRepo *repository.EmailConfirmationRepository
	userRepo              *repository.UserRepository
	userPasswordRepo      *repository.UserPasswordRepository
	createValidator       *validator.AccountCreateValidator
}

// NewCreateAccountUsecase は CreateAccountUsecase を生成する
func NewCreateAccountUsecase(
	db *sql.DB,
	emailConfirmationRepo *repository.EmailConfirmationRepository,
	userRepo *repository.UserRepository,
	userPasswordRepo *repository.UserPasswordRepository,
	createValidator *validator.AccountCreateValidator,
) *CreateAccountUsecase {
	return &CreateAccountUsecase{
		db:                    db,
		emailConfirmationRepo: emailConfirmationRepo,
		userRepo:              userRepo,
		userPasswordRepo:      userPasswordRepo,
		createValidator:       createValidator,
	}
}

// CreateAccountInput はアカウント作成の入力パラメータ
type CreateAccountInput struct {
	EmailConfirmationID string
	Atname              string
	Password            string
	Locale              model.Locale
	TimeZone            string
}

// CreateAccountOutput はアカウント作成の出力パラメータ
type CreateAccountOutput struct {
	UserID model.UserID
}

// Execute はアカウントを作成する
func (uc *CreateAccountUsecase) Execute(ctx context.Context, input CreateAccountInput) (*CreateAccountOutput, error) {
	// 1. データ取得
	emailConfirmation, err := uc.fetchEmailConfirmation(ctx, input.EmailConfirmationID)
	if err != nil {
		return nil, err
	}

	// 2. バリデーション
	if err := uc.createValidator.Validate(ctx, validator.AccountCreateValidatorInput{
		Atname:   input.Atname,
		Password: input.Password,
	}); err != nil {
		return nil, err
	}

	// 3. 永続化
	return uc.createAccount(ctx, emailConfirmation, input)
}

func (uc *CreateAccountUsecase) fetchEmailConfirmation(ctx context.Context, emailConfirmationID string) (*model.EmailConfirmation, error) {
	emailConfirmation, err := uc.emailConfirmationRepo.FindByID(ctx, emailConfirmationID)
	if err != nil {
		return nil, fmt.Errorf("メール確認情報の取得に失敗しました: %w", err)
	}
	if emailConfirmation == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	if !emailConfirmation.IsSucceeded() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeConflict,
			UserMsg: i18n.T(ctx, "error_email_not_confirmed"),
		}
	}

	return emailConfirmation, nil
}

func (uc *CreateAccountUsecase) createAccount(ctx context.Context, emailConfirmation *model.EmailConfirmation, input CreateAccountInput) (*CreateAccountOutput, error) {
	passwordDigest, err := hashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("パスワードのハッシュ化に失敗しました: %w", err)
	}

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userRepo := uc.userRepo.WithTx(tx)
	userPasswordRepo := uc.userPasswordRepo.WithTx(tx)

	now := time.Now()
	user, err := userRepo.Create(ctx, repository.CreateUserInput{
		Email:       emailConfirmation.Email,
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

	_, err = userPasswordRepo.Create(ctx, repository.CreateUserPasswordInput{
		UserID:         user.ID,
		PasswordDigest: passwordDigest,
	})
	if err != nil {
		return nil, fmt.Errorf("ユーザーパスワードの作成に失敗しました: %w", err)
	}

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
