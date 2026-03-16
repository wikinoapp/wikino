package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// SuggestionCommentBuilder は編集提案コメントテストデータのビルダー
type SuggestionCommentBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID              string
	suggestionID         string
	createdSpaceMemberID string
	body                 string
	bodyHTML             string
}

// NewSuggestionCommentBuilder は SuggestionCommentBuilder を生成します
func NewSuggestionCommentBuilder(t *testing.T, tx *sql.Tx) *SuggestionCommentBuilder {
	t.Helper()
	return &SuggestionCommentBuilder{
		t:        t,
		tx:       tx,
		body:     "テストコメント",
		bodyHTML: "<p>テストコメント</p>",
	}
}

// WithSpaceID はスペースIDを設定します
func (b *SuggestionCommentBuilder) WithSpaceID(spaceID model.SpaceID) *SuggestionCommentBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithSuggestionID は編集提案IDを設定します
func (b *SuggestionCommentBuilder) WithSuggestionID(suggestionID model.SuggestionID) *SuggestionCommentBuilder {
	b.suggestionID = string(suggestionID)
	return b
}

// WithCreatedSpaceMemberID は作成者のスペースメンバーIDを設定します
func (b *SuggestionCommentBuilder) WithCreatedSpaceMemberID(createdSpaceMemberID model.SpaceMemberID) *SuggestionCommentBuilder {
	b.createdSpaceMemberID = string(createdSpaceMemberID)
	return b
}

// WithBody は本文を設定します
func (b *SuggestionCommentBuilder) WithBody(body string) *SuggestionCommentBuilder {
	b.body = body
	return b
}

// WithBodyHTML はHTML本文を設定します
func (b *SuggestionCommentBuilder) WithBodyHTML(bodyHTML string) *SuggestionCommentBuilder {
	b.bodyHTML = bodyHTML
	return b
}

// Build は編集提案コメントを作成し、IDを返します
func (b *SuggestionCommentBuilder) Build() model.SuggestionCommentID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("SuggestionCommentBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.suggestionID == "" {
		b.t.Fatal("SuggestionCommentBuilder: suggestionIDが設定されていません。WithSuggestionID()を呼んでください")
	}
	if b.createdSpaceMemberID == "" {
		b.t.Fatal("SuggestionCommentBuilder: createdSpaceMemberIDが設定されていません。WithCreatedSpaceMemberID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO suggestion_comments (space_id, suggestion_id, created_space_member_id, body, body_html, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		b.spaceID, b.suggestionID, b.createdSpaceMemberID, b.body, b.bodyHTML, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("編集提案コメント作成に失敗: %v", err)
	}

	return model.SuggestionCommentID(id)
}
