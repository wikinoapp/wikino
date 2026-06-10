package viewmodel

import (
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// DraftPageRevision is the view-model for one entry in the page editor's edit history column.
// It carries the revision ID (used to build the diff fragment URL), the version number
// (oldest = v1), the creation time, and whether the entry is the newest revision (shown with a
// "current" badge).
//
// [Ja] DraftPageRevision はページ編集画面の編集履歴カラムの 1 エントリ用ビューモデル。
// リビジョン ID (差分フラグメント URL の組み立てに使う)・バージョン番号 (最古 = v1)・作成日時・
// 最新リビジョンかどうか (「現在」バッジの表示に使う) を保持する。
type DraftPageRevision struct {
	ID        string
	Version   int64
	CreatedAt time.Time
	IsCurrent bool
}

// NewDraftPageRevisions builds the edit history list from a newest-first revision slice.
// totalCount is the total number of revisions of the draft (not capped by the list limit):
// because revisions are never deleted and the slice is newest-first, the entry at index i is
// version totalCount-i, which keeps version numbers stable even when older revisions fall
// outside the capped list. The newest entry (index 0) is marked as current.
//
// [Ja] NewDraftPageRevisions は新しい順のリビジョンスライスから編集履歴一覧を生成する。
// totalCount は下書きのリビジョン総件数 (一覧の上限ではキャップしない)。リビジョンは削除されず
// スライスは新しい順のため、インデックス i のエントリはバージョン totalCount-i となり、古い
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

// DraftPageRevisionDiff is the view-model for the revision diff modal on the page editor.
// It compares the selected revision against the one immediately preceding it: the title is
// shown as an old → new pair and the body as diff blocks.
//
// [Ja] DraftPageRevisionDiff はページ編集画面のリビジョン差分モーダル用ビューモデル。
// 選択されたリビジョンを直前のリビジョンと比較し、タイトルは新旧のペア、本文は差分ブロックで表す。
type DraftPageRevisionDiff struct {
	CreatedAt      time.Time
	OldTitle       string
	NewTitle       string
	HasTitleChange bool
	BodyBlocks     []DiffBlock
}

// NewDraftPageRevisionDiff builds the diff between the selected revision and its predecessor.
// previous may be nil (the selected revision is the oldest one); the diff is then computed
// against empty strings, rendering the whole content as an addition.
//
// [Ja] NewDraftPageRevisionDiff は選択されたリビジョンと直前リビジョンの差分を生成する。
// previous は nil でもよい (選択されたリビジョンが最古の場合)。その場合は空文字列との比較となり、
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
