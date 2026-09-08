package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CreateExportUsecase starts an export of a space: it records the export and enqueues the job
// that writes the archive.
//
// A space runs at most one export at a time. Starting a second one while the first is still
// going would have two workers write for the same space, and the newer archive would replace the
// older one before anyone could download it.
//
// [Ja] CreateExportUsecase はスペースのエクスポートを開始する。エクスポートを記録し、アーカイブを
// 書き出すジョブを投入する。
//
// 1 つのスペースで同時に走るエクスポートは 1 つまでとする。1 つ目が続いている間に 2 つ目を始めると
// 同じスペースに対して 2 つのワーカーが書き込むことになり、新しいアーカイブが古いものを、誰も
// ダウンロードしないうちに置き換えてしまう。
type CreateExportUsecase struct {
	db              *sql.DB
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	exportRepo      *repository.ExportRepository
	dispatcher      *dispatcher.Dispatcher
}

// NewCreateExportUsecase creates a CreateExportUsecase.
//
// [Ja] NewCreateExportUsecase は CreateExportUsecase を生成する。
func NewCreateExportUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	exportRepo *repository.ExportRepository,
	d *dispatcher.Dispatcher,
) *CreateExportUsecase {
	return &CreateExportUsecase{
		db:              db,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		exportRepo:      exportRepo,
		dispatcher:      d,
	}
}

// CreateExportInput holds what it takes to start an export.
//
// [Ja] CreateExportInput はエクスポートを開始するための入力パラメータ。
type CreateExportInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
}

// CreateExportOutput carries the space and the export that was started, so that the handler can
// build the URL of the export screen without reading the request again.
//
// [Ja] CreateExportOutput は開始したエクスポートとそのスペースを返し、ハンドラーがリクエストを
// 読み直さずにエクスポート画面の URL を組み立てられるようにする。
type CreateExportOutput struct {
	Space  *model.Space
	Export *model.Export
}

// Execute records the export and enqueues the job that generates it.
//
// [Ja] Execute はエクスポートを記録し、それを生成するジョブを投入する。
func (uc *CreateExportUsecase) Execute(ctx context.Context, input CreateExportInput) (*CreateExportOutput, error) {
	space, spaceMember, err := fetchExportAccess(ctx, uc.spaceRepo, uc.spaceMemberRepo, input.SpaceIdentifier, input.UserID)
	if err != nil {
		return nil, err
	}

	export, err := uc.createExport(ctx, space.ID, spaceMember.ID)
	if err != nil {
		return nil, err
	}

	if err := uc.dispatcher.EnqueueGenerateExportFiles(ctx, export.ID.String(), string(space.ID)); err != nil {
		// Nothing will ever pick the export up, and a queued export blocks every later one for
		// this space, so the record is taken back rather than left to sit there.
		//
		// [Ja] このエクスポートを拾うものはもう無く、queued のエクスポートはこのスペースの以降の
		// エクスポートをすべて阻む。そのため、記録を残さず取り消す。
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), exportOutcomeTimeout)
		defer cancel()
		if deleteErr := uc.exportRepo.Delete(cleanupCtx, export.ID, space.ID); deleteErr != nil {
			slog.ErrorContext(ctx, "投入に失敗したエクスポートの削除に失敗しました",
				"export_id", export.ID.String(),
				"error", deleteErr,
			)
		}
		return nil, fmt.Errorf("エクスポート生成ジョブの投入に失敗: %w", err)
	}

	return &CreateExportOutput{Space: space, Export: export}, nil
}

// createExport records the export, first retiring a predecessor that has stopped.
//
// The predecessor is failed under the same condition it was read as stopped by, which
// retirePredecessor decides from its status. An export that turns out to be running after all
// keeps the space, and the new export is refused instead.
//
// [Ja] createExport はエクスポートを記録する。その前に、止まっている先行のエクスポートを終わらせる。
//
// 先行のエクスポートを失敗させる条件は、それを止まっていると読み取ったときの条件と同じにする。
// どの条件を使うかは retirePredecessor が状態から決める。結局動いていたと分かったエクスポートが
// スペースを保ち、代わりに新しいエクスポートのほうを拒否する。
func (uc *CreateExportUsecase) createExport(ctx context.Context, spaceID model.SpaceID, queuedByID model.SpaceMemberID) (*model.Export, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// The space lock must precede the latest-export read, including for a first export.
	//
	// [Ja] 初回のエクスポートでも、最新行を読む前にスペースをロックする。
	locked, err := uc.spaceRepo.WithTx(tx).LockByID(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("スペースのロックに失敗: %w", err)
	}
	if !locked {
		return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}
	exportRepo := uc.exportRepo.WithTx(tx)
	latest, err := exportRepo.FindLatestBySpace(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("最新のエクスポートの取得に失敗: %w", err)
	}
	if latest != nil && latest.InProgress(time.Now()) {
		return nil, uc.inProgressError(ctx, latest)
	}

	if err := uc.retirePredecessor(ctx, exportRepo, latest); err != nil {
		return nil, err
	}

	export, err := exportRepo.Create(ctx, repository.CreateExportInput{
		SpaceID:    spaceID,
		QueuedByID: queuedByID,
	})
	if err != nil {
		return nil, fmt.Errorf("エクスポートの作成に失敗: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return export, nil
}

// retirePredecessor fails the export that came before, which the caller has just read as stopped.
//
// Each of the two statuses that can be stopped is retired under the condition it was read as
// stopped by: a started export by its heartbeat, a queued one by how long it has been waiting for
// a worker. An export that moved on in between is running after all, so it keeps the space and
// the new export is refused instead.
//
// [Ja] retirePredecessor は、呼び出し元が止まっていると読み取った先行のエクスポートを失敗させる。
//
// 止まりうる 2 つの状態は、それぞれ止まっていると読み取ったときの条件で終わらせる。started は
// heartbeat で、queued はワーカーを待っている時間で判断する。その間に先へ進んだエクスポートは
// 結局動いているので、そのエクスポートがスペースを保ち、代わりに新しいエクスポートのほうを
// 拒否する。
func (uc *CreateExportUsecase) retirePredecessor(ctx context.Context, exportRepo *repository.ExportRepository, latest *model.Export) error {
	if latest == nil {
		return nil
	}

	var (
		failed *model.Export
		err    error
	)
	switch latest.Status {
	case model.ExportStatusStarted:
		failed, err = exportRepo.MarkFailedIfStale(ctx, latest.ID, latest.SpaceID, time.Now().Add(-model.ExportHeartbeatStaleAfter))
	case model.ExportStatusQueued:
		failed, err = exportRepo.MarkFailedIfUnclaimed(ctx, latest.ID, latest.SpaceID, time.Now().Add(-model.ExportQueuedStaleAfter))
	default:
		return nil
	}
	if err != nil {
		return fmt.Errorf("停止したエクスポートの失敗記録に失敗: %w", err)
	}
	if failed == nil {
		return uc.inProgressError(ctx, latest)
	}

	return nil
}

// inProgressError is what a caller gets when the space is already exporting. The screen hides the
// start button while that is the case, so reaching this means two requests raced.
//
// [Ja] inProgressError は、スペースが既にエクスポート中のときに呼び出し元へ返すエラー。その間は
// 画面が開始ボタンを隠すため、ここに到達するのは 2 つのリクエストが競合したときである。
func (uc *CreateExportUsecase) inProgressError(ctx context.Context, latest *model.Export) error {
	return &model.AppError{
		Code:     model.AppErrCodeConflict,
		UserMsg:  i18n.T(ctx, "validation_export_in_progress"),
		Metadata: map[string]string{"export_id": latest.ID.String()},
	}
}
