package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// uuidRegex はUUID形式を検証する正規表現
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

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
func (r *AttachmentRepository) ExistsByIDAndSpace(ctx context.Context, id model.AttachmentID, spaceID model.SpaceID) (bool, error) {
	if !uuidRegex.MatchString(string(id)) {
		return false, nil
	}
	return r.q.ExistsAttachmentByIDAndSpace(ctx, query.ExistsAttachmentByIDAndSpaceParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
}

// FindByIDsAndSpace はIDリストとスペースIDで添付ファイルを一括取得する（バッチレンダリング用）
func (r *AttachmentRepository) FindByIDsAndSpace(ctx context.Context, ids []model.AttachmentID, spaceID model.SpaceID) ([]*model.Attachment, error) {
	var idStrings []string
	for _, id := range ids {
		if uuidRegex.MatchString(string(id)) {
			idStrings = append(idStrings, string(id))
		}
	}
	if len(idStrings) == 0 {
		return nil, nil
	}
	rows, err := r.q.FindAttachmentsByIDsAndSpace(ctx, query.FindAttachmentsByIDsAndSpaceParams{
		Column1: idStrings,
		SpaceID: string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	attachments := make([]*model.Attachment, len(rows))
	for i, row := range rows {
		attachments[i] = r.toModel(query.FindAttachmentByIDAndSpaceRow(row))
	}
	return attachments, nil
}

// FindByIDAndSpace はIDとスペースIDで添付ファイルを取得する（ファイル名を含む）
func (r *AttachmentRepository) FindByIDAndSpace(ctx context.Context, id model.AttachmentID, spaceID model.SpaceID) (*model.Attachment, error) {
	if !uuidRegex.MatchString(string(id)) {
		return nil, nil
	}
	row, err := r.q.FindAttachmentByIDAndSpace(ctx, query.FindAttachmentByIDAndSpaceParams{
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

// toModel はクエリ結果をモデルに変換する
func (r *AttachmentRepository) toModel(row query.FindAttachmentByIDAndSpaceRow) *model.Attachment {
	return &model.Attachment{
		ID:       model.AttachmentID(row.ID),
		SpaceID:  model.SpaceID(row.SpaceID),
		Filename: row.Filename,
	}
}
