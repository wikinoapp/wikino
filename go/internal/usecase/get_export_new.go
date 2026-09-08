package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetExportNewUsecase gathers what the screen that starts an export shows.
//
// [Ja] GetExportNewUsecase はエクスポートを開始する画面が表示するものを集める。
type GetExportNewUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	exportRepo      *repository.ExportRepository
}

// NewGetExportNewUsecase creates a GetExportNewUsecase.
//
// [Ja] NewGetExportNewUsecase は GetExportNewUsecase を生成する。
func NewGetExportNewUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	exportRepo *repository.ExportRepository,
) *GetExportNewUsecase {
	return &GetExportNewUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		exportRepo:      exportRepo,
	}
}

// GetExportNewInput holds what it takes to render the screen.
//
// [Ja] GetExportNewInput は画面を描画するための入力パラメータ。
type GetExportNewInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
}

// GetExportNewOutput carries the space and, when the space is already exporting, the export that
// is running. The screen offers the running one instead of a start button, because starting a
// second export while the first is going is refused anyway.
//
// [Ja] GetExportNewOutput はスペースと、そのスペースが既にエクスポート中であれば実行中の
// エクスポートを返す。画面は開始ボタンの代わりに実行中のエクスポートへの導線を出す。1 つ目が
// 続いている間に 2 つ目を開始しても、どのみち拒否されるためである。
type GetExportNewOutput struct {
	Space            *model.Space
	InProgressExport *model.Export
}

// Execute resolves the space and whether it is already exporting.
//
// [Ja] Execute はスペースと、そのスペースが既にエクスポート中かどうかを解決する。
func (uc *GetExportNewUsecase) Execute(ctx context.Context, input GetExportNewInput) (*GetExportNewOutput, error) {
	space, _, err := fetchExportAccess(ctx, uc.spaceRepo, uc.spaceMemberRepo, input.SpaceIdentifier, input.UserID)
	if err != nil {
		return nil, err
	}

	latest, err := uc.exportRepo.FindLatestBySpace(ctx, space.ID)
	if err != nil {
		return nil, fmt.Errorf("最新のエクスポートの取得に失敗: %w", err)
	}

	output := &GetExportNewOutput{Space: space}
	if latest != nil && latest.InProgress(time.Now()) {
		output.InProgressExport = latest
	}

	return output, nil
}
