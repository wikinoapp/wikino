package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// SpaceMemberRepository はスペースメンバーリポジトリ
type SpaceMemberRepository struct {
	q *query.Queries
}

// NewSpaceMemberRepository は SpaceMemberRepository を生成する
func NewSpaceMemberRepository(q *query.Queries) *SpaceMemberRepository {
	return &SpaceMemberRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *SpaceMemberRepository) WithTx(tx *sql.Tx) *SpaceMemberRepository {
	return &SpaceMemberRepository{q: r.q.WithTx(tx)}
}

// FindActiveBySpaceAndUser はスペースIDとユーザーIDでアクティブなスペースメンバーを取得する
func (r *SpaceMemberRepository) FindActiveBySpaceAndUser(ctx context.Context, spaceID model.SpaceID, userID model.UserID) (*model.SpaceMember, error) {
	row, err := r.q.FindActiveSpaceMemberBySpaceAndUser(ctx, query.FindActiveSpaceMemberBySpaceAndUserParams{
		SpaceID: string(spaceID),
		UserID:  string(userID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// toModel は query.SpaceMember を model.SpaceMember に変換する
func (r *SpaceMemberRepository) toModel(row query.SpaceMember) *model.SpaceMember {
	return &model.SpaceMember{
		ID:       model.SpaceMemberID(row.ID),
		SpaceID:  model.SpaceID(row.SpaceID),
		UserID:   model.UserID(row.UserID),
		Role:     model.SpaceMemberRole(row.Role),
		JoinedAt: row.JoinedAt,
		Active:   row.Active,
	}
}
