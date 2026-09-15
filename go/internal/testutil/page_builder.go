package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// PageBuilderはページテストデータのビルダー
type PageBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID                   string
	topicID                   string
	number                    model.PageNumber
	title                     *string
	body                      string
	bodyHTML                  string
	linkedPageIDs             []string
	modifiedAt                time.Time
	publishedAt               *time.Time
	pinnedAt                  *time.Time
	trashedAt                 *time.Time
	discardedAt               *time.Time
	featuredImageAttachmentID *string
}

// NewPageBuilderはPageBuilderを生成します
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

// WithSpaceIDはスペースIDを設定します
func (b *PageBuilder) WithSpaceID(spaceID model.SpaceID) *PageBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithTopicIDはトピックIDを設定します
func (b *PageBuilder) WithTopicID(topicID model.TopicID) *PageBuilder {
	b.topicID = string(topicID)
	return b
}

// WithNumberはページ番号を設定します
func (b *PageBuilder) WithNumber(number model.PageNumber) *PageBuilder {
	b.number = number
	return b
}

// WithTitleはタイトルを設定します
func (b *PageBuilder) WithTitle(title string) *PageBuilder {
	b.title = &title
	return b
}

// WithNilTitleはタイトルをnilに設定します
func (b *PageBuilder) WithNilTitle() *PageBuilder {
	b.title = nil
	return b
}

// WithBodyは本文を設定します
func (b *PageBuilder) WithBody(body string) *PageBuilder {
	b.body = body
	return b
}

// WithBodyHTMLはHTML本文を設定します
func (b *PageBuilder) WithBodyHTML(bodyHTML string) *PageBuilder {
	b.bodyHTML = bodyHTML
	return b
}

// WithLinkedPageIDsはリンク先ページIDリストを設定します
func (b *PageBuilder) WithLinkedPageIDs(ids []model.PageID) *PageBuilder {
	b.linkedPageIDs = model.PageIDsToStrings(ids)
	return b
}

// WithModifiedAtは更新日時を設定します
func (b *PageBuilder) WithModifiedAt(modifiedAt time.Time) *PageBuilder {
	b.modifiedAt = modifiedAt
	return b
}

// WithPublishedAtは公開日時を設定します
func (b *PageBuilder) WithPublishedAt(publishedAt time.Time) *PageBuilder {
	b.publishedAt = &publishedAt
	return b
}

// WithUnpublishedは非公開状態に設定します
func (b *PageBuilder) WithUnpublished() *PageBuilder {
	b.publishedAt = nil
	return b
}

// WithPinnedAtはピン留め日時を設定します
func (b *PageBuilder) WithPinnedAt(pinnedAt time.Time) *PageBuilder {
	b.pinnedAt = &pinnedAt
	return b
}

// WithTrashedはゴミ箱状態に設定します
func (b *PageBuilder) WithTrashed() *PageBuilder {
	now := time.Now()
	b.trashedAt = &now
	return b
}

// WithDiscardedは廃棄済み状態に設定します
func (b *PageBuilder) WithDiscarded() *PageBuilder {
	now := time.Now()
	b.discardedAt = &now
	return b
}

// WithFeaturedImageAttachmentIDはアイキャッチ画像の添付ファイルIDを設定します。
func (b *PageBuilder) WithFeaturedImageAttachmentID(id model.AttachmentID) *PageBuilder {
	s := string(id)
	b.featuredImageAttachmentID = &s
	return b
}

// Buildはページを作成し、IDを返します
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
		`INSERT INTO pages (space_id, topic_id, number, title, body, body_html, linked_page_ids, modified_at, published_at, pinned_at, trashed_at, discarded_at, featured_image_attachment_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		 RETURNING id`,
		b.spaceID, b.topicID, int32(b.number), b.title, b.body, b.bodyHTML,
		pq.Array(b.linkedPageIDs), b.modifiedAt, b.publishedAt, b.pinnedAt, b.trashedAt, b.discardedAt,
		b.featuredImageAttachmentID, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("ページ作成に失敗: %v", err)
	}

	return model.PageID(id)
}

// PageBuilderDBはDBを直接使用するページテストデータのビルダー
// トランザクション管理を自前で行うUsecaseのテストに使用します
type PageBuilderDB struct {
	t  *testing.T
	db *sql.DB

	spaceID       string
	topicID       string
	number        model.PageNumber
	title         *string
	body          string
	bodyHTML      string
	linkedPageIDs []string
	modifiedAt    time.Time
	publishedAt   *time.Time
	discardedAt   *time.Time
}

// NewPageBuilderDBはPageBuilderDBを生成します
func NewPageBuilderDB(t *testing.T, db *sql.DB) *PageBuilderDB {
	t.Helper()
	now := time.Now()
	title := "Test Page"
	return &PageBuilderDB{
		t:             t,
		db:            db,
		number:        1,
		title:         &title,
		body:          "Test body",
		bodyHTML:      "<p>Test body</p>",
		linkedPageIDs: []string{},
		modifiedAt:    now,
		publishedAt:   &now,
	}
}

// WithSpaceIDはスペースIDを設定します
func (b *PageBuilderDB) WithSpaceID(spaceID model.SpaceID) *PageBuilderDB {
	b.spaceID = string(spaceID)
	return b
}

// WithTopicIDはトピックIDを設定します
func (b *PageBuilderDB) WithTopicID(topicID model.TopicID) *PageBuilderDB {
	b.topicID = string(topicID)
	return b
}

// WithNumberはページ番号を設定します
func (b *PageBuilderDB) WithNumber(number model.PageNumber) *PageBuilderDB {
	b.number = number
	return b
}

// WithTitleはタイトルを設定します
func (b *PageBuilderDB) WithTitle(title string) *PageBuilderDB {
	b.title = &title
	return b
}

// WithBodyは本文を設定します
func (b *PageBuilderDB) WithBody(body string) *PageBuilderDB {
	b.body = body
	return b
}

// WithPublishedAtは公開日時を設定します
func (b *PageBuilderDB) WithPublishedAt(publishedAt time.Time) *PageBuilderDB {
	b.publishedAt = &publishedAt
	return b
}

// WithUnpublishedは非公開状態に設定します
func (b *PageBuilderDB) WithUnpublished() *PageBuilderDB {
	b.publishedAt = nil
	return b
}

// WithDiscardedは廃棄済み状態に設定します
func (b *PageBuilderDB) WithDiscarded() *PageBuilderDB {
	now := time.Now()
	b.discardedAt = &now
	return b
}

// Buildはページを作成し、IDを返します
func (b *PageBuilderDB) Build() model.PageID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("PageBuilderDB: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.topicID == "" {
		b.t.Fatal("PageBuilderDB: topicIDが設定されていません。WithTopicID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.db.QueryRowContext(
		context.Background(),
		`INSERT INTO pages (space_id, topic_id, number, title, body, body_html, linked_page_ids, modified_at, published_at, discarded_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING id`,
		b.spaceID, b.topicID, int32(b.number), b.title, b.body, b.bodyHTML,
		pq.Array(b.linkedPageIDs), b.modifiedAt, b.publishedAt, b.discardedAt, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("ページ作成に失敗: %v", err)
	}

	return model.PageID(id)
}
