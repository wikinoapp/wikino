package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// EmailConfirmationRepositoryはメール確認リポジトリ
type EmailConfirmationRepository struct {
	q *query.Queries
}

// NewEmailConfirmationRepositoryはEmailConfirmationRepositoryを生成する
func NewEmailConfirmationRepository(q *query.Queries) *EmailConfirmationRepository {
	return &EmailConfirmationRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *EmailConfirmationRepository) WithTx(tx *sql.Tx) *EmailConfirmationRepository {
	return &EmailConfirmationRepository{q: r.q.WithTx(tx)}
}

// FindByIDはIDでメール確認情報を取得する
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

// FindActiveByEmailAndEventはメールアドレスとイベント種別で有効なメール確認情報を取得する
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

// CreateEmailConfirmationInputはメール確認情報作成の入力パラメータ
type CreateEmailConfirmationInput struct {
	UserID    *model.UserID
	Email     string
	Event     model.EmailConfirmationEvent
	Code      string
	StartedAt time.Time
}

// Createは新しいメール確認情報を作成する
func (r *EmailConfirmationRepository) Create(ctx context.Context, input CreateEmailConfirmationInput) (*model.EmailConfirmation, error) {
	now := time.Now()
	var userID *string
	if input.UserID != nil {
		id := string(*input.UserID)
		userID = &id
	}
	row, err := r.q.CreateEmailConfirmation(ctx, query.CreateEmailConfirmationParams{
		UserID:    userID,
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

// ConsumeEmailUpdateは、開始ユーザーとコードが一致する期限内の確認を一度だけ完了する。
// 条件を満たさない場合はnilを返す。メールアドレスの更新と同じトランザクションで呼ぶ。
func (r *EmailConfirmationRepository) ConsumeEmailUpdate(ctx context.Context, id string, userID model.UserID, code string) (*model.EmailConfirmation, error) {
	now := time.Now()
	ownerID := string(userID)
	row, err := r.q.ConsumeEmailUpdateConfirmation(ctx, query.ConsumeEmailUpdateConfirmationParams{
		ID:          id,
		UserID:      &ownerID,
		Code:        code,
		SucceededAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt:   now,
		ValidAfter:  now.Add(-15 * time.Minute),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// Succeedはメール確認を完了状態に更新する
func (r *EmailConfirmationRepository) Succeed(ctx context.Context, id string) error {
	now := time.Now()
	return r.q.UpdateEmailConfirmationSucceededAt(ctx, query.UpdateEmailConfirmationSucceededAtParams{
		ID:          id,
		SucceededAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt:   now,
	})
}

// DeleteStartedBeforeは、cutoffより前に開始したメール確認情報を確認の成否を問わず削除し、
// 削除した件数を返す
func (r *EmailConfirmationRepository) DeleteStartedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return r.q.DeleteEmailConfirmationsStartedBefore(ctx, cutoff)
}

// toModelはquery.EmailConfirmationをmodel.EmailConfirmationに変換する
func (r *EmailConfirmationRepository) toModel(row query.EmailConfirmation) *model.EmailConfirmation {
	var userID *model.UserID
	if row.UserID != nil {
		id := model.UserID(*row.UserID)
		userID = &id
	}
	var succeededAt *time.Time
	if row.SucceededAt.Valid {
		succeededAt = &row.SucceededAt.Time
	}
	return &model.EmailConfirmation{
		ID:          row.ID,
		UserID:      userID,
		Email:       row.Email,
		Event:       model.EmailConfirmationEvent(row.Event),
		Code:        row.Code,
		StartedAt:   row.StartedAt,
		SucceededAt: succeededAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
