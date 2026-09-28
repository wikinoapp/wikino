package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// OAuthAccessTokenRepositoryはOAuthのアクセストークンリポジトリ
type OAuthAccessTokenRepository struct {
	q *query.Queries
}

// NewOAuthAccessTokenRepositoryはOAuthAccessTokenRepositoryを生成する
func NewOAuthAccessTokenRepository(q *query.Queries) *OAuthAccessTokenRepository {
	return &OAuthAccessTokenRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *OAuthAccessTokenRepository) WithTx(tx *sql.Tx) *OAuthAccessTokenRepository {
	return &OAuthAccessTokenRepository{q: r.q.WithTx(tx)}
}

// CreateOAuthAccessTokenInputはアクセストークン作成の入力パラメータ
type CreateOAuthAccessTokenInput struct {
	OAuthGrantID model.OAuthGrantID
	SpaceID      model.SpaceID
	TokenDigest  string
	Scopes       []model.Scope
	ExpiresAt    time.Time
}

// Createはアクセストークンを作成する
func (r *OAuthAccessTokenRepository) Create(ctx context.Context, input CreateOAuthAccessTokenInput) (*model.OAuthAccessToken, error) {
	row, err := r.q.CreateOAuthAccessToken(ctx, query.CreateOAuthAccessTokenParams{
		OauthGrantID: string(input.OAuthGrantID),
		SpaceID:      string(input.SpaceID),
		TokenDigest:  input.TokenDigest,
		Scopes:       model.ScopesToStrings(input.Scopes),
		ExpiresAt:    input.ExpiresAt,
		Now:          time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByTokenDigestはトークンダイジェストでアクセストークンを取得する。見つからない場合は
// (nil, nil) を返す。失効や期限切れのトークンも返すため、有効かどうかは呼び出し側が判定する。
//
// トークン認証の入口で使うため、スペースを条件に含めない。呼び出し側は、返ったトークンの
// SpaceIDを以降の取得の条件に必ず含める。
func (r *OAuthAccessTokenRepository) FindByTokenDigest(ctx context.Context, tokenDigest string) (*model.OAuthAccessToken, error) {
	row, err := r.q.FindOAuthAccessTokenByTokenDigest(ctx, tokenDigest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// RevokeByOAuthGrantは許可に属する失効していないアクセストークンをすべて失効する
func (r *OAuthAccessTokenRepository) RevokeByOAuthGrant(ctx context.Context, oauthGrantID model.OAuthGrantID, spaceID model.SpaceID, now time.Time) error {
	return r.q.RevokeOAuthAccessTokensByOAuthGrant(ctx, query.RevokeOAuthAccessTokensByOAuthGrantParams{
		Now:          now,
		OauthGrantID: string(oauthGrantID),
		SpaceID:      string(spaceID),
	})
}

// Revokeはアクセストークンを1つ失効する。既に失効しているものは変えない
func (r *OAuthAccessTokenRepository) Revoke(ctx context.Context, id model.OAuthAccessTokenID, spaceID model.SpaceID, now time.Time) error {
	return r.q.RevokeOAuthAccessToken(ctx, query.RevokeOAuthAccessTokenParams{
		Now:     now,
		ID:      string(id),
		SpaceID: string(spaceID),
	})
}

// toModelはquery.OauthAccessTokenをmodel.OAuthAccessTokenに変換する
func (r *OAuthAccessTokenRepository) toModel(row query.OauthAccessToken) *model.OAuthAccessToken {
	var revokedAt *time.Time
	if row.RevokedAt.Valid {
		revokedAt = &row.RevokedAt.Time
	}
	return &model.OAuthAccessToken{
		ID:           model.OAuthAccessTokenID(row.ID),
		OAuthGrantID: model.OAuthGrantID(row.OauthGrantID),
		SpaceID:      model.SpaceID(row.SpaceID),
		TokenDigest:  row.TokenDigest,
		Scopes:       model.StringsToScopes(row.Scopes),
		ExpiresAt:    row.ExpiresAt,
		RevokedAt:    revokedAt,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
