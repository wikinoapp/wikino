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
