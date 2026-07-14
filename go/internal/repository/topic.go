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

// FindBySpaceAndID はスペースIDとIDでトピックを取得する（削除されていないトピックのみ）
func (r *TopicRepository) FindBySpaceAndID(ctx context.Context, spaceID model.SpaceID, topicID model.TopicID) (*model.Topic, error) {
	row, err := r.q.FindTopicBySpaceAndID(ctx, query.FindTopicBySpaceAndIDParams{
		SpaceID: string(spaceID),
		ID:      string(topicID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
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

// ListPublicBySpace returns the active public topics (not-discarded, visibility = public) in the
// given space, ordered by number. Used by the topic section shown to non-members (guests) on the
// space detail page, where only public topics are visible.
//
// [Ja] ListPublicBySpace は指定スペース内のアクティブな公開トピック (未廃棄・visibility = public) を
// number 順で返す。スペース詳細画面で非メンバー (ゲスト) に表示するトピックセクションで使用し、
// ここでは公開トピックのみが見える。
func (r *TopicRepository) ListPublicBySpace(ctx context.Context, spaceID model.SpaceID) ([]*model.Topic, error) {
	rows, err := r.q.ListPublicTopicsBySpace(ctx, string(spaceID))
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

// FindFirstJoinedBySpaceMember returns the topic with the smallest id among those the
// space member has joined (not-discarded topics only), scoped to the given space. Returns
// (nil, nil) when none is found. Used by the empty-state "create a new page" link on the
// space detail page.
//
// [Ja] FindFirstJoinedBySpaceMember はスペースメンバーが参加しているトピックのうち id が
// 最小のもの (削除されていないトピックのみ) を、指定スペースにスコープして返す。未存在の
// 場合は (nil, nil) を返す。スペース詳細画面の空状態で表示する「新しいページを作る」導線で使用する。
func (r *TopicRepository) FindFirstJoinedBySpaceMember(ctx context.Context, spaceMemberID model.SpaceMemberID, spaceID model.SpaceID) (*model.Topic, error) {
	row, err := r.q.FindFirstJoinedTopicBySpaceMember(ctx, query.FindFirstJoinedTopicBySpaceMemberParams{
		SpaceMemberID: string(spaceMemberID),
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

// FindByIDsAndSpace はIDリストとスペースIDでトピックを一括取得する
func (r *TopicRepository) FindByIDsAndSpace(ctx context.Context, ids []model.TopicID, spaceID model.SpaceID) ([]*model.Topic, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.q.FindTopicsByIDsAndSpace(ctx, query.FindTopicsByIDsAndSpaceParams{
		SpaceID: string(spaceID),
		Column2: model.TopicIDsToStrings(ids),
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// ListJoinedByUser returns the topics the user is joined to for the home page.
// Ordering and tradeoffs are documented on the underlying SQL query
// (db/queries/joined_topics.sql) — see ListJoinedTopicsByUser there.
//
// [Ja] ListJoinedByUser はホーム画面に表示する、ユーザーが参加しているトピック一覧を取得する。
// 並び順と採用理由・トレードオフは db/queries/joined_topics.sql の ListJoinedTopicsByUser のコメントを参照。
func (r *TopicRepository) ListJoinedByUser(ctx context.Context, userID model.UserID, limit int32) ([]*model.Topic, error) {
	rows, err := r.q.ListJoinedTopicsByUser(ctx, query.ListJoinedTopicsByUserParams{
		UserID: string(userID),
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}
	return r.toTopicsFromJoinedRows(rows), nil
}

// toModel は query.Topic を model.Topic に変換する
func (r *TopicRepository) toModel(row query.Topic) *model.Topic {
	var discardedAt *time.Time
	if row.DiscardedAt.Valid {
		discardedAt = &row.DiscardedAt.Time
	}

	return &model.Topic{
		ID:          model.TopicID(row.ID),
		Space:       &model.Space{ID: model.SpaceID(row.SpaceID)},
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

// toTopicsFromJoinedRows は query.ListJoinedTopicsByUserRow のスライスを model.Topic のスライスに変換する
func (r *TopicRepository) toTopicsFromJoinedRows(rows []query.ListJoinedTopicsByUserRow) []*model.Topic {
	topics := make([]*model.Topic, len(rows))
	for i, row := range rows {
		topics[i] = &model.Topic{
			ID:         model.TopicID(row.TopicID),
			Number:     row.TopicNumber,
			Name:       row.TopicName,
			Visibility: model.TopicVisibility(row.TopicVisibility),
			Space: &model.Space{
				ID:         model.SpaceID(row.SpaceID),
				Identifier: model.SpaceIdentifier(row.SpaceIdentifier),
				Name:       row.SpaceName,
			},
		}
	}
	return topics
}
