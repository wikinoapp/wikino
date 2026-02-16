package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// TopicMemberRepository はトピックメンバーリポジトリ
type TopicMemberRepository struct {
	q *query.Queries
}

// NewTopicMemberRepository は TopicMemberRepository を生成する
func NewTopicMemberRepository(q *query.Queries) *TopicMemberRepository {
	return &TopicMemberRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *TopicMemberRepository) WithTx(tx *sql.Tx) *TopicMemberRepository {
	return &TopicMemberRepository{q: r.q.WithTx(tx)}
}

// FindBySpaceMemberAndTopic はスペースメンバーIDとトピックIDでトピックメンバーを取得する
func (r *TopicMemberRepository) FindBySpaceMemberAndTopic(ctx context.Context, spaceMemberID model.SpaceMemberID, topicID model.TopicID) (*model.TopicMember, error) {
	row, err := r.q.FindTopicMemberBySpaceMemberAndTopic(ctx, query.FindTopicMemberBySpaceMemberAndTopicParams{
		SpaceMemberID: string(spaceMemberID),
		TopicID:       string(topicID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// UpdateLastPageModifiedAt はトピックメンバーのlast_page_modified_atを更新する
func (r *TopicMemberRepository) UpdateLastPageModifiedAt(ctx context.Context, topicID model.TopicID, spaceMemberID model.SpaceMemberID, modifiedAt time.Time) error {
	return r.q.UpdateTopicMemberLastPageModifiedAt(ctx, query.UpdateTopicMemberLastPageModifiedAtParams{
		LastPageModifiedAt: sql.NullTime{Time: modifiedAt, Valid: true},
		UpdatedAt:          time.Now(),
		TopicID:            string(topicID),
		SpaceMemberID:      string(spaceMemberID),
	})
}

// toModel は query.TopicMember を model.TopicMember に変換する
func (r *TopicMemberRepository) toModel(row query.TopicMember) *model.TopicMember {
	var lastPageModifiedAt *time.Time
	if row.LastPageModifiedAt.Valid {
		lastPageModifiedAt = &row.LastPageModifiedAt.Time
	}

	return &model.TopicMember{
		ID:                 model.TopicMemberID(row.ID),
		SpaceID:            model.SpaceID(row.SpaceID),
		TopicID:            model.TopicID(row.TopicID),
		SpaceMemberID:      model.SpaceMemberID(row.SpaceMemberID),
		Role:               model.TopicMemberRole(row.Role),
		JoinedAt:           row.JoinedAt,
		LastPageModifiedAt: lastPageModifiedAt,
	}
}
