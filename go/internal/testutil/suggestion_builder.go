package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// SuggestionBuilder は編集提案テストデータのビルダー
type SuggestionBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID              string
	topicID              string
	createdSpaceMemberID string
	title                string
	body                 string
	bodyHTML             string
	status               int32
	appliedAt            *time.Time
}

// NewSuggestionBuilder は SuggestionBuilder を生成します
func NewSuggestionBuilder(t *testing.T, tx *sql.Tx) *SuggestionBuilder {
	t.Helper()
	return &SuggestionBuilder{
		t:        t,
		tx:       tx,
		title:    "テスト編集提案",
		body:     "テスト本文",
		bodyHTML: "<p>テスト本文</p>",
		status:   0, // Draft
	}
}

// WithSpaceID はスペースIDを設定します
func (b *SuggestionBuilder) WithSpaceID(spaceID model.SpaceID) *SuggestionBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithTopicID はトピックIDを設定します
func (b *SuggestionBuilder) WithTopicID(topicID model.TopicID) *SuggestionBuilder {
	b.topicID = string(topicID)
	return b
}

// WithCreatedSpaceMemberID は作成者のスペースメンバーIDを設定します
func (b *SuggestionBuilder) WithCreatedSpaceMemberID(id model.SpaceMemberID) *SuggestionBuilder {
	b.createdSpaceMemberID = string(id)
	return b
}

// WithTitle はタイトルを設定します
func (b *SuggestionBuilder) WithTitle(title string) *SuggestionBuilder {
	b.title = title
	return b
}

// WithBody は本文を設定します
func (b *SuggestionBuilder) WithBody(body string) *SuggestionBuilder {
	b.body = body
	return b
}

// WithBodyHTML はHTML本文を設定します
func (b *SuggestionBuilder) WithBodyHTML(bodyHTML string) *SuggestionBuilder {
	b.bodyHTML = bodyHTML
	return b
}

// WithStatus はステータスを設定します
func (b *SuggestionBuilder) WithStatus(status model.SuggestionStatus) *SuggestionBuilder {
	b.status = int32(status)
	return b
}

// Build は編集提案を作成し、IDを返します
func (b *SuggestionBuilder) Build() model.SuggestionID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("SuggestionBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.topicID == "" {
		b.t.Fatal("SuggestionBuilder: topicIDが設定されていません。WithTopicID()を呼んでください")
	}
	if b.createdSpaceMemberID == "" {
		b.t.Fatal("SuggestionBuilder: createdSpaceMemberIDが設定されていません。WithCreatedSpaceMemberID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO suggestions (space_id, topic_id, created_space_member_id, title, body, body_html, status, applied_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id`,
		b.spaceID, b.topicID, b.createdSpaceMemberID, b.title, b.body, b.bodyHTML, b.status, b.appliedAt, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("編集提案作成に失敗: %v", err)
	}

	return model.SuggestionID(id)
}
