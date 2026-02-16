package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// PageBuilder はページテストデータのビルダー
type PageBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID       string
	topicID       string
	number        int32
	title         *string
	body          string
	bodyHTML      string
	linkedPageIDs []string
	modifiedAt    time.Time
	publishedAt   *time.Time
}

// NewPageBuilder は PageBuilder を生成します
func NewPageBuilder(t *testing.T, tx *sql.Tx) *PageBuilder {
	t.Helper()
	now := time.Now()
	title := "Test Page"
	return &PageBuilder{
		t:             t,
		tx:            tx,
		number:        1,
		title:         &title,
		body:          "Test body",
		bodyHTML:      "<p>Test body</p>",
		linkedPageIDs: []string{},
		modifiedAt:    now,
		publishedAt:   &now,
	}
}

// WithSpaceID はスペースIDを設定します
func (b *PageBuilder) WithSpaceID(spaceID model.SpaceID) *PageBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithTopicID はトピックIDを設定します
func (b *PageBuilder) WithTopicID(topicID model.TopicID) *PageBuilder {
	b.topicID = string(topicID)
	return b
}

// WithNumber はページ番号を設定します
func (b *PageBuilder) WithNumber(number int32) *PageBuilder {
	b.number = number
	return b
}

// WithTitle はタイトルを設定します
func (b *PageBuilder) WithTitle(title string) *PageBuilder {
	b.title = &title
	return b
}

// WithNilTitle はタイトルをnilに設定します
func (b *PageBuilder) WithNilTitle() *PageBuilder {
	b.title = nil
	return b
}

// WithBody は本文を設定します
func (b *PageBuilder) WithBody(body string) *PageBuilder {
	b.body = body
	return b
}

// WithBodyHTML はHTML本文を設定します
func (b *PageBuilder) WithBodyHTML(bodyHTML string) *PageBuilder {
	b.bodyHTML = bodyHTML
	return b
}

// WithLinkedPageIDs はリンク先ページIDリストを設定します
func (b *PageBuilder) WithLinkedPageIDs(ids []model.PageID) *PageBuilder {
	b.linkedPageIDs = model.PageIDsToStrings(ids)
	return b
}

// WithPublishedAt は公開日時を設定します
func (b *PageBuilder) WithPublishedAt(publishedAt time.Time) *PageBuilder {
	b.publishedAt = &publishedAt
	return b
}

// WithUnpublished は非公開状態に設定します
func (b *PageBuilder) WithUnpublished() *PageBuilder {
	b.publishedAt = nil
	return b
}

// Build はページを作成し、IDを返します
func (b *PageBuilder) Build() model.PageID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("PageBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.topicID == "" {
		b.t.Fatal("PageBuilder: topicIDが設定されていません。WithTopicID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO pages (space_id, topic_id, number, title, body, body_html, linked_page_ids, modified_at, published_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id`,
		b.spaceID, b.topicID, b.number, b.title, b.body, b.bodyHTML,
		pq.Array(b.linkedPageIDs), b.modifiedAt, b.publishedAt, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("ページ作成に失敗: %v", err)
	}

	return model.PageID(id)
}
