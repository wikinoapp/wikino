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

// FindByIDs はIDリストでスペースメンバーを一括取得する（スペースIDでスコープ）
func (r *SpaceMemberRepository) FindByIDs(ctx context.Context, ids []model.SpaceMemberID, spaceID model.SpaceID) ([]*model.SpaceMember, error) {
	if len(ids) == 0 {
		return []*model.SpaceMember{}, nil
	}
	idStrs := model.SpaceMemberIDsToStrings(ids)
	rows, err := r.q.FindSpaceMembersByIDs(ctx, query.FindSpaceMembersByIDsParams{
		Column1: idStrs,
		SpaceID: string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	members := make([]*model.SpaceMember, len(rows))
	for i, row := range rows {
		members[i] = r.toModel(row)
	}
	return members, nil
}

// ListActiveByUserAndSpaceIDs fetches the active space memberships of the given user across the
// given space ids in a single query. It replaces per-space lookups (N+1) with one query where
// permissions for topics spanning many spaces must be resolved at once, such as on the home page.
//
// [Ja] ListActiveByUserAndSpaceIDs は、ユーザーが指定したスペース群で持つアクティブな
// スペースメンバーを 1 クエリで一括取得する。ホーム画面のように複数スペースにまたがる
// トピックの権限をまとめて判定する場面で、スペースごとの単発クエリ (N+1) を 1 回のクエリに
// 置き換えるために使う。
func (r *SpaceMemberRepository) ListActiveByUserAndSpaceIDs(ctx context.Context, userID model.UserID, spaceIDs []model.SpaceID) ([]*model.SpaceMember, error) {
	if len(spaceIDs) == 0 {
		return nil, nil
	}
	rows, err := r.q.ListActiveSpaceMembersByUserAndSpaceIDs(ctx, query.ListActiveSpaceMembersByUserAndSpaceIDsParams{
		UserID:  string(userID),
		Column2: model.SpaceIDsToStrings(spaceIDs),
	})
	if err != nil {
		return nil, err
	}

	members := make([]*model.SpaceMember, len(rows))
	for i, row := range rows {
		members[i] = r.toModel(row)
	}
	return members, nil
}

// toModel は query.SpaceMember を model.SpaceMember に変換する
func (r *SpaceMemberRepository) toModel(row query.SpaceMember) *model.SpaceMember {
	return &model.SpaceMember{
		ID:       model.SpaceMemberID(row.ID),
		SpaceID:  model.SpaceID(row.SpaceID),
		UserID:   model.UserID(row.UserID),
		Scopes:   model.StringsToScopes(row.Scopes),
		JoinedAt: row.JoinedAt,
		Active:   row.Active,
	}
}
