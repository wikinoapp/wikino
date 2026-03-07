package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// DraftPageRevisionRepository は下書きページリビジョンリポジトリ
type DraftPageRevisionRepository struct {
	q *query.Queries
}

// NewDraftPageRevisionRepository は DraftPageRevisionRepository を生成する
func NewDraftPageRevisionRepository(q *query.Queries) *DraftPageRevisionRepository {
	return &DraftPageRevisionRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *DraftPageRevisionRepository) WithTx(tx *sql.Tx) *DraftPageRevisionRepository {
	return &DraftPageRevisionRepository{q: r.q.WithTx(tx)}
}

// CreateDraftPageRevisionInput は下書きページリビジョン作成の入力パラメータ
type CreateDraftPageRevisionInput struct {
	DraftPageID   model.DraftPageID
	SpaceID       model.SpaceID
	SpaceMemberID model.SpaceMemberID
	Title         string
	Body          string
	BodyHTML      string
}

// Create は下書きページリビジョンを作成する
func (r *DraftPageRevisionRepository) Create(ctx context.Context, input CreateDraftPageRevisionInput) (*model.DraftPageRevision, error) {
	row, err := r.q.CreateDraftPageRevision(ctx, query.CreateDraftPageRevisionParams{
		DraftPageID:   string(input.DraftPageID),
		SpaceID:       string(input.SpaceID),
		SpaceMemberID: string(input.SpaceMemberID),
		Title:         input.Title,
		Body:          input.Body,
		BodyHtml:      input.BodyHTML,
		CreatedAt:     time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// toModel は query.DraftPageRevision を model.DraftPageRevision に変換する
func (r *DraftPageRevisionRepository) toModel(row query.DraftPageRevision) *model.DraftPageRevision {
	return &model.DraftPageRevision{
		ID:            model.DraftPageRevisionID(row.ID),
		DraftPageID:   model.DraftPageID(row.DraftPageID),
		SpaceID:       model.SpaceID(row.SpaceID),
		SpaceMemberID: model.SpaceMemberID(row.SpaceMemberID),
		Title:         row.Title,
		Body:          row.Body,
		BodyHTML:      row.BodyHtml,
		CreatedAt:     row.CreatedAt,
	}
}
