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
func (r *TopicMemberRepository) FindBySpaceMemberAndTopic(ctx context.Context, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, topicID model.TopicID) (*model.TopicMember, error) {
	row, err := r.q.FindTopicMemberBySpaceMemberAndTopic(ctx, query.FindTopicMemberBySpaceMemberAndTopicParams{
		SpaceMemberID: string(spaceMemberID),
		TopicID:       string(topicID),
		SpaceID:       string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// ListBySpaceMemberAndTopics fetches the topic memberships of the given space member for the
// given topic ids in a single query. It replaces per-topic lookups (N+1) with one query where
// permissions for many topics must be resolved at once, such as on the space detail page.
//
// [Ja] ListBySpaceMemberAndTopics は、スペースメンバーが参加しているトピックメンバーを
// トピックIDリストで一括取得する。スペース詳細のように複数トピックの権限をまとめて
// 判定する場面で、トピックごとの単発クエリ (N+1) を 1 回のクエリに置き換えるために使う。
func (r *TopicMemberRepository) ListBySpaceMemberAndTopics(ctx context.Context, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, topicIDs []model.TopicID) ([]*model.TopicMember, error) {
	if len(topicIDs) == 0 {
		return nil, nil
	}
	rows, err := r.q.ListTopicMembersBySpaceMemberAndTopics(ctx, query.ListTopicMembersBySpaceMemberAndTopicsParams{
		SpaceMemberID: string(spaceMemberID),
		SpaceID:       string(spaceID),
		Column3:       model.TopicIDsToStrings(topicIDs),
	})
	if err != nil {
		return nil, err
	}

	topicMembers := make([]*model.TopicMember, len(rows))
	for i, row := range rows {
		topicMembers[i] = r.toModel(row)
	}
	return topicMembers, nil
}

// UpdateLastPageModifiedAt はトピックメンバーのlast_page_modified_atを更新する
func (r *TopicMemberRepository) UpdateLastPageModifiedAt(ctx context.Context, spaceID model.SpaceID, topicID model.TopicID, spaceMemberID model.SpaceMemberID, modifiedAt time.Time) error {
	return r.q.UpdateTopicMemberLastPageModifiedAt(ctx, query.UpdateTopicMemberLastPageModifiedAtParams{
		LastPageModifiedAt: sql.NullTime{Time: modifiedAt, Valid: true},
		UpdatedAt:          time.Now(),
		TopicID:            string(topicID),
		SpaceMemberID:      string(spaceMemberID),
		SpaceID:            string(spaceID),
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
		Scopes:             model.StringsToScopes(row.Scopes),
		JoinedAt:           row.JoinedAt,
		LastPageModifiedAt: lastPageModifiedAt,
	}
}
