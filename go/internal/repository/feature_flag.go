package repository

import (
	"context"
	"database/sql"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// FeatureFlagRepositoryはフィーチャーフラグリポジトリ
type FeatureFlagRepository struct {
	q *query.Queries
}

// NewFeatureFlagRepositoryはFeatureFlagRepositoryを生成する
func NewFeatureFlagRepository(q *query.Queries) *FeatureFlagRepository {
	return &FeatureFlagRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *FeatureFlagRepository) WithTx(tx *sql.Tx) *FeatureFlagRepository {
	return &FeatureFlagRepository{q: r.q.WithTx(tx)}
}

// IsEnabledは指定ユーザーに対してフラグが有効かどうかを返す
func (r *FeatureFlagRepository) IsEnabled(ctx context.Context, userID model.UserID, name model.FeatureFlagName) (bool, error) {
	uid := string(userID)
	return r.q.IsFeatureFlagEnabled(ctx, query.IsFeatureFlagEnabledParams{
		UserID: &uid,
		Name:   string(name),
	})
}

// IsEnabledForDeviceはデバイストークンまたはログインセッション経由でフラグが有効かどうかを返す
// deviceTokenとsessionTokenの両方を受け取り、1クエリで判定する
func (r *FeatureFlagRepository) IsEnabledForDevice(ctx context.Context, deviceToken string, sessionToken string, name model.FeatureFlagName) (bool, error) {
	return r.q.IsFeatureFlagEnabledForDevice(ctx, query.IsFeatureFlagEnabledForDeviceParams{
		DeviceToken: sql.NullString{String: deviceToken, Valid: deviceToken != ""},
		Token:       sessionToken,
		Name:        string(name),
	})
}
