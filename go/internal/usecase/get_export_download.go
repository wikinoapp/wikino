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

// GetExportDownloadUsecase hands out the archive of a finished export.
//
// The archive is fetched from the object storage directly by the browser rather than streamed
// through the application, so what this produces is a URL that carries its own authorization and
// stops working on its own.
//
// [Ja] GetExportDownloadUsecase は完了したエクスポートのアーカイブを渡す。
//
// アーカイブはアプリケーションを経由してストリーミングするのではなく、ブラウザがオブジェクト
// ストレージから直接取得する。そのためここが生成するのは、それ自体が認可を持ち、時間が経てば
// ひとりでに使えなくなる URL である。
type GetExportDownloadUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	exportRepo      *repository.ExportRepository
	objectStorage   storage.ObjectStorage
}

// NewGetExportDownloadUsecase creates a GetExportDownloadUsecase.
//
// [Ja] NewGetExportDownloadUsecase は GetExportDownloadUsecase を生成する。
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

// GetExportDownloadInput holds what it takes to hand out an archive.
//
// [Ja] GetExportDownloadInput はアーカイブを渡すための入力パラメータ。
type GetExportDownloadInput struct {
	SpaceIdentifier model.SpaceIdentifier
	ExportID        model.ExportID
	UserID          model.UserID
}

// GetExportDownloadOutput carries the URL the browser is sent to.
//
// [Ja] GetExportDownloadOutput はブラウザを送る先の URL を返す。
type GetExportDownloadOutput struct {
	URL string
}

// Execute resolves the export and signs a URL for its archive.
//
// An export whose archive can no longer be handed out is answered as not found rather than with a
// reason of its own. The screen shows the download only while it works, so reaching this with an
// expired export means following a link that outlived the archive it pointed at, and the archive
// itself is what is missing.
//
// [Ja] Execute はエクスポートを解決し、そのアーカイブへの URL に署名する。
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
