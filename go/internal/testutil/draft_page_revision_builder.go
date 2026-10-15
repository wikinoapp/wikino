package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// DraftPageRevisionBuilderは下書きページリビジョンテストデータのビルダー
type DraftPageRevisionBuilder struct {
	t  *testing.T
	tx *sql.Tx

	draftPageID   string
	spaceID       string
	spaceMemberID string
	title         string
	body          string
	createdAt     time.Time
}

// NewDraftPageRevisionBuilderはDraftPageRevisionBuilderを生成します
func NewDraftPageRevisionBuilder(t *testing.T, tx *sql.Tx) *DraftPageRevisionBuilder {
	t.Helper()
	return &DraftPageRevisionBuilder{
		t:         t,
		tx:        tx,
		title:     "Draft Revision Title",
		body:      "Draft revision body",
		createdAt: time.Now(),
	}
}

// WithDraftPageIDは下書きページIDを設定します
func (b *DraftPageRevisionBuilder) WithDraftPageID(draftPageID model.DraftPageID) *DraftPageRevisionBuilder {
	b.draftPageID = string(draftPageID)
	return b
}

// WithSpaceIDはスペースIDを設定します
func (b *DraftPageRevisionBuilder) WithSpaceID(spaceID model.SpaceID) *DraftPageRevisionBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithSpaceMemberIDはスペースメンバーIDを設定します
func (b *DraftPageRevisionBuilder) WithSpaceMemberID(spaceMemberID model.SpaceMemberID) *DraftPageRevisionBuilder {
	b.spaceMemberID = string(spaceMemberID)
	return b
}

// WithTitleはタイトルを設定します
func (b *DraftPageRevisionBuilder) WithTitle(title string) *DraftPageRevisionBuilder {
	b.title = title
	return b
}

// WithBodyは本文を設定します
func (b *DraftPageRevisionBuilder) WithBody(body string) *DraftPageRevisionBuilder {
	b.body = body
	return b
}

// WithCreatedAtは作成日時を設定します
func (b *DraftPageRevisionBuilder) WithCreatedAt(createdAt time.Time) *DraftPageRevisionBuilder {
	b.createdAt = createdAt
	return b
}

// Buildは下書きページリビジョンを作成し、IDを返します
func (b *DraftPageRevisionBuilder) Build() model.DraftPageRevisionID {
	b.t.Helper()

	if b.draftPageID == "" {
		b.t.Fatal("DraftPageRevisionBuilder: draftPageIDが設定されていません。WithDraftPageID()を呼んでください")
	}
	if b.spaceID == "" {
		b.t.Fatal("DraftPageRevisionBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.spaceMemberID == "" {
		b.t.Fatal("DraftPageRevisionBuilder: spaceMemberIDが設定されていません。WithSpaceMemberID()を呼んでください")
	}

	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO draft_page_revisions (draft_page_id, space_id, space_member_id, title, body, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		b.draftPageID, b.spaceID, b.spaceMemberID, b.title, b.body, b.createdAt,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("下書きページリビジョン作成に失敗: %v", err)
	}

	return model.DraftPageRevisionID(id)
}

// DraftPageRevisionBuilderDBはDBを直接使用する下書きページリビジョンテストデータのビルダー
type DraftPageRevisionBuilderDB struct {
	t  *testing.T
	db *sql.DB

	draftPageID   string
	spaceID       string
	spaceMemberID string
	title         string
	body          string
}

// NewDraftPageRevisionBuilderDBはDraftPageRevisionBuilderDBを生成します
func NewDraftPageRevisionBuilderDB(t *testing.T, db *sql.DB) *DraftPageRevisionBuilderDB {
	t.Helper()
	return &DraftPageRevisionBuilderDB{
		t:     t,
		db:    db,
		title: "Draft Revision Title",
		body:  "Draft revision body",
	}
}

// WithDraftPageIDは下書きページIDを設定します
func (b *DraftPageRevisionBuilderDB) WithDraftPageID(draftPageID model.DraftPageID) *DraftPageRevisionBuilderDB {
	b.draftPageID = string(draftPageID)
	return b
}

// WithSpaceIDはスペースIDを設定します
func (b *DraftPageRevisionBuilderDB) WithSpaceID(spaceID model.SpaceID) *DraftPageRevisionBuilderDB {
	b.spaceID = string(spaceID)
	return b
}

// WithSpaceMemberIDはスペースメンバーIDを設定します
func (b *DraftPageRevisionBuilderDB) WithSpaceMemberID(spaceMemberID model.SpaceMemberID) *DraftPageRevisionBuilderDB {
	b.spaceMemberID = string(spaceMemberID)
	return b
}

// Buildは下書きページリビジョンを作成し、IDを返します
func (b *DraftPageRevisionBuilderDB) Build() model.DraftPageRevisionID {
	b.t.Helper()

	if b.draftPageID == "" {
		b.t.Fatal("DraftPageRevisionBuilderDB: draftPageIDが設定されていません。WithDraftPageID()を呼んでください")
	}
	if b.spaceID == "" {
		b.t.Fatal("DraftPageRevisionBuilderDB: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.spaceMemberID == "" {
		b.t.Fatal("DraftPageRevisionBuilderDB: spaceMemberIDが設定されていません。WithSpaceMemberID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.db.QueryRowContext(
		context.Background(),
		`INSERT INTO draft_page_revisions (draft_page_id, space_id, space_member_id, title, body, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		b.draftPageID, b.spaceID, b.spaceMemberID, b.title, b.body, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("下書きページリビジョン作成に失敗: %v", err)
	}

	return model.DraftPageRevisionID(id)
}
