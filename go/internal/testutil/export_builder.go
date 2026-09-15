package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// ExportBuilderはエクスポートテストデータのビルダー
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

// NewExportBuilderはExportBuilderを生成します
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

// WithSpaceIDはスペースIDを設定します
func (b *ExportBuilder) WithSpaceID(spaceID model.SpaceID) *ExportBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithQueuedByIDはエクスポートを開始したスペースメンバーのIDを設定します
func (b *ExportBuilder) WithQueuedByID(id model.SpaceMemberID) *ExportBuilder {
	b.queuedByID = string(id)
	return b
}

// WithStatusは状態を設定します
func (b *ExportBuilder) WithStatus(status model.ExportStatus) *ExportBuilder {
	b.status = int32(status)
	return b
}

// WithHeartbeatAtはワーカーのheartbeatを設定します
func (b *ExportBuilder) WithHeartbeatAt(heartbeatAt time.Time) *ExportBuilder {
	b.heartbeatAt = &heartbeatAt
	return b
}

// WithObjectKeyはZIPオブジェクトのキーを設定します
func (b *ExportBuilder) WithObjectKey(objectKey string) *ExportBuilder {
	b.objectKey = &objectKey
	return b
}

// WithCreatedAtはエクスポートの作成時刻を設定します。スペースのエクスポートの並び順は
// この値で決まります
func (b *ExportBuilder) WithCreatedAt(createdAt time.Time) *ExportBuilder {
	b.createdAt = createdAt
	return b
}

// WithStatusChangedAtはエクスポートがその状態になった時刻を設定します。まだワーカーを
// 待っているqueuedかどうかはこの値で決まるため、それを扱うテストはこの値を置く必要があります
func (b *ExportBuilder) WithStatusChangedAt(statusChangedAt time.Time) *ExportBuilder {
	b.statusChangedAt = statusChangedAt
	return b
}

// Buildはエクスポートを作成し、IDを返します
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

// ExportBuilderDBはDBを直接使用するエクスポートテストデータのビルダー
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

// NewExportBuilderDBはExportBuilderDBを生成します
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

// WithSpaceIDはスペースIDを設定します
func (b *ExportBuilderDB) WithSpaceID(spaceID model.SpaceID) *ExportBuilderDB {
	b.spaceID = string(spaceID)
	return b
}

// WithQueuedByIDはエクスポートを開始したスペースメンバーのIDを設定します
func (b *ExportBuilderDB) WithQueuedByID(id model.SpaceMemberID) *ExportBuilderDB {
	b.queuedByID = string(id)
	return b
}

// WithStatusは状態を設定します
func (b *ExportBuilderDB) WithStatus(status model.ExportStatus) *ExportBuilderDB {
	b.status = int32(status)
	return b
}

// WithHeartbeatAtはワーカーのheartbeatを設定します
func (b *ExportBuilderDB) WithHeartbeatAt(heartbeatAt time.Time) *ExportBuilderDB {
	b.heartbeatAt = &heartbeatAt
	return b
}

// WithObjectKeyはZIPオブジェクトのキーを設定します
func (b *ExportBuilderDB) WithObjectKey(objectKey string) *ExportBuilderDB {
	b.objectKey = &objectKey
	return b
}

// WithCreatedAtはエクスポートの作成時刻を設定します。スペースのエクスポートの並び順は
// この値で決まります
func (b *ExportBuilderDB) WithCreatedAt(createdAt time.Time) *ExportBuilderDB {
	b.createdAt = createdAt
	return b
}

// WithStatusChangedAtはエクスポートがその状態になった時刻を設定します。まだワーカーを
// 待っているqueuedかどうかはこの値で決まるため、それを扱うテストはこの値を置く必要があります
func (b *ExportBuilderDB) WithStatusChangedAt(statusChangedAt time.Time) *ExportBuilderDB {
	b.statusChangedAt = statusChangedAt
	return b
}

// Buildはエクスポートを作成し、IDを返します
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
