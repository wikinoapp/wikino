package seed

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// demoBodies holds the bodies of the demo pages, one file per page.
//
// They are embedded as files rather than written as string literals because
// they are prose that is read and extended as prose: a page is added by writing
// a file, and the file name is what the page ends up being called. The .md.txt
// suffix is what markdownGuideBody explains — the shared Markdown linter scans
// every .md file and offers no way to exclude one.
//
// [Ja] demoBodies はデモページの本文を、1 ページ 1 ファイルで保持する。
//
// 文字列リテラルではなくファイルとして埋め込むのは、これらが地の文として読まれ、
// 地の文として書き足されるため。ページを 1 枚足す作業はファイルを 1 つ書くことで
// あり、そのファイル名がページの呼び名になる。拡張子を .md.txt にしている理由は
// markdownGuideBody が述べているとおりで、共通の Markdown リンタは .md を一律に
// 走査して除外の手段を持たない。
//
//go:embed bodies/demo/*.md.txt
var demoBodies embed.FS

const (
	// demoBodiesDir is where the embedded bodies sit.
	//
	// [Ja] demoBodiesDir は、埋め込んだ本文が置かれている場所。
	demoBodiesDir = "bodies/demo"
	// demoBodyExtension is what a body's file name carries beyond the title of
	// the page it holds.
	//
	// [Ja] demoBodyExtension は、本文のファイル名が、そこに入っているページの
	// タイトルに加えて持っている部分。
	demoBodyExtension = ".md.txt"
)

// demoPageModifiedAtStep is how far apart in time two demo pages stand.
//
// It is not a divisor of a day. A step of 24 hours, or of 12, would put every
// page at the same clock time, and a listing whose fifty rows all say the same
// hour reads as something a machine filled in — which is the one thing the demo
// space is written to avoid. At seven hours the pages spread over roughly a
// fortnight.
//
// [Ja] demoPageModifiedAtStep は、デモページ 2 枚が時刻の上でどれだけ離れるか。
//
// 24 時間の約数にしていない。24 時間や 12 時間にすると、すべてのページが同じ時刻に
// 並び、50 行のいずれもが同じ時を告げる一覧は機械が埋めたものに見える。デモスペース
// が避けたいのはまさにそれである。7 時間なら 50 枚がおよそ 2 週間の幅に散る。
const demoPageModifiedAtStep = 7 * time.Hour

// demoPage is one demo page: the title and body read from a file, together
// with the row the first pass creates for it.
//
// [Ja] demoPage はデモページ 1 件。ファイルから読んだタイトルと本文に、パス 1 が
// そのために作成する行が加わる。
type demoPage struct {
	title string
	body  string
	id    model.PageID
}

// demoStamps hands out the timestamps of the demo pages, counting back from
// the instant it was created. The pages are stamped in the order their file
// names sort in, so the first of them is the most recently written one.
//
// It is a type of its own rather than a use of draftStamps, although the two
// count the same way. One counter of draftStamps serves a whole run, which is
// what keeps the drafts of every phase in one order; these stamps are handed
// out within a single generator, and sharing the type would hide that
// difference behind a shared name.
//
// [Ja] demoStamps はデモページの時刻を、自身が作られた瞬間から遡って渡す。ページ
// はファイル名の並び順に打刻されるため、最初のページが最も新しく書かれたものになる。
//
// 数え方は draftStamps と同じだが、それを使うのではなく独自の型にしている。
// draftStamps のカウンターは 1 つで実行全体を受け持ち、それがすべてのフェーズの
// 下書きを 1 つの並びに保っている。こちらの打刻は 1 つの生成器の中で配り切るもので
// あり、型を共有すると、その前提の違いが共通の名前の裏に隠れてしまう。
type demoStamps struct {
	origin time.Time
	issued int
}

// newDemoStamps returns stamps counting back from origin.
//
// [Ja] newDemoStamps は origin から遡っていく打刻を返す。
func newDemoStamps(origin time.Time) *demoStamps {
	return &demoStamps{origin: origin}
}

// next returns the stamp for the next demo page to be published.
//
// [Ja] next は、次に公開するデモページの打刻を返す。
func (s *demoStamps) next() time.Time {
	at := s.origin.Add(-time.Duration(s.issued) * demoPageModifiedAtStep)
	s.issued++

	return at
}

// loadDemoPages reads the embedded bodies in the order their file names sort
// in, which is the order the pages are created and stamped in.
//
// A page's title is its file name without the extension, normalized to NFC.
// The bodies name one another by title, and those names are the bytes of a
// file, while a title travels through the file system and through go:embed,
// which records what it found when the binary was built. Were the two to differ
// in normalization form, the build would still pass and not one link would
// resolve to the page it names: each would create an unpublished page of its
// own, leaving fifty empty pages behind instead of the links the demo space is
// written for.
//
// [Ja] loadDemoPages は埋め込んだ本文を、ファイル名の並び順で読む。この順が、
// ページを作成し打刻していく順になる。
//
// ページのタイトルは、拡張子を取り除いたファイル名を NFC へ正規化したもの。本文は
// 互いをタイトルで名指ししており、その名前はファイルの中身のバイト列である一方、
// タイトルはファイルシステムと、ビルド時に見つけたものを記録する go:embed を経由
// する。両者の正規化形がずれてもビルドは通り、そしてリンクは 1 つも名指しした
// ページへ解決しない。どのリンクもそれぞれ未公開のページを作り、デモスペースが
// 見せたいリンクの代わりに空のページが 50 枚残ることになる。
func loadDemoPages() ([]demoPage, error) {
	entries, err := fs.ReadDir(demoBodies, demoBodiesDir)
	if err != nil {
		return nil, fmt.Errorf("デモページ本文の一覧の取得に失敗: %w", err)
	}

	pages := make([]demoPage, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()

		body, err := fs.ReadFile(demoBodies, path.Join(demoBodiesDir, name))
		if err != nil {
			return nil, fmt.Errorf("デモページ本文 %s の読み込みに失敗: %w", name, err)
		}

		pages = append(pages, demoPage{
			title: norm.NFC.String(strings.TrimSuffix(name, demoBodyExtension)),
			body:  string(body),
		})
	}

	return pages, nil
}

// generateDemoPages fills the demo topic with the pages the help pages are
// photographed from.
//
// The pages link to one another, and that is what makes them two passes rather
// than one. createPage renders a body before it inserts the page, and rendering
// has the resolver create the pages the body names; a page created that way
// would already hold the title the generator was about to create the page under,
// and the two would collide on the unique index over (topic_id, title). Creating
// every page first leaves the resolver nothing to create.
//
// [Ja] generateDemoPages は、ヘルプページのスクリーンショットを撮るためのページ
// でデモのトピックを埋める。
//
// ページは互いにリンクし合っており、それが投入を 1 パスではなく 2 パスにしている
// 理由になる。createPage は本文をレンダリングしてからページを INSERT し、
// レンダリングは resolver に本文が名指しするページを作らせる。そうして作られた
// ページは、生成器がこれから作ろうとしているタイトルを既に持っており、両者は
// (topic_id, title) の一意インデックスで衝突する。先にすべてのページを作って
// おけば、resolver に作るものは残らない。
func generateDemoPages(
	ctx context.Context,
	dbtx query.DBTX,
	out io.Writer,
	spaces *seededSpaces,
	topics *seededTopics,
) error {
	pages, err := loadDemoPages()
	if err != nil {
		return err
	}

	// Every page is written twice, once by each pass, and the counter follows
	// the writes rather than the pages. Counting the pages would leave it
	// standing still through the whole of the slower pass.
	//
	// [Ja] ページはパスごとに 1 回ずつ、計 2 回書かれる。カウンタはページではなく
	// 書き込みを数える。ページを数えると、遅いほうのパスの間じゅうカウンタが
	// 止まったままになる。
	bar := newProgress(out, "デモスペースのページ", len(pages)*2)
	defer bar.finish()

	// Every demo page belongs to roleOwner: no other role has joined the space,
	// and the screenshots are taken from that account.
	//
	// [Ja] デモページはすべて roleOwner のものになる。他の役割はこのスペースに
	// 参加しておらず、スクリーンショットもそのアカウントから撮るため。
	owner, err := spaces.demo.requireMember(roleOwner)
	if err != nil {
		return err
	}

	// The writer is used for what it already knows how to do — rendering a body
	// against the space, and the repositories to write through — while the two
	// passes are held here. createPage is left as it is: a page that links to a
	// page created later is a demo page and nothing else, and teaching the
	// generators that do not have that problem to step around it would cost
	// every one of them a branch.
	//
	// [Ja] writer は既にできること、つまりスペースに対する本文のレンダリングと、
	// 書き込みに使う Repository のために用い、2 パスの手順はここに置く。createPage
	// には手を入れない。あとから作られるページへリンクするページはデモページだけで
	// あり、その問題を持たない生成器にまで迂回路を教えると、そのすべてに分岐が
	// 増えることになる。
	writer := newPageWriter(dbtx, spaces.demo)
	if err := writer.ensureTopicInSpace(topics.demoMemo); err != nil {
		return err
	}

	if err := createDemoPages(ctx, writer, topics.demoMemo, pages, bar); err != nil {
		return err
	}

	return publishDemoPages(ctx, writer, topics.demoMemo, owner, pages, bar)
}

// createDemoPages creates every demo page as the unpublished page a wiki link
// leaves behind, and records the id each one was given. This is the first of
// the two passes, and the page numbers are handed out here, in the order the
// bodies were read in.
//
// [Ja] createDemoPages は、すべてのデモページを、Wiki リンクが残す未公開の
// ページとして作成し、それぞれに与えられた id を控える。2 つのパスのうちの
// 1 つ目にあたり、ページ番号は本文を読んだ順にここで振られる。
func createDemoPages(
	ctx context.Context,
	writer *pageWriter,
	topic *seededTopic,
	pages []demoPage,
	bar *progress,
) error {
	for i := range pages {
		number, err := writer.pageRepo.NextPageNumber(ctx, writer.space.id)
		if err != nil {
			return fmt.Errorf("次のページ番号の取得に失敗: %w", err)
		}

		page, err := writer.pageRepo.CreateLinkedPage(ctx, repository.CreateLinkedPageInput{
			SpaceID: writer.space.id,
			TopicID: topic.id,
			Number:  number,
			Title:   pages[i].title,
		})
		if err != nil {
			return fmt.Errorf("デモページ %s の作成に失敗: %w", pages[i].title, err)
		}

		pages[i].id = page.ID
		bar.advance()
	}

	return nil
}

// publishDemoPages renders each body and publishes the page it belongs to,
// leaving the revision and the editor entry that publishing from the screen
// leaves behind. This is the second of the two passes, and by the time it runs
// every title the bodies name already exists.
//
// [Ja] publishDemoPages は各本文をレンダリングしてページを公開し、画面から公開した
// ときに残るリビジョンと編集者エントリを残す。2 つのパスのうちの 2 つ目にあたり、
// これが走る時点では本文が名指しするタイトルはすべて存在している。
func publishDemoPages(
	ctx context.Context,
	writer *pageWriter,
	topic *seededTopic,
	author *seededSpaceMember,
	pages []demoPage,
	bar *progress,
) error {
	stamps := newDemoStamps(time.Now())

	for _, page := range pages {
		at := stamps.next()

		bodyHTML, linkedPageIDs, err := writer.render(ctx, createPageInput{
			topic:  topic,
			author: author,
			title:  page.title,
			body:   page.body,
		})
		if err != nil {
			return err
		}

		if _, err := writer.pageRepo.Update(ctx, repository.UpdatePageInput{
			ID:            page.id,
			SpaceID:       writer.space.id,
			TopicID:       topic.id,
			Title:         &page.title,
			Body:          page.body,
			BodyHTML:      bodyHTML,
			LinkedPageIDs: linkedPageIDs,
			ModifiedAt:    at,
			PublishedAt:   &at,
		}); err != nil {
			return fmt.Errorf("デモページ %s の公開に失敗: %w", page.title, err)
		}

		// created_at is stamped here because Update does not touch it, and
		// rightly so: a page it updates was created when it was created. A demo
		// page is written into a past that its own row has to agree with, and a
		// row that says it was created after it was last modified is a state
		// nothing could have produced.
		//
		// [Ja] created_at をここで打つのは Update がその列に触れないためで、それは
		// 正しい。Update が更新するページは、作成されたときに作成されている。一方で
		// デモページは過去へ向けて書かれるものであり、その行自身がその過去と辻褄を
		// 合わせる必要がある。最終更新より後に作成されたと述べる行は、どうやっても
		// 起こりえない状態である。
		if _, err := writer.dbtx.ExecContext(
			ctx,
			`UPDATE pages SET created_at = $3 WHERE id = $1 AND space_id = $2`,
			string(page.id), string(writer.space.id), at,
		); err != nil {
			return fmt.Errorf("デモページ %s の作成日時の打刻に失敗: %w", page.title, err)
		}

		if _, err := writer.pageRevisionRepo.Create(ctx, repository.CreatePageRevisionInput{
			SpaceID:       writer.space.id,
			SpaceMemberID: author.id,
			PageID:        page.id,
			Title:         page.title,
			Body:          page.body,
			BodyHTML:      bodyHTML,
		}); err != nil {
			return fmt.Errorf("デモページ %s のリビジョンの作成に失敗: %w", page.title, err)
		}

		if _, err := writer.pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
			SpaceID:            writer.space.id,
			PageID:             page.id,
			SpaceMemberID:      author.id,
			LastPageModifiedAt: at,
		}); err != nil {
			return fmt.Errorf("デモページ %s の編集者の登録に失敗: %w", page.title, err)
		}

		bar.advance()
	}

	return nil
}
