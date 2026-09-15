package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// ExportRepositoryはエクスポートリポジトリ
type ExportRepository struct {
	q *query.Queries
}

// NewExportRepositoryはExportRepositoryを生成する
func NewExportRepository(q *query.Queries) *ExportRepository {
	return &ExportRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *ExportRepository) WithTx(tx *sql.Tx) *ExportRepository {
	return &ExportRepository{q: r.q.WithTx(tx)}
}

// CreateExportInputはエクスポート作成の入力パラメータ
type CreateExportInput struct {
	SpaceID    model.SpaceID
	QueuedByID model.SpaceMemberID
}

// Createはキュー投入済みの状態でエクスポートを記録する。生成するジョブを投入するのは
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

// FindByIDAndSpaceは指定IDのエクスポートを返す。スペースにそのエクスポートが無い場合は
// (nil, nil) を返す。IDはURLからこのメソッドへ渡ってくるため、UUIDでない文字列はデータベースへ
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

// FindLatestBySpaceはスペースの最新のエクスポートを返す。一度もエクスポートされていない
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

// ListOlderBySpaceは指定したエクスポートより古い、同じスペースのエクスポートを古い順に
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

// MarkStartedはエクスポートをExportStatusStartedへ進め、heartbeatを新しくする。
//
// リトライでは、直前の試行が終端の状態に到達していないためエクスポートがstartedのままになって
// いる。そのためstartedも開始点として受け付ける。既に完了しているエクスポートは巻き戻さず、
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

// MarkSucceededはエクスポートをExportStatusSucceededへ進め、ZIPの位置を記録する。進める
// のはstartedのエクスポートだけで、それ以外では (nil, nil) を返す。
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

// MarkFailedはエクスポートをExportStatusFailedへ進める。進めるのはまだ結果に至っていない
// エクスポートだけで、それ以外では (nil, nil) を返す。
//
// startedと並んでqueuedも受け付けるのは、開始を記録する前に試行が諦めることがあるためである。
// エクスポートを進めるはずだった書き込み自体が失敗した場合がこれにあたる。queuedのまま残すと、
// 理由を誰にも伝えないままそのスペースがエクスポートできなくなる。
func (r *ExportRepository) MarkFailed(ctx context.Context, id model.ExportID, spaceID model.SpaceID) (*model.Export, error) {
	return r.updateStatus(ctx, updateExportStatusInput{
		ID:      id,
		SpaceID: spaceID,
		Status:  model.ExportStatusFailed,
		ExpectedStatuses: []model.ExportStatus{
			model.ExportStatusQueued,
			model.ExportStatusStarted,
		},
		Now: time.Now(),
	})
}

// MarkFailedIfStaleはエクスポートをExportStatusFailedへ進める。ただし、状態がstarted
// でheartbeatがstaleBeforeより古い、つまり止まって見える間に限る。読み取りからこの呼び出し
// までの間に生存を報告したエクスポートには手を触れず (nil, nil) を返す。アーカイブを書いている
// 最中のワーカーを、その足元で失敗させないためである。
//
// (nil, nil) を受け取った呼び出し元は、そのエクスポートが結局動いていたことを知る。新しい
// エクスポートを開始せずに拒否し、1つのスペースに対して2つのワーカーが書き込む状態を作らない。
func (r *ExportRepository) MarkFailedIfStale(ctx context.Context, id model.ExportID, spaceID model.SpaceID, staleBefore time.Time) (*model.Export, error) {
	return r.updateStatus(ctx, updateExportStatusInput{
		ID:                   id,
		SpaceID:              spaceID,
		Status:               model.ExportStatusFailed,
		ExpectedStatuses:     []model.ExportStatus{model.ExportStatusStarted},
		StaleHeartbeatBefore: &staleBefore,
		Now:                  time.Now(),
	})
}

// MarkFailedIfUnclaimedはエクスポートをExportStatusFailedへ進める。ただし、来ることの
// なかったワーカーを待ち続けている間、つまり状態がqueuedで、その状態になったのが
// unclaimedBeforeより前である間に限る。読み取りからこの呼び出しまでの間にジョブがワーカーへ
// 届いたエクスポートには手を触れず (nil, nil) を返す。開始したばかりのワーカーを、その足元で
// 失敗させないためである。
//
// heartbeatを参照できない状態のための、MarkFailedIfStaleの対になるメソッドである。queuedの
// エクスポートは鼓動を打たないため、まだ道半ばのものと、ジョブがもう存在しないものを分けられる
// のは、その状態でいる時間だけである。
func (r *ExportRepository) MarkFailedIfUnclaimed(ctx context.Context, id model.ExportID, spaceID model.SpaceID, unclaimedBefore time.Time) (*model.Export, error) {
	return r.updateStatus(ctx, updateExportStatusInput{
		ID:                       id,
		SpaceID:                  spaceID,
		Status:                   model.ExportStatusFailed,
		ExpectedStatuses:         []model.ExportStatus{model.ExportStatusQueued},
		StaleStatusChangedBefore: &unclaimedBefore,
		Now:                      time.Now(),
	})
}

// UpdateHeartbeatはstartedのエクスポートに新しいheartbeatを打ち、長くかかっている
// エクスポートが止まったものと見なされないようにする。それ以外の状態のエクスポートには触れず
// falseを返す。この場合、走っている間にエクスポートが停止したと判断されて置き換えられている
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

// Deleteはエクスポートを、旧ファイル関連ごと削除する。
// 共有されていない旧blobのメタデータも関連とともに削除する。参照している
// ZIPオブジェクトの削除は、オブジェクトストレージへ到達できる唯一の側である呼び出し側が行う。
func (r *ExportRepository) Delete(ctx context.Context, id model.ExportID, spaceID model.SpaceID) error {
	if err := r.q.DeleteLegacyExportFiles(ctx, query.DeleteLegacyExportFilesParams{ExportID: string(id), SpaceID: string(spaceID)}); err != nil {
		return err
	}
	return r.q.DeleteExport(ctx, query.DeleteExportParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
}

// updateExportStatusInputは状態遷移の入力パラメータ
type updateExportStatusInput struct {
	ID               model.ExportID
	SpaceID          model.SpaceID
	Status           model.ExportStatus
	ExpectedStatuses []model.ExportStatus
	HeartbeatAt      *time.Time
	ObjectKey        *string

	// StaleHeartbeatBeforeは、エクスポートがこの時刻以降に生存を報告している場合に遷移を
	// 拒否する。停止していると読み取ったエクスポートを失敗させる呼び出し元だけが設定し、その間に
	// 復帰したワーカーには手を触れないようにする。
	StaleHeartbeatBefore *time.Time

	// StaleStatusChangedBeforeは、エクスポートがこの時刻以降に現在の状態へ移っていた場合に
	// 遷移を拒否する。ワーカーに拾われないままだと読み取ったqueuedのエクスポートを失敗させる
	// 呼び出し元だけが設定し、その間にワーカーへ届いたジョブには手を触れないようにする。
	StaleStatusChangedBefore *time.Time

	Now time.Time
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

	var staleHeartbeatBefore sql.NullTime
	if input.StaleHeartbeatBefore != nil {
		staleHeartbeatBefore = sql.NullTime{Time: *input.StaleHeartbeatBefore, Valid: true}
	}

	var staleStatusChangedBefore sql.NullTime
	if input.StaleStatusChangedBefore != nil {
		staleStatusChangedBefore = sql.NullTime{Time: *input.StaleStatusChangedBefore, Valid: true}
	}

	row, err := r.q.UpdateExportStatus(ctx, query.UpdateExportStatusParams{
		Status:                   int32(input.Status),
		Now:                      input.Now,
		HeartbeatAt:              heartbeatAt,
		ObjectKey:                objectKey,
		ID:                       string(input.ID),
		SpaceID:                  string(input.SpaceID),
		ExpectedStatuses:         expectedStatuses,
		StaleHeartbeatBefore:     staleHeartbeatBefore,
		StaleStatusChangedBefore: staleStatusChangedBefore,
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

// LegacyExportFileは旧Railsアーカイブと、別のレコードが使用中かどうかを表す。
type LegacyExportFile struct {
	Key    string
	Shared bool
}

// ListLegacyFilesはエクスポートを削除する前に回収するRailsアーカイブを取得する。
func (r *ExportRepository) ListLegacyFiles(ctx context.Context, id model.ExportID, spaceID model.SpaceID) ([]LegacyExportFile, error) {
	rows, err := r.q.ListLegacyExportFiles(ctx, query.ListLegacyExportFilesParams{ExportID: string(id), SpaceID: string(spaceID)})
	if err != nil {
		return nil, err
	}
	files := make([]LegacyExportFile, len(rows))
	for i, row := range rows {
		files[i] = LegacyExportFile{Key: row.Key, Shared: row.Shared}
	}
	return files, nil
}
