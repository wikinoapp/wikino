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
// 並び順はユーザーがスペースに参加した日の降順
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

// Discardはスペースを論理削除し、論理削除したかを返す。既に論理削除したスペースと、
// 見つからないスペースではfalseを返す。
func (r *SpaceRepository) Discard(ctx context.Context, id model.SpaceID, discardedAt time.Time) (bool, error) {
	n, err := r.q.DiscardSpace(ctx, query.DiscardSpaceParams{
		ID:          string(id),
		DiscardedAt: sql.NullTime{Time: discardedAt, Valid: true},
	})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Undiscardは、discardedAtに論理削除したスペースの論理削除を取り消す。
func (r *SpaceRepository) Undiscard(ctx context.Context, id model.SpaceID, discardedAt time.Time, updatedAt time.Time) error {
	return r.q.UndiscardSpace(ctx, query.UndiscardSpaceParams{
		ID:          string(id),
		DiscardedAt: sql.NullTime{Time: discardedAt, Valid: true},
		UpdatedAt:   updatedAt,
	})
}

// ExistsDiscardedは、論理削除したスペースがあるかを返す。論理削除していないスペースと、行の無い
// スペースではfalseを返す。
func (r *SpaceRepository) ExistsDiscarded(ctx context.Context, id model.SpaceID) (bool, error) {
	return r.q.ExistsDiscardedSpaceByID(ctx, string(id))
}

// LockDiscardedは論理削除したスペースをロックし、ロックできたかを返す。論理削除していない
// スペースと、見つからないスペースではfalseを返す。
func (r *SpaceRepository) LockDiscarded(ctx context.Context, id model.SpaceID) (bool, error) {
	_, err := r.q.LockDiscardedSpace(ctx, string(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DeleteDiscardedは論理削除したスペースを物理削除し、削除したかを返す。
func (r *SpaceRepository) DeleteDiscarded(ctx context.Context, id model.SpaceID) (bool, error) {
	n, err := r.q.DeleteDiscardedSpace(ctx, string(id))
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ExistsByIdentifierはその識別子を持つスペースが既にあるかを返す。削除済みのスペースも数え、
// 大文字と小文字を区別しない。識別子の一意インデックスと同じ範囲で重複を判定するためである。
func (r *SpaceRepository) ExistsByIdentifier(ctx context.Context, identifier model.SpaceIdentifier) (bool, error) {
	return r.q.ExistsSpaceByIdentifier(ctx, string(identifier))
}

// ExistsOtherByIdentifierは、指定したスペースのほかにその識別子を持つスペースがあるかを返す。
// 範囲はExistsByIdentifierと同じく、削除済みのスペースを含み、大文字と小文字を区別しない。
func (r *SpaceRepository) ExistsOtherByIdentifier(ctx context.Context, id model.SpaceID, identifier model.SpaceIdentifier) (bool, error) {
	return r.q.ExistsOtherSpaceByIdentifier(ctx, query.ExistsOtherSpaceByIdentifierParams{
		Identifier: string(identifier),
		ID:         string(id),
	})
}

// CreateSpaceInputはスペースの作成に必要な値を保持する。
type CreateSpaceInput struct {
	Identifier model.SpaceIdentifier
	Name       string
	Plan       model.Plan
}

// Createはスペースを作成する。
func (r *SpaceRepository) Create(ctx context.Context, input CreateSpaceInput) (*model.Space, error) {
	row, err := r.q.CreateSpace(ctx, query.CreateSpaceParams{
		Identifier: string(input.Identifier),
		Name:       input.Name,
		Plan:       int32(input.Plan),
		Now:        time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// UpdateSpaceInputはスペースの一般設定の更新に必要な値を保持する。
type UpdateSpaceInput struct {
	ID         model.SpaceID
	Identifier model.SpaceIdentifier
	Name       string
}

// Updateはスペースの識別子と名前を更新する。スペースが削除済みか存在しない場合は (nil, nil) を返す。
func (r *SpaceRepository) Update(ctx context.Context, input UpdateSpaceInput) (*model.Space, error) {
	row, err := r.q.UpdateSpace(ctx, query.UpdateSpaceParams{
		Identifier: string(input.Identifier),
		Name:       input.Name,
		Now:        time.Now(),
		ID:         string(input.ID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}
