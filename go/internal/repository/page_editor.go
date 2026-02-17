package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// PageEditorRepository はページ編集者リポジトリ
type PageEditorRepository struct {
	q *query.Queries
}

// NewPageEditorRepository は PageEditorRepository を生成する
func NewPageEditorRepository(q *query.Queries) *PageEditorRepository {
	return &PageEditorRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *PageEditorRepository) WithTx(tx *sql.Tx) *PageEditorRepository {
	return &PageEditorRepository{q: r.q.WithTx(tx)}
}

// FindOrCreateInput はページ編集者のFindOrCreateの入力パラメータ
type FindOrCreateInput struct {
	SpaceID            model.SpaceID
	PageID             model.PageID
	SpaceMemberID      model.SpaceMemberID
	LastPageModifiedAt time.Time
}

// FindOrCreate はページ編集者を取得し、存在しない場合は作成する
func (r *PageEditorRepository) FindOrCreate(ctx context.Context, input FindOrCreateInput) (*model.PageEditor, error) {
	row, err := r.q.FindPageEditorByPageAndSpaceMember(ctx, query.FindPageEditorByPageAndSpaceMemberParams{
		PageID:        string(input.PageID),
		SpaceMemberID: string(input.SpaceMemberID),
		SpaceID:       string(input.SpaceID),
	})
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}

		// 存在しない場合は作成する
		now := time.Now()
		row, err = r.q.CreatePageEditor(ctx, query.CreatePageEditorParams{
			SpaceID:            string(input.SpaceID),
			PageID:             string(input.PageID),
			SpaceMemberID:      string(input.SpaceMemberID),
			LastPageModifiedAt: input.LastPageModifiedAt,
			CreatedAt:          now,
			UpdatedAt:          now,
		})
		if err != nil {
			return nil, err
		}
	}

	return r.toModel(row), nil
}

// UpdateLastPageModifiedAtInput はページ編集者のlast_page_modified_at更新の入力パラメータ
type UpdateLastPageModifiedAtInput struct {
	ID                 model.PageEditorID
	SpaceID            model.SpaceID
	LastPageModifiedAt time.Time
}

// UpdateLastPageModifiedAt はページ編集者のlast_page_modified_atを更新する
func (r *PageEditorRepository) UpdateLastPageModifiedAt(ctx context.Context, input UpdateLastPageModifiedAtInput) (*model.PageEditor, error) {
	row, err := r.q.UpdatePageEditorLastPageModifiedAt(ctx, query.UpdatePageEditorLastPageModifiedAtParams{
		ID:                 string(input.ID),
		LastPageModifiedAt: input.LastPageModifiedAt,
		UpdatedAt:          time.Now(),
		SpaceID:            string(input.SpaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// toModel は query.PageEditor を model.PageEditor に変換する
func (r *PageEditorRepository) toModel(row query.PageEditor) *model.PageEditor {
	return &model.PageEditor{
		ID:                 model.PageEditorID(row.ID),
		SpaceID:            model.SpaceID(row.SpaceID),
		PageID:             model.PageID(row.PageID),
		SpaceMemberID:      model.SpaceMemberID(row.SpaceMemberID),
		LastPageModifiedAt: row.LastPageModifiedAt,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}
