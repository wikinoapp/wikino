package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// UserPasswordRepository はユーザーパスワードリポジトリ
type UserPasswordRepository struct {
	q *query.Queries
}

// NewUserPasswordRepository は UserPasswordRepository を生成する
func NewUserPasswordRepository(q *query.Queries) *UserPasswordRepository {
	return &UserPasswordRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *UserPasswordRepository) WithTx(tx *sql.Tx) *UserPasswordRepository {
	return &UserPasswordRepository{q: r.q.WithTx(tx)}
}

// FindByUserID はユーザーIDでパスワード情報を取得する
func (r *UserPasswordRepository) FindByUserID(ctx context.Context, userID model.UserID) (*model.UserPassword, error) {
	row, err := r.q.GetUserPasswordByUserID(ctx, string(userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// CreateUserPasswordInput はユーザーパスワード作成の入力パラメータ
type CreateUserPasswordInput struct {
	UserID         model.UserID
	PasswordDigest string
}

// Create は新しいユーザーパスワードを作成する
func (r *UserPasswordRepository) Create(ctx context.Context, input CreateUserPasswordInput) (*model.UserPassword, error) {
	now := time.Now()
	row, err := r.q.CreateUserPassword(ctx, query.CreateUserPasswordParams{
		UserID:         string(input.UserID),
		PasswordDigest: input.PasswordDigest,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// UpdatePasswordDigest はユーザーIDでパスワードダイジェストを更新する
func (r *UserPasswordRepository) UpdatePasswordDigest(ctx context.Context, userID model.UserID, passwordDigest string) error {
	now := time.Now()
	return r.q.UpdateUserPasswordDigest(ctx, query.UpdateUserPasswordDigestParams{
		UserID:         string(userID),
		PasswordDigest: passwordDigest,
		UpdatedAt:      now,
	})
}

// toModel は query.UserPassword を model.UserPassword に変換する
func (r *UserPasswordRepository) toModel(row query.UserPassword) *model.UserPassword {
	return &model.UserPassword{
		ID:             row.ID,
		UserID:         model.UserID(row.UserID),
		PasswordDigest: row.PasswordDigest,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}
