package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetExportShowUsecase gathers what the screen that follows an export shows.
//
// [Ja] GetExportShowUsecase はエクスポートの経過を追う画面が表示するものを集める。
type GetExportShowUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	exportRepo      *repository.ExportRepository
}

// NewGetExportShowUsecase creates a GetExportShowUsecase.
//
// [Ja] NewGetExportShowUsecase は GetExportShowUsecase を生成する。
func NewGetExportShowUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	exportRepo *repository.ExportRepository,
) *GetExportShowUsecase {
	return &GetExportShowUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		exportRepo:      exportRepo,
	}
}

// GetExportShowInput holds what it takes to render the screen.
//
// [Ja] GetExportShowInput は画面を描画するための入力パラメータ。
type GetExportShowInput struct {
	SpaceIdentifier model.SpaceIdentifier
	ExportID        model.ExportID
	UserID          model.UserID
}

// GetExportShowOutput carries the space and the export the screen is about.
//
// [Ja] GetExportShowOutput は画面が対象とするスペースとエクスポートを返す。
type GetExportShowOutput struct {
	Space  *model.Space
	Export *model.Export
}

// Execute resolves the export, scoped to the space it belongs to.
//
// [Ja] Execute はエクスポートを、それが属するスペースにスコープして解決する。
func (uc *GetExportShowUsecase) Execute(ctx context.Context, input GetExportShowInput) (*GetExportShowOutput, error) {
	space, _, err := fetchExportAccess(ctx, uc.spaceRepo, uc.spaceMemberRepo, input.SpaceIdentifier, input.UserID)
	if err != nil {
		return nil, err
	}

	export, err := uc.exportRepo.FindByIDAndSpace(ctx, input.ExportID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("エクスポートの取得に失敗: %w", err)
	}
	if export == nil {
		return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}

	return &GetExportShowOutput{Space: space, Export: export}, nil
}
