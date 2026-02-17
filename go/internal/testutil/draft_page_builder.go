package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// DraftPageBuilder は下書きページテストデータのビルダー
type DraftPageBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID       string
	pageID        string
	spaceMemberID string
	topicID       string
	title         *string
	body          string
	bodyHTML      string
	linkedPageIDs []string
	modifiedAt    time.Time
}

// NewDraftPageBuilder は DraftPageBuilder を生成します
func NewDraftPageBuilder(t *testing.T, tx *sql.Tx) *DraftPageBuilder {
	t.Helper()
	now := time.Now()
	title := "Draft Title"
	return &DraftPageBuilder{
		t:             t,
		tx:            tx,
		title:         &title,
		body:          "Draft body",
		bodyHTML:      "<p>Draft body</p>",
		linkedPageIDs: []string{},
		modifiedAt:    now,
	}
}

// WithSpaceID はスペースIDを設定します
func (b *DraftPageBuilder) WithSpaceID(spaceID model.SpaceID) *DraftPageBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithPageID はページIDを設定します
func (b *DraftPageBuilder) WithPageID(pageID model.PageID) *DraftPageBuilder {
	b.pageID = string(pageID)
	return b
}

// WithSpaceMemberID はスペースメンバーIDを設定します
func (b *DraftPageBuilder) WithSpaceMemberID(spaceMemberID model.SpaceMemberID) *DraftPageBuilder {
	b.spaceMemberID = string(spaceMemberID)
	return b
}

// WithTopicID はトピックIDを設定します
func (b *DraftPageBuilder) WithTopicID(topicID model.TopicID) *DraftPageBuilder {
	b.topicID = string(topicID)
	return b
}

// WithTitle はタイトルを設定します
func (b *DraftPageBuilder) WithTitle(title string) *DraftPageBuilder {
	b.title = &title
	return b
}

// WithNilTitle はタイトルをnilに設定します
func (b *DraftPageBuilder) WithNilTitle() *DraftPageBuilder {
	b.title = nil
	return b
}

// WithBody は本文を設定します
func (b *DraftPageBuilder) WithBody(body string) *DraftPageBuilder {
	b.body = body
	return b
}

// WithBodyHTML はHTML本文を設定します
func (b *DraftPageBuilder) WithBodyHTML(bodyHTML string) *DraftPageBuilder {
	b.bodyHTML = bodyHTML
	return b
}

// WithLinkedPageIDs はリンク先ページIDリストを設定します
func (b *DraftPageBuilder) WithLinkedPageIDs(ids []model.PageID) *DraftPageBuilder {
	b.linkedPageIDs = model.PageIDsToStrings(ids)
	return b
}

// Build は下書きページを作成し、IDを返します
func (b *DraftPageBuilder) Build() model.DraftPageID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("DraftPageBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.pageID == "" {
		b.t.Fatal("DraftPageBuilder: pageIDが設定されていません。WithPageID()を呼んでください")
	}
	if b.spaceMemberID == "" {
		b.t.Fatal("DraftPageBuilder: spaceMemberIDが設定されていません。WithSpaceMemberID()を呼んでください")
	}
	if b.topicID == "" {
		b.t.Fatal("DraftPageBuilder: topicIDが設定されていません。WithTopicID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO draft_pages (space_id, page_id, space_member_id, topic_id, title, body, body_html, linked_page_ids, modified_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id`,
		b.spaceID, b.pageID, b.spaceMemberID, b.topicID, b.title, b.body, b.bodyHTML,
		pq.Array(b.linkedPageIDs), b.modifiedAt, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("下書きページ作成に失敗: %v", err)
	}

	return model.DraftPageID(id)
}
