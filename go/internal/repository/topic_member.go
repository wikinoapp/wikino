package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// TopicMemberRepositoryはトピックメンバーリポジトリ
type TopicMemberRepository struct {
	q *query.Queries
}

// NewTopicMemberRepositoryはTopicMemberRepositoryを生成する
func NewTopicMemberRepository(q *query.Queries) *TopicMemberRepository {
	return &TopicMemberRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *TopicMemberRepository) WithTx(tx *sql.Tx) *TopicMemberRepository {
	return &TopicMemberRepository{q: r.q.WithTx(tx)}
}

// FindBySpaceMemberAndTopicはスペースメンバーIDとトピックIDでトピックメンバーを取得する
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

// ListBySpaceMemberAndTopicsは、スペースメンバーが参加しているトピックメンバーを
// トピックIDリストで一括取得する。スペース詳細のように複数トピックの権限をまとめて
// 判定する場面で、トピックごとの単発クエリ (N+1) を1回のクエリに置き換えるために使う。
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

// ListByUserAndTopicsは、ユーザーが指定したトピック群で持つトピックメンバーを1クエリで
// 一括取得する (space_membersとJOINしてユーザーを解決する)。ホーム画面のように複数スペースに
// またがるトピックの作成権限をまとめて判定する場面で、トピックごとの単発クエリ (N+1) を1回の
// クエリに置き換えるために使う。
func (r *TopicMemberRepository) ListByUserAndTopics(ctx context.Context, userID model.UserID, spaceIDs []model.SpaceID, topicIDs []model.TopicID) ([]*model.TopicMember, error) {
	if len(spaceIDs) == 0 || len(topicIDs) == 0 {
		return nil, nil
	}
	rows, err := r.q.ListTopicMembersByUserAndTopics(ctx, query.ListTopicMembersByUserAndTopicsParams{
		UserID:  string(userID),
		Column2: model.SpaceIDsToStrings(spaceIDs),
		Column3: model.TopicIDsToStrings(topicIDs),
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

// UpdateLastPageModifiedAtはトピックメンバーのlast_page_modified_atを更新する
func (r *TopicMemberRepository) UpdateLastPageModifiedAt(ctx context.Context, spaceID model.SpaceID, topicID model.TopicID, spaceMemberID model.SpaceMemberID, modifiedAt time.Time) error {
	return r.q.UpdateTopicMemberLastPageModifiedAt(ctx, query.UpdateTopicMemberLastPageModifiedAtParams{
		LastPageModifiedAt: sql.NullTime{Time: modifiedAt, Valid: true},
		UpdatedAt:          time.Now(),
		TopicID:            string(topicID),
		SpaceMemberID:      string(spaceMemberID),
		SpaceID:            string(spaceID),
	})
}

// CreateTopicMemberInputはトピックメンバーの作成に必要な値を保持する。
type CreateTopicMemberInput struct {
	SpaceID       model.SpaceID
	TopicID       model.TopicID
	SpaceMemberID model.SpaceMemberID
}

// Createはスペースメンバーをトピックに参加させる。
func (r *TopicMemberRepository) Create(ctx context.Context, input CreateTopicMemberInput) (*model.TopicMember, error) {
	row, err := r.q.CreateTopicMember(ctx, query.CreateTopicMemberParams{
		SpaceID:       string(input.SpaceID),
		TopicID:       string(input.TopicID),
		SpaceMemberID: string(input.SpaceMemberID),
		Now:           time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// toModelはquery.TopicMemberをmodel.TopicMemberに変換する
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
