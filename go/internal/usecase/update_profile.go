package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// userAtnameUniqueIndexはユーザーのアットネームの一意インデックスの名前
const userAtnameUniqueIndex = "index_users_on_atname"

// UpdateProfileUsecaseはログイン中のユーザーのプロフィール (アットネーム・名前・自己紹介) を
// 更新する
type UpdateProfileUsecase struct {
	updateValidator *validator.ProfileUpdateValidator
	userRepo        *repository.UserRepository
}

// NewUpdateProfileUsecaseはUpdateProfileUsecaseを生成する
func NewUpdateProfileUsecase(
	updateValidator *validator.ProfileUpdateValidator,
	userRepo *repository.UserRepository,
) *UpdateProfileUsecase {
	return &UpdateProfileUsecase{
		updateValidator: updateValidator,
		userRepo:        userRepo,
	}
}

// UpdateProfileInputはプロフィールの更新の入力パラメータ
type UpdateProfileInput struct {
	UserID      model.UserID
	Atname      string
	Name        string
	Description string
}

// UpdateProfileOutputは更新したユーザーを保持する
type UpdateProfileOutput struct {
	User *model.User
}

// Executeはプロフィールを更新する。更新するのはログイン中のユーザー自身だけのため、認可チェック
// は無い。
func (uc *UpdateProfileUsecase) Execute(ctx context.Context, input UpdateProfileInput) (*UpdateProfileOutput, error) {
	if err := uc.updateValidator.Validate(ctx, validator.ProfileUpdateValidatorInput{
		UserID:      input.UserID,
		Atname:      input.Atname,
		Name:        input.Name,
		Description: input.Description,
	}); err != nil {
		return nil, err
	}

	user, err := uc.userRepo.UpdateProfile(ctx, repository.UpdateProfileInput{
		ID:          input.UserID,
		Atname:      input.Atname,
		Name:        input.Name,
		Description: input.Description,
	})
	if err != nil {
		// バリデーションの後に同じアットネームのユーザーが作られた場合は、重複を見つけたときと
		// 同じくフォームのエラーとして返す
		if isUniqueViolationOn(err, userAtnameUniqueIndex) {
			ve := model.NewValidationError()
			ve.AddField("atname", i18n.T(ctx, "validation_atname_already_taken"))
			return nil, ve
		}
		return nil, fmt.Errorf("プロフィールの更新に失敗: %w", err)
	}
	if user == nil {
		return nil, &model.AppError{
			Code:     model.AppErrCodeResourceNotFound,
			UserMsg:  i18n.T(ctx, "error_not_found_message"),
			Internal: fmt.Errorf("削除済みのユーザー%sのプロフィールは更新できない", input.UserID),
		}
	}

	return &UpdateProfileOutput{User: user}, nil
}
