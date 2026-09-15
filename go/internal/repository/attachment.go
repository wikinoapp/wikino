package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// uuidRegexはUUID形式を検証する正規表現
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// AttachmentRepositoryは添付ファイルリポジトリ
type AttachmentRepository struct {
	q *query.Queries
}

// NewAttachmentRepositoryはAttachmentRepositoryを生成する
func NewAttachmentRepository(q *query.Queries) *AttachmentRepository {
	return &AttachmentRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *AttachmentRepository) WithTx(tx *sql.Tx) *AttachmentRepository {
	return &AttachmentRepository{q: r.q.WithTx(tx)}
}

// ExistsByIDAndSpaceはIDとスペースIDで添付ファイルの存在を確認する
func (r *AttachmentRepository) ExistsByIDAndSpace(ctx context.Context, id model.AttachmentID, spaceID model.SpaceID) (bool, error) {
	if !uuidRegex.MatchString(string(id)) {
		return false, nil
	}
	return r.q.ExistsAttachmentByIDAndSpace(ctx, query.ExistsAttachmentByIDAndSpaceParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
}

// FindByIDsAndSpaceはIDリストとスペースIDで添付ファイルを一括取得する (バッチレンダリング用)
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

// FindPubliclyReferencedBlobByIDは公開og:image配信用: 「生きている公開トピックの
// ページからのみ参照されている」場合に限りblob情報を返す。
//
// Rails版AttachmentRecord#all_referencing_pages_public? と等価な判定を1 SQLに統合する
// ことで、呼び出し側でvisibility検証を忘れる構造的事故を排除している。判定スコープからは
// 論理削除済みのページ・トピック (`discarded_at IS NOT NULL`) に加えて、ゴミ箱に入った
// ページ (`trashed_at IS NOT NULL`) も除外する。ゴミ箱に入ったページのog:imageをSNSの
// リンクプレビューに残さないため。レスポンスはキャッシュされる前提のため、ゴミ箱を開ける
// メンバーであってもメンバー判定は行わない (ページ表示画面の移行計画を参照)。
//
// 戻り値のAttachmentはBlobKey / ContentTypeをpopulateするが、Filenameは空のまま
// (このメソッドでは取得していない)。og:image配信用途ではFilenameを使わないため問題ない。
// 参照集合はattachmentと同じspaceに内部で限定する。呼び出し元からspaceスコープを
// 受け取る必要はなく、この判定を通過した画像はURL文字列を知っている誰でも (ゲスト含む)
// 閲覧可能であることを前提にする。
func (r *AttachmentRepository) FindPubliclyReferencedBlobByID(ctx context.Context, id model.AttachmentID) (*model.Attachment, error) {
	if !uuidRegex.MatchString(string(id)) {
		return nil, nil
	}
	row, err := r.q.FindPubliclyReferencedAttachmentBlobByID(ctx, string(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &model.Attachment{
		ID:          model.AttachmentID(row.ID),
		SpaceID:     model.SpaceID(row.SpaceID),
		BlobKey:     row.BlobKey,
		ContentType: row.BlobContentType.String,
	}, nil
}

// FindByIDAndSpaceはIDとスペースIDで添付ファイルを取得する (ファイル名を含む)
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

// toModelはクエリ結果をモデルに変換する
func (r *AttachmentRepository) toModel(row query.FindAttachmentByIDAndSpaceRow) *model.Attachment {
	return &model.Attachment{
		ID:       model.AttachmentID(row.ID),
		SpaceID:  model.SpaceID(row.SpaceID),
		Filename: row.Filename,
	}
}

// PageAttachmentは添付ファイルと、それを参照しているページの組。エクスポートにはこの組が
// 要る。添付ファイルの複製は参照元のページが属するトピックのディレクトリへ置かれ、1つの添付
// ファイルが複数のトピックから参照されうるためである。
type PageAttachment struct {
	PageID     model.PageID
	Attachment *model.Attachment
}

// ListByPageIDsAndSpaceは指定したページが参照している添付ファイルを、参照元のページとの組
// で返す。populateされるのはFilenameとBlobKeyで、エクスポートは前者から複製の名前を決め、
// 後者でオブジェクトを取得する。
func (r *AttachmentRepository) ListByPageIDsAndSpace(ctx context.Context, pageIDs []model.PageID, spaceID model.SpaceID) ([]*PageAttachment, error) {
	var idStrings []string
	for _, id := range pageIDs {
		if uuidRegex.MatchString(string(id)) {
			idStrings = append(idStrings, string(id))
		}
	}
	if len(idStrings) == 0 {
		return nil, nil
	}

	rows, err := r.q.ListAttachmentsByPageIDsAndSpace(ctx, query.ListAttachmentsByPageIDsAndSpaceParams{
		PageIds: idStrings,
		SpaceID: string(spaceID),
	})
	if err != nil {
		return nil, err
	}

	pageAttachments := make([]*PageAttachment, len(rows))
	for i, row := range rows {
		pageAttachments[i] = &PageAttachment{
			PageID: model.PageID(row.PageID),
			Attachment: &model.Attachment{
				ID:       model.AttachmentID(row.ID),
				SpaceID:  model.SpaceID(row.SpaceID),
				Filename: row.Filename,
				BlobKey:  row.BlobKey,
			},
		}
	}
	return pageAttachments, nil
}
