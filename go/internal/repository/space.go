package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// SpaceRepositoryはスペースリポジトリ
type SpaceRepository struct {
	q *query.Queries
}

// NewSpaceRepositoryはSpaceRepositoryを生成する
func NewSpaceRepository(q *query.Queries) *SpaceRepository {
	return &SpaceRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *SpaceRepository) WithTx(tx *sql.Tx) *SpaceRepository {
	return &SpaceRepository{q: r.q.WithTx(tx)}
}

// FindByIdentifierは識別子でスペースを取得する (削除されていないスペースのみ)
func (r *SpaceRepository) FindByIdentifier(ctx context.Context, identifier model.SpaceIdentifier) (*model.Space, error) {
	row, err := r.q.GetSpaceByIdentifier(ctx, string(identifier))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// ListActiveByUserはユーザーが参加中 (アクティブ) かつ削除されていないスペースの一覧を返す
// Rails版current_user.active_space_records相当。並び順はユーザーがスペースに参加した日の降順
func (r *SpaceRepository) ListActiveByUser(ctx context.Context, userID model.UserID) ([]*model.Space, error) {
	rows, err := r.q.ListActiveSpacesByUser(ctx, string(userID))
	if err != nil {
		return nil, err
	}

	spaces := make([]*model.Space, len(rows))
	for i, row := range rows {
		spaces[i] = r.toModel(row)
	}
	return spaces, nil
}

// toModelはquery.Spaceをmodel.Spaceに変換する
func (r *SpaceRepository) toModel(row query.Space) *model.Space {
	var discardedAt *time.Time
	if row.DiscardedAt.Valid {
		discardedAt = &row.DiscardedAt.Time
	}

	return &model.Space{
		ID:          model.SpaceID(row.ID),
		Identifier:  model.SpaceIdentifier(row.Identifier),
		Name:        row.Name,
		Plan:        model.Plan(row.Plan),
		JoinedAt:    row.JoinedAt,
		DiscardedAt: discardedAt,
	}
}

// FindByIDは指定IDのスペースを返す。そのIDを持つ生きたスペースが無い場合は (nil, nil)
// を返す。エクスポートのワーカーはエクスポートの行から処理を始めるが、その行はスペースをIDで
// 指している。
func (r *SpaceRepository) FindByID(ctx context.Context, id model.SpaceID) (*model.Space, error) {
	row, err := r.q.GetSpaceByID(ctx, string(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// LockByIDは呼び出し元のトランザクションが終わるまで有効なスペースをロックする。
// スペースが存在しない場合はfalseを返す。
func (r *SpaceRepository) LockByID(ctx context.Context, id model.SpaceID) (bool, error) {
	_, err := r.q.LockSpaceByID(ctx, string(id))
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
