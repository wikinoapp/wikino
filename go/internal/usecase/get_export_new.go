package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetExportNewUsecaseはエクスポートを開始する画面が表示するものを集める。
type GetExportNewUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	exportRepo      *repository.ExportRepository
}

// NewGetExportNewUsecaseはGetExportNewUsecaseを生成する。
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

// GetExportNewInputは画面を描画するための入力パラメータ。
type GetExportNewInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
}

// GetExportNewOutputはスペースと、そのスペースが既にエクスポート中であれば実行中の
// エクスポートを返す。画面は開始ボタンの代わりに実行中のエクスポートへの導線を出す。1つ目が
// 続いている間に2つ目を開始しても、どのみち拒否されるためである。
type GetExportNewOutput struct {
	Space            *model.Space
	InProgressExport *model.Export
}

// Executeはスペースと、そのスペースが既にエクスポート中かどうかを解決する。
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
