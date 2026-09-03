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

// FindPubliclyReferencedBlobByID returns the blob info for public og:image delivery only when
// the attachment is referenced exclusively by live pages in public topics.
//
// Folding the Rails AttachmentRecord#all_referencing_pages_public? check into a single SQL
// statement removes the structural risk of a caller forgetting the visibility check. The
// reference set excludes discarded pages and topics (`discarded_at IS NOT NULL`) as well as
// pages moved to the trash (`trashed_at IS NOT NULL`); a trashed page must not keep its
// og:image alive in social link previews. Membership is deliberately not checked, even for
// members who may open the trash, because the response is cached (see the page-show migration
// plan).
//
// The returned Attachment populates BlobKey / ContentType but leaves Filename empty (this
// method does not fetch it), which is fine because og:image delivery does not use Filename.
// References are internally constrained to the attachment's space. No caller-provided space
// scope is needed because the endpoint assumes anyone who knows the URL (guests included) may
// view the image once this check succeeds.
//
// [Ja] FindPubliclyReferencedBlobByID は公開 og:image 配信用: 「生きている公開トピックの
// ページからのみ参照されている」場合に限り blob 情報を返す。
//
// Rails 版 AttachmentRecord#all_referencing_pages_public? と等価な判定を 1 SQL に統合する
// ことで、呼び出し側で visibility 検証を忘れる構造的事故を排除している。判定スコープからは
// 論理削除済みのページ・トピック (`discarded_at IS NOT NULL`) に加えて、ゴミ箱に入った
// ページ (`trashed_at IS NOT NULL`) も除外する。ゴミ箱に入ったページの og:image を SNS の
// リンクプレビューに残さないため。レスポンスはキャッシュされる前提のため、ゴミ箱を開ける
// メンバーであってもメンバー判定は行わない (ページ表示画面の移行計画を参照)。
//
// 戻り値の Attachment は BlobKey / ContentType を populate するが、Filename は空のまま
// (このメソッドでは取得していない)。og:image 配信用途では Filename を使わないため問題ない。
// 参照集合は attachment と同じ space に内部で限定する。呼び出し元から space スコープを
// 受け取る必要はなく、この判定を通過した画像は URL 文字列を知っている誰でも (ゲスト含む)
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
