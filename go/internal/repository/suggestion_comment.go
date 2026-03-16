package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// SuggestionCommentRepository は編集提案コメントリポジトリ
type SuggestionCommentRepository struct {
	q *query.Queries
}

// NewSuggestionCommentRepository は SuggestionCommentRepository を生成する
func NewSuggestionCommentRepository(q *query.Queries) *SuggestionCommentRepository {
	return &SuggestionCommentRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *SuggestionCommentRepository) WithTx(tx *sql.Tx) *SuggestionCommentRepository {
	return &SuggestionCommentRepository{q: r.q.WithTx(tx)}
}

// CreateSuggestionCommentInput は編集提案コメント作成の入力パラメータ
type CreateSuggestionCommentInput struct {
	SpaceID              model.SpaceID
	SuggestionID         model.SuggestionID
	CreatedSpaceMemberID model.SpaceMemberID
	Body                 string
	BodyHTML             string
}

// Create は編集提案コメントを作成する
func (r *SuggestionCommentRepository) Create(ctx context.Context, input CreateSuggestionCommentInput) (*model.SuggestionComment, error) {
	now := time.Now()

	row, err := r.q.CreateSuggestionComment(ctx, query.CreateSuggestionCommentParams{
		SpaceID:              string(input.SpaceID),
		SuggestionID:         string(input.SuggestionID),
		CreatedSpaceMemberID: string(input.CreatedSpaceMemberID),
		Body:                 input.Body,
		BodyHtml:             input.BodyHTML,
		CreatedAt:            now,
		UpdatedAt:            now,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByID はIDで編集提案コメントを取得する（スペースIDでスコープ）
func (r *SuggestionCommentRepository) FindByID(ctx context.Context, id model.SuggestionCommentID, spaceID model.SpaceID) (*model.SuggestionComment, error) {
	row, err := r.q.FindSuggestionCommentByID(ctx, query.FindSuggestionCommentByIDParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// ListBySuggestionID は編集提案IDでコメント一覧を取得する（作成日時の昇順）
func (r *SuggestionCommentRepository) ListBySuggestionID(ctx context.Context, suggestionID model.SuggestionID, spaceID model.SpaceID) ([]*model.SuggestionComment, error) {
	rows, err := r.q.ListSuggestionCommentsBySuggestionID(ctx, query.ListSuggestionCommentsBySuggestionIDParams{
		SuggestionID: string(suggestionID),
		SpaceID:      string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// CountBySuggestionID は編集提案IDでコメント数を取得する
func (r *SuggestionCommentRepository) CountBySuggestionID(ctx context.Context, suggestionID model.SuggestionID, spaceID model.SpaceID) (int64, error) {
	return r.q.CountSuggestionCommentsBySuggestionID(ctx, query.CountSuggestionCommentsBySuggestionIDParams{
		SuggestionID: string(suggestionID),
		SpaceID:      string(spaceID),
	})
}

// toModel は query.SuggestionComment を model.SuggestionComment に変換する
func (r *SuggestionCommentRepository) toModel(row query.SuggestionComment) *model.SuggestionComment {
	return &model.SuggestionComment{
		ID:                   model.SuggestionCommentID(row.ID),
		SpaceID:              model.SpaceID(row.SpaceID),
		SuggestionID:         model.SuggestionID(row.SuggestionID),
		CreatedSpaceMemberID: model.SpaceMemberID(row.CreatedSpaceMemberID),
		Body:                 row.Body,
		BodyHTML:             row.BodyHtml,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

// toModels は query.SuggestionComment のスライスを model.SuggestionComment のスライスに変換する
func (r *SuggestionCommentRepository) toModels(rows []query.SuggestionComment) []*model.SuggestionComment {
	comments := make([]*model.SuggestionComment, len(rows))
	for i, row := range rows {
		comments[i] = r.toModel(row)
	}
	return comments
}
