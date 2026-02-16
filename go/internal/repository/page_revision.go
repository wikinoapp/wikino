package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// PageRevisionRepository はページリビジョンリポジトリ
type PageRevisionRepository struct {
	q *query.Queries
}

// NewPageRevisionRepository は PageRevisionRepository を生成する
func NewPageRevisionRepository(q *query.Queries) *PageRevisionRepository {
	return &PageRevisionRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *PageRevisionRepository) WithTx(tx *sql.Tx) *PageRevisionRepository {
	return &PageRevisionRepository{q: r.q.WithTx(tx)}
}

// CreatePageRevisionInput はページリビジョン作成の入力パラメータ
type CreatePageRevisionInput struct {
	SpaceID       model.SpaceID
	SpaceMemberID model.SpaceMemberID
	PageID        model.PageID
	Title         string
	Body          string
	BodyHTML      string
}

// Create はページリビジョンを作成する
func (r *PageRevisionRepository) Create(ctx context.Context, input CreatePageRevisionInput) (*model.PageRevision, error) {
	now := time.Now()
	row, err := r.q.CreatePageRevision(ctx, query.CreatePageRevisionParams{
		SpaceID:       string(input.SpaceID),
		SpaceMemberID: string(input.SpaceMemberID),
		PageID:        string(input.PageID),
		Title:         input.Title,
		Body:          input.Body,
		BodyHtml:      input.BodyHTML,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// toModel は query.PageRevision を model.PageRevision に変換する
func (r *PageRevisionRepository) toModel(row query.PageRevision) *model.PageRevision {
	return &model.PageRevision{
		ID:            row.ID,
		SpaceID:       model.SpaceID(row.SpaceID),
		SpaceMemberID: model.SpaceMemberID(row.SpaceMemberID),
		PageID:        model.PageID(row.PageID),
		Title:         row.Title,
		Body:          row.Body,
		BodyHTML:      row.BodyHtml,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}
