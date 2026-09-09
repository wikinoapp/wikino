package seed

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGenerateSuggestions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tx, spaces, topics, amt := generateSuggestionsForTest(ctx, t, "seed-suggestion")

	suggestions := listSuggestions(ctx, t, tx, spaces.wiki.id, topics.handbook.id)

	// The listing has a tab for the suggestions still being decided and one for
	// the settled ones, and the settled tab holds two statuses. All three have
	// to be created, or a tab is looked at while it is empty or while it shows
	// only one of the two ways a suggestion can end.
	//
	// [Ja] 一覧には、まだ決着していない編集提案のタブと決着済みのタブがあり、決着済みの
	// タブは 2 つのステータスを持つ。3 つとも作られている必要がある。そうでないと、
	// タブが空のまま、あるいは編集提案の終わり方 2 つのうち片方しか出ていない状態で
	// 眺めることになる。
	byStatus := make(map[model.SuggestionStatus]int, 3)
	for _, suggestion := range suggestions {
		byStatus[suggestion.status]++
	}
	for _, want := range []struct {
		name   string
		status model.SuggestionStatus
		count  int
	}{
		{name: "オープンな編集提案", status: model.SuggestionStatusOpen, count: amt.openSuggestions},
		{name: "反映済みの編集提案", status: model.SuggestionStatusApplied, count: amt.appliedSuggestions},
		{name: "クローズされた編集提案", status: model.SuggestionStatusClosed, count: amt.closedSuggestions},
	} {
		if got := byStatus[want.status]; got != want.count {
			t.Errorf("%sが %d 件であることを期待したが %d 件だった", want.name, want.count, got)
		}
	}

	// What a member may do with a suggestion depends on whether they opened it,
	// so the two accounts take turns opening them. The expected account is
	// derived here rather than through suggestionCreator, so a regression in
	// that chooser cannot change both the generated data and its expectation.
	//
	// [Ja] メンバーが編集提案に対して何をできるかは、それを自分が開いたかどうかで
	// 変わるため、2 つのアカウントで交互に開く。期待するアカウントを
	// suggestionCreator から求めないのは、選択処理の退行によって生成結果と期待値が
	// 同時に変わることを避けるため。
	for _, suggestion := range suggestions {
		wantCreatorID := spaces.wiki.member(roleOwner).id
		if suggestion.number%2 == 0 {
			wantCreatorID = spaces.wiki.member(roleCollaborator).id
		}
		if suggestion.creatorID != wantCreatorID {
			t.Errorf("編集提案 %d の作成者が交互になっていない", suggestion.number)
		}
	}

	for _, suggestion := range suggestions {
		// A suggestion that has been applied is marked with when it happened,
		// and one that has not been must not be: the mark is what the screen
		// reports the status by.
		//
		// [Ja] 反映された編集提案には、それがいつ行われたかが記録され、反映されて
		// いない編集提案には記録されていてはならない。画面がステータスを報告する
		// 根拠がこの記録であるため。
		if suggestion.status == model.SuggestionStatusApplied && suggestion.appliedAt == nil {
			t.Errorf("反映済みの編集提案 %q に反映時刻が無い", suggestion.title)
		}
		if suggestion.status != model.SuggestionStatusApplied && suggestion.appliedAt != nil {
			t.Errorf("反映されていない編集提案 %q に反映時刻が入っている", suggestion.title)
		}

		pages := listSuggestionPages(ctx, t, tx, spaces.wiki.id, suggestion.id)
		if len(pages) == 0 {
			t.Fatalf("編集提案 %q が 1 件も提案ページを持っていない", suggestion.title)
		}

		for _, suggestionPage := range pages {
			assertSuggestionPageShowsADiff(ctx, t, tx, spaces.wiki.id, suggestion, suggestionPage)

			if got := countSuggestionPageRevisions(ctx, t, tx, spaces.wiki.id, suggestionPage.id); got == 0 {
				t.Errorf("編集提案 %q の提案ページにリビジョンが 1 件も無い", suggestion.title)
			}
		}
	}
}

// assertSuggestionPageShowsADiff checks that a proposed page can be read as a
// diff, and that the page behind it holds what its suggestion's status says it
// should.
//
// The changed pages screen takes the diff between the revision the proposal was
// written against and the proposal itself, so a proposal needs a revision to
// point at, something changed against it, and something left unchanged to be
// read as context.
//
// [Ja] assertSuggestionPageShowsADiff は、提案されたページが差分として読めること、
// そしてその先のページが、編集提案のステータスが言うとおりの内容を保持していることを
// 確認する。
//
// 変更差分画面は、提案が書かれた時点のリビジョンと提案自身との差分を取る。そのため
// 提案には、指すべきリビジョンと、それに対して変わった箇所と、文脈として読まれる
// 変わらなかった箇所が要る。
func assertSuggestionPageShowsADiff(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	suggestion suggestionRow,
	suggestionPage suggestionPageRow,
) {
	t.Helper()

	if suggestionPage.pageRevisionID == nil {
		t.Errorf("編集提案 %q の提案ページに差分の基準となるリビジョンが無い", suggestion.title)

		return
	}

	base := readPageRevisionBody(ctx, t, tx, spaceID, *suggestionPage.pageRevisionID)
	if suggestionPage.body == base {
		t.Errorf("編集提案 %q の提案ページが、基準リビジョンと同じ本文だった", suggestion.title)
	}
	if sharedLineCount(base, suggestionPage.body) == 0 {
		t.Errorf("編集提案 %q の提案ページに、基準リビジョンと共通する行が 1 行も無い", suggestion.title)
	}
	if suggestionPage.bodyHTML == "" {
		t.Errorf("編集提案 %q の提案ページの本文HTMLが空だった", suggestion.title)
	}

	page := readPageUnderSuggestion(ctx, t, tx, spaceID, suggestionPage.pageID)

	// Applying a suggestion is what puts the proposal onto the page, and it is
	// the only thing that does. Until then the page keeps what was published
	// last, which is what the diff is taken against.
	//
	// [Ja] 提案の内容がページに載るのは反映によってであり、それ以外では載らない。
	// それまでページは最後に公開された内容を保持しており、差分はそれとの間で
	// 取られる。
	if suggestion.status == model.SuggestionStatusApplied {
		if page.body != suggestionPage.body {
			t.Errorf("反映済みの編集提案 %q の内容が、ページに反映されていない", suggestion.title)
		}
		if got := countPageRevisions(ctx, t, tx, spaceID, suggestionPage.pageID); got < 2 {
			t.Errorf("反映済みの編集提案 %q のページのリビジョンが %d 件しか無い", suggestion.title, got)
		}

		return
	}

	if page.body != base {
		t.Errorf("反映されていない編集提案 %q が、ページの本文を書き換えている", suggestion.title)
	}
}

func TestGenerateSuggestionsShowcaseStates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tx, spaces, topics, amt := generateSuggestionsForTest(ctx, t, "seed-suggestion-showcase")

	suggestions := listSuggestions(ctx, t, tx, spaces.wiki.id, topics.handbook.id)
	byTitle := make(map[string]suggestionRow, len(suggestions))
	for _, suggestion := range suggestions {
		byTitle[suggestion.title] = suggestion
	}

	specs := showcaseSuggestionSpecs()
	for _, spec := range specs {
		if _, ok := byTitle[spec.title]; !ok {
			t.Fatalf("編集提案 %q が作られていない", spec.title)
		}
	}

	// The listing shows the newest suggestions first, and the ones written to
	// show a state of their own are the ones worth meeting at the top of it.
	// The comparison allows equal instants: the seed does not decide when a
	// suggestion was created — the row is written by the same Create the screen
	// calls — so two of them landing on the same instant is not a failure of
	// the order they were created in.
	//
	// [Ja] 一覧は新しい編集提案から順に見せる。その先頭で出会う価値があるのは、固有の
	// 状態を見せるために書いた編集提案である。比較で同時刻を許しているのは、作成時刻を
	// シードが決めていないため。行を書くのは画面が呼ぶのと同じ Create であり、2 件が
	// 同じ時刻になったとしても、作成順そのものが崩れたわけではない。
	var newestOrdinary time.Time
	for _, suggestion := range suggestions {
		if _, isShowcase := showcaseTitles(specs)[suggestion.title]; isShowcase {
			continue
		}
		if suggestion.createdAt.After(newestOrdinary) {
			newestOrdinary = suggestion.createdAt
		}
	}
	for _, spec := range specs {
		if byTitle[spec.title].createdAt.Before(newestOrdinary) {
			t.Errorf("編集提案 %q が、通常の編集提案より前に作成されている", spec.title)
		}
	}

	// One suggestion changes several pages at once, and renames the last of
	// them. Both are states the changed pages screen shows apart from an
	// ordinary body edit.
	//
	// [Ja] 1 件の編集提案が複数のページを一度に変更し、その最後のページの名前を
	// 変更する。どちらも、変更差分画面が通常の本文の編集とは別に見せる状態になる。
	multiPage := byTitle[multiPageSuggestionTitle]
	pages := listSuggestionPages(ctx, t, tx, spaces.wiki.id, multiPage.id)
	if want := showcaseSpecByTitle(specs, multiPageSuggestionTitle).changedPages; len(pages) != want {
		t.Fatalf("編集提案 %q の提案ページが %d 件であることを期待したが %d 件だった", multiPage.title, want, len(pages))
	}

	renamed := 0
	for _, suggestionPage := range pages {
		if suggestionPage.title == nil {
			t.Fatalf("編集提案 %q の提案ページにタイトルが無い", multiPage.title)
		}

		page := readPageUnderSuggestion(ctx, t, tx, spaces.wiki.id, suggestionPage.pageID)
		if page.title == nil {
			t.Fatalf("編集提案 %q の対象ページにタイトルが無い", multiPage.title)
		}
		if *suggestionPage.title != *page.title {
			renamed++
			if !strings.HasSuffix(*suggestionPage.title, renamedPageTitleSuffix) {
				t.Errorf("名前の変更を提案しているタイトル %q が想定の形になっていない", *suggestionPage.title)
			}
		}
	}
	if renamed != 1 {
		t.Errorf("名前の変更を提案している提案ページが 1 件であることを期待したが %d 件だった", renamed)
	}

	// One suggestion carries a discussion. The comments are shown in the order
	// they were written and numbered in the same order, and the two accounts
	// take turns so that a thread is not read as one member talking to
	// themselves.
	//
	// [Ja] 1 件の編集提案が議論を持つ。コメントは書かれた順に表示され、同じ順に採番
	// される。また 2 つのアカウントが交互に発言するため、スレッドが 1 人の独り言として
	// 読まれることがない。
	comments := listSuggestionComments(ctx, t, tx, spaces.wiki.id, byTitle[discussionSuggestionTitle].id)
	if len(comments) != amt.suggestionComments {
		t.Fatalf("編集提案のコメントが %d 件であることを期待したが %d 件だった", amt.suggestionComments, len(comments))
	}
	for i, comment := range comments {
		position := i + 1
		if comment.number != int32(position) {
			t.Errorf("%d 件目のコメントの番号が %d であることを期待したが %d だった", position, position, comment.number)
		}
		wantCreatorID := spaces.wiki.member(roleOwner).id
		if position%2 == 0 {
			wantCreatorID = spaces.wiki.member(roleCollaborator).id
		}
		if comment.creatorID != wantCreatorID {
			t.Errorf("%d 件目のコメントが、交互に発言する順番どおりの投稿者になっていない", position)
		}
		if comment.body == "" {
			t.Errorf("%d 件目のコメントの本文が空だった", position)
		}
	}
	for _, suggestion := range suggestions {
		if suggestion.title == discussionSuggestionTitle {
			continue
		}
		if got := len(listSuggestionComments(ctx, t, tx, spaces.wiki.id, suggestion.id)); got != 0 {
			t.Errorf("編集提案 %q にコメントが %d 件付いている", suggestion.title, got)
		}
	}

	// One suggestion is in the middle of being edited. The draft the editor
	// left behind is linked to the proposed page, which is what the page detail
	// screen reads to report that the page is being changed under a suggestion.
	//
	// [Ja] 1 件の編集提案は編集の途中にある。編集画面が残した下書きは提案ページに
	// 紐づいており、ページ詳細画面が「このページは編集提案の下で変更されようとして
	// いる」と知らせるために読むのがこの紐づけになる。
	editStarted := byTitle[editStartedSuggestionTitle]
	editedPages := listSuggestionPages(ctx, t, tx, spaces.wiki.id, editStarted.id)
	if len(editedPages) == 0 {
		t.Fatalf("編集提案 %q が提案ページを持っていない", editStarted.title)
	}

	drafts := listDraftPages(ctx, t, tx, spaces.wiki.id, spaces.wiki.member(roleOwner).id)
	if len(drafts) != 1 {
		t.Fatalf("編集中の下書きが 1 件であることを期待したが %d 件だった", len(drafts))
	}
	if drafts[0].pageID != editedPages[0].pageID {
		t.Errorf("編集中の下書きが、提案されているページに付いていない")
	}
	if drafts[0].body != editedPages[0].body {
		t.Errorf("編集中の下書きが、提案されている本文を持っていない")
	}
	if got := readDraftSuggestionPageID(ctx, t, tx, spaces.wiki.id, drafts[0].id); got == nil || *got != editedPages[0].id {
		t.Errorf("編集中の下書きが、提案ページに紐づいていない")
	}
}

// TestGenerateSuggestionsStampsTheStartedEditDraftBehindEarlierDrafts checks
// that the draft a started edit leaves behind takes the run's next stamp
// instead of the current time. The drafts on unpublished pages are written
// before it and are the ones the home screen is meant to keep, so a draft
// stamped here from the current time would stand ahead of them and push one of
// them off that screen.
//
// [Ja] TestGenerateSuggestionsStampsTheStartedEditDraftBehindEarlierDrafts は、
// 編集を始めたときに残る下書きが、現在時刻ではなく実行の次の打刻を受け取ることを
// 確認する。未公開のページに付く下書きはそれより先に書かれ、ホーム画面が残すべき
// 下書きであるため、ここで現在時刻を打刻するとそれらより前に並び、どれかを
// その画面から押し出してしまう。
func TestGenerateSuggestionsStampsTheStartedEditDraftBehindEarlierDrafts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tx, spaces, topics, amt := prepareSuggestionsForTest(ctx, t, "seed-suggestion-stamp")

	// The stamp taken here stands for a draft an earlier phase wrote. The
	// suggestion generator is handed the same counter, so the draft it writes
	// has to land behind that one.
	//
	// [Ja] ここで受け取る打刻は、先行するフェーズが書いた下書きを表す。編集提案の
	// 生成器には同じカウンターを渡すため、そこで書かれる下書きはその後ろに並ぶ
	// はずである。
	stamps := newDraftStamps(time.Now())
	earlier := stamps.next()

	if err := generateSuggestions(ctx, tx, io.Discard, amt, spaces, topics, stamps); err != nil {
		t.Fatalf("編集提案の生成に失敗: %v", err)
	}

	drafts := listDraftPages(ctx, t, tx, spaces.wiki.id, spaces.wiki.member(roleOwner).id)
	if len(drafts) != 1 {
		t.Fatalf("編集中の下書きが 1 件であることを期待したが %d 件だった", len(drafts))
	}
	if !drafts[0].modifiedAt.Before(earlier) {
		t.Errorf(
			"編集中の下書きの最終更新 %v が、先に書かれた下書きの %v より後になっている",
			drafts[0].modifiedAt, earlier,
		)
	}
}
func TestSuggestionPageBodyKeepsTheBaseAsContext(t *testing.T) {
	t.Parallel()

	// The diff is taken against what the page holds now. A proposal that shares
	// no lines with it is shown as the whole page being replaced, and one that
	// rewrites a heading moves the change into the outline of the page instead
	// of into a paragraph of it.
	//
	// The paragraph is written over two lines, which the bodies the seed writes
	// are not. The sentence has to land at the end of the paragraph, and only a
	// paragraph of several lines tells that apart from landing on its first line,
	// where it would sit in the middle of a sentence.
	//
	// [Ja] 差分は、ページが現在保持している内容との間で取られる。共通する行を持たない
	// 提案はページ全体の置き換えとして表示され、見出しを書き換える提案は、変更を段落の
	// 中ではなくページの見出し構成へ移してしまう。
	//
	// 段落を 2 行にまたがらせているのは、シードが書く本文がそうなっていないため。文は
	// 段落の末尾に足される必要があり、それを最初の行へ足す挙動と区別できるのは複数行の
	// 段落だけになる。最初の行へ足すと文の途中に入り込む。
	const base = `## 概要

ページの最初の段落が 1 行に収まらず、
2 行にまたがっている場合を基準にしています。

- リストの項目。
- もう 1 つのリストの項目。
`

	body := suggestionPageBody(base)

	if !strings.Contains(body, "## 概要") {
		t.Error("提案された本文が、基準の見出しを書き換えている")
	}
	if !strings.Contains(body, "- リストの項目。") {
		t.Error("提案された本文に、変更していないはずの行が残っていない")
	}
	if !strings.Contains(body, "2 行にまたがっている場合を基準にしています。"+suggestionRewrittenSentence) {
		t.Error("提案された本文で、見出しでない最初の段落の最後の行が書き換えられていない")
	}
	if !strings.HasSuffix(body, suggestionAddedSection) {
		t.Error("提案された本文の末尾に、追加された節が無い")
	}
	if sharedLineCount(base, body) == 0 {
		t.Error("提案された本文に、基準と共通する行が 1 行も無い")
	}
}

func TestSuggestionPageBodyEndsTheParagraphItRewrites(t *testing.T) {
	t.Parallel()

	// The pages a suggestion is written against hold the bodies the bulk page
	// generator writes. Adding the sentence anywhere but at the end of a
	// paragraph puts it in the middle of another sentence, which is read as one
	// broken sentence on the screen.
	//
	// [Ja] 編集提案の対象になるページは、ページネーション用ページの生成器が書いた本文を
	// 持っている。段落の末尾以外へ文を足すと別の文の途中へ入り込み、画面では壊れた 1 文
	// として読まれる。
	for i, template := range bulkPageBodies {
		body := suggestionPageBody(fmt.Sprintf(template, topicNameHandbook+" 001"))

		lines := strings.Split(body, "\n")
		rewritten := 0
		for j, line := range lines {
			if !strings.HasSuffix(line, suggestionRewrittenSentence) {
				continue
			}
			rewritten++

			if j+1 < len(lines) && strings.TrimSpace(lines[j+1]) != "" {
				t.Errorf("%d 件目の本文で、書き換えた行の後に段落が続いている: %q", i+1, line)
			}
		}
		if rewritten != 1 {
			t.Errorf("%d 件目の本文で書き換えられた行が %d 行だった", i+1, rewritten)
		}
	}
}

func TestSuggestionCopyUsesJapanese(t *testing.T) {
	t.Parallel()

	// These expectations intentionally stay independent of the production
	// titles and prose. A test derived from the constants or specs would follow
	// them back to English instead of catching the regression. None of this copy
	// has a reason to retain English words, so an ASCII letter anywhere in it is
	// also a failure.
	//
	// [Ja] 期待値は、実装側のタイトルや本文から意図的に独立させる。定数や spec から
	// 導出すると、実装が英語へ戻ったときにテストも追従し、退行を検出できないため。
	// ここに英単語を残す理由は無いため、ASCII の英字が 1 文字でも含まれている場合も
	// 失敗とする。
	pageTitle := "ハンドブック 001"
	for _, tt := range []struct {
		name string
		got  string
		want string
	}{
		{name: "通常の編集提案のタイトル", got: fmt.Sprintf(ordinarySuggestionTitleFormat, pageTitle), want: "ハンドブック 001 を修正する"},
		{name: "複数ページを変更する編集提案のタイトル", got: multiPageSuggestionTitle, want: "3 ページをまとめて書き直す"},
		{name: "議論を持つ編集提案のタイトル", got: discussionSuggestionTitle, want: "どのページも導入文を繰り返すべきか?"},
		{name: "編集途中の編集提案のタイトル", got: editStartedSuggestionTitle, want: "ページの書き出しを言い換える"},
		{name: "改題するページのタイトル接尾辞", got: renamedPageTitleSuffix, want: " (改題)"},
		{name: "提案ページが書き換える文", got: suggestionRewrittenSentence, want: "この文は、レビュー中の編集がこの行に足したものです。"},
	} {
		if tt.got != tt.want {
			t.Errorf("%sが %q であることを期待したが %q だった", tt.name, tt.want, tt.got)
		}
	}

	type copyExpectation struct {
		name    string
		copy    string
		markers []string
	}
	copies := []copyExpectation{
		{
			name: "通常の編集提案の本文",
			copy: ordinarySuggestionBody(pageTitle),
			markers: []string{
				"この編集提案はハンドブック 001 に小さな編集を提案します",
				"1 行を書き換え、末尾に節を 1 つ足すものです",
			},
		},
		{
			name: "提案ページへ追加する節",
			copy: suggestionAddedSection,
			markers: []string{
				"## レビューメモ",
				"追加されたまとまりも見せるようになります",
			},
		},
	}

	showcaseWants := []struct {
		title   string
		markers []string
	}{
		{
			title:   "3 ページをまとめて書き直す",
			markers: []string{"3 つのページを一度に変更", "名前を変更したページ"},
		},
		{
			title:   "どのページも導入文を繰り返すべきか?",
			markers: []string{"議論を持たせるため", "2 つのアカウントが交互に書いた"},
		},
		{
			title:   "ページの書き出しを言い換える",
			markers: []string{"いま誰かが編集している最中", "下の提案ページに紐づいて"},
		},
	}
	showcases := showcaseSuggestionSpecs()
	if len(showcases) != len(showcaseWants) {
		t.Fatalf("固有の状態を見せる編集提案が %d 件であることを期待したが %d 件だった", len(showcaseWants), len(showcases))
	}
	for i, want := range showcaseWants {
		if showcases[i].title != want.title {
			t.Errorf("%d 件目の固有の編集提案のタイトルが %q であることを期待したが %q だった", i+1, want.title, showcases[i].title)
		}
		copies = append(copies, copyExpectation{
			name:    fmt.Sprintf("固有の編集提案 %q の本文", want.title),
			copy:    showcases[i].body,
			markers: want.markers,
		})
	}

	commentMarkers := [][]string{
		{"コメント 1。書き換えられた行は", "繰り返しているように見えます"},
		{"コメント 2。節の件は同意です", "他のページに揃えられないでしょうか"},
		{"コメント 3。差分に「追加されたまとまり」", "反映する前に削って構いません"},
		{"コメント 4。もう 1 点あります", "編集提案が終われば意味を持たなくなります"},
		{"コメント 5。このページが並ぶ他のページ", "私はこのままでよいと思います"},
		{"コメント 6。私からは以上です", "あとは反映してよい程度の小さな変更です"},
	}
	if len(suggestionCommentBodies) != len(commentMarkers) {
		t.Fatalf("編集提案のコメント本文が %d 件であることを期待したが %d 件だった", len(commentMarkers), len(suggestionCommentBodies))
	}
	for i, markers := range commentMarkers {
		copies = append(copies, copyExpectation{
			name:    fmt.Sprintf("%d 件目のコメント本文", i+1),
			copy:    suggestionCommentBody(i + 1),
			markers: markers,
		})
	}

	for _, tt := range copies {
		for _, marker := range tt.markers {
			if !strings.Contains(tt.copy, marker) {
				t.Errorf("%sに日本語のマーカー %q が含まれていない: %q", tt.name, marker, tt.copy)
			}
		}
		if containsASCIILetter(tt.copy) {
			t.Errorf("%sにASCIIの英字が含まれている: %q", tt.name, tt.copy)
		}
	}
}

func TestSuggestionBodiesCarryNoWikilinks(t *testing.T) {
	t.Parallel()

	// A wiki link in a proposed body would have the resolver create the page it
	// names, adding a page to a topic whose listing counts are decided
	// elsewhere. A suggestion exists to show what a proposed edit looks like,
	// not to add pages.
	//
	// [Ja] 提案された本文に Wiki リンクがあると、resolver がその名前のページを作成し、
	// 一覧の件数を別の場所で決めているトピックにページが増える。編集提案は、提案された
	// 編集がどう見えるかを示すために存在するのであって、ページを増やすためではない。
	bodies := []string{suggestionPageBody("ページがいま持っている本文。\n")}
	for position := 1; position <= len(suggestionCommentBodies); position++ {
		bodies = append(bodies, suggestionCommentBody(position))
	}
	bodies = append(bodies, ordinarySuggestionBody(topicNameHandbook+" 001"))
	for _, spec := range showcaseSuggestionSpecs() {
		bodies = append(bodies, spec.body)
	}

	for _, body := range bodies {
		if got := len(markup.ScanWikilinks(body, topicNameHandbook)); got != 0 {
			t.Errorf("編集提案の本文にWikiリンクが %d 件含まれている: %q", got, body)
		}
	}
}

func TestGenerateSuggestionsRejectsTooFewOpen(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	spaces := buildSeedSpaces(t, tx, "seed-suggestion-few")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	// The suggestions written to show a state of their own are a fixed list, so
	// a count of open suggestions below its length cannot be satisfied. Making
	// fewer of them would silently drop one of the states they exist to show.
	//
	// [Ja] 固有の状態を見せるために書いた編集提案は固定の一覧であるため、オープンな
	// 編集提案の件数がその長さを下回る指定は満たせない。黙って減らすと、それらが見せる
	// ために存在する状態のどれかが失われる。
	amt := amounts{openSuggestions: len(showcaseSuggestionSpecs()) - 1}
	err = generateSuggestions(ctx, tx, io.Discard, amt, spaces, topics, newDraftStamps(time.Now()))
	if err == nil {
		t.Fatal("オープンな編集提案の件数が足りない場合にエラーになることを期待した")
	}

	// The topic holds no pages either, so a run that stopped for want of target
	// pages would return an error as well. The message is what tells the two
	// apart, and without checking it the guard could be removed without the
	// test noticing.
	//
	// [Ja] このトピックはページも持たないため、対象ページが足りずに止まった実行も
	// エラーを返す。両者を見分けるのはメッセージであり、これを確認しないとガードを
	// 外してもテストが気づかない。
	if !strings.Contains(err.Error(), "固定で作る") {
		t.Errorf("オープン件数不足のエラーを期待したが %v だった", err)
	}
}

func TestGenerateSuggestionsUpdatesAppliedEditorTimes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tx, spaces, topics, amt := prepareSuggestionsForTest(ctx, t, "seed-suggestion-editor-times")

	// Ordinary open suggestions are created before the applied ones. The next
	// target therefore belongs to the first applied suggestion. Its bulk page
	// was already written by roleOwner, which exercises the FindOrCreate path
	// that finds an existing PageEditor instead of creating one.
	//
	// [Ja] 通常のオープンな編集提案は、反映済みのものより先に作られる。その次の対象が
	// 最初の反映済み編集提案に使われる。この一括生成ページは既に roleOwner が書いており、
	// PageEditor を新規作成せず既存のものを返す FindOrCreate の経路を通る。
	ordinaryOpen := amt.openSuggestions - len(showcaseSuggestionSpecs())
	targets, err := collectSuggestionTargets(ctx, tx, topics.handbook, ordinaryOpen+1)
	if err != nil {
		t.Fatalf("反映済み編集提案の対象ページの取得に失敗: %v", err)
	}
	appliedTarget := targets[ordinaryOpen]
	oldModifiedAt := time.Now().Add(-24 * time.Hour)

	result, err := tx.ExecContext(
		ctx,
		`UPDATE page_editors
         SET last_page_modified_at = $1
         WHERE space_id = $2 AND page_id = $3 AND space_member_id = $4`,
		oldModifiedAt, string(spaces.wiki.id), string(appliedTarget.page.id), string(spaces.wiki.member(roleOwner).id),
	)
	if err != nil {
		t.Fatalf("既存ページ編集者の更新時刻を戻せなかった: %v", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("既存ページ編集者の更新件数を取得できなかった: %v", err)
	}
	if affected != 1 {
		t.Fatalf("反映対象に owner の既存ページ編集者が 1 件あることを期待したが %d 件だった", affected)
	}

	result, err = tx.ExecContext(
		ctx,
		`UPDATE topic_members
         SET last_page_modified_at = $1
         WHERE space_id = $2 AND topic_id = $3 AND space_member_id = $4`,
		oldModifiedAt, string(spaces.wiki.id), string(topics.handbook.id), string(spaces.wiki.member(roleOwner).id),
	)
	if err != nil {
		t.Fatalf("トピックメンバーの更新時刻を戻せなかった: %v", err)
	}
	affected, err = result.RowsAffected()
	if err != nil {
		t.Fatalf("トピックメンバーの更新件数を取得できなかった: %v", err)
	}
	if affected != 1 {
		t.Fatalf("「ハンドブック」に owner のトピックメンバーが 1 件あることを期待したが %d 件だった", affected)
	}

	if err := generateSuggestions(ctx, tx, io.Discard, amt, spaces, topics, newDraftStamps(time.Now())); err != nil {
		t.Fatalf("編集提案の生成に失敗: %v", err)
	}

	var pageEditorModifiedAt time.Time
	err = tx.QueryRowContext(
		ctx,
		`SELECT last_page_modified_at
         FROM page_editors
         WHERE space_id = $1 AND page_id = $2 AND space_member_id = $3`,
		string(spaces.wiki.id), string(appliedTarget.page.id), string(spaces.wiki.member(roleOwner).id),
	).Scan(&pageEditorModifiedAt)
	if err != nil {
		t.Fatalf("反映者のページ編集時刻を取得できなかった: %v", err)
	}
	if !pageEditorModifiedAt.After(oldModifiedAt) {
		t.Errorf("反映者のページ編集時刻が更新されていない: %s", pageEditorModifiedAt)
	}

	var topicMemberModifiedAt sql.NullTime
	err = tx.QueryRowContext(
		ctx,
		`SELECT last_page_modified_at
         FROM topic_members
         WHERE space_id = $1 AND topic_id = $2 AND space_member_id = $3`,
		string(spaces.wiki.id), string(topics.handbook.id), string(spaces.wiki.member(roleOwner).id),
	).Scan(&topicMemberModifiedAt)
	if err != nil {
		t.Fatalf("反映者のトピック編集時刻を取得できなかった: %v", err)
	}
	if !topicMemberModifiedAt.Valid || !topicMemberModifiedAt.Time.After(oldModifiedAt) {
		t.Errorf("反映者のトピック編集時刻が更新されていない: %v", topicMemberModifiedAt)
	}
}

func TestGenerateSuggestionsExcludesPagesWithDrafts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tx, spaces, topics, amt := prepareSuggestionsForTest(ctx, t, "seed-suggestion-draft-target")

	// Put a draft on the first page the target query would otherwise return.
	// Enough later pages remain for every suggestion, so a successful run must
	// skip this page rather than stop before creating suggestion pages.
	//
	// [Ja] 対象取得クエリが本来最初に返すページへ下書きを付ける。後続ページだけで
	// すべての編集提案を作れる件数が残るため、生成が成功したなら、提案ページを作らず
	// 終わったのではなく、このページを飛ばしたことになる。
	targets, err := collectSuggestionTargets(ctx, tx, topics.handbook, 1)
	if err != nil {
		t.Fatalf("下書きを付ける候補ページの取得に失敗: %v", err)
	}
	excluded := targets[0]
	draftTitle := excluded.page.title
	writer := newSuggestionWriter(tx, spaces.wiki, spaces.wiki.member(roleOwner), newDraftStamps(time.Now()))
	if _, err := writer.draftPageRepo.Create(ctx, repository.CreateDraftPageInput{
		SpaceID:       spaces.wiki.id,
		PageID:        excluded.page.id,
		SpaceMemberID: spaces.wiki.member(roleOwner).id,
		TopicID:       topics.handbook.id,
		Title:         &draftTitle,
		Body:          excluded.body,
		BodyHTML:      "<p>Draft on a suggestion target.</p>",
		LinkedPageIDs: []model.PageID{},
		ModifiedAt:    time.Now(),
	}); err != nil {
		t.Fatalf("編集提案候補ページへの下書き作成に失敗: %v", err)
	}

	if err := generateSuggestions(ctx, tx, io.Discard, amt, spaces, topics, newDraftStamps(time.Now())); err != nil {
		t.Fatalf("編集提案の生成に失敗: %v", err)
	}

	var count int
	err = tx.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
         FROM suggestion_pages
         WHERE space_id = $1 AND page_id = $2`,
		string(spaces.wiki.id), string(excluded.page.id),
	).Scan(&count)
	if err != nil {
		t.Fatalf("下書き付きページを参照する提案ページ数の取得に失敗: %v", err)
	}
	if count != 0 {
		t.Errorf("下書き付きページが %d 件の編集提案に使われている", count)
	}
}

func TestGenerateSuggestionsRejectsTooFewTargets(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	spaces := buildSeedSpaces(t, tx, "seed-suggestion-few-targets")
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	amt := amounts{
		handbookPages:   1,
		openSuggestions: len(showcaseSuggestionSpecs()),
	}
	if err := generateBulkPages(ctx, tx, io.Discard, amt, spaces, topics); err != nil {
		t.Fatalf("ページネーション用ページの生成に失敗: %v", err)
	}

	err = generateSuggestions(ctx, tx, io.Discard, amt, spaces, topics, newDraftStamps(time.Now()))
	if err == nil {
		t.Fatal("編集提案の対象ページが足りない場合にエラーになることを期待した")
	}
	if !strings.Contains(err.Error(), "編集提案の対象にできるページ") {
		t.Errorf("対象ページ不足のエラーを期待したが %v だった", err)
	}
}

func TestCreateSuggestionRejectsAnUnknownStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tx, spaces, topics, _ := prepareSuggestionsForTest(ctx, t, "seed-suggestion-status")

	targets, err := collectSuggestionTargets(ctx, tx, topics.handbook, 1)
	if err != nil {
		t.Fatalf("編集提案の対象ページの取得に失敗: %v", err)
	}

	// Each status a suggestion comes to rest at needs writing down: applying
	// one rewrites the pages it names, while closing one leaves them alone. A
	// status this generator has nothing written for is refused, rather than
	// left to fall through and produce an open suggestion the caller did not
	// ask for. The draft status is one such: it exists on the model and no part
	// of the seed creates it.
	//
	// [Ja] 編集提案が落ち着く先のステータスは、それぞれ書き下しておく必要がある。
	// 反映は名指ししたページを書き換え、クローズはページに触れないためである。この
	// 生成器が何も書いていないステータスは、素通りして呼び出し側が求めていない
	// オープンな編集提案になるのではなく、拒否される。下書きステータスがそれにあたる。
	// モデルには存在するが、シードのどこもそれを作らない。
	err = newSuggestionWriter(tx, spaces.wiki, spaces.wiki.member(roleOwner), newDraftStamps(time.Now())).createSuggestion(ctx, createSuggestionInput{
		topic:   topics.handbook,
		creator: spaces.wiki.member(roleOwner),
		title:   "未知のステータスで止まった編集提案",
		body:    "この編集提案は、生成器が何も書いていないステータスが拒否されることを確認するために存在します。",
		targets: targets,
		status:  model.SuggestionStatusDraft,
	})
	if err == nil {
		t.Fatal("未知のステータスが指定された場合にエラーになることを期待した")
	}
	if !strings.Contains(err.Error(), "未知のステータス") {
		t.Errorf("未知のステータスのエラーを期待したが %v だった", err)
	}
}

func TestDefaultAmountsFillBothSuggestionTabs(t *testing.T) {
	t.Parallel()

	// The listing does not paginate, so there is no page size to check the
	// counts against. What has to hold is that neither tab is met empty, that
	// the settled tab holds both of the ways a suggestion can end, and that the
	// open tab holds more than the suggestions written to show a state of their
	// own — otherwise it is a listing of special cases rather than a listing.
	//
	// [Ja] この一覧はページングしないため、件数を突き合わせる 1 ページあたりの件数は
	// 無い。守るべきなのは、どちらのタブも空のまま出会わないこと、決着済みのタブが
	// 編集提案の終わり方 2 つを両方持つこと、そしてオープンのタブが、固有の状態を
	// 見せるために書いた編集提案より多くを持つことである。そうでなければ、一覧ではなく
	// 特殊例の並びになってしまう。
	if got, least := defaultAmounts.openSuggestions, len(showcaseSuggestionSpecs()); got <= least {
		t.Errorf("オープンな編集提案 %d 件が、固有の状態を見せる %d 件を超えることを期待した", got, least)
	}
	for _, tt := range []struct {
		name  string
		count int
	}{
		{name: "反映済みの編集提案", count: defaultAmounts.appliedSuggestions},
		{name: "クローズされた編集提案", count: defaultAmounts.closedSuggestions},
		{name: "議論を持つ編集提案のコメント", count: defaultAmounts.suggestionComments},
	} {
		if tt.count < 1 {
			t.Errorf("%sが 1 件以上であることを期待したが %d 件だった", tt.name, tt.count)
		}
	}

	// The comment bodies are written as one thread that answers itself and is
	// settled by its last remark. Asking for more comments than there are
	// bodies cycles back to the opening remark, which puts it under the one
	// that settled the thread.
	//
	// [Ja] コメントの本文は、互いに答えながら最後の 1 件で収まる 1 つのスレッドとして
	// 書かれている。本文の数より多いコメントを求めると冒頭の発言へ循環し、スレッドを
	// 収めた発言の下にそれが並ぶことになる。
	if got, least := len(suggestionCommentBodies), defaultAmounts.suggestionComments; got < least {
		t.Errorf("コメントの本文が %d 件しかなく、既定の %d 件に足りない", got, least)
	}
}

// generateSuggestionsForTest prepares the space, topics and published pages a
// suggestion can be written against, and runs the generator over them.
//
// The pages come from the generator that fills a topic's listing, because the
// diff of a proposal is taken against the body the page already holds: pages
// created here with a body of their own would say less about what a proposal
// looks like against a real one.
//
// [Ja] generateSuggestionsForTest は、編集提案の対象にできるスペース・トピック・
// 公開済みページを用意し、その上で生成器を実行する。
//
// ページはトピックの一覧を埋める生成器から取る。提案の差分は、ページが既に保持して
// いる本文との間で取られるため、ここで独自の本文を持つページを作ると、実際のページに
// 対する提案がどう見えるかについて語れることが減ってしまう。
func generateSuggestionsForTest(
	ctx context.Context,
	t *testing.T,
	prefix string,
) (*sql.Tx, *seededSpaces, *seededTopics, amounts) {
	t.Helper()

	tx, spaces, topics, amt := prepareSuggestionsForTest(ctx, t, prefix)
	if err := generateSuggestions(ctx, tx, io.Discard, amt, spaces, topics, newDraftStamps(time.Now())); err != nil {
		t.Fatalf("編集提案の生成に失敗: %v", err)
	}

	return tx, spaces, topics, amt
}

// prepareSuggestionsForTest prepares the space, topics and published pages a
// suggestion can be written against, leaving the suggestion generator for the
// test to run after arranging the state it needs.
//
// [Ja] prepareSuggestionsForTest は、編集提案の対象にできるスペース・トピック・
// 公開済みページを用意する。テストが必要な状態を整えてから実行できるよう、編集提案の
// 生成自体は呼び出し元に残す。
func prepareSuggestionsForTest(
	ctx context.Context,
	t *testing.T,
	prefix string,
) (*sql.Tx, *seededSpaces, *seededTopics, amounts) {
	t.Helper()

	_, tx := testutil.SetupTx(t)

	spaces := buildSeedSpaces(t, tx, prefix)
	topics, err := generateTopics(ctx, tx, io.Discard, spaces)
	if err != nil {
		t.Fatalf("トピック生成に失敗: %v", err)
	}

	specs := showcaseSuggestionSpecs()
	amt := amounts{
		handbookPages:      12,
		openSuggestions:    len(specs) + 2,
		appliedSuggestions: 1,
		closedSuggestions:  1,
		suggestionComments: 3,
	}

	if err := generateBulkPages(ctx, tx, io.Discard, amt, spaces, topics); err != nil {
		t.Fatalf("ページネーション用ページの生成に失敗: %v", err)
	}

	return tx, spaces, topics, amt
}

// showcaseTitles lists the titles of the suggestions written to show a state of
// their own, for telling them apart from the ordinary ones.
//
// [Ja] showcaseTitles は、固有の状態を見せるために書いた編集提案のタイトルを列挙する。
// 通常の編集提案と見分けるために使う。
func showcaseTitles(specs []showcaseSuggestionSpec) map[string]bool {
	titles := make(map[string]bool, len(specs))
	for _, spec := range specs {
		titles[spec.title] = true
	}

	return titles
}

// showcaseSpecByTitle returns the spec with the given title.
//
// [Ja] showcaseSpecByTitle は、指定タイトルの仕様を返す。
func showcaseSpecByTitle(specs []showcaseSuggestionSpec, title string) showcaseSuggestionSpec {
	for _, spec := range specs {
		if spec.title == title {
			return spec
		}
	}

	return showcaseSuggestionSpec{}
}

// sharedLineCount counts the lines the two bodies have in common, which is what
// a diff shows as unchanged context.
//
// [Ja] sharedLineCount は、2 つの本文に共通する行を数える。差分が変更のない文脈として
// 見せるのはこの行になる。
func sharedLineCount(base string, body string) int {
	inBase := make(map[string]bool)
	for _, line := range strings.Split(base, "\n") {
		if strings.TrimSpace(line) != "" {
			inBase[line] = true
		}
	}

	shared := 0
	for _, line := range strings.Split(body, "\n") {
		if inBase[line] {
			shared++
		}
	}

	return shared
}

// suggestionRow is a stored suggestion.
//
// [Ja] suggestionRow は保存された編集提案。
type suggestionRow struct {
	id        model.SuggestionID
	number    int32
	title     string
	status    model.SuggestionStatus
	creatorID model.SpaceMemberID
	appliedAt *time.Time
	createdAt time.Time
}

// listSuggestions reads back a topic's suggestions in the order the listing
// shows them.
//
// [Ja] listSuggestions は、あるトピックの編集提案を、一覧が見せる順序で読み戻す。
func listSuggestions(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	topicID model.TopicID,
) []suggestionRow {
	t.Helper()

	rows, err := tx.QueryContext(
		ctx,
		`SELECT id, number, title, status, created_space_member_id, applied_at, created_at
         FROM suggestions
         WHERE space_id = $1 AND topic_id = $2
         ORDER BY created_at DESC`,
		string(spaceID), string(topicID),
	)
	if err != nil {
		t.Fatalf("編集提案の取得に失敗: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var suggestions []suggestionRow
	for rows.Next() {
		var (
			suggestion suggestionRow
			id         string
			status     int32
			creatorID  string
		)
		if err := rows.Scan(
			&id, &suggestion.number, &suggestion.title, &status, &creatorID,
			&suggestion.appliedAt, &suggestion.createdAt,
		); err != nil {
			t.Fatalf("編集提案の読み取りに失敗: %v", err)
		}

		suggestion.id = model.SuggestionID(id)
		suggestion.status = model.SuggestionStatus(status)
		suggestion.creatorID = model.SpaceMemberID(creatorID)
		suggestions = append(suggestions, suggestion)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("編集提案の読み取りに失敗: %v", err)
	}

	return suggestions
}

// suggestionPageRow is one stored page of a suggestion.
//
// [Ja] suggestionPageRow は、編集提案が保存している提案ページ 1 件。
type suggestionPageRow struct {
	id             model.SuggestionPageID
	pageID         model.PageID
	pageRevisionID *model.PageRevisionID
	title          *string
	body           string
	bodyHTML       string
}

// listSuggestionPages reads back a suggestion's pages in the order the changed
// pages screen shows them.
//
// [Ja] listSuggestionPages は、編集提案の提案ページを、変更差分画面が見せる順序で
// 読み戻す。
func listSuggestionPages(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	suggestionID model.SuggestionID,
) []suggestionPageRow {
	t.Helper()

	rows, err := tx.QueryContext(
		ctx,
		`SELECT id, page_id, page_revision_id, title, body, body_html
         FROM suggestion_pages
         WHERE space_id = $1 AND suggestion_id = $2
         ORDER BY created_at ASC, id ASC`,
		string(spaceID), string(suggestionID),
	)
	if err != nil {
		t.Fatalf("提案ページの取得に失敗: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var pages []suggestionPageRow
	for rows.Next() {
		var (
			page           suggestionPageRow
			id             string
			pageID         string
			pageRevisionID *string
		)
		if err := rows.Scan(&id, &pageID, &pageRevisionID, &page.title, &page.body, &page.bodyHTML); err != nil {
			t.Fatalf("提案ページの読み取りに失敗: %v", err)
		}

		page.id = model.SuggestionPageID(id)
		page.pageID = model.PageID(pageID)
		if pageRevisionID != nil {
			revisionID := model.PageRevisionID(*pageRevisionID)
			page.pageRevisionID = &revisionID
		}
		pages = append(pages, page)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("提案ページの読み取りに失敗: %v", err)
	}

	return pages
}

// suggestionCommentRow is one stored comment of a suggestion.
//
// [Ja] suggestionCommentRow は、編集提案に保存されたコメント 1 件。
type suggestionCommentRow struct {
	number    int32
	body      string
	creatorID model.SpaceMemberID
}

// listSuggestionComments reads back a suggestion's comments in the order the
// discussion shows them. The ordering falls back to the comment number so that
// two comments written in the same instant are still read in the order they
// were numbered.
//
// [Ja] listSuggestionComments は、編集提案のコメントを、議論が見せる順序で読み戻す。
// 並び順がコメント番号に落ちるようにしているのは、同じ時刻に書かれた 2 件も採番された
// 順に読めるようにするため。
func listSuggestionComments(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	suggestionID model.SuggestionID,
) []suggestionCommentRow {
	t.Helper()

	rows, err := tx.QueryContext(
		ctx,
		`SELECT number, body, created_space_member_id
         FROM suggestion_comments
         WHERE space_id = $1 AND suggestion_id = $2
         ORDER BY created_at ASC, number ASC`,
		string(spaceID), string(suggestionID),
	)
	if err != nil {
		t.Fatalf("編集提案のコメントの取得に失敗: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var comments []suggestionCommentRow
	for rows.Next() {
		var (
			comment   suggestionCommentRow
			creatorID string
		)
		if err := rows.Scan(&comment.number, &comment.body, &creatorID); err != nil {
			t.Fatalf("編集提案のコメントの読み取りに失敗: %v", err)
		}

		comment.creatorID = model.SpaceMemberID(creatorID)
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("編集提案のコメントの読み取りに失敗: %v", err)
	}

	return comments
}

// countSuggestionPageRevisions counts the revisions a proposed page has been
// through.
//
// [Ja] countSuggestionPageRevisions は、提案ページが辿ったリビジョンの件数を数える。
func countSuggestionPageRevisions(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	suggestionPageID model.SuggestionPageID,
) int {
	t.Helper()

	var count int
	err := tx.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM suggestion_page_revisions WHERE space_id = $1 AND suggestion_page_id = $2`,
		string(spaceID), string(suggestionPageID),
	).Scan(&count)
	if err != nil {
		t.Fatalf("提案ページのリビジョン数の取得に失敗: %v", err)
	}

	return count
}

// countPageRevisions counts the revisions a page has been through. A page that
// a suggestion was applied to keeps one for what was published and one for what
// the suggestion proposed.
//
// [Ja] countPageRevisions は、ページが辿ったリビジョンの件数を数える。編集提案が反映
// されたページは、公開された内容の分と編集提案が提案した内容の分を持つ。
func countPageRevisions(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	pageID model.PageID,
) int {
	t.Helper()

	var count int
	err := tx.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM page_revisions WHERE space_id = $1 AND page_id = $2`,
		string(spaceID), string(pageID),
	).Scan(&count)
	if err != nil {
		t.Fatalf("ページのリビジョン数の取得に失敗: %v", err)
	}

	return count
}

// readPageRevisionBody reads back the body a page held at one revision, which
// is what a proposal is compared against.
//
// [Ja] readPageRevisionBody は、あるリビジョン時点でページが保持していた本文を読み
// 戻す。提案が比較される相手がこの本文になる。
func readPageRevisionBody(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	pageRevisionID model.PageRevisionID,
) string {
	t.Helper()

	var body string
	err := tx.QueryRowContext(
		ctx,
		`SELECT body FROM page_revisions WHERE space_id = $1 AND id = $2`,
		string(spaceID), string(pageRevisionID),
	).Scan(&body)
	if err != nil {
		t.Fatalf("ページリビジョンの取得に失敗: %v", err)
	}

	return body
}

// pageUnderSuggestion is the page a suggestion proposes to change.
//
// [Ja] pageUnderSuggestion は、編集提案が変更を提案する先のページ。
type pageUnderSuggestion struct {
	title *string
	body  string
}

// readPageUnderSuggestion reads back the page a suggestion proposes to change.
//
// [Ja] readPageUnderSuggestion は、編集提案が変更を提案する先のページを読み戻す。
func readPageUnderSuggestion(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	pageID model.PageID,
) pageUnderSuggestion {
	t.Helper()

	var page pageUnderSuggestion
	err := tx.QueryRowContext(
		ctx,
		`SELECT title, body FROM pages WHERE space_id = $1 AND id = $2`,
		string(spaceID), string(pageID),
	).Scan(&page.title, &page.body)
	if err != nil {
		t.Fatalf("編集提案の対象ページの取得に失敗: %v", err)
	}

	return page
}

// readDraftSuggestionPageID reads back the proposed page a draft is linked to,
// which is what puts the editor into the mode of editing a suggestion.
//
// [Ja] readDraftSuggestionPageID は、下書きが紐づいている提案ページを読み戻す。
// 編集画面が編集提案の編集モードに入るのはこの紐づけによる。
func readDraftSuggestionPageID(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	spaceID model.SpaceID,
	draftPageID model.DraftPageID,
) *model.SuggestionPageID {
	t.Helper()

	var id *string
	err := tx.QueryRowContext(
		ctx,
		`SELECT suggestion_page_id FROM draft_pages WHERE space_id = $1 AND id = $2`,
		string(spaceID), string(draftPageID),
	).Scan(&id)
	if err != nil {
		t.Fatalf("下書きの取得に失敗: %v", err)
	}
	if id == nil {
		return nil
	}

	suggestionPageID := model.SuggestionPageID(*id)

	return &suggestionPageID
}
