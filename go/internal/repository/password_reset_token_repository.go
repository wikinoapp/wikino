package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// PasswordResetTokenRepository はパスワードリセットトークンリポジトリ
type PasswordResetTokenRepository struct {
	q *query.Queries
}

// NewPasswordResetTokenRepository は PasswordResetTokenRepository を生成する
func NewPasswordResetTokenRepository(q *query.Queries) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *PasswordResetTokenRepository) WithTx(tx *sql.Tx) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{q: r.q.WithTx(tx)}
}

// FindByTokenDigest はトークンダイジェストでパスワードリセットトークンを取得する
func (r *PasswordResetTokenRepository) FindByTokenDigest(ctx context.Context, tokenDigest string) (*model.PasswordResetToken, error) {
	row, err := r.q.GetPasswordResetTokenByTokenDigest(ctx, tokenDigest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// CreatePasswordResetTokenInput はパスワードリセットトークン作成の入力パラメータ
type CreatePasswordResetTokenInput struct {
	UserID      model.UserID
	TokenDigest string
	ExpiresAt   time.Time
}

// Create は新しいパスワードリセットトークンを作成する
func (r *PasswordResetTokenRepository) Create(ctx context.Context, input CreatePasswordResetTokenInput) (*model.PasswordResetToken, error) {
	now := time.Now()
	row, err := r.q.CreatePasswordResetToken(ctx, query.CreatePasswordResetTokenParams{
		UserID:      string(input.UserID),
		TokenDigest: input.TokenDigest,
		ExpiresAt:   input.ExpiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// MarkAsUsed はパスワードリセットトークンを使用済みにマークする
func (r *PasswordResetTokenRepository) MarkAsUsed(ctx context.Context, id string) error {
	now := time.Now()
	return r.q.UpdatePasswordResetTokenUsedAt(ctx, query.UpdatePasswordResetTokenUsedAtParams{
		ID:        id,
		UsedAt:    sql.NullTime{Time: now, Valid: true},
		UpdatedAt: now,
	})
}

// DeleteUnusedByUserID はユーザーIDで未使用のパスワードリセットトークンを削除する
func (r *PasswordResetTokenRepository) DeleteUnusedByUserID(ctx context.Context, userID model.UserID) error {
	return r.q.DeleteUnusedPasswordResetTokensByUserID(ctx, string(userID))
}

// toModel は query.PasswordResetToken を model.PasswordResetToken に変換する
func (r *PasswordResetTokenRepository) toModel(row query.PasswordResetToken) *model.PasswordResetToken {
	var usedAt *time.Time
	if row.UsedAt.Valid {
		usedAt = &row.UsedAt.Time
	}
	return &model.PasswordResetToken{
		ID:          row.ID,
		UserID:      model.UserID(row.UserID),
		TokenDigest: row.TokenDigest,
		ExpiresAt:   row.ExpiresAt,
		UsedAt:      usedAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
