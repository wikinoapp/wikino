package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

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
	idStrings := attachmentIDStrings(ids)
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
// 参照するページがすべて公開トピックにあるかの判定を1 SQLに統合する
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

// FindPubliclyReferencedBlobByIDAndSpaceは、指定したスペースの添付ファイルが、生きている公開トピックの
// ページだけから参照される場合にblob情報を返す。所属スペースを確定した後の公開判定に使い、
// IDが存在しない場合や別のスペースに属する場合、参照が無い場合や非公開の場合はnilを返す。
// 戻り値はID / SpaceID / BlobKey / ContentTypeをpopulateする
func (r *AttachmentRepository) FindPubliclyReferencedBlobByIDAndSpace(ctx context.Context, id model.AttachmentID, spaceID model.SpaceID) (*model.Attachment, error) {
	if !uuidRegex.MatchString(string(id)) {
		return nil, nil
	}
	row, err := r.q.FindPubliclyReferencedAttachmentBlobByIDAndSpace(ctx, query.FindPubliclyReferencedAttachmentBlobByIDAndSpaceParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
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

// FindBlobByIDAndSpaceはIDとスペースIDで添付ファイルを取得する。ファイル名に加えて、
// 種類・バイト数・ストレージキーをpopulateする。公開APIの添付ファイルの取得が、メタデータの
// 応答と署名付きURLの生成に使う。閲覧の判定は行わないため、呼び出し側がPolicyで判定する
func (r *AttachmentRepository) FindBlobByIDAndSpace(ctx context.Context, id model.AttachmentID, spaceID model.SpaceID) (*model.Attachment, error) {
	if !uuidRegex.MatchString(string(id)) {
		return nil, nil
	}
	row, err := r.q.FindAttachmentBlobByIDAndSpace(ctx, query.FindAttachmentBlobByIDAndSpaceParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &model.Attachment{
		ID:          model.AttachmentID(row.ID),
		SpaceID:     model.SpaceID(row.SpaceID),
		Filename:    row.Filename,
		BlobKey:     row.BlobKey,
		ContentType: row.BlobContentType.String,
		ByteSize:    row.BlobByteSize,
	}, nil
}

// FindBlobByIDはIDだけで添付ファイルを取得する。ファイル名・種類・バイト数・ストレージキーを
// populateする。Web画面の `/attachments/{id}` がURLにスペースを含まないため、添付ファイルと
// そのスペースを最初に引くのに使う。閲覧の判定は行わないため、呼び出し側が添付ファイルの
// スペースでPolicyを使って判定する
func (r *AttachmentRepository) FindBlobByID(ctx context.Context, id model.AttachmentID) (*model.Attachment, error) {
	if !uuidRegex.MatchString(string(id)) {
		return nil, nil
	}
	row, err := r.q.FindAttachmentBlobByID(ctx, string(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &model.Attachment{
		ID:          model.AttachmentID(row.ID),
		SpaceID:     model.SpaceID(row.SpaceID),
		Filename:    row.Filename,
		BlobKey:     row.BlobKey,
		ContentType: row.BlobContentType.String,
		ByteSize:    row.BlobByteSize,
	}, nil
}

// FindForProcessingByIDAndSpaceは、IDとスペースIDで、アップロード後の処理に使う添付ファイルを取得する。
// ストレージキー・種類・バイト数・処理の状態・作った時刻をpopulateする。存在しなければ (nil, nil) を返す
func (r *AttachmentRepository) FindForProcessingByIDAndSpace(ctx context.Context, id model.AttachmentID, spaceID model.SpaceID) (*model.Attachment, error) {
	if !uuidRegex.MatchString(string(id)) {
		return nil, nil
	}
	row, err := r.q.FindAttachmentForProcessingByIDAndSpace(ctx, query.FindAttachmentForProcessingByIDAndSpaceParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &model.Attachment{
		ID:               model.AttachmentID(row.ID),
		SpaceID:          model.SpaceID(row.SpaceID),
		BlobKey:          row.BlobKey,
		ContentType:      row.BlobContentType.String,
		ByteSize:         row.BlobByteSize,
		ProcessingStatus: model.AttachmentProcessingStatus(row.ProcessingStatus),
		AttachedAt:       row.AttachedAt,
	}, nil
}

// UpdateProcessingStatusは、添付ファイルの処理の状態を、今の状態がfromのときだけtoに変える。
// 状態を変えたかを返し、ほかの処理が先に状態を変えていたらfalseを返す
func (r *AttachmentRepository) UpdateProcessingStatus(ctx context.Context, id model.AttachmentID, spaceID model.SpaceID, from, to model.AttachmentProcessingStatus, now time.Time) (bool, error) {
	rows, err := r.q.UpdateAttachmentProcessingStatus(ctx, query.UpdateAttachmentProcessingStatusParams{
		ToStatus:   int32(to),
		UpdatedAt:  now,
		ID:         string(id),
		SpaceID:    string(spaceID),
		FromStatus: int32(from),
	})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// UpdateBlobContentは、添付ファイルのblobのバイト数とチェックサム (MD5のBase64) を更新する
func (r *AttachmentRepository) UpdateBlobContent(ctx context.Context, id model.AttachmentID, spaceID model.SpaceID, byteSize int64, checksum string) error {
	return r.q.UpdateAttachmentBlobContent(ctx, query.UpdateAttachmentBlobContentParams{
		ByteSize: byteSize,
		Checksum: sql.NullString{String: checksum, Valid: true},
		ID:       string(id),
		SpaceID:  string(spaceID),
	})
}

// CreateAttachmentInputは、事前署名した添付ファイルの行を作るための入力
type CreateAttachmentInput struct {
	SpaceID model.SpaceID
	// UserIDは事前署名したユーザー。Rails期に作られたblobと同じくmetadataに残す
	UserID model.UserID
	// SpaceMemberIDは事前署名したスペースメンバー
	SpaceMemberID model.SpaceMemberID
	BlobKey       string
	Filename      string
	ContentType   string
	ByteSize      int64
	// Checksumは内容のMD5をBase64にした値
	Checksum string
	// AttachedAtは事前署名した時刻。行の作成と更新の時刻にも使う
	AttachedAt time.Time
}

// attachmentBlobMetadataは、事前署名したblobのmetadataに書くJSON。Rails期に作られたblobと同じキーと順にする
type attachmentBlobMetadata struct {
	SpaceID string `json:"space_id"`
	UserID  string `json:"user_id"`
}

// Createは、事前署名した添付ファイルのactive_storage_blobs・active_storage_attachments・attachmentsの行を、
// Rails期に作られた行と同じ形で作り、処理待ちの状態にする。3つの行をまとめて作るため、呼び出し側がWithTxで
// トランザクションの中のRepositoryにしてから呼ぶ
func (r *AttachmentRepository) Create(ctx context.Context, input CreateAttachmentInput) (model.AttachmentID, error) {
	metadata, err := json.Marshal(attachmentBlobMetadata{SpaceID: string(input.SpaceID), UserID: string(input.UserID)})
	if err != nil {
		return "", fmt.Errorf("blobのmetadataの組み立てに失敗: %w", err)
	}
	// 列はタイムゾーンを持たないため、Rails期に作られた行と同じくUTCの時刻を書く
	now := input.AttachedAt.UTC()

	blobID, err := r.q.CreateAttachmentBlob(ctx, query.CreateAttachmentBlobParams{
		Key:         input.BlobKey,
		Filename:    input.Filename,
		ContentType: sql.NullString{String: input.ContentType, Valid: true},
		Metadata:    sql.NullString{String: string(metadata), Valid: true},
		ServiceName: model.AttachmentBlobServiceName,
		ByteSize:    input.ByteSize,
		Checksum:    sql.NullString{String: input.Checksum, Valid: true},
		CreatedAt:   now,
	})
	if err != nil {
		return "", fmt.Errorf("blobの作成に失敗: %w", err)
	}

	storageAttachmentID, err := r.q.CreateAttachmentStorageAttachment(ctx, query.CreateAttachmentStorageAttachmentParams{
		SpaceID:   string(input.SpaceID),
		BlobID:    blobID,
		CreatedAt: now,
	})
	if err != nil {
		return "", fmt.Errorf("blobの関連付けの作成に失敗: %w", err)
	}

	id, err := r.q.CreateAttachment(ctx, query.CreateAttachmentParams{
		SpaceID:                   string(input.SpaceID),
		ActiveStorageAttachmentID: storageAttachmentID,
		AttachedSpaceMemberID:     string(input.SpaceMemberID),
		AttachedAt:                now,
		ProcessingStatus:          int32(model.AttachmentProcessingStatusPending),
	})
	if err != nil {
		return "", fmt.Errorf("添付ファイルの作成に失敗: %w", err)
	}
	return model.AttachmentID(id), nil
}

// FindBlobsByIDsはIDリストだけで添付ファイルを一括取得する。ファイル名・種類・バイト数・ストレージキーを
// populateする。Web画面の署名付きURLの一括取得が、複数のスペースにまたがりうるIDから添付ファイルと
// そのスペースを1回で引くのに使う。UUIDでないIDと存在しないIDは結果に含めない。閲覧の判定は行わないため、
// 呼び出し側が添付ファイルごとのスペースでPolicyを使って判定する
func (r *AttachmentRepository) FindBlobsByIDs(ctx context.Context, ids []model.AttachmentID) ([]*model.Attachment, error) {
	idStrings := attachmentIDStrings(ids)
	if len(idStrings) == 0 {
		return nil, nil
	}
	rows, err := r.q.FindAttachmentBlobsByIDs(ctx, idStrings)
	if err != nil {
		return nil, err
	}
	attachments := make([]*model.Attachment, len(rows))
	for i, row := range rows {
		attachments[i] = &model.Attachment{
			ID:          model.AttachmentID(row.ID),
			SpaceID:     model.SpaceID(row.SpaceID),
			Filename:    row.Filename,
			BlobKey:     row.BlobKey,
			ContentType: row.BlobContentType.String,
			ByteSize:    row.BlobByteSize,
		}
	}
	return attachments, nil
}

// ListPubliclyReferencedIDsByAttachmentsは、取得済みの添付ファイルのIDと所属スペースの組で公開判定を行い、
// 生きている公開トピックのページだけが参照するIDを返す。入力はID / SpaceIDをpopulateした添付ファイルで、
// UUIDでないIDとスペースの組は除く。参照集合の条件はFindPubliclyReferencedBlobByIDと同じで、
// 複数スペースにまたがる添付ファイルを1回のクエリで判定する
func (r *AttachmentRepository) ListPubliclyReferencedIDsByAttachments(ctx context.Context, attachments []*model.Attachment) ([]model.AttachmentID, error) {
	var ids, spaceIDs []string
	for _, attachment := range attachments {
		if uuidRegex.MatchString(string(attachment.ID)) && uuidRegex.MatchString(string(attachment.SpaceID)) {
			ids = append(ids, string(attachment.ID))
			spaceIDs = append(spaceIDs, string(attachment.SpaceID))
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.q.ListPubliclyReferencedAttachmentIDsByIDAndSpacePairs(ctx, query.ListPubliclyReferencedAttachmentIDsByIDAndSpacePairsParams{
		Ids:      ids,
		SpaceIds: spaceIDs,
	})
	if err != nil {
		return nil, err
	}
	publicIDs := make([]model.AttachmentID, len(rows))
	for i, id := range rows {
		publicIDs[i] = model.AttachmentID(id)
	}
	return publicIDs, nil
}

// attachmentIDStringsは、UUIDの形のIDだけをクエリに渡す文字列にする。UUIDでない値をuuid[]に
// キャストするとクエリ全体が失敗するため、一括取得の前に取り除く
func attachmentIDStrings(ids []model.AttachmentID) []string {
	var idStrings []string
	for _, id := range ids {
		if uuidRegex.MatchString(string(id)) {
			idStrings = append(idStrings, string(id))
		}
	}
	return idStrings
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

// ListObjectKeysByIDsAndSpaceは、指定した添付ファイルの実体のキーを返す。添付ファイルのblobのキーと、
// Rails期に作られたサムネイル (variant) のblobのキーを含む。ほかのスペースの添付ファイルと存在しないIDは
// 含めない。
func (r *AttachmentRepository) ListObjectKeysByIDsAndSpace(ctx context.Context, ids []model.AttachmentID, spaceID model.SpaceID) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return r.q.ListAttachmentObjectKeysByIDsAndSpace(ctx, query.ListAttachmentObjectKeysByIDsAndSpaceParams{
		Ids:     attachmentIDStrings(ids),
		SpaceID: string(spaceID),
	})
}

// DeleteByIDsAndSpaceは、指定した添付ファイルの行を消す。サムネイル (variant) の行・ページからの参照・
// 添付ファイル・Active Storageの行である。2つの文で消すため、呼び出し側がトランザクションの中で呼ぶ
// (WithTxで得たリポジトリを使う)。ほかのスペースの添付ファイルと存在しないIDは消さない。
// 実際に削除したblobのキーを返す。最初に実体を消してから増えたvariantも、後始末ジョブで回収できる。
func (r *AttachmentRepository) DeleteByIDsAndSpace(ctx context.Context, ids []model.AttachmentID, spaceID model.SpaceID) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	idStrings := attachmentIDStrings(ids)
	variantKeys, err := r.q.DeleteAttachmentVariantsByIDsAndSpace(ctx, query.DeleteAttachmentVariantsByIDsAndSpaceParams{
		Ids:     idStrings,
		SpaceID: string(spaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("サムネイルの行の削除に失敗: %w", err)
	}
	keys, err := r.q.DeleteAttachmentsByIDsAndSpace(ctx, query.DeleteAttachmentsByIDsAndSpaceParams{
		Ids:     idStrings,
		SpaceID: string(spaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("添付ファイルの行の削除に失敗: %w", err)
	}
	return append(keys, variantKeys...), nil
}

// firstAttachmentIDは、ListIDsBySpaceAfterに最初のページを読ませるときに渡すID (どのIDよりも小さい値)
const firstAttachmentID model.AttachmentID = "00000000-0000-0000-0000-000000000000"

// ListIDsBySpaceAfterは、スペースの添付ファイルのIDを、IDの昇順にafterIDより後ろからlimit件返す。
// afterIDが空なら最初から読む。スペースの物理削除が、添付ファイルを一定の件数ずつ読むのに使う
func (r *AttachmentRepository) ListIDsBySpaceAfter(ctx context.Context, spaceID model.SpaceID, afterID model.AttachmentID, limit int32) ([]model.AttachmentID, error) {
	if afterID == "" {
		afterID = firstAttachmentID
	}
	rows, err := r.q.ListAttachmentIDsBySpaceAfter(ctx, query.ListAttachmentIDsBySpaceAfterParams{
		SpaceID:  string(spaceID),
		AfterID:  string(afterID),
		RowLimit: limit,
	})
	if err != nil {
		return nil, err
	}
	ids := make([]model.AttachmentID, len(rows))
	for i, row := range rows {
		ids[i] = model.AttachmentID(row)
	}
	return ids, nil
}

// PaginatedAttachmentsは、オフセットページネーションで取得した添付ファイルと、絞り込みに一致する総件数
type PaginatedAttachments struct {
	Attachments []*model.Attachment
	TotalCount  int64
}

// ListBySpacePaginatedは、スペースの添付ファイルをアップロードの新しい順 (同じ時刻はIDの降順) に、
// pageページ目のlimit件を返す。ID・SpaceID・Filename・BlobKey・ContentType・ByteSize・AttachedAtをpopulateする。
// filenameQueryが空白だけでなければ、ファイル名に含むもの (大文字と小文字を区別しない部分一致) に絞る。
// filenameQueryの `%`・`_`・`\` は文字として扱う。件数と一覧は、ページの件数によらず2回のクエリで引く
func (r *AttachmentRepository) ListBySpacePaginated(ctx context.Context, spaceID model.SpaceID, filenameQuery string, page int32, limit int32) (*PaginatedAttachments, error) {
	filenamePattern := ""
	if strings.TrimSpace(filenameQuery) != "" {
		filenamePattern = fmt.Sprintf("%%%s%%", escapeLikePattern(filenameQuery))
	}

	totalCount, err := r.q.CountAttachmentsBySpace(ctx, query.CountAttachmentsBySpaceParams{
		SpaceID:         string(spaceID),
		FilenamePattern: filenamePattern,
	})
	if err != nil {
		return nil, err
	}

	rows, err := r.q.ListAttachmentsBySpacePaginated(ctx, query.ListAttachmentsBySpacePaginatedParams{
		SpaceID:         string(spaceID),
		FilenamePattern: filenamePattern,
		RowLimit:        limit,
		RowOffset:       (page - 1) * limit,
	})
	if err != nil {
		return nil, err
	}

	attachments := make([]*model.Attachment, len(rows))
	for i, row := range rows {
		attachments[i] = &model.Attachment{
			ID:          model.AttachmentID(row.ID),
			SpaceID:     model.SpaceID(row.SpaceID),
			Filename:    row.Filename,
			BlobKey:     row.BlobKey,
			ContentType: row.BlobContentType.String,
			ByteSize:    row.BlobByteSize,
			AttachedAt:  row.AttachedAt,
		}
	}
	return &PaginatedAttachments{Attachments: attachments, TotalCount: totalCount}, nil
}
