package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// TopicRepository はトピックリポジトリ
type TopicRepository struct {
	q *query.Queries
}

// NewTopicRepository は TopicRepository を生成する
func NewTopicRepository(q *query.Queries) *TopicRepository {
	return &TopicRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *TopicRepository) WithTx(tx *sql.Tx) *TopicRepository {
	return &TopicRepository{q: r.q.WithTx(tx)}
}

// FindBySpaceAndNumber はスペースIDとナンバーでトピックを取得する（削除されていないトピックのみ）
func (r *TopicRepository) FindBySpaceAndNumber(ctx context.Context, spaceID model.SpaceID, number int32) (*model.Topic, error) {
	row, err := r.q.FindTopicBySpaceAndNumber(ctx, query.FindTopicBySpaceAndNumberParams{
		SpaceID: string(spaceID),
		Number:  number,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// ListActiveBySpace はスペースIDでアクティブなトピック一覧を取得する
func (r *TopicRepository) ListActiveBySpace(ctx context.Context, spaceID model.SpaceID) ([]*model.Topic, error) {
	rows, err := r.q.ListActiveTopicsBySpace(ctx, string(spaceID))
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// FindBySpaceAndNames はスペースIDと名前リストでトピックを取得する（Wikiリンク解析時のトピック一括検索用）
func (r *TopicRepository) FindBySpaceAndNames(ctx context.Context, spaceID model.SpaceID, names []string) ([]*model.Topic, error) {
	rows, err := r.q.FindTopicsBySpaceAndNames(ctx, query.FindTopicsBySpaceAndNamesParams{
		SpaceID: string(spaceID),
		Column2: names,
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// ListJoinedBySpaceMember はスペースメンバーが参加しているトピック一覧を取得する（編集画面のトピックセレクター用）
func (r *TopicRepository) ListJoinedBySpaceMember(ctx context.Context, spaceMemberID model.SpaceMemberID, spaceID model.SpaceID) ([]*model.Topic, error) {
	rows, err := r.q.ListTopicsJoinedBySpaceMember(ctx, query.ListTopicsJoinedBySpaceMemberParams{
		SpaceMemberID: string(spaceMemberID),
		SpaceID:       string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// toModel は query.Topic を model.Topic に変換する
func (r *TopicRepository) toModel(row query.Topic) *model.Topic {
	var discardedAt *time.Time
	if row.DiscardedAt.Valid {
		discardedAt = &row.DiscardedAt.Time
	}

	return &model.Topic{
		ID:          model.TopicID(row.ID),
		SpaceID:     model.SpaceID(row.SpaceID),
		Number:      row.Number,
		Name:        row.Name,
		Description: row.Description,
		Visibility:  model.TopicVisibility(row.Visibility),
		DiscardedAt: discardedAt,
	}
}

// toModels は query.Topic のスライスを model.Topic のスライスに変換する
func (r *TopicRepository) toModels(rows []query.Topic) []*model.Topic {
	topics := make([]*model.Topic, len(rows))
	for i, row := range rows {
		topics[i] = r.toModel(row)
	}
	return topics
}
