package repository

import (
	"context"
	"database/sql"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// FeatureFlagRepository はフィーチャーフラグリポジトリ
type FeatureFlagRepository struct {
	q *query.Queries
}

// NewFeatureFlagRepository は FeatureFlagRepository を生成する
func NewFeatureFlagRepository(q *query.Queries) *FeatureFlagRepository {
	return &FeatureFlagRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *FeatureFlagRepository) WithTx(tx *sql.Tx) *FeatureFlagRepository {
	return &FeatureFlagRepository{q: r.q.WithTx(tx)}
}

// IsEnabled は指定ユーザーに対してフラグが有効かどうかを返す
func (r *FeatureFlagRepository) IsEnabled(ctx context.Context, userID model.UserID, name model.FeatureFlagName) (bool, error) {
	return r.q.IsFeatureFlagEnabled(ctx, query.IsFeatureFlagEnabledParams{
		UserID: string(userID),
		Name:   string(name),
	})
}

// IsEnabledBySessionToken はセッショントークンからユーザーを特定し、フラグが有効かどうかを返す
func (r *FeatureFlagRepository) IsEnabledBySessionToken(ctx context.Context, sessionToken string, name model.FeatureFlagName) (bool, error) {
	return r.q.IsFeatureFlagEnabledBySessionToken(ctx, query.IsFeatureFlagEnabledBySessionTokenParams{
		Token: sessionToken,
		Name:  string(name),
	})
}
