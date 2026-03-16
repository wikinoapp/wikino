package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// SuggestionPageBuilder は編集提案ページテストデータのビルダー
type SuggestionPageBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID        string
	suggestionID   string
	pageID         string
	pageRevisionID string
	title          *string
	body           string
	bodyHTML       string
}

// NewSuggestionPageBuilder は SuggestionPageBuilder を生成します
func NewSuggestionPageBuilder(t *testing.T, tx *sql.Tx) *SuggestionPageBuilder {
	t.Helper()
	title := "テスト提案ページ"
	return &SuggestionPageBuilder{
		t:        t,
		tx:       tx,
		title:    &title,
		body:     "テスト本文",
		bodyHTML: "<p>テスト本文</p>",
	}
}

// WithSpaceID はスペースIDを設定します
func (b *SuggestionPageBuilder) WithSpaceID(spaceID model.SpaceID) *SuggestionPageBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithSuggestionID は編集提案IDを設定します
func (b *SuggestionPageBuilder) WithSuggestionID(suggestionID model.SuggestionID) *SuggestionPageBuilder {
	b.suggestionID = string(suggestionID)
	return b
}

// WithPageID はページIDを設定します
func (b *SuggestionPageBuilder) WithPageID(pageID model.PageID) *SuggestionPageBuilder {
	b.pageID = string(pageID)
	return b
}

// WithPageRevisionID はページリビジョンIDを設定します
func (b *SuggestionPageBuilder) WithPageRevisionID(pageRevisionID model.PageRevisionID) *SuggestionPageBuilder {
	b.pageRevisionID = string(pageRevisionID)
	return b
}

// WithTitle はタイトルを設定します
func (b *SuggestionPageBuilder) WithTitle(title string) *SuggestionPageBuilder {
	b.title = &title
	return b
}

// WithNilTitle はタイトルをnilに設定します
func (b *SuggestionPageBuilder) WithNilTitle() *SuggestionPageBuilder {
	b.title = nil
	return b
}

// WithBody は本文を設定します
func (b *SuggestionPageBuilder) WithBody(body string) *SuggestionPageBuilder {
	b.body = body
	return b
}

// WithBodyHTML はHTML本文を設定します
func (b *SuggestionPageBuilder) WithBodyHTML(bodyHTML string) *SuggestionPageBuilder {
	b.bodyHTML = bodyHTML
	return b
}

// Build は編集提案ページを作成し、IDを返します
func (b *SuggestionPageBuilder) Build() model.SuggestionPageID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("SuggestionPageBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.suggestionID == "" {
		b.t.Fatal("SuggestionPageBuilder: suggestionIDが設定されていません。WithSuggestionID()を呼んでください")
	}
	if b.pageID == "" {
		b.t.Fatal("SuggestionPageBuilder: pageIDが設定されていません。WithPageID()を呼んでください")
	}
	if b.pageRevisionID == "" {
		b.t.Fatal("SuggestionPageBuilder: pageRevisionIDが設定されていません。WithPageRevisionID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO suggestion_pages (space_id, suggestion_id, page_id, page_revision_id, title, body, body_html, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id`,
		b.spaceID, b.suggestionID, b.pageID, b.pageRevisionID, b.title, b.body, b.bodyHTML, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("編集提案ページ作成に失敗: %v", err)
	}

	return model.SuggestionPageID(id)
}
