package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// PageAttachmentReferenceRepositoryはページ添付ファイル参照リポジトリ
type PageAttachmentReferenceRepository struct {
	q *query.Queries
}

// NewPageAttachmentReferenceRepositoryはPageAttachmentReferenceRepositoryを生成する
func NewPageAttachmentReferenceRepository(q *query.Queries) *PageAttachmentReferenceRepository {
	return &PageAttachmentReferenceRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *PageAttachmentReferenceRepository) WithTx(tx *sql.Tx) *PageAttachmentReferenceRepository {
	return &PageAttachmentReferenceRepository{q: r.q.WithTx(tx)}
}

// ListByPageIDはページIDに紐づく添付ファイル参照を取得する (スペースIDでスコープ)
func (r *PageAttachmentReferenceRepository) ListByPageID(ctx context.Context, pageID model.PageID, spaceID model.SpaceID) ([]*model.PageAttachmentReference, error) {
	rows, err := r.q.ListPageAttachmentReferencesByPageID(ctx, query.ListPageAttachmentReferencesByPageIDParams{
		PageID:  string(pageID),
		SpaceID: string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// CreateBatchは複数の添付ファイル参照を一括作成する (スペースIDでスコープ)
func (r *PageAttachmentReferenceRepository) CreateBatch(ctx context.Context, pageID model.PageID, spaceID model.SpaceID, attachmentIDs []model.AttachmentID) ([]*model.PageAttachmentReference, error) {
	now := time.Now()
	refs := make([]*model.PageAttachmentReference, 0, len(attachmentIDs))

	for _, attachmentID := range attachmentIDs {
		row, err := r.q.CreatePageAttachmentReference(ctx, query.CreatePageAttachmentReferenceParams{
			AttachmentID: string(attachmentID),
			PageID:       string(pageID),
			CreatedAt:    now,
			UpdatedAt:    now,
			SpaceID:      string(spaceID),
		})
		if err != nil {
			return nil, err
		}
		refs = append(refs, r.toModel(row))
	}

	return refs, nil
}

// DeleteByPageAndAttachmentIDsはページIDと添付ファイルIDリストに該当する参照を削除する (スペースIDでスコープ)
func (r *PageAttachmentReferenceRepository) DeleteByPageAndAttachmentIDs(ctx context.Context, pageID model.PageID, spaceID model.SpaceID, attachmentIDs []model.AttachmentID) error {
	strIDs := make([]string, len(attachmentIDs))
	for i, id := range attachmentIDs {
		strIDs[i] = string(id)
	}
	return r.q.DeletePageAttachmentReferencesByPageAndAttachmentIDs(ctx, query.DeletePageAttachmentReferencesByPageAndAttachmentIDsParams{
		PageID:  string(pageID),
		SpaceID: string(spaceID),
		Column3: strIDs,
	})
}

// toModelはquery.PageAttachmentReferenceをmodel.PageAttachmentReferenceに変換する
func (r *PageAttachmentReferenceRepository) toModel(row query.PageAttachmentReference) *model.PageAttachmentReference {
	return &model.PageAttachmentReference{
		ID:           model.PageAttachmentReferenceID(row.ID),
		AttachmentID: model.AttachmentID(row.AttachmentID),
		PageID:       model.PageID(row.PageID),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

// toModelsはquery.PageAttachmentReferenceのスライスをmodel.PageAttachmentReferenceのスライスに変換する
func (r *PageAttachmentReferenceRepository) toModels(rows []query.PageAttachmentReference) []*model.PageAttachmentReference {
	refs := make([]*model.PageAttachmentReference, len(rows))
	for i, row := range rows {
		refs[i] = r.toModel(row)
	}
	return refs
}
