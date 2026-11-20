package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// UpdateSpaceUsecaseはスペースの一般設定 (識別子と名前) を保存する。
type UpdateSpaceUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	updateValidator *validator.SpaceUpdateValidator
}

// NewUpdateSpaceUsecaseはUpdateSpaceUsecaseを生成する
func NewUpdateSpaceUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	updateValidator *validator.SpaceUpdateValidator,
) *UpdateSpaceUsecase {
	return &UpdateSpaceUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		updateValidator: updateValidator,
	}
}

// UpdateSpaceInputはスペースの一般設定の保存の入力パラメータ。SpaceIdentifierは保存されている
// 識別子 (URLのもの) で、Identifierはフォームが送信した新しい識別子である。
type UpdateSpaceInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
	Identifier      string
	Name            string
}

// UpdateSpaceOutputは更新されたスペースを保持する
type UpdateSpaceOutput struct {
	Space *model.Space
}

// Executeはスペースの一般設定を更新する
func (uc *UpdateSpaceUsecase) Execute(ctx context.Context, input UpdateSpaceInput) (*UpdateSpaceOutput, error) {
	// 1. データ取得と認可チェック
	space, err := fetchSpaceUpdateAccess(ctx, uc.spaceRepo, uc.spaceMemberRepo, input.SpaceIdentifier, input.UserID)
	if err != nil {
		return nil, err
	}

	// 2. バリデーション
	if err := uc.updateValidator.Validate(ctx, validator.SpaceUpdateValidatorInput{
		SpaceID:    space.ID,
		Identifier: input.Identifier,
		Name:       input.Name,
	}); err != nil {
		return nil, err
	}

	// 3. 永続化
	//
	// 更新は1文で済むため、専用のトランザクションを開かずに実行する。
	updated, err := uc.spaceRepo.Update(ctx, repository.UpdateSpaceInput{
		ID:         space.ID,
		Identifier: model.SpaceIdentifier(input.Identifier),
		Name:       input.Name,
	})
	if err != nil {
		// バリデーションの後に別のスペースが同じ識別子を取った場合は、一意性チェックで見つけた
		// ときと同じくフォームのエラーとして返す
		if isUniqueViolationOn(err, spaceIdentifierUniqueIndex) {
			ve := model.NewValidationError()
			ve.AddField("identifier", i18n.T(ctx, "validation_space_identifier_uniqueness"))
			return nil, ve
		}
		return nil, fmt.Errorf("スペースの更新に失敗: %w", err)
	}
	// 読み取ったあとにスペースが削除された場合は、削除済みのスペースを書き換えない
	if updated == nil {
		return nil, notFoundError(ctx)
	}

	return &UpdateSpaceOutput{Space: updated}, nil
}
