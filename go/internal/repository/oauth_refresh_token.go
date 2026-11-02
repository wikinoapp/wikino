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
	Audience     model.OAuthAudience
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
		Audience:     int32(input.Audience),
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

// ReissueOAuthRefreshTokenInputは、猶予の間に再提示された使用済みのリフレッシュトークンから
// 発行し直す入力パラメータ。ReuseAllowedAfterより後に使われたトークンだけを発行し直す
type ReissueOAuthRefreshTokenInput struct {
	RotateOAuthRefreshTokenInput
	ReuseAllowedAfter time.Time
}

// Reissueは、使用済みのリフレッシュトークンのローテーションで発行した後継のリフレッシュトークンを
// 失効させ、元のトークンの後継として新しいリフレッシュトークンとアクセストークンを作成して、新しい
// リフレッシュトークンを返す。元のトークンがReuseAllowedAfterより前に使われた・失効している場合と、
// 後継が使用済み・失効済みの場合は何もせず (nil, nil) を返す。判定と更新を1つの文で行う
func (r *OAuthRefreshTokenRepository) Reissue(ctx context.Context, input ReissueOAuthRefreshTokenInput) (*model.OAuthRefreshToken, error) {
	row, err := r.q.ReissueOAuthRefreshToken(ctx, query.ReissueOAuthRefreshTokenParams{
		TokenDigest:          input.TokenDigest,
		ExpiresAt:            input.ExpiresAt,
		Now:                  input.Now,
		ID:                   string(input.ID),
		SpaceID:              string(input.SpaceID),
		ReuseAllowedAfter:    input.ReuseAllowedAfter,
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

// Revokeは、リフレッシュトークンが属する許可と、許可に属するアクセストークン・リフレッシュ
// トークンをすべて1文で失効する (RFC 7009 §2.1)。元のトークンが使用済み・失効済み・期限切れなら
// 何も変えない
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
		Audience:               model.OAuthAudience(row.Audience),
		PreviousRefreshTokenID: previousRefreshTokenID,
		ExpiresAt:              row.ExpiresAt,
		UsedAt:                 usedAt,
		RevokedAt:              revokedAt,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}
