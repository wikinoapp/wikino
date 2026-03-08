package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// UserSessionRepository はユーザーセッションリポジトリ
type UserSessionRepository struct {
	q *query.Queries
}

// NewUserSessionRepository は UserSessionRepository を生成する
func NewUserSessionRepository(q *query.Queries) *UserSessionRepository {
	return &UserSessionRepository{q: q}
}

// FindByID はセッションIDでセッションを取得する
func (r *UserSessionRepository) FindByID(ctx context.Context, id string) (*model.UserSession, error) {
	row, err := r.q.GetUserSessionByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByToken はセッショントークンでセッションを取得する
func (r *UserSessionRepository) FindByToken(ctx context.Context, token string) (*model.UserSession, error) {
	row, err := r.q.GetUserSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// CreateInput はセッション作成の入力パラメータ
type CreateInput struct {
	UserID     model.UserID
	Token      string
	IPAddress  string
	UserAgent  string
	SignedInAt time.Time
}

// Create は新しいセッションを作成する
func (r *UserSessionRepository) Create(ctx context.Context, input CreateInput) (*model.UserSession, error) {
	now := time.Now()
	row, err := r.q.CreateUserSession(ctx, query.CreateUserSessionParams{
		UserID:     string(input.UserID),
		Token:      input.Token,
		IpAddress:  input.IPAddress,
		UserAgent:  input.UserAgent,
		SignedInAt: input.SignedInAt,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// Delete はセッションを削除する
func (r *UserSessionRepository) Delete(ctx context.Context, id string) error {
	return r.q.DeleteUserSession(ctx, id)
}

// DeleteByToken はセッショントークンでセッションを削除する
func (r *UserSessionRepository) DeleteByToken(ctx context.Context, token string) error {
	return r.q.DeleteUserSessionByToken(ctx, token)
}

// toModel は query.UserSession を model.UserSession に変換する
func (r *UserSessionRepository) toModel(row query.UserSession) *model.UserSession {
	return &model.UserSession{
		ID:         row.ID,
		UserID:     model.UserID(row.UserID),
		Token:      row.Token,
		IPAddress:  row.IpAddress,
		UserAgent:  row.UserAgent,
		SignedInAt: row.SignedInAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}
