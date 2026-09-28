package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// OAuthGrantRepositoryはOAuthアプリの許可リポジトリ
type OAuthGrantRepository struct {
	q *query.Queries
}

// NewOAuthGrantRepositoryはOAuthGrantRepositoryを生成する
func NewOAuthGrantRepository(q *query.Queries) *OAuthGrantRepository {
	return &OAuthGrantRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *OAuthGrantRepository) WithTx(tx *sql.Tx) *OAuthGrantRepository {
	return &OAuthGrantRepository{q: r.q.WithTx(tx)}
}

// CreateOAuthGrantInputはOAuthアプリの許可作成の入力パラメータ
type CreateOAuthGrantInput struct {
	OAuthApplicationID model.OAuthApplicationID
	SpaceID            model.SpaceID
	SpaceMemberID      model.SpaceMemberID
	Scopes             []model.Scope
}

// CreateはOAuthアプリの許可を作成する。同じアプリ・メンバーに失効していない許可が既に
// あれば、一意制約違反のエラーを返す
func (r *OAuthGrantRepository) Create(ctx context.Context, input CreateOAuthGrantInput) (*model.OAuthGrant, error) {
	row, err := r.q.CreateOAuthGrant(ctx, query.CreateOAuthGrantParams{
		OauthApplicationID: string(input.OAuthApplicationID),
		SpaceID:            string(input.SpaceID),
		SpaceMemberID:      string(input.SpaceMemberID),
		Scopes:             model.ScopesToStrings(input.Scopes),
		Now:                time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByIDはIDでOAuthアプリの許可を取得する。見つからない場合は (nil, nil) を返す。
// 失効した許可も返すため、有効かどうかは呼び出し側が判定する
func (r *OAuthGrantRepository) FindByID(ctx context.Context, id model.OAuthGrantID, spaceID model.SpaceID) (*model.OAuthGrant, error) {
	row, err := r.q.FindOAuthGrantByID(ctx, query.FindOAuthGrantByIDParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindUnrevokedByApplicationAndSpaceMemberは、スペースメンバーがOAuthアプリに与えている
// 失効していない許可を取得する。見つからない場合は (nil, nil) を返す
func (r *OAuthGrantRepository) FindUnrevokedByApplicationAndSpaceMember(ctx context.Context, oauthApplicationID model.OAuthApplicationID, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID) (*model.OAuthGrant, error) {
	row, err := r.q.FindUnrevokedOAuthGrantByApplicationAndSpaceMember(ctx, query.FindUnrevokedOAuthGrantByApplicationAndSpaceMemberParams{
		OauthApplicationID: string(oauthApplicationID),
		SpaceID:            string(spaceID),
		SpaceMemberID:      string(spaceMemberID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// ListUnrevokedBySpaceMemberは、スペースメンバーが与えている失効していない許可を新しい順に取得する
func (r *OAuthGrantRepository) ListUnrevokedBySpaceMember(ctx context.Context, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID) ([]*model.OAuthGrant, error) {
	rows, err := r.q.ListUnrevokedOAuthGrantsBySpaceMember(ctx, query.ListUnrevokedOAuthGrantsBySpaceMemberParams{
		SpaceID:       string(spaceID),
		SpaceMemberID: string(spaceMemberID),
	})
	if err != nil {
		return nil, err
	}
	grants := make([]*model.OAuthGrant, len(rows))
	for i, row := range rows {
		grants[i] = r.toModel(row)
	}
	return grants, nil
}

// Revokeは、スペースメンバー自身のOAuthアプリの許可と、許可に属するアクセストークン・
// リフレッシュトークンをすべて失効する。1文で行うため、許可だけが失効してトークンが使える
// 状態は途中にも残らない。対象が無いか既に失効している場合は何も変えずに (nil, nil) を返す。
// IDはURLからこのメソッドへ渡ってくるため、UUIDでない文字列はデータベースへ送らず
// 「見つからない」として答える。
func (r *OAuthGrantRepository) Revoke(ctx context.Context, id model.OAuthGrantID, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID) (*model.OAuthGrant, error) {
	if !uuidRegex.MatchString(string(id)) {
		return nil, nil
	}

	row, err := r.q.RevokeOAuthGrant(ctx, query.RevokeOAuthGrantParams{
		Now:           time.Now(),
		ID:            string(id),
		SpaceID:       string(spaceID),
		SpaceMemberID: string(spaceMemberID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(query.OauthGrant(row)), nil
}

// toModelはquery.OauthGrantをmodel.OAuthGrantに変換する
func (r *OAuthGrantRepository) toModel(row query.OauthGrant) *model.OAuthGrant {
	var revokedAt *time.Time
	if row.RevokedAt.Valid {
		revokedAt = &row.RevokedAt.Time
	}
	return &model.OAuthGrant{
		ID:                 model.OAuthGrantID(row.ID),
		OAuthApplicationID: model.OAuthApplicationID(row.OauthApplicationID),
		SpaceID:            model.SpaceID(row.SpaceID),
		SpaceMemberID:      model.SpaceMemberID(row.SpaceMemberID),
		Scopes:             model.StringsToScopes(row.Scopes),
		RevokedAt:          revokedAt,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}
