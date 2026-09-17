package viewmodel

import (
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// DraftPageRevisionはページ編集画面の編集履歴カラムの1エントリ用ビューモデル。
// リビジョンID (差分フラグメントURLの組み立てに使う)・バージョン番号 (最古 = v1)・作成日時・
// 最新リビジョンかどうか (「現在」バッジの表示に使う) を保持する。
type DraftPageRevision struct {
	ID        string
	Version   int64
	CreatedAt time.Time
	IsCurrent bool
}

// NewDraftPageRevisionsは新しい順のリビジョンスライスから編集履歴一覧を生成する。
// totalCountは下書きのリビジョン総件数 (一覧の上限ではキャップしない)。リビジョンは削除されず
// スライスは新しい順のため、インデックスiのエントリはバージョンtotalCount-iとなり、古い
// リビジョンが上限から溢れてもバージョン番号は安定する。先頭 (最新) のエントリを現在として扱う。
func NewDraftPageRevisions(revisions []*model.DraftPageRevision, totalCount int64) []DraftPageRevision {
	result := make([]DraftPageRevision, len(revisions))
	for i, r := range revisions {
		result[i] = DraftPageRevision{
			ID:        string(r.ID),
			Version:   totalCount - int64(i),
			CreatedAt: r.CreatedAt,
			IsCurrent: i == 0,
		}
	}
	return result
}

// DraftPageRevisionDiffはページ編集画面のリビジョン差分モーダル用ビューモデル。
// 選択されたリビジョンを直前のリビジョンと比較し、タイトルは新旧のペア、本文は差分ブロックで表す。
type DraftPageRevisionDiff struct {
	CreatedAt      time.Time
	OldTitle       string
	NewTitle       string
	HasTitleChange bool
	BodyBlocks     []DiffBlock
}

// NewDraftPageRevisionDiffは選択されたリビジョンと直前リビジョンの差分を生成する。
// previousはnilでもよい (選択されたリビジョンが最古の場合)。その場合は空文字列との比較となり、
// 全文追加として表示される。
func NewDraftPageRevisionDiff(revision, previous *model.DraftPageRevision) DraftPageRevisionDiff {
	var oldTitle, oldBody string
	if previous != nil {
		oldTitle = previous.Title
		oldBody = previous.Body
	}

	return DraftPageRevisionDiff{
		CreatedAt:      revision.CreatedAt,
		OldTitle:       oldTitle,
		NewTitle:       revision.Title,
		HasTitleChange: oldTitle != revision.Title,
		BodyBlocks:     ComputeDiffBlocks(oldBody, revision.Body, 3),
	}
}
