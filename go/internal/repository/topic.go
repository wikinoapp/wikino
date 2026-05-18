package repository

import (
	"context"
	"database/sql"
	"errors"
	"math"
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

// ListJoinedByUser はユーザーが参加しているトピック一覧を取得する（サイドバー表示用）
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

// JoinedTopicWithStats is the Repository return type that augments a topic the user is
// joined to with display stats for the home page (published page count).
//
// PublishedPagesCount is the number of pages in the topic that are published and not
// discarded or trashed. Topics with zero published pages return PublishedPagesCount = 0.
//
// [Ja] JoinedTopicWithStats はユーザーが参加しているトピックに、ホーム画面で表示する
// 統計情報 (公開中ページ数) を付加した Repository 戻り値型。
//
// PublishedPagesCount は対象トピック内で公開済み・未廃棄・未ゴミ箱のページ数。
// 公開ページが 0 件のトピックでは PublishedPagesCount = 0 となる。
type JoinedTopicWithStats struct {
	Topic               *model.Topic
	PublishedPagesCount int32
}

// ListJoinedWithStatsByUser returns the topics the user is joined to, with the published
// page count attached (for the home page).
// Results are ordered by topic_members.last_page_modified_at DESC NULLS LAST, then topic
// number DESC. See db/queries/joined_topics.sql for the rationale and tradeoffs.
//
// [Ja] ListJoinedWithStatsByUser はユーザーが参加しているトピック一覧を、公開中ページ数付きで
// 取得する (ホーム画面表示用)。
// 並び順は topic_members.last_page_modified_at の降順 (NULLS LAST)、同点はトピック番号の降順。
// 採用理由とトレードオフは db/queries/joined_topics.sql のコメントを参照。
func (r *TopicRepository) ListJoinedWithStatsByUser(ctx context.Context, userID model.UserID, limit int32) ([]JoinedTopicWithStats, error) {
	rows, err := r.q.ListJoinedTopicsWithStatsByUser(ctx, query.ListJoinedTopicsWithStatsByUserParams{
		UserID: string(userID),
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}
	return r.toJoinedTopicsWithStatsFromRows(rows), nil
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

// toJoinedTopicsWithStatsFromRows converts query.ListJoinedTopicsWithStatsByUserRow
// slice into a JoinedTopicWithStats slice.
//
// [Ja] toJoinedTopicsWithStatsFromRows は query.ListJoinedTopicsWithStatsByUserRow を
// JoinedTopicWithStats のスライスに変換する。
func (r *TopicRepository) toJoinedTopicsWithStatsFromRows(rows []query.ListJoinedTopicsWithStatsByUserRow) []JoinedTopicWithStats {
	stats := make([]JoinedTopicWithStats, len(rows))
	for i, row := range rows {
		// Saturate published_pages_count to MaxInt32 before narrowing to int32.
		// [Ja] sqlc は COUNT(*) を int64 で生成するが、ホーム表示の用途では int32 で十分なため
		// MaxInt32 を上限に飽和させて変換する (実運用で int32 を超える件数は想定しない)。
		count := row.PublishedPagesCount
		if count > math.MaxInt32 {
			count = math.MaxInt32
		}

		stats[i] = JoinedTopicWithStats{
			Topic: &model.Topic{
				ID:         model.TopicID(row.TopicID),
				Number:     row.TopicNumber,
				Name:       row.TopicName,
				Visibility: model.TopicVisibility(row.TopicVisibility),
				Space: &model.Space{
					ID:         model.SpaceID(row.SpaceID),
					Identifier: model.SpaceIdentifier(row.SpaceIdentifier),
					Name:       row.SpaceName,
				},
			},
			PublishedPagesCount: int32(count),
		}
	}
	return stats
}
