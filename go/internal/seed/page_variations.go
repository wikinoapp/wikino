package seed

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// Titles of the pages that exist to show one state a page can be in. The two
// states a run creates several of are numbered within their group, so that the
// section they appear in shows at a glance how many there are and in which
// order. The states a single page is enough for are named outright.
//
// [Ja] ページが取りうる状態を 1 つずつ見せるために存在するページのタイトル。実行 1 回で
// 複数作る 2 つの状態は組の中で採番し、それらが並ぶ場所で件数と順序が一目で分かる
// ようにする。1 ページで足りる状態はそのまま名前で表す。
const (
	pinnedPageTitleFormat  = "ピン留めページ %02d"
	trashedPageTitleFormat = "ゴミ箱のページ %02d"
	unwrittenPageTitle     = "未公開のページ"
)

// generatePageVariations creates the pages that show what a page can be
// besides published and readable: pinned, trashed, and the two shapes a page
// has before anything has been written into it.
//
// Publishing produces none of these states, so each page here is either
// stamped with its state after being published, or written in the shape the
// screen that creates it leaves behind.
//
// [Ja] generatePageVariations は、公開されて読める状態以外にページが取りうる姿を
// 見せるページを作成する。ピン留め・ゴミ箱、そして何も書かれる前のページが取る
// 2 つの形。
//
// これらの状態はいずれも公開では生まれないため、ここのページは公開後に状態を
// 打刻するか、それを作る画面が残す形のまま書くかのどちらかになる。
func generatePageVariations(
	ctx context.Context,
	dbtx query.DBTX,
	out io.Writer,
	amt amounts,
	spaces *seededSpaces,
	topics *seededTopics,
) error {
	// The two unwritten pages are counted as the two pages they are: unlike the
	// pages a wiki link creates, nothing else is created along the way.
	//
	// [Ja] 何も書かれていない 2 ページは、そのまま 2 件として数える。Wiki リンクが
	// 作るページと違い、途中で他のページが作られることが無いため。
	bar := newProgress(out, "状態バリエーションページ", amt.pinnedPages+amt.trashedPages+2)
	defer bar.finish()

	owner, err := spaces.wiki.requireMember(roleOwner)
	if err != nil {
		return err
	}

	writer := newPageWriter(dbtx, spaces.wiki)
	now := time.Now()

	for number := 1; number <= amt.pinnedPages; number++ {
		title := fmt.Sprintf(pinnedPageTitleFormat, number)

		author, err := variationAuthor(spaces.wiki, number)
		if err != nil {
			return err
		}

		page, err := writer.createPage(ctx, createPageInput{
			topic:  topics.notes,
			author: author,
			title:  title,
			body:   pinnedPageBody(title),
		})
		if err != nil {
			return err
		}

		// The pinned section is ordered by pinned_at descending, so the pages
		// are pinned a minute apart and in the order their numbers read.
		// Stamping them all with the same instant would leave the order to the
		// page ids, and the section would read differently from the numbers
		// printed on it.
		//
		// [Ja] ピン留めの区画は pinned_at の降順で並ぶため、番号の読み順どおりに
		// 1 分ずつずらしてピン留めする。同じ時刻で打刻すると並び順がページ ID 任せに
		// なり、区画の並びがそこに出ている番号と食い違って見える。
		if err := writer.pinPage(ctx, page, now.Add(-time.Duration(number)*time.Minute)); err != nil {
			return err
		}
		bar.advance()
	}

	for number := 1; number <= amt.trashedPages; number++ {
		title := fmt.Sprintf(trashedPageTitleFormat, number)

		author, err := variationAuthor(spaces.wiki, number)
		if err != nil {
			return err
		}

		page, err := writer.createPage(ctx, createPageInput{
			topic:  topics.notes,
			author: author,
			title:  title,
			body:   trashedPageBody(title),
		})
		if err != nil {
			return err
		}

		// Trashing is one of the few state changes the Go side already performs
		// in production, so it goes through the repository instead of an UPDATE
		// written here.
		//
		// [Ja] ゴミ箱への移動は、Go 側が本番で既に行っている数少ない状態変更の 1 つで
		// あるため、ここで UPDATE を書かずに Repository を経由する。
		if err := writer.pageRepo.TrashByID(ctx, page.id, spaces.wiki.id, now); err != nil {
			return fmt.Errorf("ページ %s のゴミ箱への移動に失敗: %w", title, err)
		}
		bar.advance()
	}

	if err := writer.createUnwrittenPage(ctx, topics.notes, owner, unwrittenPageTitle); err != nil {
		return err
	}
	bar.advance()

	if _, err := writer.createBlankPage(ctx, topics.notes, owner); err != nil {
		return err
	}
	bar.advance()

	return nil
}

// pinPage stamps pinned_at on a page, which moves it out of the page listing
// and into the pinned section shown above it.
//
// The UPDATE is written here rather than taken from a repository because
// pinning is still handled by the Rails side, and the Go side has no query for
// it.
//
// [Ja] pinPage はページに pinned_at を打刻する。これによりページは通常の一覧から
// 外れ、その上に表示されるピン留めの区画へ移る。
//
// Repository ではなくここで UPDATE を書くのは、ピン留めを担当しているのが今も
// Rails 側であり、Go 側にクエリが無いため。
func (w *pageWriter) pinPage(ctx context.Context, page *seededPage, pinnedAt time.Time) error {
	_, err := w.dbtx.ExecContext(
		ctx,
		`UPDATE pages SET pinned_at = $3, updated_at = $3 WHERE id = $1 AND space_id = $2`,
		string(page.id), string(w.space.id), pinnedAt,
	)
	if err != nil {
		return fmt.Errorf("ページ %s のピン留めに失敗: %w", page.title, err)
	}

	return nil
}

// createUnwrittenPage creates the page a wiki link leaves behind when it names
// a title nobody has written yet: a title, an empty body and no publication.
//
// The seed's other unpublished pages are all named by the link hub, so they are
// reached from a link list. This one is linked from nowhere, which is how the
// state looks when the page that once named it has been rewritten since.
//
// [Ja] createUnwrittenPage は、まだ誰も書いていないタイトルを Wiki リンクが名指し
// したときに残るページを作成する。タイトルがあり、本文は空で、公開されていない。
//
// シードの他の未公開ページはいずれもリンク集中ページが名指ししたものであり、
// リンク一覧から辿り着ける。このページはどこからもリンクされておらず、これは
// かつて名指ししていたページがその後書き換えられたときの見え方にあたる。
func (w *pageWriter) createUnwrittenPage(
	ctx context.Context,
	topic *seededTopic,
	author *seededSpaceMember,
	title string,
) error {
	if err := w.ensureTopicInSpace(topic); err != nil {
		return err
	}

	number, err := w.pageRepo.NextPageNumber(ctx, w.space.id)
	if err != nil {
		return fmt.Errorf("次のページ番号の取得に失敗: %w", err)
	}

	page, err := w.pageRepo.CreateLinkedPage(ctx, repository.CreateLinkedPageInput{
		SpaceID: w.space.id,
		TopicID: topic.id,
		Number:  number,
		Title:   title,
	})
	if err != nil {
		return fmt.Errorf("ページ %s の作成に失敗: %w", title, err)
	}

	if _, err := w.pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
		SpaceID:            w.space.id,
		PageID:             page.ID,
		SpaceMemberID:      author.id,
		LastPageModifiedAt: time.Now(),
	}); err != nil {
		return fmt.Errorf("ページ %s の編集者の登録に失敗: %w", title, err)
	}

	return nil
}

// createBlankPage creates the page the new page button leaves behind: no title,
// no body and no publication. Its NULL title is what the untitled label the
// listings and the page detail screen fall back to is read from.
//
// The row is written here rather than through a repository because creating a
// blank page is still handled by the Rails side (Pages::CreateBlankedService),
// and the Go side has no Create to call. body_html is stored empty because
// rendering an empty body produces an empty string, which is what Rails stores
// for such a page too.
//
// [Ja] createBlankPage は、ページ作成ボタンが残すページを作成する。タイトルも本文も
// 無く、公開もされていない。一覧とページ詳細画面がフォールバックする「無題」の表示は、
// このページの NULL のタイトルから来る。
//
// Repository ではなくここで行を書くのは、空のページの作成を担当しているのが今も
// Rails 側 (Pages::CreateBlankedService) であり、Go 側に呼べる Create が無いため。
// body_html を空で保存するのは、空の本文をレンダリングすると空文字列になるためで、
// Rails もこの種のページには同じものを保存している。
func (w *pageWriter) createBlankPage(
	ctx context.Context,
	topic *seededTopic,
	author *seededSpaceMember,
) (*seededPage, error) {
	if err := w.ensureTopicInSpace(topic); err != nil {
		return nil, err
	}

	number, err := w.pageRepo.NextPageNumber(ctx, w.space.id)
	if err != nil {
		return nil, fmt.Errorf("次のページ番号の取得に失敗: %w", err)
	}

	now := time.Now()

	var id string
	err = w.dbtx.QueryRowContext(
		ctx,
		`INSERT INTO pages
           (space_id, topic_id, number, title, body, body_html, linked_page_ids,
            modified_at, published_at, created_at, updated_at)
         VALUES ($1, $2, $3, NULL, '', '', '{}', $4, NULL, $4, $4)
         RETURNING id`,
		string(w.space.id), string(topic.id), int32(number), now,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("タイトル未設定のページの作成に失敗: %w", err)
	}

	pageID := model.PageID(id)

	if _, err := w.pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
		SpaceID:            w.space.id,
		PageID:             pageID,
		SpaceMemberID:      author.id,
		LastPageModifiedAt: now,
	}); err != nil {
		return nil, fmt.Errorf("タイトル未設定のページの編集者の登録に失敗: %w", err)
	}

	return &seededPage{id: pageID, number: number}, nil
}

// variationAuthor picks the account a page of the given position is attributed
// to, handing the pages round the roles so that no home screen is left without
// pages in these states.
//
// [Ja] variationAuthor は、その位置のページを誰の書いたものとして記録するかを選ぶ。
// ページを役割へ順に回すことで、これらの状態のページを持たないままのホーム画面が
// 残らないようにする。
func variationAuthor(space *seededSpace, number int) (*seededSpaceMember, error) {
	return space.memberInTurn(contentAuthorRoles, number)
}

// pinnedPageBody builds the body of a pinned page.
//
// [Ja] pinnedPageBody はピン留めされたページの本文を組み立てる。
func pinnedPageBody(title string) string {
	return fmt.Sprintf(`%s はピン留めされているため、トピックとスペースのページ一覧では、更新日順の並びの中ではなく一覧の上に表示されます。

ピン留めは、先に読んでほしいページが、新しく書かれたページに押し流されないようにするための仕組みです。ここでは複数のページをピン留めしているため、ピン留めの区画の並び順も読み取れます。並び順はピン留めした日時の新しい順で、これらのページはタイトルの番号の読み順どおりにピン留めしてあります。
`, title)
}

// trashedPageBody builds the body of a page that has been moved to the trash.
//
// [Ja] trashedPageBody はゴミ箱へ移されたページの本文を組み立てる。
func trashedPageBody(title string) string {
	return fmt.Sprintf(`%s はゴミ箱へ移されたページです。

ゴミ箱のページは、元に戻せるようにタイトルと本文をそのまま保持しますが、戻すまでの間はページ一覧からも検索結果からもリンク一覧からも消えます。そこへ辿り着ける画面はゴミ箱だけです。
`, title)
}
