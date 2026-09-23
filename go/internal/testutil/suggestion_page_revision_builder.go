package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// SuggestionPageRevisionBuilderは編集提案ページリビジョンテストデータのビルダー
type SuggestionPageRevisionBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID             string
	suggestionPageID    string
	editorSpaceMemberID string
	title               *string
	body                string
}

// NewSuggestionPageRevisionBuilderはSuggestionPageRevisionBuilderを生成します
func NewSuggestionPageRevisionBuilder(t *testing.T, tx *sql.Tx) *SuggestionPageRevisionBuilder {
	t.Helper()
	title := "テストリビジョン"
	return &SuggestionPageRevisionBuilder{
		t:     t,
		tx:    tx,
		title: &title,
		body:  "テストリビジョン本文",
	}
}

// WithSpaceIDはスペースIDを設定します
func (b *SuggestionPageRevisionBuilder) WithSpaceID(spaceID model.SpaceID) *SuggestionPageRevisionBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithSuggestionPageIDは編集提案ページIDを設定します
func (b *SuggestionPageRevisionBuilder) WithSuggestionPageID(suggestionPageID model.SuggestionPageID) *SuggestionPageRevisionBuilder {
	b.suggestionPageID = string(suggestionPageID)
	return b
}

// WithEditorSpaceMemberIDは編集者のスペースメンバーIDを設定します
func (b *SuggestionPageRevisionBuilder) WithEditorSpaceMemberID(editorSpaceMemberID model.SpaceMemberID) *SuggestionPageRevisionBuilder {
	b.editorSpaceMemberID = string(editorSpaceMemberID)
	return b
}

// WithTitleはタイトルを設定します
func (b *SuggestionPageRevisionBuilder) WithTitle(title string) *SuggestionPageRevisionBuilder {
	b.title = &title
	return b
}

// WithNilTitleはタイトルをnilに設定します
func (b *SuggestionPageRevisionBuilder) WithNilTitle() *SuggestionPageRevisionBuilder {
	b.title = nil
	return b
}

// WithBodyは本文を設定します
func (b *SuggestionPageRevisionBuilder) WithBody(body string) *SuggestionPageRevisionBuilder {
	b.body = body
	return b
}

// Buildは編集提案ページリビジョンを作成し、IDを返します
func (b *SuggestionPageRevisionBuilder) Build() model.SuggestionPageRevisionID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("SuggestionPageRevisionBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.suggestionPageID == "" {
		b.t.Fatal("SuggestionPageRevisionBuilder: suggestionPageIDが設定されていません。WithSuggestionPageID()を呼んでください")
	}
	if b.editorSpaceMemberID == "" {
		b.t.Fatal("SuggestionPageRevisionBuilder: editorSpaceMemberIDが設定されていません。WithEditorSpaceMemberID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO suggestion_page_revisions (space_id, suggestion_page_id, editor_space_member_id, title, body, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		b.spaceID, b.suggestionPageID, b.editorSpaceMemberID, b.title, b.body, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("編集提案ページリビジョン作成に失敗: %v", err)
	}

	return model.SuggestionPageRevisionID(id)
}
