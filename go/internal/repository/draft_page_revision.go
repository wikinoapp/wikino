package repository

import (
	"context"
	"database/sql"
	"errors"
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

// DeleteByDraftPageID は下書きページIDに紐づくリビジョンをすべて削除する
func (r *DraftPageRevisionRepository) DeleteByDraftPageID(ctx context.Context, draftPageID model.DraftPageID, spaceID model.SpaceID) error {
	return r.q.DeleteDraftPageRevisionsByDraftPageID(ctx, query.DeleteDraftPageRevisionsByDraftPageIDParams{
		DraftPageID: string(draftPageID),
		SpaceID:     string(spaceID),
	})
}

// CountByDraftPageID は下書きページIDに紐づくリビジョン件数を返す
func (r *DraftPageRevisionRepository) CountByDraftPageID(ctx context.Context, draftPageID model.DraftPageID, spaceID model.SpaceID) (int64, error) {
	return r.q.CountDraftPageRevisionsByDraftPageID(ctx, query.CountDraftPageRevisionsByDraftPageIDParams{
		DraftPageID: string(draftPageID),
		SpaceID:     string(spaceID),
	})
}

// ListByDraftPageID returns the draft page's revisions newest-first, up to limit, scoped by space_id.
// [Ja] 下書きページのリビジョンを新しい順に最大 limit 件取得する (スペース ID でスコープ)。
func (r *DraftPageRevisionRepository) ListByDraftPageID(ctx context.Context, draftPageID model.DraftPageID, spaceID model.SpaceID, limit int32) ([]*model.DraftPageRevision, error) {
	rows, err := r.q.ListDraftPageRevisionsByDraftPageID(ctx, query.ListDraftPageRevisionsByDraftPageIDParams{
		DraftPageID: string(draftPageID),
		SpaceID:     string(spaceID),
		Limit:       limit,
	})
	if err != nil {
		return nil, err
	}
	revisions := make([]*model.DraftPageRevision, len(rows))
	for i, row := range rows {
		revisions[i] = r.toModel(row)
	}
	return revisions, nil
}

// FindByID returns a draft page revision by ID, scoped by space_id. Returns (nil, nil) when not found.
// [Ja] ID で下書きページリビジョンを取得する (スペース ID でスコープ)。見つからない場合は (nil, nil) を返す。
func (r *DraftPageRevisionRepository) FindByID(ctx context.Context, id model.DraftPageRevisionID, spaceID model.SpaceID) (*model.DraftPageRevision, error) {
	row, err := r.q.FindDraftPageRevisionByID(ctx, query.FindDraftPageRevisionByIDParams{
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

// FindPrevious returns the revision immediately preceding the given one (the diff comparison
// target), located by the passed revision's DraftPageID / SpaceID / CreatedAt / ID. "Preceding"
// uses the same (created_at, id) total order as ListByDraftPageID. Returns (nil, nil) when the
// given revision is the oldest one (no predecessor).
//
// [Ja] 対象リビジョンの直前のリビジョンを取得する (差分の比較対象)。比較対象は渡された
// リビジョンの DraftPageID / SpaceID / CreatedAt / ID で特定し、「直前」は ListByDraftPageID と
// 同じ (created_at, id) の全順序で定義する。対象が最古で直前が存在しない場合は (nil, nil) を返す。
func (r *DraftPageRevisionRepository) FindPrevious(ctx context.Context, revision *model.DraftPageRevision) (*model.DraftPageRevision, error) {
	row, err := r.q.FindPreviousDraftPageRevision(ctx, query.FindPreviousDraftPageRevisionParams{
		DraftPageID: string(revision.DraftPageID),
		SpaceID:     string(revision.SpaceID),
		CreatedAt:   revision.CreatedAt,
		ID:          string(revision.ID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
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
