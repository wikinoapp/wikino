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

// LockByIDは、トランザクションが終わるまで許可行をロックし、最新の状態を返す。
// 許可の更新とトークンの発行を直列化するときに、WithTxで束縛したRepositoryから呼ぶ。
// 見つからない場合は (nil, nil) を返す
func (r *OAuthGrantRepository) LockByID(ctx context.Context, id model.OAuthGrantID, spaceID model.SpaceID) (*model.OAuthGrant, error) {
	row, err := r.q.LockOAuthGrantByID(ctx, query.LockOAuthGrantByIDParams{ID: string(id), SpaceID: string(spaceID)})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// ListUsableBySpaceMemberは、スペースメンバーが与えている失効していない許可のうち、有効な
// リフレッシュトークンを持つものを新しい順に取得する
func (r *OAuthGrantRepository) ListUsableBySpaceMember(ctx context.Context, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, now time.Time) ([]*model.OAuthGrant, error) {
	rows, err := r.q.ListUsableOAuthGrantsBySpaceMember(ctx, query.ListUsableOAuthGrantsBySpaceMemberParams{
		SpaceID:       string(spaceID),
		SpaceMemberID: string(spaceMemberID),
		Now:           now,
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

// SetLastUsedAtは、失効していない許可の最終利用日時をnowにする。トークンの発行に使い、
// 発行と同じトランザクションのRepository (WithTx) から呼ぶ
func (r *OAuthGrantRepository) SetLastUsedAt(ctx context.Context, id model.OAuthGrantID, spaceID model.SpaceID, now time.Time) error {
	return r.q.UpdateOAuthGrantLastUsedAt(ctx, query.UpdateOAuthGrantLastUsedAtParams{
		Now:         now,
		ID:          string(id),
		SpaceID:     string(spaceID),
		StaleBefore: now,
	})
}

// UpdateLastUsedAtは、失効していない許可の最終利用日時をnowにする。アクセストークンの照合に使う。
// 最終利用日時がmodel.OAuthGrantLastUsedAtUpdateIntervalより新しい場合は書き込まない
func (r *OAuthGrantRepository) UpdateLastUsedAt(ctx context.Context, id model.OAuthGrantID, spaceID model.SpaceID, now time.Time) error {
	return r.q.UpdateOAuthGrantLastUsedAt(ctx, query.UpdateOAuthGrantLastUsedAtParams{
		Now:         now,
		ID:          string(id),
		SpaceID:     string(spaceID),
		StaleBefore: now.Add(-model.OAuthGrantLastUsedAtUpdateInterval),
	})
}

// EvictInputは、許可を作る前の押し出しの入力パラメータ
type EvictInput struct {
	OAuthApplicationID model.OAuthApplicationID
	SpaceID            model.SpaceID
	SpaceMemberID      model.SpaceMemberID
	Now                time.Time
}

// Evictは、同じアプリ・メンバーに許可を1つ作れるよう、失効していない許可が
// model.OAuthGrantMaxPerApplicationMember - 1件以下になるまで、使われていないものから順に許可と
// 許可に属するトークンを失効する。許可の作成と同じトランザクションのRepository (WithTx) から呼ぶ
func (r *OAuthGrantRepository) Evict(ctx context.Context, input EvictInput) error {
	return r.q.EvictOAuthGrants(ctx, query.EvictOAuthGrantsParams{
		Now:                input.Now,
		SpaceID:            string(input.SpaceID),
		OauthApplicationID: string(input.OAuthApplicationID),
		SpaceMemberID:      string(input.SpaceMemberID),
		GraceBefore:        input.Now.Add(-model.OAuthGrantEvictionGracePeriod),
		MaxGrants:          model.OAuthGrantMaxPerApplicationMember,
	})
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
	var deviceName *string
	if row.DeviceName.Valid {
		deviceName = &row.DeviceName.String
	}
	var lastUsedAt *time.Time
	if row.LastUsedAt.Valid {
		lastUsedAt = &row.LastUsedAt.Time
	}
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
		DeviceName:         deviceName,
		LastUsedAt:         lastUsedAt,
		RevokedAt:          revokedAt,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}
