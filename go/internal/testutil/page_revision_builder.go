package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// PageRevisionBuilder はページリビジョンテストデータのビルダー
type PageRevisionBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID       string
	spaceMemberID string
	pageID        string
	title         string
	body          string
	bodyHTML      string
}

// NewPageRevisionBuilder は PageRevisionBuilder を生成します
func NewPageRevisionBuilder(t *testing.T, tx *sql.Tx) *PageRevisionBuilder {
	t.Helper()
	return &PageRevisionBuilder{
		t:        t,
		tx:       tx,
		title:    "Revision Title",
		body:     "Revision body",
		bodyHTML: "<p>Revision body</p>",
	}
}

// WithSpaceID はスペースIDを設定します
func (b *PageRevisionBuilder) WithSpaceID(spaceID model.SpaceID) *PageRevisionBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithSpaceMemberID はスペースメンバーIDを設定します
func (b *PageRevisionBuilder) WithSpaceMemberID(spaceMemberID model.SpaceMemberID) *PageRevisionBuilder {
	b.spaceMemberID = string(spaceMemberID)
	return b
}

// WithPageID はページIDを設定します
func (b *PageRevisionBuilder) WithPageID(pageID model.PageID) *PageRevisionBuilder {
	b.pageID = string(pageID)
	return b
}

// WithTitle はタイトルを設定します
func (b *PageRevisionBuilder) WithTitle(title string) *PageRevisionBuilder {
	b.title = title
	return b
}

// WithBody は本文を設定します
func (b *PageRevisionBuilder) WithBody(body string) *PageRevisionBuilder {
	b.body = body
	return b
}

// WithBodyHTML はHTML本文を設定します
func (b *PageRevisionBuilder) WithBodyHTML(bodyHTML string) *PageRevisionBuilder {
	b.bodyHTML = bodyHTML
	return b
}

// Build はページリビジョンを作成し、IDを返します
func (b *PageRevisionBuilder) Build() string {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("PageRevisionBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.spaceMemberID == "" {
		b.t.Fatal("PageRevisionBuilder: spaceMemberIDが設定されていません。WithSpaceMemberID()を呼んでください")
	}
	if b.pageID == "" {
		b.t.Fatal("PageRevisionBuilder: pageIDが設定されていません。WithPageID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO page_revisions (space_id, space_member_id, page_id, title, body, body_html, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id`,
		b.spaceID, b.spaceMemberID, b.pageID, b.title, b.body, b.bodyHTML, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("ページリビジョン作成に失敗: %v", err)
	}

	return id
}
