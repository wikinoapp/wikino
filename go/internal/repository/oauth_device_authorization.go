package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// OAuthDeviceAuthorizationRepositoryはOAuthのデバイス認可の要求のリポジトリ
type OAuthDeviceAuthorizationRepository struct {
	q *query.Queries
}

// NewOAuthDeviceAuthorizationRepositoryはOAuthDeviceAuthorizationRepositoryを生成する
func NewOAuthDeviceAuthorizationRepository(q *query.Queries) *OAuthDeviceAuthorizationRepository {
	return &OAuthDeviceAuthorizationRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *OAuthDeviceAuthorizationRepository) WithTx(tx *sql.Tx) *OAuthDeviceAuthorizationRepository {
	return &OAuthDeviceAuthorizationRepository{q: r.q.WithTx(tx)}
}

// CreateOAuthDeviceAuthorizationInputはデバイス認可の要求の作成の入力パラメータ
type CreateOAuthDeviceAuthorizationInput struct {
	OAuthApplicationID model.OAuthApplicationID
	SpaceID            model.SpaceID
	DeviceCodeDigest   string
	UserCodeDigest     string
	Scopes             []model.Scope
	Interval           time.Duration
	ExpiresAt          time.Time
}

// Createはデバイス認可の要求を許可待ちで作成する
func (r *OAuthDeviceAuthorizationRepository) Create(ctx context.Context, input CreateOAuthDeviceAuthorizationInput) (*model.OAuthDeviceAuthorization, error) {
	row, err := r.q.CreateOAuthDeviceAuthorization(ctx, query.CreateOAuthDeviceAuthorizationParams{
		OauthApplicationID: string(input.OAuthApplicationID),
		SpaceID:            string(input.SpaceID),
		DeviceCodeDigest:   input.DeviceCodeDigest,
		UserCodeDigest:     input.UserCodeDigest,
		Scopes:             model.ScopesToStrings(input.Scopes),
		// #nosec G115 -- ポーリングの間隔は数秒から数十秒で、int32に収まる。
		IntervalSeconds: int32(input.Interval / time.Second),
		ExpiresAt:       input.ExpiresAt,
		Now:             time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByDeviceCodeDigestはデバイスコードのダイジェストで要求を取得する。見つからない場合は
// (nil, nil) を返す。期限切れや交換済みの要求も返すため、使えるかどうかは呼び出し側が判定する。
//
// トークン要求の入口で使うため、スペースを条件に含めない。呼び出し側は、返った要求の
// SpaceIDを以降の取得の条件に必ず含める。
func (r *OAuthDeviceAuthorizationRepository) FindByDeviceCodeDigest(ctx context.Context, deviceCodeDigest string) (*model.OAuthDeviceAuthorization, error) {
	row, err := r.q.FindOAuthDeviceAuthorizationByDeviceCodeDigest(ctx, deviceCodeDigest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByUserCodeDigestはユーザーコードのダイジェストで要求を取得する。見つからない場合は
// (nil, nil) を返す。期限切れや許可済みの要求も返すため、使えるかどうかは呼び出し側が判定する。
//
// ユーザーコードの入力の入口で使うため、スペースを条件に含めない。呼び出し側は、返った要求の
// SpaceIDを以降の取得の条件に必ず含める。
func (r *OAuthDeviceAuthorizationRepository) FindByUserCodeDigest(ctx context.Context, userCodeDigest string) (*model.OAuthDeviceAuthorization, error) {
	row, err := r.q.FindOAuthDeviceAuthorizationByUserCodeDigest(ctx, userCodeDigest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// LockByIDは、交換前の要求をロックして取得する。トランザクションの中で使い、
// 呼び出し側でロック待機後の状態と期限を確認する。見つからない場合は (nil, nil) を返す
func (r *OAuthDeviceAuthorizationRepository) LockByID(ctx context.Context, id model.OAuthDeviceAuthorizationID, spaceID model.SpaceID) (*model.OAuthDeviceAuthorization, error) {
	row, err := r.q.LockOAuthDeviceAuthorizationByID(ctx, query.LockOAuthDeviceAuthorizationByIDParams{
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

// LockPendingは、許可待ちで期限内の要求をロックして取得する。許可待ちでない要求と期限切れの
// 要求は (nil, nil) を返す。トランザクションの中で使う。
//
// 期限はロック待機前の時刻で判定するため、待機中に期限が切れた要求も返りうる。呼び出し側は
// 取得した要求の期限を確かめ直す
func (r *OAuthDeviceAuthorizationRepository) LockPending(ctx context.Context, id model.OAuthDeviceAuthorizationID, spaceID model.SpaceID) (*model.OAuthDeviceAuthorization, error) {
	row, err := r.q.LockPendingOAuthDeviceAuthorization(ctx, query.LockPendingOAuthDeviceAuthorizationParams{
		ID:      string(id),
		SpaceID: string(spaceID),
		Now:     time.Now(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// Denyは、許可待ちで期限内の要求を拒否済みにする。許可待ちでない要求と期限切れの要求は
// 更新せず (nil, nil) を返す。
//
// 期限はロック待機前の時刻で判定するため、同じ要求への許可の送信を待つ間に期限が切れた要求も
// 拒否済みになりうる。拒否はアプリに権限を与えないため、確かめ直さない
func (r *OAuthDeviceAuthorizationRepository) Deny(ctx context.Context, id model.OAuthDeviceAuthorizationID, spaceID model.SpaceID) (*model.OAuthDeviceAuthorization, error) {
	row, err := r.q.DenyOAuthDeviceAuthorization(ctx, query.DenyOAuthDeviceAuthorizationParams{
		ID:      string(id),
		SpaceID: string(spaceID),
		Now:     time.Now(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// ApproveOAuthDeviceAuthorizationWithGrantInputは、デバイス認可の要求の許可の入力パラメータ。
// Scopesは利用者が選んだ付与するスコープ
type ApproveOAuthDeviceAuthorizationWithGrantInput struct {
	ID                 model.OAuthDeviceAuthorizationID
	OAuthApplicationID model.OAuthApplicationID
	SpaceID            model.SpaceID
	SpaceMemberID      model.SpaceMemberID
	Scopes             []model.Scope
}

// ApproveWithGrantは、スペースメンバーのOAuthアプリへの許可を作成または選んだ範囲に置き換え、
// 許可待ちの要求を許可済みにして許可を結び付ける。許可の外に出たスコープを持つトークンは
// 失効させる。要求が許可待ちでなければ何も変えず (nil, nil) を返す。
//
// 同じトランザクションで、先にLockPendingで要求を、OAuthGrantRepository.LockOrCreateで許可を
// ロックしておく
func (r *OAuthDeviceAuthorizationRepository) ApproveWithGrant(ctx context.Context, input ApproveOAuthDeviceAuthorizationWithGrantInput) (*model.OAuthDeviceAuthorization, error) {
	row, err := r.q.ApproveOAuthDeviceAuthorizationWithGrant(ctx, query.ApproveOAuthDeviceAuthorizationWithGrantParams{
		ID:                 string(input.ID),
		OauthApplicationID: string(input.OAuthApplicationID),
		SpaceID:            string(input.SpaceID),
		SpaceMemberID:      string(input.SpaceMemberID),
		Scopes:             model.ScopesToStrings(input.Scopes),
		Now:                time.Now(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// RecordPollは、許可待ちの要求へのポーリングを記録し、前回のポーリングから間隔を空けずに届いたか
// (tooFast) を返す。間隔を空けずに届いたときは、以降の間隔をslowDownだけ延ばす (RFC 8628 §3.5)。
// 許可待ちでない要求は更新せず、falseを返す
func (r *OAuthDeviceAuthorizationRepository) RecordPoll(ctx context.Context, id model.OAuthDeviceAuthorizationID, spaceID model.SpaceID, slowDown time.Duration, now time.Time) (tooFast bool, err error) {
	tooFast, err = r.q.RecordOAuthDeviceAuthorizationPoll(ctx, query.RecordOAuthDeviceAuthorizationPollParams{
		ID:      string(id),
		SpaceID: string(spaceID),
		// #nosec G115 -- 延ばす幅は数秒で、int32に収まる。
		SlowDownSeconds: int32(slowDown / time.Second),
		Now:             now,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return tooFast, nil
}

// ExchangeOAuthDeviceAuthorizationInputは、デバイスコードの交換の入力パラメータ。
// アクセストークンとリフレッシュトークンは、要求の許可・スコープで作る
type ExchangeOAuthDeviceAuthorizationInput struct {
	ID                    model.OAuthDeviceAuthorizationID
	SpaceID               model.SpaceID
	AccessTokenDigest     string
	AccessTokenExpiresAt  time.Time
	RefreshTokenDigest    string
	RefreshTokenExpiresAt time.Time
	Now                   time.Time
}

// Exchangeは、許可済みの要求を交換済みにし、要求の許可・スコープでアクセストークンとリフレッシュ
// トークンを作成して、交換済みにした要求を返す。判定と更新を1つの文で行うため、同じデバイスコードの
// 同時の交換でもトークンを作るのは1回だけである。許可済みでない場合は何もせず (nil, nil) を返す。
// 有効期限は呼び出し側が交換の前に確かめる
func (r *OAuthDeviceAuthorizationRepository) Exchange(ctx context.Context, input ExchangeOAuthDeviceAuthorizationInput) (*model.OAuthDeviceAuthorization, error) {
	row, err := r.q.ExchangeOAuthDeviceAuthorization(ctx, query.ExchangeOAuthDeviceAuthorizationParams{
		Now:                   input.Now,
		ID:                    string(input.ID),
		SpaceID:               string(input.SpaceID),
		AccessTokenDigest:     input.AccessTokenDigest,
		AccessTokenExpiresAt:  input.AccessTokenExpiresAt,
		RefreshTokenDigest:    input.RefreshTokenDigest,
		RefreshTokenExpiresAt: input.RefreshTokenExpiresAt,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(query.OauthDeviceAuthorization(row)), nil
}

// toModelはquery.OauthDeviceAuthorizationをmodel.OAuthDeviceAuthorizationに変換する
func (r *OAuthDeviceAuthorizationRepository) toModel(row query.OauthDeviceAuthorization) *model.OAuthDeviceAuthorization {
	var grantID *model.OAuthGrantID
	if row.OauthGrantID != nil {
		id := model.OAuthGrantID(*row.OauthGrantID)
		grantID = &id
	}
	var lastPolledAt *time.Time
	if row.LastPolledAt.Valid {
		lastPolledAt = &row.LastPolledAt.Time
	}
	return &model.OAuthDeviceAuthorization{
		ID:                 model.OAuthDeviceAuthorizationID(row.ID),
		OAuthApplicationID: model.OAuthApplicationID(row.OauthApplicationID),
		SpaceID:            model.SpaceID(row.SpaceID),
		OAuthGrantID:       grantID,
		DeviceCodeDigest:   row.DeviceCodeDigest,
		UserCodeDigest:     row.UserCodeDigest,
		Scopes:             model.StringsToScopes(row.Scopes),
		Status:             model.OAuthDeviceAuthorizationStatus(row.Status),
		Interval:           time.Duration(row.IntervalSeconds) * time.Second,
		ExpiresAt:          row.ExpiresAt,
		LastPolledAt:       lastPolledAt,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}
