package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// SpaceMemberRepositoryはスペースメンバーリポジトリ
type SpaceMemberRepository struct {
	q *query.Queries
}

// NewSpaceMemberRepositoryはSpaceMemberRepositoryを生成する
func NewSpaceMemberRepository(q *query.Queries) *SpaceMemberRepository {
	return &SpaceMemberRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *SpaceMemberRepository) WithTx(tx *sql.Tx) *SpaceMemberRepository {
	return &SpaceMemberRepository{q: r.q.WithTx(tx)}
}

// FindActiveBySpaceAndUserはスペースIDとユーザーIDでアクティブなスペースメンバーを取得する
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

// FindByIDsはIDリストでスペースメンバーを一括取得する (スペースIDでスコープ)
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

// ListActiveByUserAndSpaceIDsは、ユーザーが指定したスペース群で持つアクティブな
// スペースメンバーを1クエリで一括取得する。ホーム画面のように複数スペースにまたがる
// トピックの権限をまとめて判定する場面で、スペースごとの単発クエリ (N+1) を1回のクエリに
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

// CreateSpaceMemberInputはスペースメンバーの作成に必要な値を保持する。
type CreateSpaceMemberInput struct {
	SpaceID model.SpaceID
	UserID  model.UserID
	Role    model.SpaceRole
}

// Createはユーザーをスペースに参加させる。Rails版が権限を判定できるよう、
// ロールに応じたスコープもspace_members.scopesへ書く。
func (r *SpaceMemberRepository) Create(ctx context.Context, input CreateSpaceMemberInput) (*model.SpaceMember, error) {
	row, err := r.q.CreateSpaceMember(ctx, query.CreateSpaceMemberParams{
		SpaceID: string(input.SpaceID),
		UserID:  string(input.UserID),
		Role:    string(input.Role),
		Scopes:  model.ScopesToStrings(input.Role.RailsScopes()),
		Now:     time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// toModelはquery.SpaceMemberをmodel.SpaceMemberに変換する
func (r *SpaceMemberRepository) toModel(row query.SpaceMember) *model.SpaceMember {
	return &model.SpaceMember{
		ID:       model.SpaceMemberID(row.ID),
		SpaceID:  model.SpaceID(row.SpaceID),
		UserID:   model.UserID(row.UserID),
		Role:     model.SpaceRole(row.Role),
		JoinedAt: row.JoinedAt,
		Active:   row.Active,
	}
}
