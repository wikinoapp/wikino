package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// ExportBuilder builds export test data.
//
// [Ja] ExportBuilder はエクスポートテストデータのビルダー
type ExportBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID         string
	queuedByID      string
	status          int32
	statusChangedAt time.Time
	heartbeatAt     *time.Time
	objectKey       *string
	createdAt       time.Time
}

// NewExportBuilder builds an ExportBuilder.
//
// [Ja] NewExportBuilder は ExportBuilder を生成します
func NewExportBuilder(t *testing.T, tx *sql.Tx) *ExportBuilder {
	t.Helper()
	now := time.Now()
	return &ExportBuilder{
		t:               t,
		tx:              tx,
		status:          int32(model.ExportStatusQueued),
		statusChangedAt: now,
		createdAt:       now,
	}
}

// WithSpaceID sets the space.
//
// [Ja] WithSpaceID はスペースIDを設定します
func (b *ExportBuilder) WithSpaceID(spaceID model.SpaceID) *ExportBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithQueuedByID sets the space member who started the export.
//
// [Ja] WithQueuedByID はエクスポートを開始したスペースメンバーのIDを設定します
func (b *ExportBuilder) WithQueuedByID(id model.SpaceMemberID) *ExportBuilder {
	b.queuedByID = string(id)
	return b
}

// WithStatus sets the status of the export.
//
// [Ja] WithStatus は状態を設定します
func (b *ExportBuilder) WithStatus(status model.ExportStatus) *ExportBuilder {
	b.status = int32(status)
	return b
}

// WithHeartbeatAt sets the heartbeat of the worker.
//
// [Ja] WithHeartbeatAt はワーカーの heartbeat を設定します
func (b *ExportBuilder) WithHeartbeatAt(heartbeatAt time.Time) *ExportBuilder {
	b.heartbeatAt = &heartbeatAt
	return b
}

// WithObjectKey sets the key of the ZIP object.
//
// [Ja] WithObjectKey は ZIP オブジェクトのキーを設定します
func (b *ExportBuilder) WithObjectKey(objectKey string) *ExportBuilder {
	b.objectKey = &objectKey
	return b
}

// WithCreatedAt sets when the export was created, which is what orders the exports of a space.
//
// [Ja] WithCreatedAt はエクスポートの作成時刻を設定します。スペースのエクスポートの並び順は
// この値で決まります
func (b *ExportBuilder) WithCreatedAt(createdAt time.Time) *ExportBuilder {
	b.createdAt = createdAt
	return b
}

// WithStatusChangedAt sets when the export entered its status. A queued export is told apart from
// one still waiting for a worker by this value, so a test about that needs to place it.
//
// [Ja] WithStatusChangedAt はエクスポートがその状態になった時刻を設定します。まだワーカーを
// 待っている queued かどうかはこの値で決まるため、それを扱うテストはこの値を置く必要があります
func (b *ExportBuilder) WithStatusChangedAt(statusChangedAt time.Time) *ExportBuilder {
	b.statusChangedAt = statusChangedAt
	return b
}

// Build creates the export and returns its ID.
//
// [Ja] Build はエクスポートを作成し、IDを返します
func (b *ExportBuilder) Build() model.ExportID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("ExportBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.queuedByID == "" {
		b.t.Fatal("ExportBuilder: queuedByIDが設定されていません。WithQueuedByID()を呼んでください")
	}

	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO exports (space_id, queued_by_id, status, status_changed_at, heartbeat_at, object_key, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		 RETURNING id`,
		b.spaceID, b.queuedByID, b.status, b.statusChangedAt, b.heartbeatAt, b.objectKey, b.createdAt,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("エクスポート作成に失敗: %v", err)
	}

	return model.ExportID(id)
}

// ExportBuilderDB builds export test data directly against the database.
// Used by the tests of UseCases that manage their own transactions.
//
// [Ja] ExportBuilderDB はDBを直接使用するエクスポートテストデータのビルダー
// トランザクション管理を自前で行うUsecaseのテストに使用します
type ExportBuilderDB struct {
	t  *testing.T
	db *sql.DB

	spaceID         string
	queuedByID      string
	status          int32
	statusChangedAt time.Time
	heartbeatAt     *time.Time
	objectKey       *string
	createdAt       time.Time
}

// NewExportBuilderDB builds an ExportBuilderDB.
//
// [Ja] NewExportBuilderDB は ExportBuilderDB を生成します
func NewExportBuilderDB(t *testing.T, db *sql.DB) *ExportBuilderDB {
	t.Helper()
	now := time.Now()
	return &ExportBuilderDB{
		t:               t,
		db:              db,
		status:          int32(model.ExportStatusQueued),
		statusChangedAt: now,
		createdAt:       now,
	}
}

// WithSpaceID sets the space.
//
// [Ja] WithSpaceID はスペースIDを設定します
func (b *ExportBuilderDB) WithSpaceID(spaceID model.SpaceID) *ExportBuilderDB {
	b.spaceID = string(spaceID)
	return b
}

// WithQueuedByID sets the space member who started the export.
//
// [Ja] WithQueuedByID はエクスポートを開始したスペースメンバーのIDを設定します
func (b *ExportBuilderDB) WithQueuedByID(id model.SpaceMemberID) *ExportBuilderDB {
	b.queuedByID = string(id)
	return b
}

// WithStatus sets the status of the export.
//
// [Ja] WithStatus は状態を設定します
func (b *ExportBuilderDB) WithStatus(status model.ExportStatus) *ExportBuilderDB {
	b.status = int32(status)
	return b
}

// WithHeartbeatAt sets the heartbeat of the worker.
//
// [Ja] WithHeartbeatAt はワーカーの heartbeat を設定します
func (b *ExportBuilderDB) WithHeartbeatAt(heartbeatAt time.Time) *ExportBuilderDB {
	b.heartbeatAt = &heartbeatAt
	return b
}

// WithObjectKey sets the key of the ZIP object.
//
// [Ja] WithObjectKey は ZIP オブジェクトのキーを設定します
func (b *ExportBuilderDB) WithObjectKey(objectKey string) *ExportBuilderDB {
	b.objectKey = &objectKey
	return b
}

// WithCreatedAt sets when the export was created, which is what orders the exports of a space.
//
// [Ja] WithCreatedAt はエクスポートの作成時刻を設定します。スペースのエクスポートの並び順は
// この値で決まります
func (b *ExportBuilderDB) WithCreatedAt(createdAt time.Time) *ExportBuilderDB {
	b.createdAt = createdAt
	return b
}

// WithStatusChangedAt sets when the export entered its status. A queued export is told apart from
// one still waiting for a worker by this value, so a test about that needs to place it.
//
// [Ja] WithStatusChangedAt はエクスポートがその状態になった時刻を設定します。まだワーカーを
// 待っている queued かどうかはこの値で決まるため、それを扱うテストはこの値を置く必要があります
func (b *ExportBuilderDB) WithStatusChangedAt(statusChangedAt time.Time) *ExportBuilderDB {
	b.statusChangedAt = statusChangedAt
	return b
}

// Build creates the export and returns its ID.
//
// [Ja] Build はエクスポートを作成し、IDを返します
func (b *ExportBuilderDB) Build() model.ExportID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("ExportBuilderDB: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.queuedByID == "" {
		b.t.Fatal("ExportBuilderDB: queuedByIDが設定されていません。WithQueuedByID()を呼んでください")
	}

	var id string
	err := b.db.QueryRowContext(
		context.Background(),
		`INSERT INTO exports (space_id, queued_by_id, status, status_changed_at, heartbeat_at, object_key, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		 RETURNING id`,
		b.spaceID, b.queuedByID, b.status, b.statusChangedAt, b.heartbeatAt, b.objectKey, b.createdAt,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("エクスポート作成に失敗: %v", err)
	}

	return model.ExportID(id)
}
