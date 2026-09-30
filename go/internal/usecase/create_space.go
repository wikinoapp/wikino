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

// spaceIdentifierUniqueIndexはスペースの識別子の一意インデックスの名前
const spaceIdentifierUniqueIndex = "index_spaces_on_identifier"

// CreateSpaceUsecaseはスペースを作成し、作成者をそのスペースのメンバーにする。
type CreateSpaceUsecase struct {
	db              *sql.DB
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	createValidator *validator.SpaceCreateValidator
}

// NewCreateSpaceUsecaseはCreateSpaceUsecaseを生成する
func NewCreateSpaceUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	createValidator *validator.SpaceCreateValidator,
) *CreateSpaceUsecase {
	return &CreateSpaceUsecase{
		db:              db,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		createValidator: createValidator,
	}
}

// CreateSpaceInputはスペース作成の入力パラメータ
type CreateSpaceInput struct {
	UserID     model.UserID
	Identifier string
	Name       string
}

// CreateSpaceOutputは作成されたスペースを保持する
type CreateSpaceOutput struct {
	Space *model.Space
}

// Executeはスペースを作成する。スペースはログインしている誰もが作れるため、認可チェックは無い。
func (uc *CreateSpaceUsecase) Execute(ctx context.Context, input CreateSpaceInput) (*CreateSpaceOutput, error) {
	// 1. バリデーション
	if err := uc.createValidator.Validate(ctx, validator.SpaceCreateValidatorInput{
		Identifier: input.Identifier,
		Name:       input.Name,
	}); err != nil {
		return nil, err
	}

	// 2. 永続化 (トランザクション)
	space, err := uc.createSpace(ctx, input)
	if err != nil {
		// バリデーションの後に同じ識別子のスペースが作られた場合は、一意性チェックで見つけた
		// ときと同じくフォームのエラーとして返す
		if isUniqueViolationOn(err, spaceIdentifierUniqueIndex) {
			ve := model.NewValidationError()
			ve.AddField("identifier", i18n.T(ctx, "validation_space_identifier_uniqueness"))
			return nil, ve
		}
		return nil, err
	}

	return &CreateSpaceOutput{Space: space}, nil
}

// createSpaceはスペースの作成と作成者の参加を1つのトランザクションで行う。誰も参加して
// いないスペースが残らないようにするためである。
//
// 作成者のロールは管理者にする。スペースを作ったメンバーがそのスペースのすべてを
// 扱えるようにするためである。
func (uc *CreateSpaceUsecase) createSpace(ctx context.Context, input CreateSpaceInput) (*model.Space, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	space, err := uc.spaceRepo.WithTx(tx).Create(ctx, repository.CreateSpaceInput{
		Identifier: model.SpaceIdentifier(input.Identifier),
		Name:       input.Name,
		Plan:       model.PlanFree,
	})
	if err != nil {
		return nil, fmt.Errorf("スペースの作成に失敗: %w", err)
	}

	if _, err := uc.spaceMemberRepo.WithTx(tx).Create(ctx, repository.CreateSpaceMemberInput{
		SpaceID: space.ID,
		UserID:  input.UserID,
		Role:    model.SpaceRoleAdmin,
	}); err != nil {
		return nil, fmt.Errorf("スペースメンバーの作成に失敗: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return space, nil
}
