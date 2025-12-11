package repository

import (
	"context"
	"database/sql"
	"errors"

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

// FindByUserID はユーザーIDでパスワード情報を取得する
func (r *UserPasswordRepository) FindByUserID(ctx context.Context, userID string) (*model.UserPassword, error) {
	row, err := r.q.GetUserPasswordByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// toModel は query.UserPassword を model.UserPassword に変換する
func (r *UserPasswordRepository) toModel(row query.UserPassword) *model.UserPassword {
	return &model.UserPassword{
		ID:             row.ID,
		UserID:         row.UserID,
		PasswordDigest: row.PasswordDigest,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}
