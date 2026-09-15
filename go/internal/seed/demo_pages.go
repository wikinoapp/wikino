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

// demoBodiesはデモページの本文を、1ページ1ファイルで保持する。
//
// 文字列リテラルではなくファイルとして埋め込むのは、これらが地の文として読まれ、
// 地の文として書き足されるため。ページを1枚足す作業はファイルを1つ書くことで
// あり、そのファイル名がページの呼び名になる。拡張子を .md.txtにしている理由は
// markdownGuideBodyが述べているとおりで、共通のMarkdownリンタは .mdを一律に
// 走査して除外の手段を持たない。
//
//go:embed bodies/demo/*.md.txt
var demoBodies embed.FS

const (
	// demoBodiesDirは、埋め込んだ本文が置かれている場所。
	demoBodiesDir = "bodies/demo"
	// demoBodyExtensionは、本文のファイル名が、そこに入っているページの
	// タイトルに加えて持っている部分。
	demoBodyExtension = ".md.txt"
)

// demoPageModifiedAtStepは、デモページ2枚が時刻の上でどれだけ離れるか。
//
// 24時間の約数にしていない。24時間や12時間にすると、すべてのページが同じ時刻に
// 並び、50行のいずれもが同じ時を告げる一覧は機械が埋めたものに見える。デモスペース
// が避けたいのはまさにそれである。7時間なら50枚がおよそ2週間の幅に散る。
const demoPageModifiedAtStep = 7 * time.Hour

// demoPageはデモページ1件。ファイルから読んだタイトルと本文に、パス1が
// そのために作成する行が加わる。
type demoPage struct {
	title string
	body  string
	id    model.PageID
}

// demoStampsはデモページの時刻を、自身が作られた瞬間から遡って渡す。ページ
// はファイル名の並び順に打刻されるため、最初のページが最も新しく書かれたものになる。
//
// 数え方はdraftStampsと同じだが、それを使うのではなく独自の型にしている。
// draftStampsのカウンターは1つで実行全体を受け持ち、それがすべてのフェーズの
// 下書きを1つの並びに保っている。こちらの打刻は1つの生成器の中で配り切るもので
// あり、型を共有すると、その前提の違いが共通の名前の裏に隠れてしまう。
type demoStamps struct {
	origin time.Time
	issued int
}

// newDemoStampsはoriginから遡っていく打刻を返す。
func newDemoStamps(origin time.Time) *demoStamps {
	return &demoStamps{origin: origin}
}

// nextは、次に公開するデモページの打刻を返す。
func (s *demoStamps) next() time.Time {
	at := s.origin.Add(-time.Duration(s.issued) * demoPageModifiedAtStep)
	s.issued++

	return at
}

// loadDemoPagesは埋め込んだ本文を、ファイル名の並び順で読む。この順が、
// ページを作成し打刻していく順になる。
//
// ページのタイトルは、拡張子を取り除いたファイル名をNFCへ正規化したもの。本文は
// 互いをタイトルで名指ししており、その名前はファイルの中身のバイト列である一方、
// タイトルはファイルシステムと、ビルド時に見つけたものを記録するgo:embedを経由
// する。両者の正規化形がずれてもビルドは通り、そしてリンクは1つも名指しした
// ページへ解決しない。どのリンクもそれぞれ未公開のページを作り、デモスペースが
// 見せたいリンクの代わりに空のページが50枚残ることになる。
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
			return nil, fmt.Errorf("デモページ本文 %sの読み込みに失敗: %w", name, err)
		}

		pages = append(pages, demoPage{
			title: norm.NFC.String(strings.TrimSuffix(name, demoBodyExtension)),
			body:  string(body),
		})
	}

	return pages, nil
}

// generateDemoPagesは、ヘルプページのスクリーンショットを撮るためのページ
// でデモのトピックを埋める。
//
// ページは互いにリンクし合っており、それが投入を1パスではなく2パスにしている
// 理由になる。createPageは本文をレンダリングしてからページをINSERTし、
// レンダリングはresolverに本文が名指しするページを作らせる。そうして作られた
// ページは、生成器がこれから作ろうとしているタイトルを既に持っており、両者は
// (topic_id, title) の一意インデックスで衝突する。先にすべてのページを作って
// おけば、resolverに作るものは残らない。
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

	// ページはパスごとに1回ずつ、計2回書かれる。カウンタはページではなく
	// 書き込みを数える。ページを数えると、遅いほうのパスの間じゅうカウンタが
	// 止まったままになる。
	bar := newProgress(out, "デモスペースのページ", len(pages)*2)
	defer bar.finish()

	// デモページはすべてroleOwnerのものになる。他の役割はこのスペースに
	// 参加しておらず、スクリーンショットもそのアカウントから撮るため。
	owner, err := spaces.demo.requireMember(roleOwner)
	if err != nil {
		return err
	}

	// writerは既にできること、つまりスペースに対する本文のレンダリングと、
	// 書き込みに使うRepositoryのために用い、2パスの手順はここに置く。createPage
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

// createDemoPagesは、すべてのデモページを、Wikiリンクが残す未公開の
// ページとして作成し、それぞれに与えられたidを控える。2つのパスのうちの
// 1つ目にあたり、ページ番号は本文を読んだ順にここで振られる。
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
			return fmt.Errorf("デモページ %sの作成に失敗: %w", pages[i].title, err)
		}

		pages[i].id = page.ID
		bar.advance()
	}

	return nil
}

// publishDemoPagesは各本文をレンダリングしてページを公開し、画面から公開した
// ときに残るリビジョンと編集者エントリを残す。2つのパスのうちの2つ目にあたり、
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
			return fmt.Errorf("デモページ %sの公開に失敗: %w", page.title, err)
		}

		// created_atをここで打つのはUpdateがその列に触れないためで、それは
		// 正しい。Updateが更新するページは、作成されたときに作成されている。一方で
		// デモページは過去へ向けて書かれるものであり、その行自身がその過去と辻褄を
		// 合わせる必要がある。最終更新より後に作成されたと述べる行は、どうやっても
		// 起こりえない状態である。
		if _, err := writer.dbtx.ExecContext(
			ctx,
			`UPDATE pages SET created_at = $3 WHERE id = $1 AND space_id = $2`,
			string(page.id), string(writer.space.id), at,
		); err != nil {
			return fmt.Errorf("デモページ %sの作成日時の打刻に失敗: %w", page.title, err)
		}

		if _, err := writer.pageRevisionRepo.Create(ctx, repository.CreatePageRevisionInput{
			SpaceID:       writer.space.id,
			SpaceMemberID: author.id,
			PageID:        page.id,
			Title:         page.title,
			Body:          page.body,
			BodyHTML:      bodyHTML,
		}); err != nil {
			return fmt.Errorf("デモページ %sのリビジョンの作成に失敗: %w", page.title, err)
		}

		if _, err := writer.pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
			SpaceID:            writer.space.id,
			PageID:             page.id,
			SpaceMemberID:      author.id,
			LastPageModifiedAt: at,
		}); err != nil {
			return fmt.Errorf("デモページ %sの編集者の登録に失敗: %w", page.title, err)
		}

		bar.advance()
	}

	return nil
}
