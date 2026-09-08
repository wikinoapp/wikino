package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// SpaceRepository はスペースリポジトリ
type SpaceRepository struct {
	q *query.Queries
}

// NewSpaceRepository は SpaceRepository を生成する
func NewSpaceRepository(q *query.Queries) *SpaceRepository {
	return &SpaceRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *SpaceRepository) WithTx(tx *sql.Tx) *SpaceRepository {
	return &SpaceRepository{q: r.q.WithTx(tx)}
}

// FindByIdentifier は識別子でスペースを取得する（削除されていないスペースのみ）
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

// ListActiveByUser はユーザーが参加中（アクティブ）かつ削除されていないスペースの一覧を返す
// Rails 版 current_user.active_space_records 相当。並び順はユーザーがスペースに参加した日の降順
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

// toModel は query.Space を model.Space に変換する
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

// FindByID returns the space with the given ID, or (nil, nil) when no live space has it. The
// export worker starts from an export row, which names its space by ID.
//
// [Ja] FindByID は指定 ID のスペースを返す。その ID を持つ生きたスペースが無い場合は (nil, nil)
// を返す。エクスポートのワーカーはエクスポートの行から処理を始めるが、その行はスペースを ID で
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

// LockByID locks an active space until the caller's transaction ends.
// It returns false when the space no longer exists.
//
// [Ja] LockByID は呼び出し元のトランザクションが終わるまで有効なスペースをロックする。
// スペースが存在しない場合は false を返す。
func (r *SpaceRepository) LockByID(ctx context.Context, id model.SpaceID) (bool, error) {
	_, err := r.q.LockSpaceByID(ctx, string(id))
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
