package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// ExportRepository is the repository of space exports.
//
// [Ja] ExportRepository はエクスポートリポジトリ
type ExportRepository struct {
	q *query.Queries
}

// NewExportRepository builds an ExportRepository.
//
// [Ja] NewExportRepository は ExportRepository を生成する
func NewExportRepository(q *query.Queries) *ExportRepository {
	return &ExportRepository{q: q}
}

// WithTx returns a new Repository that runs inside the given transaction.
//
// [Ja] WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *ExportRepository) WithTx(tx *sql.Tx) *ExportRepository {
	return &ExportRepository{q: r.q.WithTx(tx)}
}

// CreateExportInput carries what it takes to record an export.
//
// [Ja] CreateExportInput はエクスポート作成の入力パラメータ
type CreateExportInput struct {
	SpaceID    model.SpaceID
	QueuedByID model.SpaceMemberID
}

// Create records a queued export. The job that generates it is enqueued by the caller.
//
// [Ja] Create はキュー投入済みの状態でエクスポートを記録する。生成するジョブを投入するのは
// 呼び出し側である。
func (r *ExportRepository) Create(ctx context.Context, input CreateExportInput) (*model.Export, error) {
	row, err := r.q.CreateExport(ctx, query.CreateExportParams{
		SpaceID:    string(input.SpaceID),
		QueuedByID: string(input.QueuedByID),
		Status:     int32(model.ExportStatusQueued),
		Now:        time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByIDAndSpace returns the export with the given ID, or (nil, nil) when the space holds
// no such export. The ID reaches this method from the URL, so a string that is not a UUID is
// answered as not found rather than sent to the database, which would reject it as a malformed
// literal.
//
// [Ja] FindByIDAndSpace は指定 ID のエクスポートを返す。スペースにそのエクスポートが無い場合は
// (nil, nil) を返す。ID は URL からこのメソッドへ渡ってくるため、UUID でない文字列はデータベースへ
// 送らず「見つからない」として答える。送れば不正なリテラルとして拒否されるためである。
func (r *ExportRepository) FindByIDAndSpace(ctx context.Context, id model.ExportID, spaceID model.SpaceID) (*model.Export, error) {
	if !uuidRegex.MatchString(string(id)) {
		return nil, nil
	}

	row, err := r.q.FindExportByIDAndSpace(ctx, query.FindExportByIDAndSpaceParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindLatestBySpace returns the most recent export of the space, or (nil, nil) when the space
// has never been exported.
//
// [Ja] FindLatestBySpace はスペースの最新のエクスポートを返す。一度もエクスポートされていない
// 場合は (nil, nil) を返す。
func (r *ExportRepository) FindLatestBySpace(ctx context.Context, spaceID model.SpaceID) (*model.Export, error) {
	row, err := r.q.FindLatestExportBySpace(ctx, string(spaceID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// ListOlderBySpace returns the exports of the space older than the given one, oldest first.
// A successful export uses it to find the exports it replaces. A newer replacement is never
// returned, even if the given export's worker comes back after it was considered stale.
//
// [Ja] ListOlderBySpace は指定したエクスポートより古い、同じスペースのエクスポートを古い順に
// 返す。成功したエクスポートが、自分が置き換えるエクスポートを見つけるために使う。指定した
// エクスポートが停止したと見なされた後でワーカーが復帰しても、新しい後継は返さない。
func (r *ExportRepository) ListOlderBySpace(ctx context.Context, spaceID model.SpaceID, currentID model.ExportID) ([]*model.Export, error) {
	rows, err := r.q.ListOlderExportsBySpace(ctx, query.ListOlderExportsBySpaceParams{
		CurrentID: string(currentID),
		SpaceID:   string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// MarkStarted moves the export into ExportStatusStarted and stamps a fresh heartbeat.
//
// A retry finds the export already started, because the attempt before it did not reach a
// terminal status, so that status is accepted as a starting point too. An export that has
// already finished is not moved back, and (nil, nil) is returned.
//
// [Ja] MarkStarted はエクスポートを ExportStatusStarted へ進め、heartbeat を新しくする。
//
// リトライでは、直前の試行が終端の状態に到達していないためエクスポートが started のままになって
// いる。そのため started も開始点として受け付ける。既に完了しているエクスポートは巻き戻さず、
// (nil, nil) を返す。
func (r *ExportRepository) MarkStarted(ctx context.Context, id model.ExportID, spaceID model.SpaceID) (*model.Export, error) {
	now := time.Now()
	return r.updateStatus(ctx, updateExportStatusInput{
		ID:      id,
		SpaceID: spaceID,
		Status:  model.ExportStatusStarted,
		ExpectedStatuses: []model.ExportStatus{
			model.ExportStatusQueued,
			model.ExportStatusStarted,
		},
		HeartbeatAt: &now,
		Now:         now,
	})
}

// MarkSucceeded moves the export into ExportStatusSucceeded and records where its ZIP is.
// Only a started export is moved; (nil, nil) is returned otherwise.
//
// [Ja] MarkSucceeded はエクスポートを ExportStatusSucceeded へ進め、ZIP の位置を記録する。進める
// のは started のエクスポートだけで、それ以外では (nil, nil) を返す。
func (r *ExportRepository) MarkSucceeded(ctx context.Context, id model.ExportID, spaceID model.SpaceID, objectKey string) (*model.Export, error) {
	return r.updateStatus(ctx, updateExportStatusInput{
		ID:               id,
		SpaceID:          spaceID,
		Status:           model.ExportStatusSucceeded,
		ExpectedStatuses: []model.ExportStatus{model.ExportStatusStarted},
		ObjectKey:        &objectKey,
		Now:              time.Now(),
	})
}

// MarkFailed moves the export into ExportStatusFailed. Only a started export is moved;
// (nil, nil) is returned otherwise.
//
// [Ja] MarkFailed はエクスポートを ExportStatusFailed へ進める。進めるのは started のエクスポート
// だけで、それ以外では (nil, nil) を返す。
func (r *ExportRepository) MarkFailed(ctx context.Context, id model.ExportID, spaceID model.SpaceID) (*model.Export, error) {
	return r.updateStatus(ctx, updateExportStatusInput{
		ID:               id,
		SpaceID:          spaceID,
		Status:           model.ExportStatusFailed,
		ExpectedStatuses: []model.ExportStatus{model.ExportStatusStarted},
		Now:              time.Now(),
	})
}

// UpdateHeartbeat stamps a fresh heartbeat on a started export, so that a long export is not
// taken for a stopped one. An export in any other status is left alone, and false is returned:
// the export was declared stopped and replaced while this worker was running, so the caller can
// give up instead of finishing work that no longer counts.
//
// [Ja] UpdateHeartbeat は started のエクスポートに新しい heartbeat を打ち、長くかかっている
// エクスポートが止まったものと見なされないようにする。それ以外の状態のエクスポートには触れず
// false を返す。この場合、走っている間にエクスポートが停止したと判断されて置き換えられている
// ので、呼び出し側は既に意味を失った処理を続けずに打ち切れる。
func (r *ExportRepository) UpdateHeartbeat(ctx context.Context, id model.ExportID, spaceID model.SpaceID) (bool, error) {
	_, err := r.q.UpdateExportHeartbeat(ctx, query.UpdateExportHeartbeatParams{
		Now:     time.Now(),
		ID:      string(id),
		SpaceID: string(spaceID),
		Status:  int32(model.ExportStatusStarted),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Delete removes the export together with the Rails-era status history that references it.
// The ZIP object it points at is deleted by the caller, which is the only one that can reach
// the object storage.
//
// [Ja] Delete はエクスポートを、それを参照する Rails 時代の状態履歴ごと削除する。参照している
// ZIP オブジェクトの削除は、オブジェクトストレージへ到達できる唯一の側である呼び出し側が行う。
func (r *ExportRepository) Delete(ctx context.Context, id model.ExportID, spaceID model.SpaceID) error {
	if err := r.q.DeleteExportStatusesByExport(ctx, query.DeleteExportStatusesByExportParams{
		ExportID: string(id),
		SpaceID:  string(spaceID),
	}); err != nil {
		return err
	}
	return r.q.DeleteExport(ctx, query.DeleteExportParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
}

// updateExportStatusInput carries one conditional status transition.
//
// [Ja] updateExportStatusInput は状態遷移の入力パラメータ
type updateExportStatusInput struct {
	ID               model.ExportID
	SpaceID          model.SpaceID
	Status           model.ExportStatus
	ExpectedStatuses []model.ExportStatus
	HeartbeatAt      *time.Time
	ObjectKey        *string
	Now              time.Time
}

func (r *ExportRepository) updateStatus(ctx context.Context, input updateExportStatusInput) (*model.Export, error) {
	expectedStatuses := make([]int32, len(input.ExpectedStatuses))
	for i, status := range input.ExpectedStatuses {
		expectedStatuses[i] = int32(status)
	}

	var heartbeatAt sql.NullTime
	if input.HeartbeatAt != nil {
		heartbeatAt = sql.NullTime{Time: *input.HeartbeatAt, Valid: true}
	}

	var objectKey sql.NullString
	if input.ObjectKey != nil {
		objectKey = sql.NullString{String: *input.ObjectKey, Valid: true}
	}

	row, err := r.q.UpdateExportStatus(ctx, query.UpdateExportStatusParams{
		Status:           int32(input.Status),
		Now:              input.Now,
		HeartbeatAt:      heartbeatAt,
		ObjectKey:        objectKey,
		ID:               string(input.ID),
		SpaceID:          string(input.SpaceID),
		ExpectedStatuses: expectedStatuses,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

func (r *ExportRepository) toModel(row query.Export) *model.Export {
	var heartbeatAt *time.Time
	if row.HeartbeatAt.Valid {
		heartbeatAt = &row.HeartbeatAt.Time
	}

	var objectKey *string
	if row.ObjectKey.Valid {
		objectKey = &row.ObjectKey.String
	}

	return &model.Export{
		ID:              model.ExportID(row.ID),
		SpaceID:         model.SpaceID(row.SpaceID),
		QueuedByID:      model.SpaceMemberID(row.QueuedByID),
		Status:          model.ExportStatus(row.Status),
		StatusChangedAt: row.StatusChangedAt,
		HeartbeatAt:     heartbeatAt,
		ObjectKey:       objectKey,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func (r *ExportRepository) toModels(rows []query.Export) []*model.Export {
	exports := make([]*model.Export, len(rows))
	for i, row := range rows {
		exports[i] = r.toModel(row)
	}
	return exports
}
