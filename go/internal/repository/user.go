// Package repositoryはデータアクセス層を提供します
package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// UserRepositoryはユーザーリポジトリ
type UserRepository struct {
	q *query.Queries
}

// NewUserRepositoryはUserRepositoryを生成する
func NewUserRepository(q *query.Queries) *UserRepository {
	return &UserRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *UserRepository) WithTx(tx *sql.Tx) *UserRepository {
	return &UserRepository{q: r.q.WithTx(tx)}
}

// FindByIDはIDでユーザーを取得する
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

// FindByEmailはメールアドレスでユーザーを取得する (削除されていないユーザーのみ)
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

// FindByAtnameはアットネームでユーザーを取得する (削除されていないユーザーのみ)
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

// FindByIDsはIDリストでユーザーを一括取得する
func (r *UserRepository) FindByIDs(ctx context.Context, ids []model.UserID) ([]*model.User, error) {
	if len(ids) == 0 {
		return []*model.User{}, nil
	}
	rows, err := r.q.FindUsersByIDs(ctx, model.UserIDsToStrings(ids))
	if err != nil {
		return nil, err
	}
	users := make([]*model.User, len(rows))
	for i, row := range rows {
		users[i] = r.toModel(row)
	}
	return users, nil
}

// CreateUserInputはユーザー作成の入力パラメータ
type CreateUserInput struct {
	Email       string
	Atname      string
	Name        string
	Description string
	Locale      model.Locale
	TimeZone    string
	JoinedAt    time.Time
}

// Createは新しいユーザーを作成する
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

// ExistsByAtnameはアットネームを使っているユーザーがいるかを返す。excludeIDを渡すと、その
// ユーザーを数えない。アットネームの一意インデックスは削除済みのユーザーも含むため、削除済みの
// ユーザーも数える。
func (r *UserRepository) ExistsByAtname(ctx context.Context, atname string, excludeID *model.UserID) (bool, error) {
	if excludeID == nil {
		return r.q.ExistsUserByAtname(ctx, atname)
	}
	return r.q.ExistsOtherUserByAtname(ctx, query.ExistsOtherUserByAtnameParams{
		Atname: atname,
		UserID: string(*excludeID),
	})
}

// UpdateProfileInputはプロフィールの更新の入力パラメータ
type UpdateProfileInput struct {
	ID          model.UserID
	Atname      string
	Name        string
	Description string
}

// UpdateProfileはユーザーのアットネーム・名前・自己紹介を更新する。削除されたユーザーは
// 更新せず、(nil, nil) を返す。
func (r *UserRepository) UpdateProfile(ctx context.Context, input UpdateProfileInput) (*model.User, error) {
	row, err := r.q.UpdateUserProfile(ctx, query.UpdateUserProfileParams{
		Atname:      input.Atname,
		Name:        input.Name,
		Description: input.Description,
		Now:         time.Now(),
		ID:          string(input.ID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// ExistsByEmailはメールアドレスを使っているユーザーがいるかを返す。メールアドレスの一意
// インデックスは削除済みのユーザーも含むため、削除済みのユーザーも数える。
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.q.ExistsUserByEmail(ctx, email)
}

// UpdateEmailはユーザーのメールアドレスを更新する。削除されたユーザーは更新せず、(nil, nil) を
// 返す。
func (r *UserRepository) UpdateEmail(ctx context.Context, id model.UserID, email string) (*model.User, error) {
	row, err := r.q.UpdateUserEmail(ctx, query.UpdateUserEmailParams{
		Email: email,
		Now:   time.Now(),
		ID:    string(id),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// Discardはユーザーを論理削除し、論理削除したかを返す。既に論理削除したユーザーではfalseを返す。
func (r *UserRepository) Discard(ctx context.Context, id model.UserID, discardedAt time.Time) (bool, error) {
	n, err := r.q.DiscardUser(ctx, query.DiscardUserParams{
		ID:          string(id),
		DiscardedAt: sql.NullTime{Time: discardedAt, Valid: true},
	})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Undiscardは、discardedAtに論理削除したユーザーの論理削除を取り消す。
func (r *UserRepository) Undiscard(ctx context.Context, id model.UserID, discardedAt time.Time, updatedAt time.Time) error {
	return r.q.UndiscardUser(ctx, query.UndiscardUserParams{
		ID:          string(id),
		DiscardedAt: sql.NullTime{Time: discardedAt, Valid: true},
		UpdatedAt:   updatedAt,
	})
}

// LockDiscardedは論理削除したユーザーをロックし、ロックできたかを返す。論理削除していない
// ユーザーと、見つからないユーザーではfalseを返す。
func (r *UserRepository) LockDiscarded(ctx context.Context, id model.UserID) (bool, error) {
	if _, err := r.q.LockDiscardedUser(ctx, string(id)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DeleteDiscardedは論理削除したユーザーを物理削除し、削除したかを返す。
func (r *UserRepository) DeleteDiscarded(ctx context.Context, id model.UserID) (bool, error) {
	n, err := r.q.DeleteDiscardedUser(ctx, string(id))
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// toModelはquery.Userをmodel.Userに変換する
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
