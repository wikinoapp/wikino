package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// DraftPageRevisionBuilderDB はDBを直接使用する下書きページリビジョンテストデータのビルダー
type DraftPageRevisionBuilderDB struct {
	t  *testing.T
	db *sql.DB

	draftPageID   string
	spaceID       string
	spaceMemberID string
	title         string
	body          string
	bodyHTML      string
}

// NewDraftPageRevisionBuilderDB は DraftPageRevisionBuilderDB を生成します
func NewDraftPageRevisionBuilderDB(t *testing.T, db *sql.DB) *DraftPageRevisionBuilderDB {
	t.Helper()
	return &DraftPageRevisionBuilderDB{
		t:        t,
		db:       db,
		title:    "Draft Revision Title",
		body:     "Draft revision body",
		bodyHTML: "<p>Draft revision body</p>",
	}
}

// WithDraftPageID は下書きページIDを設定します
func (b *DraftPageRevisionBuilderDB) WithDraftPageID(draftPageID model.DraftPageID) *DraftPageRevisionBuilderDB {
	b.draftPageID = string(draftPageID)
	return b
}

// WithSpaceID はスペースIDを設定します
func (b *DraftPageRevisionBuilderDB) WithSpaceID(spaceID model.SpaceID) *DraftPageRevisionBuilderDB {
	b.spaceID = string(spaceID)
	return b
}

// WithSpaceMemberID はスペースメンバーIDを設定します
func (b *DraftPageRevisionBuilderDB) WithSpaceMemberID(spaceMemberID model.SpaceMemberID) *DraftPageRevisionBuilderDB {
	b.spaceMemberID = string(spaceMemberID)
	return b
}

// Build は下書きページリビジョンを作成し、IDを返します
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
		`INSERT INTO draft_page_revisions (draft_page_id, space_id, space_member_id, title, body, body_html, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		b.draftPageID, b.spaceID, b.spaceMemberID, b.title, b.body, b.bodyHTML, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("下書きページリビジョン作成に失敗: %v", err)
	}

	return model.DraftPageRevisionID(id)
}
