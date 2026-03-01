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
