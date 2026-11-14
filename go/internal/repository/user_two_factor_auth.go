package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// UserTwoFactorAuthRepositoryはユーザーの二要素認証リポジトリ
type UserTwoFactorAuthRepository struct {
	q *query.Queries
}

// NewUserTwoFactorAuthRepositoryはUserTwoFactorAuthRepositoryを生成する
func NewUserTwoFactorAuthRepository(q *query.Queries) *UserTwoFactorAuthRepository {
	return &UserTwoFactorAuthRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *UserTwoFactorAuthRepository) WithTx(tx *sql.Tx) *UserTwoFactorAuthRepository {
	return &UserTwoFactorAuthRepository{q: r.q.WithTx(tx)}
}

// FindByUserIDはユーザーIDで二要素認証設定を取得する
func (r *UserTwoFactorAuthRepository) FindByUserID(ctx context.Context, userID model.UserID) (*model.UserTwoFactorAuth, error) {
	row, err := r.q.GetUserTwoFactorAuthByUserID(ctx, string(userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindEnabledByUserIDはユーザーIDで有効な二要素認証設定を取得する
func (r *UserTwoFactorAuthRepository) FindEnabledByUserID(ctx context.Context, userID model.UserID) (*model.UserTwoFactorAuth, error) {
	row, err := r.q.GetEnabledUserTwoFactorAuthByUserID(ctx, string(userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// UpdateRecoveryCodesはリカバリーコードを更新する
func (r *UserTwoFactorAuthRepository) UpdateRecoveryCodes(ctx context.Context, userID model.UserID, recoveryCodes []string) error {
	return r.q.UpdateUserTwoFactorAuthRecoveryCodes(ctx, query.UpdateUserTwoFactorAuthRecoveryCodesParams{
		UserID:        string(userID),
		RecoveryCodes: recoveryCodes,
		UpdatedAt:     time.Now(),
	})
}

// UpsertPendingは、登録の途中の二要素認証設定を新しい秘密鍵で作るか書き換え、保存した設定を返す。
// 既に有効な設定は書き換えず、(nil, nil) を返す。
func (r *UserTwoFactorAuthRepository) UpsertPending(ctx context.Context, userID model.UserID, secret string) (*model.UserTwoFactorAuth, error) {
	row, err := r.q.UpsertPendingUserTwoFactorAuth(ctx, query.UpsertPendingUserTwoFactorAuthParams{
		UserID: string(userID),
		Secret: secret,
		Now:    time.Now(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// Enableは、秘密鍵がsecretのままの登録の途中の設定を有効にし、リカバリーコードを保存する。
// 対象の設定が無い (秘密鍵が差し替わった・既に有効になった) ときはfalseを返す。
func (r *UserTwoFactorAuthRepository) Enable(ctx context.Context, userID model.UserID, secret string, recoveryCodes []string) (bool, error) {
	rows, err := r.q.EnableUserTwoFactorAuth(ctx, query.EnableUserTwoFactorAuthParams{
		Now:           time.Now(),
		RecoveryCodes: recoveryCodes,
		UserID:        string(userID),
		Secret:        secret,
	})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// Disableは有効な二要素認証設定を無効にし、リカバリーコードを空にする。
// 有効な設定が無い (既に無効になった) ときはfalseを返す。
func (r *UserTwoFactorAuthRepository) Disable(ctx context.Context, userID model.UserID) (bool, error) {
	rows, err := r.q.DisableUserTwoFactorAuth(ctx, query.DisableUserTwoFactorAuthParams{
		Now:    time.Now(),
		UserID: string(userID),
	})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// RegenerateRecoveryCodesは有効な二要素認証設定のリカバリーコードを置き換える。
// 有効な設定が無い (無効になった) ときはfalseを返す。
func (r *UserTwoFactorAuthRepository) RegenerateRecoveryCodes(ctx context.Context, userID model.UserID, recoveryCodes []string) (bool, error) {
	rows, err := r.q.RegenerateUserTwoFactorAuthRecoveryCodes(ctx, query.RegenerateUserTwoFactorAuthRecoveryCodesParams{
		RecoveryCodes: recoveryCodes,
		Now:           time.Now(),
		UserID:        string(userID),
	})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// toModelはquery.UserTwoFactorAuthをmodel.UserTwoFactorAuthに変換する
func (r *UserTwoFactorAuthRepository) toModel(row query.UserTwoFactorAuth) *model.UserTwoFactorAuth {
	var enabledAt *time.Time
	if row.EnabledAt.Valid {
		enabledAt = &row.EnabledAt.Time
	}

	return &model.UserTwoFactorAuth{
		ID:            row.ID,
		UserID:        model.UserID(row.UserID),
		Secret:        row.Secret,
		Enabled:       row.Enabled,
		EnabledAt:     enabledAt,
		RecoveryCodes: row.RecoveryCodes,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}
