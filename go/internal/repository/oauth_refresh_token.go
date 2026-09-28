package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// OAuthRefreshTokenRepositoryはOAuthのリフレッシュトークンリポジトリ
type OAuthRefreshTokenRepository struct {
	q *query.Queries
}

// NewOAuthRefreshTokenRepositoryはOAuthRefreshTokenRepositoryを生成する
func NewOAuthRefreshTokenRepository(q *query.Queries) *OAuthRefreshTokenRepository {
	return &OAuthRefreshTokenRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *OAuthRefreshTokenRepository) WithTx(tx *sql.Tx) *OAuthRefreshTokenRepository {
	return &OAuthRefreshTokenRepository{q: r.q.WithTx(tx)}
}

// CreateOAuthRefreshTokenInputはリフレッシュトークン作成の入力パラメータ
type CreateOAuthRefreshTokenInput struct {
	OAuthGrantID model.OAuthGrantID
	SpaceID      model.SpaceID
	TokenDigest  string
	Scopes       []model.Scope
	ExpiresAt    time.Time
}

// Createはリフレッシュトークンを作成する。認可コードの交換で最初のトークンを発行する
// ときに使い、ローテーションではRotateを使う
func (r *OAuthRefreshTokenRepository) Create(ctx context.Context, input CreateOAuthRefreshTokenInput) (*model.OAuthRefreshToken, error) {
	row, err := r.q.CreateOAuthRefreshToken(ctx, query.CreateOAuthRefreshTokenParams{
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

// FindByTokenDigestはトークンダイジェストでリフレッシュトークンを取得する。見つからない
// 場合は (nil, nil) を返す。使用済み・失効・期限切れのトークンも返すため、有効かどうかは
// 呼び出し側が判定する。
//
// トークン要求の入口で使うため、スペースを条件に含めない。呼び出し側は、返ったトークンの
// SpaceIDを以降の取得の条件に必ず含める。
func (r *OAuthRefreshTokenRepository) FindByTokenDigest(ctx context.Context, tokenDigest string) (*model.OAuthRefreshToken, error) {
	row, err := r.q.FindOAuthRefreshTokenByTokenDigest(ctx, tokenDigest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// RotateOAuthRefreshTokenInputはリフレッシュトークンのローテーションの入力パラメータ。
// TokenDigestとExpiresAtは新しいリフレッシュトークンの値で、AccessTokenで始まる値は
// 同時に作るアクセストークンの値である
type RotateOAuthRefreshTokenInput struct {
	ID                   model.OAuthRefreshTokenID
	SpaceID              model.SpaceID
	TokenDigest          string
	ExpiresAt            time.Time
	AccessTokenDigest    string
	AccessTokenScopes    []model.Scope
	AccessTokenExpiresAt time.Time
	Now                  time.Time
}

// Rotateはリフレッシュトークンを使用済みにし、同じ許可・スコープの新しいリフレッシュトークンと、
// 同じ許可のアクセストークンを作成して、新しいリフレッシュトークンを返す。判定と更新を1つの文で
// 行うため、同じトークンの同時の更新でも新しいトークンを返すのは1回だけである。元のトークンが
// 見つからないか、使用済み・失効・期限切れの場合は何もせず (nil, nil) を返す。
func (r *OAuthRefreshTokenRepository) Rotate(ctx context.Context, input RotateOAuthRefreshTokenInput) (*model.OAuthRefreshToken, error) {
	row, err := r.q.RotateOAuthRefreshToken(ctx, query.RotateOAuthRefreshTokenParams{
		TokenDigest:          input.TokenDigest,
		ExpiresAt:            input.ExpiresAt,
		Now:                  input.Now,
		ID:                   string(input.ID),
		SpaceID:              string(input.SpaceID),
		AccessTokenDigest:    input.AccessTokenDigest,
		AccessTokenScopes:    model.ScopesToStrings(input.AccessTokenScopes),
		AccessTokenExpiresAt: input.AccessTokenExpiresAt,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// RevokeByOAuthGrantは許可に属する失効していないリフレッシュトークンをすべて失効する
func (r *OAuthRefreshTokenRepository) RevokeByOAuthGrant(ctx context.Context, oauthGrantID model.OAuthGrantID, spaceID model.SpaceID, now time.Time) error {
	return r.q.RevokeOAuthRefreshTokensByOAuthGrant(ctx, query.RevokeOAuthRefreshTokensByOAuthGrantParams{
		Now:          now,
		OauthGrantID: string(oauthGrantID),
		SpaceID:      string(spaceID),
	})
}

// Revokeは、リフレッシュトークンを1つ失効し、同じ許可のアクセストークンをすべて失効する
// (RFC 7009 §2.1)。同じ許可のほかのリフレッシュトークンは失効しない。元のトークンが使用済み・
// 失効済み・期限切れなら何も変えない
func (r *OAuthRefreshTokenRepository) Revoke(ctx context.Context, id model.OAuthRefreshTokenID, spaceID model.SpaceID, now time.Time) error {
	return r.q.RevokeOAuthRefreshToken(ctx, query.RevokeOAuthRefreshTokenParams{
		Now:     now,
		ID:      string(id),
		SpaceID: string(spaceID),
	})
}

// toModelはquery.OauthRefreshTokenをmodel.OAuthRefreshTokenに変換する
func (r *OAuthRefreshTokenRepository) toModel(row query.OauthRefreshToken) *model.OAuthRefreshToken {
	var previousRefreshTokenID *model.OAuthRefreshTokenID
	if row.PreviousRefreshTokenID != nil {
		id := model.OAuthRefreshTokenID(*row.PreviousRefreshTokenID)
		previousRefreshTokenID = &id
	}
	var usedAt *time.Time
	if row.UsedAt.Valid {
		usedAt = &row.UsedAt.Time
	}
	var revokedAt *time.Time
	if row.RevokedAt.Valid {
		revokedAt = &row.RevokedAt.Time
	}
	return &model.OAuthRefreshToken{
		ID:                     model.OAuthRefreshTokenID(row.ID),
		OAuthGrantID:           model.OAuthGrantID(row.OauthGrantID),
		SpaceID:                model.SpaceID(row.SpaceID),
		TokenDigest:            row.TokenDigest,
		Scopes:                 model.StringsToScopes(row.Scopes),
		PreviousRefreshTokenID: previousRefreshTokenID,
		ExpiresAt:              row.ExpiresAt,
		UsedAt:                 usedAt,
		RevokedAt:              revokedAt,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}
