package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// SuggestionPageRevisionRepository は編集提案ページリビジョンリポジトリ
type SuggestionPageRevisionRepository struct {
	q *query.Queries
}

// NewSuggestionPageRevisionRepository は SuggestionPageRevisionRepository を生成する
func NewSuggestionPageRevisionRepository(q *query.Queries) *SuggestionPageRevisionRepository {
	return &SuggestionPageRevisionRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *SuggestionPageRevisionRepository) WithTx(tx *sql.Tx) *SuggestionPageRevisionRepository {
	return &SuggestionPageRevisionRepository{q: r.q.WithTx(tx)}
}

// CreateSuggestionPageRevisionInput は編集提案ページリビジョン作成の入力パラメータ
type CreateSuggestionPageRevisionInput struct {
	SpaceID             model.SpaceID
	SuggestionPageID    model.SuggestionPageID
	EditorSpaceMemberID model.SpaceMemberID
	Title               *string
	Body                string
	BodyHTML            string
}

// Create は編集提案ページリビジョンを作成する
func (r *SuggestionPageRevisionRepository) Create(ctx context.Context, input CreateSuggestionPageRevisionInput) (*model.SuggestionPageRevision, error) {
	now := time.Now()

	var title sql.NullString
	if input.Title != nil {
		title = sql.NullString{String: *input.Title, Valid: true}
	}

	row, err := r.q.CreateSuggestionPageRevision(ctx, query.CreateSuggestionPageRevisionParams{
		SpaceID:             string(input.SpaceID),
		SuggestionPageID:    string(input.SuggestionPageID),
		EditorSpaceMemberID: string(input.EditorSpaceMemberID),
		Title:               title,
		Body:                input.Body,
		BodyHtml:            input.BodyHTML,
		CreatedAt:           now,
		UpdatedAt:           now,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// ListBySuggestionPageID は編集提案ページIDでリビジョン一覧を取得する
func (r *SuggestionPageRevisionRepository) ListBySuggestionPageID(ctx context.Context, suggestionPageID model.SuggestionPageID, spaceID model.SpaceID) ([]*model.SuggestionPageRevision, error) {
	rows, err := r.q.ListSuggestionPageRevisionsBySuggestionPageID(ctx, query.ListSuggestionPageRevisionsBySuggestionPageIDParams{
		SuggestionPageID: string(suggestionPageID),
		SpaceID:          string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// FindLatest は編集提案ページの最新リビジョンを取得する（スペースIDでスコープ）
func (r *SuggestionPageRevisionRepository) FindLatest(ctx context.Context, suggestionPageID model.SuggestionPageID, spaceID model.SpaceID) (*model.SuggestionPageRevision, error) {
	row, err := r.q.FindLatestSuggestionPageRevision(ctx, query.FindLatestSuggestionPageRevisionParams{
		SuggestionPageID: string(suggestionPageID),
		SpaceID:          string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// DeleteBySuggestionPageID は編集提案ページIDでリビジョンを一括削除する
func (r *SuggestionPageRevisionRepository) DeleteBySuggestionPageID(ctx context.Context, suggestionPageID model.SuggestionPageID, spaceID model.SpaceID) error {
	return r.q.DeleteSuggestionPageRevisionsBySuggestionPageID(ctx, query.DeleteSuggestionPageRevisionsBySuggestionPageIDParams{
		SuggestionPageID: string(suggestionPageID),
		SpaceID:          string(spaceID),
	})
}

// toModel は query.SuggestionPageRevision を model.SuggestionPageRevision に変換する
func (r *SuggestionPageRevisionRepository) toModel(row query.SuggestionPageRevision) *model.SuggestionPageRevision {
	var title *string
	if row.Title.Valid {
		title = &row.Title.String
	}

	return &model.SuggestionPageRevision{
		ID:                  model.SuggestionPageRevisionID(row.ID),
		SpaceID:             model.SpaceID(row.SpaceID),
		SuggestionPageID:    model.SuggestionPageID(row.SuggestionPageID),
		EditorSpaceMemberID: model.SpaceMemberID(row.EditorSpaceMemberID),
		Title:               title,
		Body:                row.Body,
		BodyHTML:            row.BodyHtml,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

// toModels は query.SuggestionPageRevision のスライスを model.SuggestionPageRevision のスライスに変換する
func (r *SuggestionPageRevisionRepository) toModels(rows []query.SuggestionPageRevision) []*model.SuggestionPageRevision {
	revisions := make([]*model.SuggestionPageRevision, len(rows))
	for i, row := range rows {
		revisions[i] = r.toModel(row)
	}
	return revisions
}
