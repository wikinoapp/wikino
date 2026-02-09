package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// UserTwoFactorAuthRepository はユーザーの二要素認証リポジトリ
type UserTwoFactorAuthRepository struct {
	q *query.Queries
}

// NewUserTwoFactorAuthRepository は UserTwoFactorAuthRepository を生成する
func NewUserTwoFactorAuthRepository(q *query.Queries) *UserTwoFactorAuthRepository {
	return &UserTwoFactorAuthRepository{q: q}
}

// FindByUserID はユーザーIDで二要素認証設定を取得する
func (r *UserTwoFactorAuthRepository) FindByUserID(ctx context.Context, userID string) (*model.UserTwoFactorAuth, error) {
	row, err := r.q.GetUserTwoFactorAuthByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindEnabledByUserID はユーザーIDで有効な二要素認証設定を取得する
func (r *UserTwoFactorAuthRepository) FindEnabledByUserID(ctx context.Context, userID string) (*model.UserTwoFactorAuth, error) {
	row, err := r.q.GetEnabledUserTwoFactorAuthByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// UpdateRecoveryCodes はリカバリーコードを更新する
func (r *UserTwoFactorAuthRepository) UpdateRecoveryCodes(ctx context.Context, userID string, recoveryCodes []string) error {
	return r.q.UpdateUserTwoFactorAuthRecoveryCodes(ctx, query.UpdateUserTwoFactorAuthRecoveryCodesParams{
		UserID:        userID,
		RecoveryCodes: recoveryCodes,
		UpdatedAt:     time.Now(),
	})
}

// toModel は query.UserTwoFactorAuth を model.UserTwoFactorAuth に変換する
func (r *UserTwoFactorAuthRepository) toModel(row query.UserTwoFactorAuth) *model.UserTwoFactorAuth {
	var enabledAt *time.Time
	if row.EnabledAt.Valid {
		enabledAt = &row.EnabledAt.Time
	}

	return &model.UserTwoFactorAuth{
		ID:            row.ID,
		UserID:        row.UserID,
		Secret:        row.Secret,
		Enabled:       row.Enabled,
		EnabledAt:     enabledAt,
		RecoveryCodes: row.RecoveryCodes,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}
