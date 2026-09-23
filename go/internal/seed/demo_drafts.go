package seed

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path"
	"time"

	"golang.org/x/text/unicode/norm"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// demoDraftBodiesはデモスペースの下書きの本文を、1つの保存につき1ファイルで
// 保持する。ディレクトリ名が下書きの対象とするデモページのタイトルで、その中の
// ファイル名が何回目の保存かを表す。
//
// デモページの本文と同じくファイルとして埋め込むのは、保存ごとの本文が地の文として
// 読まれ、編集履歴の差分として並べて見られるため。
//
//go:embed bodies/demo-drafts/*/*.md.txt
var demoDraftBodies embed.FS

// demoDraftBodiesDirは、埋め込んだ下書きの本文が置かれている場所。
const demoDraftBodiesDir = "bodies/demo-drafts"

// demoFeaturedPageTitleは、トップページのスクリーンショットに写すデモページ。
//
// このページは編集画面に並ぶ一覧のどれもが埋まるように選んでいる。下書きは
// 何度も保存されて編集履歴を持ち、リンク先の1つが関連リンクを持ち、ページ自身も
// バックリンクを持つ。
//
// 一方で、下書きの本文がリンクするページは2件に絞り、関連リンクを持つのは
// そのうち1件だけにしている。関連リンクはリンク先ごとに1段ずつ並ぶため、リンク先が
// それぞれ関連リンクを持つと画面が縦に伸び、スクリーンショットに収まらなくなる。
// もう1件のリンク先である集中できる読書環境は、このページ以外にはもう一方の
// リンク先からしかリンクされておらず、関連リンクの一覧はリンク先そのものを除くため、
// 関連リンクを持たない。
// ピン留めしておくのは、スペースやトピックの画面を開いたときに、撮る画面へ
// 真っ先に辿り着けるようにするため。
const demoFeaturedPageTitle = "月と栞"

// demoDraftSaveAgesは、下書きの各保存が最後の保存からどれだけ前に行われたかを、
// 新しい保存から順に持つ。
//
// 間隔を揃えずに広げていくのは、編集履歴の時刻が「数分前」「数時間前」「数日前」、
// そして日付の表示のすべてを見せるようにするため。手で書き進めた下書きの履歴は
// そのように散らばる。最も古い保存でもデモページが公開されてから後になるよう、
// デモページの打刻の間隔に比べて小さく抑えている。
var demoDraftSaveAges = []time.Duration{
	0,
	6 * time.Minute,
	3 * time.Hour,
	26 * time.Hour,
	3 * 24 * time.Hour,
	5 * 24 * time.Hour,
}

// demoDraftは、デモページ1件に対して書かれた下書き。
type demoDraft struct {
	pageTitle string
	// bodiesは各保存が書いた本文を、古い保存から順に持つ。
	bodies []string
}

// loadDemoDraftsは埋め込んだ下書きの本文を読む。スクリーンショットに写す
// ページの下書きを先頭に置き、残りはディレクトリ名の並び順にする。この順が
// 下書きを作成し打刻していく順になるため、先頭の下書きが最も新しく編集された
// ものとして下書きの一覧の先頭に並ぶ。
//
// タイトルをNFCへ正規化する理由はloadDemoPagesが述べているとおり。
func loadDemoDrafts() ([]demoDraft, error) {
	dirs, err := fs.ReadDir(demoDraftBodies, demoDraftBodiesDir)
	if err != nil {
		return nil, fmt.Errorf("デモ下書き本文の一覧の取得に失敗: %w", err)
	}

	drafts := make([]demoDraft, 0, len(dirs))
	for _, dir := range dirs {
		dirPath := path.Join(demoDraftBodiesDir, dir.Name())

		entries, err := fs.ReadDir(demoDraftBodies, dirPath)
		if err != nil {
			return nil, fmt.Errorf("デモ下書き %sの本文の一覧の取得に失敗: %w", dir.Name(), err)
		}

		draft := demoDraft{pageTitle: norm.NFC.String(dir.Name())}
		for i, entry := range entries {
			// 保存の順はファイル名の並び順で決まる。1から順に振られていることを
			// ここで確かめるのは、欠番や10回目以降の保存がその並び順を崩すため。
			if want := fmt.Sprintf("%d%s", i+1, demoBodyExtension); entry.Name() != want {
				return nil, fmt.Errorf("デモ下書き %sの%d回目の保存の本文は %sであることを期待したが %sだった", dir.Name(), i+1, want, entry.Name())
			}

			body, err := fs.ReadFile(demoDraftBodies, path.Join(dirPath, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("デモ下書き %sの本文 %sの読み込みに失敗: %w", dir.Name(), entry.Name(), err)
			}
			draft.bodies = append(draft.bodies, string(body))
		}

		if len(draft.bodies) > len(demoDraftSaveAges) {
			return nil, fmt.Errorf("デモ下書き %sの保存は %d回で、打刻できる %d回を超えている", draft.pageTitle, len(draft.bodies), len(demoDraftSaveAges))
		}

		if draft.pageTitle == demoFeaturedPageTitle {
			drafts = append([]demoDraft{draft}, drafts...)
		} else {
			drafts = append(drafts, draft)
		}
	}

	if len(drafts) == 0 || drafts[0].pageTitle != demoFeaturedPageTitle {
		return nil, fmt.Errorf("デモ下書きに %sの下書きが無い", demoFeaturedPageTitle)
	}

	return drafts, nil
}

// generateDemoDraftsは、デモスペースの下書きを作成し、スクリーンショットに写す
// ページをピン留めする。これにより、そのページの編集画面の下書き一覧と編集履歴の
// 2つのカラムが埋まる。
//
// 下書きの最終更新は、他の下書きと同じくstampsから受け取る。デモの下書きは
// 他のスペースの下書きより後に作られるため、スペースを跨いで下書きを並べる
// ホーム画面では、それらより古いものとして並ぶ。
func generateDemoDrafts(
	ctx context.Context,
	dbtx query.DBTX,
	out io.Writer,
	spaces *seededSpaces,
	topics *seededTopics,
	stamps *draftStamps,
) error {
	drafts, err := loadDemoDrafts()
	if err != nil {
		return err
	}

	bar := newProgress(out, "デモスペースの下書き", len(drafts)+1)
	defer bar.finish()

	owner, err := spaces.demo.requireMember(roleOwner)
	if err != nil {
		return err
	}

	writer := newDraftWriter(dbtx, spaces.demo)
	topic := topics.demoMemo

	var featured *seededPage
	for _, draft := range drafts {
		page, err := findDemoPage(ctx, writer.pages, topic, draft.pageTitle)
		if err != nil {
			return err
		}
		if draft.pageTitle == demoFeaturedPageTitle {
			featured = page
		}

		modifiedAt := stamps.next()
		last := len(draft.bodies) - 1
		savedAt := make([]time.Time, len(draft.bodies))
		for i := range savedAt {
			savedAt[i] = modifiedAt.Add(-demoDraftSaveAges[last-i])
		}

		title := page.title
		if err := writer.createDraft(ctx, createDraftInput{
			topic:      topic,
			member:     owner,
			page:       page,
			title:      &title,
			bodies:     draft.bodies,
			modifiedAt: modifiedAt,
			savedAt:    savedAt,
		}); err != nil {
			return err
		}
		bar.advance()
	}

	if err := writer.pages.pinPage(ctx, featured, time.Now()); err != nil {
		return err
	}
	bar.advance()

	return nil
}

// findDemoPageは、デモのトピックからタイトルでページを引く。
func findDemoPage(ctx context.Context, writer *pageWriter, topic *seededTopic, title string) (*seededPage, error) {
	page, err := writer.pageRepo.FindByTopicAndTitle(ctx, topic.id, title, writer.space.id)
	if err != nil {
		return nil, fmt.Errorf("デモページ %sの取得に失敗: %w", title, err)
	}
	if page == nil || page.PublishedAt == nil {
		return nil, fmt.Errorf("デモ下書きの対象とする公開済みのデモページ %sが無い", title)
	}

	return &seededPage{id: page.ID, number: page.Number, title: title}, nil
}
