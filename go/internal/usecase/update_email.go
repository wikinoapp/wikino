package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// userEmailUniqueIndexはユーザーのメールアドレスの一意インデックスの名前
const userEmailUniqueIndex = "index_users_on_email"

// UpdateEmailUsecaseは、メールアドレスの変更の確認コードを検証し、ログイン中のユーザーの
// メールアドレスを確認したメールアドレスに変更する
type UpdateEmailUsecase struct {
	db                    *sql.DB
	updateValidator       *validator.SettingsEmailConfirmationUpdateValidator
	emailConfirmationRepo *repository.EmailConfirmationRepository
	userRepo              *repository.UserRepository
}

// NewUpdateEmailUsecaseはUpdateEmailUsecaseを生成する
func NewUpdateEmailUsecase(
	db *sql.DB,
	updateValidator *validator.SettingsEmailConfirmationUpdateValidator,
	emailConfirmationRepo *repository.EmailConfirmationRepository,
	userRepo *repository.UserRepository,
) *UpdateEmailUsecase {
	return &UpdateEmailUsecase{
		db:                    db,
		updateValidator:       updateValidator,
		emailConfirmationRepo: emailConfirmationRepo,
		userRepo:              userRepo,
	}
}

// UpdateEmailInputはメールアドレスの変更を確定する入力パラメータ
type UpdateEmailInput struct {
	UserID              model.UserID
	EmailConfirmationID string
	Code                string
}

// UpdateEmailOutputは更新したユーザーを保持する
type UpdateEmailOutput struct {
	User *model.User
}

// Executeは確認コードを検証し、確認の完了とメールアドレスの更新を1つのトランザクションで行う
func (uc *UpdateEmailUsecase) Execute(ctx context.Context, input UpdateEmailInput) (*UpdateEmailOutput, error) {
	confirmation, err := uc.updateValidator.Validate(ctx, validator.SettingsEmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: input.EmailConfirmationID,
		Code:                input.Code,
	})
	if err != nil {
		return nil, err
	}

	return uc.updateEmail(ctx, input.UserID, confirmation)
}

func (uc *UpdateEmailUsecase) updateEmail(ctx context.Context, userID model.UserID, confirmation *model.EmailConfirmation) (*UpdateEmailOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := uc.emailConfirmationRepo.WithTx(tx).Succeed(ctx, confirmation.ID); err != nil {
		return nil, fmt.Errorf("メール確認の完了状態の更新に失敗: %w", err)
	}

	user, err := uc.userRepo.WithTx(tx).UpdateEmail(ctx, userID, confirmation.Email)
	if err != nil {
		// 検証の後に同じメールアドレスのユーザーが作られた場合は、検証で見つけたときと同じく
		// フォームのエラーとして返す
		if isUniqueViolationOn(err, userEmailUniqueIndex) {
			ve := model.NewValidationError()
			ve.AddGlobal(i18n.T(ctx, "validation_email_already_registered"))
			return nil, ve
		}
		return nil, fmt.Errorf("メールアドレスの更新に失敗: %w", err)
	}
	if user == nil {
		return nil, &model.AppError{
			Code:     model.AppErrCodeResourceNotFound,
			UserMsg:  i18n.T(ctx, "error_not_found_message"),
			Internal: fmt.Errorf("削除済みのユーザー%sのメールアドレスは更新できない", userID),
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return &UpdateEmailOutput{User: user}, nil
}
