package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// PersonalAccessTokenRepositoryは個人アクセストークンリポジトリ
type PersonalAccessTokenRepository struct {
	q *query.Queries
}

// NewPersonalAccessTokenRepositoryはPersonalAccessTokenRepositoryを生成する
func NewPersonalAccessTokenRepository(q *query.Queries) *PersonalAccessTokenRepository {
	return &PersonalAccessTokenRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *PersonalAccessTokenRepository) WithTx(tx *sql.Tx) *PersonalAccessTokenRepository {
	return &PersonalAccessTokenRepository{q: r.q.WithTx(tx)}
}

// CreatePersonalAccessTokenInputは個人アクセストークン作成の入力パラメータ
type CreatePersonalAccessTokenInput struct {
	SpaceID        model.SpaceID
	SpaceMemberID  model.SpaceMemberID
	Name           string
	TokenDigest    string
	TokenLastChars string
	Scopes         []model.Scope
	ExpiresAt      time.Time
}

// Createは個人アクセストークンを作成する
func (r *PersonalAccessTokenRepository) Create(ctx context.Context, input CreatePersonalAccessTokenInput) (*model.PersonalAccessToken, error) {
	row, err := r.q.CreatePersonalAccessToken(ctx, query.CreatePersonalAccessTokenParams{
		SpaceID:        string(input.SpaceID),
		SpaceMemberID:  string(input.SpaceMemberID),
		Name:           input.Name,
		TokenDigest:    input.TokenDigest,
		TokenLastChars: input.TokenLastChars,
		Scopes:         model.ScopesToStrings(input.Scopes),
		ExpiresAt:      input.ExpiresAt,
		Now:            time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByTokenDigestはトークンダイジェストで個人アクセストークンを取得する。見つからない
// 場合は (nil, nil) を返す。失効や期限切れのトークンも返すため、有効かどうかは呼び出し側が
// 判定する。
//
// トークン認証の入口で使うため、スペースを条件に含めない。呼び出し側は、返ったトークンの
// SpaceIDを以降の取得の条件に必ず含める。
func (r *PersonalAccessTokenRepository) FindByTokenDigest(ctx context.Context, tokenDigest string) (*model.PersonalAccessToken, error) {
	row, err := r.q.FindPersonalAccessTokenByTokenDigest(ctx, tokenDigest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// ListUnrevokedBySpaceMemberは、スペースメンバー自身の失効していない個人アクセストークンを
// 新しい順に返す。期限切れのトークンも含めるため、有効かどうかは呼び出し側が判定する。
func (r *PersonalAccessTokenRepository) ListUnrevokedBySpaceMember(ctx context.Context, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID) ([]*model.PersonalAccessToken, error) {
	rows, err := r.q.ListUnrevokedPersonalAccessTokensBySpaceMember(ctx, query.ListUnrevokedPersonalAccessTokensBySpaceMemberParams{
		SpaceID:       string(spaceID),
		SpaceMemberID: string(spaceMemberID),
	})
	if err != nil {
		return nil, err
	}

	tokens := make([]*model.PersonalAccessToken, len(rows))
	for i, row := range rows {
		tokens[i] = r.toModel(row)
	}
	return tokens, nil
}

// UpdateLastUsedAtは最終使用日時をnowにする。最終使用日時が
// model.PersonalAccessTokenLastUsedAtUpdateIntervalより新しい場合は書き込まない
func (r *PersonalAccessTokenRepository) UpdateLastUsedAt(ctx context.Context, id model.PersonalAccessTokenID, spaceID model.SpaceID, now time.Time) error {
	return r.q.UpdatePersonalAccessTokenLastUsedAt(ctx, query.UpdatePersonalAccessTokenLastUsedAtParams{
		Now:         now,
		ID:          string(id),
		SpaceID:     string(spaceID),
		StaleBefore: now.Add(-model.PersonalAccessTokenLastUsedAtUpdateInterval),
	})
}

// Revokeはスペースメンバー自身の個人アクセストークンを失効する。対象が無いか既に失効して
// いる場合は (nil, nil) を返す。IDはURLからこのメソッドへ渡ってくるため、UUIDでない文字列は
// データベースへ送らず「見つからない」として答える。
func (r *PersonalAccessTokenRepository) Revoke(ctx context.Context, id model.PersonalAccessTokenID, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID) (*model.PersonalAccessToken, error) {
	if !uuidRegex.MatchString(string(id)) {
		return nil, nil
	}

	row, err := r.q.RevokePersonalAccessToken(ctx, query.RevokePersonalAccessTokenParams{
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
	return r.toModel(row), nil
}

// toModelはquery.PersonalAccessTokenをmodel.PersonalAccessTokenに変換する
func (r *PersonalAccessTokenRepository) toModel(row query.PersonalAccessToken) *model.PersonalAccessToken {
	var lastUsedAt *time.Time
	if row.LastUsedAt.Valid {
		lastUsedAt = &row.LastUsedAt.Time
	}
	var revokedAt *time.Time
	if row.RevokedAt.Valid {
		revokedAt = &row.RevokedAt.Time
	}
	return &model.PersonalAccessToken{
		ID:             model.PersonalAccessTokenID(row.ID),
		SpaceID:        model.SpaceID(row.SpaceID),
		SpaceMemberID:  model.SpaceMemberID(row.SpaceMemberID),
		Name:           row.Name,
		TokenDigest:    row.TokenDigest,
		TokenLastChars: row.TokenLastChars,
		Scopes:         model.StringsToScopes(row.Scopes),
		ExpiresAt:      row.ExpiresAt,
		LastUsedAt:     lastUsedAt,
		RevokedAt:      revokedAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}
