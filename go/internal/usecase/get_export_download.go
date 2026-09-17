package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
)

// GetExportDownloadUsecaseは完了したエクスポートのアーカイブを渡す。
//
// アーカイブはアプリケーションを経由してストリーミングするのではなく、ブラウザがオブジェクト
// ストレージから直接取得する。そのためここが生成するのは、それ自体が認可を持ち、時間が経てば
// ひとりでに使えなくなるURLである。
type GetExportDownloadUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	exportRepo      *repository.ExportRepository
	objectStorage   storage.ObjectStorage
}

// NewGetExportDownloadUsecaseはGetExportDownloadUsecaseを生成する。
func NewGetExportDownloadUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	exportRepo *repository.ExportRepository,
	objectStorage storage.ObjectStorage,
) *GetExportDownloadUsecase {
	return &GetExportDownloadUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		exportRepo:      exportRepo,
		objectStorage:   objectStorage,
	}
}

// GetExportDownloadInputはアーカイブを渡すための入力パラメータ。
type GetExportDownloadInput struct {
	SpaceIdentifier model.SpaceIdentifier
	ExportID        model.ExportID
	UserID          model.UserID
}

// GetExportDownloadOutputはブラウザを送る先のURLを返す。
type GetExportDownloadOutput struct {
	URL string
}

// Executeはエクスポートを解決し、そのアーカイブへのURLに署名する。
//
// アーカイブをもう渡せないエクスポートは、固有の理由ではなく「見つからない」として答える。画面は
// ダウンロードが使える間だけそれを表示するため、期限切れのエクスポートでここに到達するのは、
// アーカイブより長生きしたリンクをたどったときであり、実際に無いのはそのアーカイブである。
func (uc *GetExportDownloadUsecase) Execute(ctx context.Context, input GetExportDownloadInput) (*GetExportDownloadOutput, error) {
	if uc.objectStorage == nil {
		return nil, errors.New("オブジェクトストレージが設定されていないためエクスポートをダウンロードできません")
	}

	space, _, err := fetchExportAccess(ctx, uc.spaceRepo, uc.spaceMemberRepo, input.SpaceIdentifier, input.UserID)
	if err != nil {
		return nil, err
	}

	export, err := uc.exportRepo.FindByIDAndSpace(ctx, input.ExportID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("エクスポートの取得に失敗: %w", err)
	}
	if export == nil || !export.Downloadable(time.Now()) {
		return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}

	url, err := uc.objectStorage.PresignedGetURL(ctx, *export.ObjectKey, model.ExportDownloadExpiration)
	if err != nil {
		return nil, fmt.Errorf("エクスポートのダウンロードURLの生成に失敗: %w", err)
	}

	return &GetExportDownloadOutput{URL: url}, nil
}
