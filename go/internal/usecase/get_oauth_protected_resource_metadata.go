package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetOAuthProtectedResourceMetadataUsecaseは、スペースのAPIの保護リソースのメタデータ (RFC 9728) を
// 返すために、パスのスペースが存在することを確かめる
type GetOAuthProtectedResourceMetadataUsecase struct {
	spaceRepo *repository.SpaceRepository
}

// NewGetOAuthProtectedResourceMetadataUsecaseはGetOAuthProtectedResourceMetadataUsecaseを生成する
func NewGetOAuthProtectedResourceMetadataUsecase(spaceRepo *repository.SpaceRepository) *GetOAuthProtectedResourceMetadataUsecase {
	return &GetOAuthProtectedResourceMetadataUsecase{spaceRepo: spaceRepo}
}

// GetOAuthProtectedResourceMetadataInputは入力パラメータ
type GetOAuthProtectedResourceMetadataInput struct {
	SpaceIdentifier model.SpaceIdentifier
}

// GetOAuthProtectedResourceMetadataOutputは、メタデータが記述するスペース
type GetOAuthProtectedResourceMetadataOutput struct {
	Space *model.Space
}

// Executeはスペースを取得する。存在しない・削除済みのスペースは、トークンを発行できる
// 保護リソースではないため、未存在として扱う。
//
// メタデータは個人のデータを含まないため、未ログインでも返し、フィーチャーフラグで隠さない
func (uc *GetOAuthProtectedResourceMetadataUsecase) Execute(ctx context.Context, input GetOAuthProtectedResourceMetadataInput) (*GetOAuthProtectedResourceMetadataOutput, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}
	return &GetOAuthProtectedResourceMetadataOutput{Space: space}, nil
}
