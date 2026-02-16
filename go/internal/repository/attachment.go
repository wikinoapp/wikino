package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// AttachmentRepository は添付ファイルリポジトリ
type AttachmentRepository struct {
	q *query.Queries
}

// NewAttachmentRepository は AttachmentRepository を生成する
func NewAttachmentRepository(q *query.Queries) *AttachmentRepository {
	return &AttachmentRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *AttachmentRepository) WithTx(tx *sql.Tx) *AttachmentRepository {
	return &AttachmentRepository{q: r.q.WithTx(tx)}
}

// ExistsByIDAndSpace はIDとスペースIDで添付ファイルの存在を確認する
func (r *AttachmentRepository) ExistsByIDAndSpace(ctx context.Context, id string, spaceID model.SpaceID) (bool, error) {
	return r.q.ExistsAttachmentByIDAndSpace(ctx, query.ExistsAttachmentByIDAndSpaceParams{
		ID:      id,
		SpaceID: string(spaceID),
	})
}

// FindByIDAndSpace はIDとスペースIDで添付ファイルを取得する（ファイル名を含む）
func (r *AttachmentRepository) FindByIDAndSpace(ctx context.Context, id string, spaceID model.SpaceID) (*model.Attachment, error) {
	row, err := r.q.FindAttachmentByIDAndSpace(ctx, query.FindAttachmentByIDAndSpaceParams{
		ID:      id,
		SpaceID: string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &model.Attachment{
		ID:       row.ID,
		SpaceID:  model.SpaceID(row.SpaceID),
		Filename: row.Filename,
	}, nil
}
