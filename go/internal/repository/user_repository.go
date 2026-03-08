// Package repository はデータアクセス層を提供します
package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// UserRepository はユーザーリポジトリ
type UserRepository struct {
	q *query.Queries
}

// NewUserRepository は UserRepository を生成する
func NewUserRepository(q *query.Queries) *UserRepository {
	return &UserRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *UserRepository) WithTx(tx *sql.Tx) *UserRepository {
	return &UserRepository{q: r.q.WithTx(tx)}
}

// FindByID はIDでユーザーを取得する
func (r *UserRepository) FindByID(ctx context.Context, id model.UserID) (*model.User, error) {
	row, err := r.q.GetUserByID(ctx, string(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByEmail はメールアドレスでユーザーを取得する（削除されていないユーザーのみ）
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByAtname はアットネームでユーザーを取得する（削除されていないユーザーのみ）
func (r *UserRepository) FindByAtname(ctx context.Context, atname string) (*model.User, error) {
	row, err := r.q.GetUserByAtname(ctx, atname)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// CreateUserInput はユーザー作成の入力パラメータ
type CreateUserInput struct {
	Email       string
	Atname      string
	Name        string
	Description string
	Locale      model.Locale
	TimeZone    string
	JoinedAt    time.Time
}

// Create は新しいユーザーを作成する
func (r *UserRepository) Create(ctx context.Context, input CreateUserInput) (*model.User, error) {
	now := time.Now()
	row, err := r.q.CreateUser(ctx, query.CreateUserParams{
		Email:       input.Email,
		Atname:      input.Atname,
		Name:        input.Name,
		Description: input.Description,
		Locale:      int32(input.Locale),
		TimeZone:    input.TimeZone,
		JoinedAt:    input.JoinedAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// toModel は query.User を model.User に変換する
func (r *UserRepository) toModel(row query.User) *model.User {
	var discardedAt *time.Time
	if row.DiscardedAt.Valid {
		discardedAt = &row.DiscardedAt.Time
	}

	return &model.User{
		ID:          model.UserID(row.ID),
		Email:       row.Email,
		Atname:      row.Atname,
		Name:        row.Name,
		Description: row.Description,
		Locale:      model.Locale(row.Locale),
		TimeZone:    row.TimeZone,
		JoinedAt:    row.JoinedAt,
		DiscardedAt: discardedAt,
	}
}
