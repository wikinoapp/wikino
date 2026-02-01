package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// EmailConfirmationRepository はメール確認リポジトリ
type EmailConfirmationRepository struct {
	q *query.Queries
}

// NewEmailConfirmationRepository は EmailConfirmationRepository を生成する
func NewEmailConfirmationRepository(q *query.Queries) *EmailConfirmationRepository {
	return &EmailConfirmationRepository{q: q}
}

// FindByID はIDでメール確認情報を取得する
func (r *EmailConfirmationRepository) FindByID(ctx context.Context, id string) (*model.EmailConfirmation, error) {
	row, err := r.q.GetEmailConfirmationByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindActiveByEmailAndEvent はメールアドレスとイベント種別で有効なメール確認情報を取得する
func (r *EmailConfirmationRepository) FindActiveByEmailAndEvent(ctx context.Context, email string, event model.EmailConfirmationEvent) (*model.EmailConfirmation, error) {
	row, err := r.q.GetActiveEmailConfirmationByEmailAndEvent(ctx, query.GetActiveEmailConfirmationByEmailAndEventParams{
		Email: email,
		Event: int32(event),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// CreateEmailConfirmationInput はメール確認情報作成の入力パラメータ
type CreateEmailConfirmationInput struct {
	Email     string
	Event     model.EmailConfirmationEvent
	Code      string
	StartedAt time.Time
}

// Create は新しいメール確認情報を作成する
func (r *EmailConfirmationRepository) Create(ctx context.Context, input CreateEmailConfirmationInput) (*model.EmailConfirmation, error) {
	now := time.Now()
	row, err := r.q.CreateEmailConfirmation(ctx, query.CreateEmailConfirmationParams{
		Email:     input.Email,
		Event:     int32(input.Event),
		Code:      input.Code,
		StartedAt: input.StartedAt,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// Succeed はメール確認を完了状態に更新する
func (r *EmailConfirmationRepository) Succeed(ctx context.Context, id string) error {
	now := time.Now()
	return r.q.UpdateEmailConfirmationSucceededAt(ctx, query.UpdateEmailConfirmationSucceededAtParams{
		ID:          id,
		SucceededAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt:   now,
	})
}

// toModel は query.EmailConfirmation を model.EmailConfirmation に変換する
func (r *EmailConfirmationRepository) toModel(row query.EmailConfirmation) *model.EmailConfirmation {
	var succeededAt *time.Time
	if row.SucceededAt.Valid {
		succeededAt = &row.SucceededAt.Time
	}
	return &model.EmailConfirmation{
		ID:          row.ID,
		Email:       row.Email,
		Event:       model.EmailConfirmationEvent(row.Event),
		Code:        row.Code,
		StartedAt:   row.StartedAt,
		SucceededAt: succeededAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
