package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// PageAttachmentReferenceRepository はページ添付ファイル参照リポジトリ
type PageAttachmentReferenceRepository struct {
	q *query.Queries
}

// NewPageAttachmentReferenceRepository は PageAttachmentReferenceRepository を生成する
func NewPageAttachmentReferenceRepository(q *query.Queries) *PageAttachmentReferenceRepository {
	return &PageAttachmentReferenceRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *PageAttachmentReferenceRepository) WithTx(tx *sql.Tx) *PageAttachmentReferenceRepository {
	return &PageAttachmentReferenceRepository{q: r.q.WithTx(tx)}
}

// ListByPageID はページIDに紐づく添付ファイル参照を取得する（スペースIDでスコープ）
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

// CreateBatch は複数の添付ファイル参照を一括作成する
func (r *PageAttachmentReferenceRepository) CreateBatch(ctx context.Context, pageID model.PageID, attachmentIDs []string) ([]*model.PageAttachmentReference, error) {
	now := time.Now()
	refs := make([]*model.PageAttachmentReference, 0, len(attachmentIDs))

	for _, attachmentID := range attachmentIDs {
		row, err := r.q.CreatePageAttachmentReference(ctx, query.CreatePageAttachmentReferenceParams{
			AttachmentID: attachmentID,
			PageID:       string(pageID),
			CreatedAt:    now,
			UpdatedAt:    now,
		})
		if err != nil {
			return nil, err
		}
		refs = append(refs, r.toModel(row))
	}

	return refs, nil
}

// DeleteByPageAndAttachmentIDs はページIDと添付ファイルIDリストに該当する参照を削除する
func (r *PageAttachmentReferenceRepository) DeleteByPageAndAttachmentIDs(ctx context.Context, pageID model.PageID, attachmentIDs []string) error {
	return r.q.DeletePageAttachmentReferencesByPageAndAttachmentIDs(ctx, query.DeletePageAttachmentReferencesByPageAndAttachmentIDsParams{
		PageID:  string(pageID),
		Column2: attachmentIDs,
	})
}

// toModel は query.PageAttachmentReference を model.PageAttachmentReference に変換する
func (r *PageAttachmentReferenceRepository) toModel(row query.PageAttachmentReference) *model.PageAttachmentReference {
	return &model.PageAttachmentReference{
		ID:           row.ID,
		AttachmentID: row.AttachmentID,
		PageID:       model.PageID(row.PageID),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

// toModels は query.PageAttachmentReference のスライスを model.PageAttachmentReference のスライスに変換する
func (r *PageAttachmentReferenceRepository) toModels(rows []query.PageAttachmentReference) []*model.PageAttachmentReference {
	refs := make([]*model.PageAttachmentReference, len(rows))
	for i, row := range rows {
		refs[i] = r.toModel(row)
	}
	return refs
}
